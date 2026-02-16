import os
import sys
import numpy as np
import pandas as pd
import pdfplumber
from tabulate import tabulate

# ------------------------------------------------------------
# Configuration
# ------------------------------------------------------------
pdf_path = os.path.abspath(os.path.join(os.path.dirname(__file__), '../data/input/HKA-2026.pdf'))
output_path = os.path.abspath(os.path.join(os.path.dirname(__file__), '../data/output/HKA-2026_final.csv'))

EXPECTED_COLUMNS = [
    "BIL", "HARI KELEPASAN AM", "TARIKH", "HARI",
    "W.P.K.LUMPUR", "W.P.LABUAN", "W.P.PUTRAJAYA",
    "JOHOR", "KEDAH", "KELANTAN", "MELAKA", "N.SEMBILAN",
    "PAHANG", "PERAK", "PERLIS", "P.PINANG",
    "SABAH", "SARAWAK", "SELANGOR", "TERENGGANU"
]

STATE_COLS = EXPECTED_COLUMNS[4:]  # The 16 state/territory columns

print(f"Processing: {os.path.basename(pdf_path)}")


def extract_table_from_page(page):
    """Extract the first table from a pdfplumber page."""
    table_settings = {
        "vertical_strategy": "lines",
        "horizontal_strategy": "lines",
        "snap_tolerance": 5,
        "join_tolerance": 5,
        "edge_min_length": 10,
        "intersection_tolerance": 15,
    }
    tables = page.extract_tables(table_settings)
    if tables:
        return tables[0]
    return None


def find_header_row(raw_table):
    """Find the header row index by looking for 'BIL' keyword."""
    for i, row in enumerate(raw_table[:5]):
        row_text = " ".join(str(c).upper() for c in row if c)
        if "BIL" in row_text and ("HARI" in row_text or "KELEPASAN" in row_text):
            return i
    return 0


def normalize_cell(val):
    """Normalize a cell value: √ -> √, empty/None -> -, strip whitespace."""
    if val is None:
        return ""
    val = str(val).strip()
    if not val:
        return ""
    return val


def split_multiline_rows(df):
    """Split rows where TARIKH/HARI have multiple lines (true multi-day holidays).

    Only split when the date/day columns have multiple lines, indicating distinct
    holidays packed into one row. If only the holiday name wraps, join the lines
    instead of splitting.
    """
    expanded = []

    for _, row in df.iterrows():
        tarikh_val = str(row.get("TARIKH", ""))
        hari_val = str(row.get("HARI", ""))
        tarikh_lines = [l.strip() for l in tarikh_val.split("\n") if l.strip()]
        hari_lines = [l.strip() for l in hari_val.split("\n") if l.strip()]

        num_days = max(len(tarikh_lines), len(hari_lines))

        if num_days <= 1:
            # Single-day holiday: join any wrapped text in name column
            new_row = row.to_dict()
            name_val = str(new_row.get("HARI KELEPASAN AM", ""))
            name_lines = [l.strip() for l in name_val.split("\n") if l.strip()]
            new_row["HARI KELEPASAN AM"] = " ".join(name_lines)
            expanded.append(new_row)
            continue

        # Multi-day holiday: split into separate rows
        name_val = str(row.get("HARI KELEPASAN AM", ""))
        name_lines = [l.strip() for l in name_val.split("\n") if l.strip()]

        for line_idx in range(num_days):
            new_row = {}
            for col in df.columns:
                if col == "HARI KELEPASAN AM":
                    if line_idx < len(name_lines):
                        new_row[col] = name_lines[line_idx]
                    elif len(name_lines) == 1:
                        new_row[col] = name_lines[0]
                    else:
                        new_row[col] = ""
                elif col == "TARIKH":
                    new_row[col] = tarikh_lines[line_idx] if line_idx < len(tarikh_lines) else ""
                elif col == "HARI":
                    new_row[col] = hari_lines[line_idx] if line_idx < len(hari_lines) else ""
                elif col == "BIL":
                    new_row[col] = row[col] if line_idx == 0 else ""
                elif col == "JENIS":
                    new_row[col] = row[col]  # always carry forward
                elif col in STATE_COLS:
                    val = str(row[col])
                    lines = [l.strip() for l in val.split("\n") if l.strip()]
                    if line_idx < len(lines):
                        new_row[col] = lines[line_idx]
                    elif len(lines) == 1:
                        new_row[col] = lines[0]  # carry forward single value
                    else:
                        new_row[col] = ""
                else:
                    new_row[col] = row[col]
            expanded.append(new_row)

    return pd.DataFrame(expanded, columns=df.columns)


def clean_state_value(val):
    """Normalize state column values to consistent √ or - markers."""
    val = str(val).strip()
    if not val or val == "nan":
        return "-"
    # Check for check mark variants
    if "\u221a" in val or "V" in val.upper() or "v" in val:
        # Filter out cases where it's part of a word
        if val in ["\u221a", "V", "v", "√"]:
            return "\u221a"
        # Contains a check mark among other chars
        if "\u221a" in val:
            return "\u221a"
    if val == "-":
        return "-"
    # If it's something else unexpected, keep it
    return val


# ------------------------------------------------------------
# Extract tables from all pages
# ------------------------------------------------------------
print("Opening PDF...")

all_dfs = []

with pdfplumber.open(pdf_path) as pdf:
    total_pages = len(pdf.pages)
    print(f"PDF has {total_pages} pages")

    for page_idx, page in enumerate(pdf.pages):
        page_num = page_idx + 1
        print(f"\nProcessing page {page_num}...")

        raw_table = extract_table_from_page(page)
        if not raw_table:
            print(f"  No table found on page {page_num}, skipping")
            continue

        print(f"  Raw table: {len(raw_table)} rows x {len(raw_table[0])} cols")

        # Find header row
        header_idx = find_header_row(raw_table)
        print(f"  Header row index: {header_idx}")

        # Extract header and data rows
        header_raw = raw_table[header_idx]
        data_rows = raw_table[header_idx + 1:]

        # Map columns: use expected columns if the count matches
        n_cols = len(header_raw)
        if n_cols == len(EXPECTED_COLUMNS):
            columns = EXPECTED_COLUMNS
        else:
            # Try to build columns from raw header, cleaning up
            columns = []
            for i, h in enumerate(header_raw):
                h_clean = normalize_cell(h).replace("\n", " ").strip()
                if h_clean:
                    columns.append(h_clean)
                else:
                    columns.append(f"COL_{i}")
            print(f"  Warning: got {n_cols} columns, expected {len(EXPECTED_COLUMNS)}")

        # Build dataframe from data rows
        rows = []
        for raw_row in data_rows:
            row = [normalize_cell(c) for c in raw_row]
            # Pad or trim to match column count
            while len(row) < len(columns):
                row.append("")
            row = row[:len(columns)]
            rows.append(row)

        if not rows:
            print(f"  No data rows on page {page_num}")
            continue

        df = pd.DataFrame(rows, columns=columns)

        # Detect table type from the title line on the page
        page_text = page.extract_text() or ""
        page_lines = page_text.split("\n")
        table_type = "UNKNOWN"
        for line in page_lines:
            line_upper = line.strip().upper()
            if "JADUAL" in line_upper and "KELEPASAN" in line_upper:
                if "PERSEKUTUAN" in line_upper:
                    table_type = "PERSEKUTUAN"
                elif "NEGERI" in line_upper:
                    table_type = "NEGERI"
                break

        df["JENIS"] = table_type
        print(f"  Table type: {table_type}, data rows: {len(df)}")

        all_dfs.append(df)

if not all_dfs:
    print("No tables extracted from any page.")
    sys.exit(1)

# ------------------------------------------------------------
# Combine all pages
# ------------------------------------------------------------
combined = pd.concat(all_dfs, ignore_index=True)
print(f"\nCombined: {len(combined)} rows")

# Remove fully empty data rows
combined = combined.replace(["", "nan", "NaN", "None"], np.nan)
combined = combined.dropna(subset=["HARI KELEPASAN AM"], how="all")
combined = combined.fillna("")
combined = combined.reset_index(drop=True)

# ------------------------------------------------------------
# Split multi-line rows
# ------------------------------------------------------------
print("Splitting multi-line rows...")
combined = split_multiline_rows(combined)
combined = combined.reset_index(drop=True)

# Remove rows that are completely empty (no holiday name)
combined = combined[combined["HARI KELEPASAN AM"].str.strip() != ""]
combined = combined.reset_index(drop=True)

# ------------------------------------------------------------
# Clean state columns
# ------------------------------------------------------------
for col in STATE_COLS:
    if col in combined.columns:
        combined[col] = combined[col].apply(clean_state_value)

# ------------------------------------------------------------
# Re-number BIL within each JENIS group
# ------------------------------------------------------------
from collections import defaultdict
counter = defaultdict(int)
new_bils = []
for _, row in combined.iterrows():
    jenis = row.get("JENIS", "UNKNOWN")
    counter[jenis] += 1
    new_bils.append(counter[jenis])
combined["BIL"] = new_bils

# ------------------------------------------------------------
# Reorder columns: JENIS first, then the rest
# ------------------------------------------------------------
col_order = ["JENIS"] + EXPECTED_COLUMNS
col_order = [c for c in col_order if c in combined.columns]
combined = combined[col_order]

# ------------------------------------------------------------
# Output
# ------------------------------------------------------------
print(f"\nFinal table: {len(combined)} rows x {len(combined.columns)} cols")
print("\nPreview:")
print(tabulate(combined.head(50), headers="keys", tablefmt="pretty", maxcolwidths=30))

os.makedirs(os.path.dirname(output_path), exist_ok=True)
combined.to_csv(output_path, index=False)
print(f"\nSaved to: {output_path}")

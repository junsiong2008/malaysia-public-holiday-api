import os
import sys
import numpy as np
import pandas as pd
import camelot
import pdfplumber
import pytesseract
from pathlib import Path
from pdf2image import convert_from_path
from tabulate import tabulate

# ------------------------------------------------------------
# Configuration
# ------------------------------------------------------------
pdf_path = os.path.abspath(os.path.join(os.path.dirname(__file__), '../data/input/HKA-2026.pdf'))
output_path = os.path.abspath(os.path.join(os.path.dirname(__file__), '../data/output/HKA-2026_final.csv'))

print(f"🔄 Processing: {os.path.basename(pdf_path)}")

# ------------------------------------------------------------
# Determine total pages
# ------------------------------------------------------------
with pdfplumber.open(pdf_path) as pdf:
    total_pages = len(pdf.pages)

pages_range = f"1-{total_pages}"

# ------------------------------------------------------------
# Extract tables with Camelot (ALL pages)
# ------------------------------------------------------------
tables = camelot.read_pdf(
    pdf_path,
    flavor="lattice",
    line_scale=40,
    pages=pages_range
)

if not tables:
    print("❌ No tables detected by Camelot.")
    sys.exit(1)

print(f"📄 Detected {len(tables)} tables across {total_pages} pages")

final_dfs = []

# ------------------------------------------------------------
# Process each table independently (IMPORTANT)
# ------------------------------------------------------------
for table_idx, table in enumerate(tables):
    print(f"\n📘 Processing table {table_idx + 1} (Page {table.page})")

    df = table.df.copy()
    df = df.applymap(lambda x: x.strip() if isinstance(x, str) else x)

    # --------------------------------------------------------
    # 1. Detect header row
    # --------------------------------------------------------
    header_row_index = -1
    for i in range(min(5, len(df))):
        row_values = [str(v).upper() for v in df.iloc[i].values]
        if any("BIL" in v or "HARI" in v for v in row_values):
            header_row_index = i
            break

    if header_row_index == -1:
        header_row_index = 0  # fallback

    # --------------------------------------------------------
    # 2. Find missing header columns
    # --------------------------------------------------------
    missing_header_cols = [
        col_idx
        for col_idx, val in enumerate(df.iloc[header_row_index])
        if not str(val).strip()
    ]

    # --------------------------------------------------------
    # 3. OCR fallback (PAGE-AWARE)
    # --------------------------------------------------------
    if missing_header_cols:
        print(f"🔍 OCR fallback for {len(missing_header_cols)} header cells")

        page_num = int(table.page)

        try:
            # Convert correct page to image
            images = convert_from_path(
                pdf_path,
                first_page=page_num,
                last_page=page_num
            )
            page_image = images[0]
            img_w, img_h = page_image.size

            # PDF dimensions
            with pdfplumber.open(pdf_path) as pdf:
                pdf_page = pdf.pages[page_num - 1]
                pdf_w, pdf_h = pdf_page.width, pdf_page.height

            scale_x = img_w / pdf_w
            scale_y = img_h / pdf_h

            for col_idx in missing_header_cols:
                cell = table.cells[header_row_index][col_idx]

                img_x1 = cell.x1 * scale_x
                img_x2 = cell.x2 * scale_x
                img_y1 = (pdf_h - cell.y2) * scale_y
                img_y2 = (pdf_h - cell.y1) * scale_y

                padding = 2 * scale_x
                crop_box = (
                    img_x1 + padding,
                    img_y1 + padding,
                    img_x2 - padding,
                    img_y2 - padding
                )

                cell_img = page_image.crop(crop_box)

                # Rotate for vertical headers (very common)
                rotated_img = cell_img.rotate(-90, expand=True)

                ocr_text = pytesseract.image_to_string(
                    rotated_img,
                    config="--psm 7"
                ).strip()

                if ocr_text:
                    print(f"✅ Recovered header col {col_idx}: {ocr_text}")
                    df.iloc[header_row_index, col_idx] = ocr_text

        except Exception as e:
            print(f"⚠️ OCR failed on page {page_num}: {e}")

    # --------------------------------------------------------
    # 4. Promote header row & clean table
    # --------------------------------------------------------
    df.columns = df.iloc[header_row_index]
    df = df.iloc[header_row_index + 1 :]

    df = df.replace(["", "nan", "NaN", "NULL"], np.nan)
    df = df.dropna(how="all")
    df = df.fillna("")
    df = df.reset_index(drop=True)

    final_dfs.append(df)

# ------------------------------------------------------------
# 5. Merge ALL pages
# ------------------------------------------------------------
final_df = pd.concat(final_dfs, ignore_index=True)

# ------------------------------------------------------------
# Output
# ------------------------------------------------------------
print("\n✨ Final Table Preview:")
print(tabulate(final_df.head(50), headers="keys", tablefmt="pretty"))

final_df.to_csv(output_path, index=False)
print(f"\n💾 Saved to: {output_path}")
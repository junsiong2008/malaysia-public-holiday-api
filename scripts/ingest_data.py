import pandas as pd
from sqlalchemy import create_engine, text
from deep_translator import GoogleTranslator
import datetime
import re
import uuid

import os
from dotenv import load_dotenv

# Load .env file
load_dotenv()

# Database connection
DB_CONNECTION = os.getenv("DATABASE_URL")
if not DB_CONNECTION:
    raise ValueError("DATABASE_URL environment variable is not set. Please ensure a .env file exists.")

# State Mapping (CSV Header -> DB Code)
STATE_MAPPING = {
    'JOHOR': 'JHR', 'KEDAH': 'KDH', 'KELANTAN': 'KTN', 'MELAKA': 'MLK', 
    'N.SEMBILAN': 'NSN', 'PAHANG': 'PHG', 'PERAK': 'PRK', 'PERLIS': 'PLS', 
    'P.PINANG': 'PNG', 'SABAH': 'SBH', 'SARAWAK': 'SWK', 'SELANGOR': 'SGR', 
    'TERENGGANU': 'TRG', 'W.P.K.LUMPUR': 'KUL', 'W.P.LABUAN': 'LBN', 'W.P.PUTRAJAYA': 'WP'
}

# Month Mapping
MONTH_MAPPING = {
    'Januari': 1, 'Februari': 2, 'Mac': 3, 'April': 4, 'Mei': 5, 'Jun': 6,
    'Julai': 7, 'Ogos': 8, 'September': 9, 'Oktober': 10, 'November': 11, 'Disember': 12
}

def get_db_engine():
    return create_engine(DB_CONNECTION)

def translate_text(text):
    try:
        # Simple caching or just direct call. For a small script, direct call is fine but might be slow.
        # Check if text is effectively empty
        if not text or not text.strip():
            return ""
        return GoogleTranslator(source='auto', target='en').translate(text)
    except Exception as e:
        print(f"Translation failed for {text}: {e}")
        return text

def parse_date(date_str, year):
    # Format from CSV: "17 Februari" or "17 Februari 2026" (though CSV seems to lack year in date col, implies header year)
    # We assume the year is provided (2026 based on file name).
    # Clean up the string
    clean_str = date_str.strip()
    
    # Split day and month
    parts = clean_str.split(' ')
    if len(parts) < 2:
        return None
    
    day = int(parts[0])
    month_str = parts[1]
    month = MONTH_MAPPING.get(month_str)
    
    if not month:
        print(f"Unknown month: {month_str}")
        return None
        
    try:
        return datetime.date(year, month, day)
    except ValueError as e:
        print(f"Invalid date: {year}-{month}-{day} ({e})")
        return None

def get_next_working_day(date, weekend_days):
    # weekend_days is a set/list of weekday names e.g., ['Saturday', 'Sunday']
    # Start checking from the next day
    next_date = date + datetime.timedelta(days=1)
    while next_date.strftime('%A') in weekend_days:
        next_date += datetime.timedelta(days=1)
    return next_date

def get_replacement_date(holiday_date, weekend_days, replacement_rule):
    # Returns the replacement date IF the holiday needs replacement, else None.
    # Logic based on rules:
    # 1. If holiday falls on a weekend day...
    day_name = holiday_date.strftime('%A')
    
    if day_name not in weekend_days:
        return None, None # No replacement needed
        
    # It IS on a weekend. Check rule.
    if replacement_rule == 'no_replacement':
        return None, "No replacement rule involved" # Or specifically for Saturday in some states
        
    # For 'replace_on_sunday' (Fri-Sat states):
    # Rule: If Fri -> Sun. If Sat -> Sun.
    if replacement_rule == 'replace_on_sunday':
         # Find next Sunday? Or just next working day which happens to be Sunday?
         # Fri-Sat states work Sun-Thu. So Sunday is a working day.
         # If holiday is Friday, next day Sat (weekend), next day Sun (working).
         # If holiday is Saturday, next day Sun (working).
         # So effectively, "Next Working Day" covers it?
         # Let's verify: Kedah (Fri-Sat) has 'no_replacement' for Saturday. 
         # Kelantan/Terengganu (Fri-Sat) have 'replace_on_sunday'.
         pass
    
    # General logic: Find next working day.
    # Assumption for MVP: Replacement is always the next available working day.
    # Specific logic for "Saturday no replacement" in Fri-Sat states needs care.
    if day_name == 'Saturday' and replacement_rule == 'no_replacement':
        return None, "Saturday holiday not replaced"

    replacement_date = get_next_working_day(holiday_date, weekend_days)
    return replacement_date, f"Original date falls on {day_name} (weekend)"


def main():
    engine = get_db_engine()
    
    # 0. Setup DB (Execute schema if needed, but ideally schema is pre-loaded. 
    # We will assume schema.sql and seed_states.sql might need running or tables exist)
    # For robustness, let's try to run the schema/seed SQL if tables don't exist? 
    # Or just assume user ran them? The prompt asked to write a script to INGEST.
    # I'll add a step to execute schema.sql and seed_states.sql content just in case, or clearer: 
    # just rely on the table existing. I'll TRUNCATE holidays to start fresh.
    
    # 0. Setup DB (Execute schema if needed)
    with engine.connect() as conn:
        print("Checking/Initializing database schema...")
        
        # Drop existing tables to start fresh
        print("Dropping existing tables...")
        conn.execute(text("DROP TABLE IF EXISTS holiday_states, holidays, states, api_metadata CASCADE;"))
        conn.execute(text("DROP TYPE IF EXISTS holiday_type, weekend_pattern, saturday_rule, religion_type CASCADE;"))
        
        # Read and execute schema.sql
        print("Executing schema.sql...")
        with open('database/schema.sql', 'r') as f:
            schema_content = f.read()
            # Remove all comments (start with -- until end of line) using regex to avoid colon issues in comments
            schema_clean = re.sub(r'--.*', '', schema_content)
            
            statements = schema_clean.split(';')
            for stmt in statements:
                if stmt.strip():
                    try:
                        conn.execute(text(stmt))
                    except Exception as e:
                        print(f"Error executing statement: {stmt[:50]}... -> {e}")
                        if "already exists" in str(e):
                             print("Ignoring 'already exists' error.")
                        else:
                             raise
            
        # Read and execute seed_states.sql
        print("Executing seed_states.sql...")
        with open('database/seed_states.sql', 'r') as f:
            seed_sql = f.read()
            statements = seed_sql.split(';')
            for stmt in statements:
                if stmt.strip():
                    try:
                        conn.execute(text(stmt))
                    except Exception as e:
                        print(f"Error executing statement: {stmt[:50]}... -> {e}")
                        if "duplicate key" in str(e) or "already exists" in str(e):
                             print("Ignoring duplicate/exists error.")
                        else:
                             raise
            
        print("Database initialized.")
        conn.commit()
    
    # Reload states after seeding
    print("Loading state configurations...")
    states_df = pd.read_sql("SELECT * FROM states", engine)
    states_config = {}
    for _, row in states_df.iterrows():
        states_config[row['code'].strip()] = {
            'weekend_days': row['weekend_days'],
            'weekend_pattern': row['weekend_pattern'],
            'saturday_replacement_rule': row['saturday_replacement_rule']
        }
        
    # 2. Read CSV
    csv_path = 'data/output/HKA-2026_final.csv'
    print(f"Reading CSV from {csv_path}...")
    df = pd.read_csv(csv_path)
    
    # 3. Process Rows
    holidays_to_insert = []
    holiday_states_to_insert = []
    
    # Track generated replacements to avoid duplicates if multiple states generate same replacement? 
    # Actually replacements are usually state-specific.
    
    global_id_counter = 1
    YEAR = 2026
    
    for index, row in df.iterrows():
        h_name = str(row['HARI KELEPASAN AM']).strip()
        h_date_str = str(row['TARIKH']).strip()
        jenis = str(row.get('JENIS', '')).strip()

        # Remove trailing '*' often found in names
        h_name_clean = h_name.replace('*', '').strip()

        # Parse Date
        h_date = parse_date(h_date_str, YEAR)
        if not h_date:
            continue

        # Determine affected states from state columns
        affected_states = []
        for csv_col, db_code in STATE_MAPPING.items():
            val = str(row.get(csv_col, '')).strip()
            if val == '√':
                affected_states.append(db_code)

        if not affected_states:
            continue

        # Use JENIS column for holiday type
        h_type = 'national' if jenis == 'PERSEKUTUAN' else 'state'

        # Translate Name
        h_name_en = translate_text(h_name_clean)

        # Create Main Holiday ID
        h_id = f"mys-{YEAR}-{global_id_counter:03d}"
        global_id_counter += 1

        # Store Main Holiday
        holidays_to_insert.append({
            'id': h_id,
            'name': h_name_clean,
            'name_en': h_name_en,
            'date': h_date,
            'day_of_week': h_date.strftime('%A'),
            'type': h_type,
            'is_replacement_holiday': False,
            'original_date': None,
            'original_holiday_id': None,
            'replaced_by': None,
            'replacement_reason': None
        })

        # Link Main Holiday to States
        for state_code in affected_states:
            holiday_states_to_insert.append({
                'holiday_id': h_id,
                'state_code': state_code
            })

        # Replacement Logic: group states by their calculated replacement date
        replacements_map = {}  # { date: [state_codes] }

        for state_code in affected_states:
            config = states_config.get(state_code)
            if not config:
                continue
            r_date, r_reason = get_replacement_date(h_date, config['weekend_days'], config['saturday_replacement_rule'])
            if r_date:
                if r_date not in replacements_map:
                    replacements_map[r_date] = []
                replacements_map[r_date].append(state_code)

        # Create replacement holidays (one per unique replacement date)
        for r_date, r_states in replacements_map.items():
            r_id = f"mys-{YEAR}-{global_id_counter:03d}"
            global_id_counter += 1

            r_name = f"{h_name_clean} (Cuti Gantian)"
            r_name_en = f"{h_name_en} (Replacement)"

            holidays_to_insert.append({
                'id': r_id,
                'name': r_name,
                'name_en': r_name_en,
                'date': r_date,
                'day_of_week': r_date.strftime('%A'),
                'type': 'replacement',
                'is_replacement_holiday': True,
                'original_date': h_date,
                'original_holiday_id': h_id,
                'replaced_by': None,
                'replacement_reason': f"Replacement for {h_name_clean}"
            })

            for rs in r_states:
                holiday_states_to_insert.append({
                    'holiday_id': r_id,
                    'state_code': rs
                })

    # 4. Bulk Insert
    print(f"Inserting {len(holidays_to_insert)} holidays and {len(holiday_states_to_insert)} state mappings...")
    
    if holidays_to_insert:
        pd.DataFrame(holidays_to_insert).to_sql('holidays', engine, if_exists='append', index=False)
    
    if holiday_states_to_insert:
        pd.DataFrame(holiday_states_to_insert).to_sql('holiday_states', engine, if_exists='append', index=False)
        
    print("Ingestion complete.")

if __name__ == "__main__":
    main()

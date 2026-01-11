# Database Schema Design for Malaysian Public Holidays API

## 1. Overview
This schema is designed to support the `Malaysian Public Holidays API` (OpenAPI 3.0.0). It prioritizes:
- **Data Integrity**: Using Enums and Foreign Keys to ensure valid states and holiday types.
- **Query Performance**: Indexes on frequently filtered columns (`date`, `year`, `state`, `type`).
- **Flexibility**: Supporting complex replacement rules and ad-hoc holidays.

## 2. Table Structure

### `states`
Static configuration table for the 16 states/territories.
- **Primary Key**: `code` (ISO 3166-2, e.g., 'JHR')
- **Weekend Logic**: Stores `weekend_days` (Array) and `weekend_pattern` to support logic like "Friday is weekend in Kedah".
- **Replacement Rules**: `saturday_replacement_rule` explicitly defines behavior for Saturday holidays.

### `holidays`
The main table storing holiday events.
- **ID Strategy**: `id` is a string to support the `mys-YYYY-XXX` format specified in the API example.
- **Computed Columns**: `day_of_week` is automatically generated from `date` to ensure consistency.
- **Replacement Handling**:
    - `is_replacement_holiday`: Boolean flag.
    - `original_holiday_id`: Links a replacement holiday (e.g., Monday) back to the original holiday (Sunday).
    - `replaced_by`: Links the original holiday to its replacement.
- **Ad-hoc Support**: Fields for `gazette_reference` and `declared_date` support `Cuti Peristiwa`.

### `holiday_states`
Many-to-Many relationship table.
- Allows a holiday to be associated with multiple states (or 'ALL' logic which can be expanded to all state codes during insertion).
- Supports efficient joining to find "all holidays for state JHR".

### `api_metadata`
Key-value store for API meta information (version, last updated).

## 3. API Mapping Strategy

| API Endpoint | SQL Query Strategy |
|--------------|-------------------|
| `GET /holidays` | `SELECT * FROM holidays h JOIN holiday_states hs ON h.id = hs.holiday_id WHERE hs.state_code = ?` |
| `GET /states` | `SELECT * FROM states` |
| `GET /holidays/check` | Query `holidays` for specific date + `states` for weekend check. |
| `GET /holidays/working-days` | specific logic needed: Count days between range minus (count of weekends + count of holidays). |

## 4. Import Strategy for CSV
The source CSV (`HKA-2026_final.csv`) requires a transformation script (ETL) to be loaded into this schema:
1. **Parse Dates**: Convert Malaysian dates (e.g., "17 Februari") to ISO format (`2026-02-17`).
2. **Expand Merged Rows**: The CSV has merged cells for holidays spanning multiple days. These must be split into individual rows.
3. **Map States**: Convert column headers (`JOHOR`, `KEDAH`) to codes (`JHR`, `KDH`).
4. **Generate IDs**: Sequentially generate `mys-2026-001`, etc.
5. **Determine Type**: If a holiday is marked '√' for all states, `type = 'national'`. Else `type = 'state'`.
6. **Calculate Replacements**:
    - For each holiday, check the state's `weekend_pattern`.
    - If holiday falls on a weekend, create a new `replacement` entry for the next working day (Monday or Sunday) based on rules.
    - Link original and replacement via `original_holiday_id` / `replaced_by`.

## 5. Design Decision: Replacement Holidays (Stored vs. Calculated)

**Question:** Should replacement holidays be calculated on-the-fly during API requests or pre-calculated and stored in the database?

**Decision: STORE in Database**

### Rationale:

1.  **API Query Requirements**:
    - The API allows filtering by `type=replacement` and `include_replacements=true/false`.
    - It also supports filtering by `month`. A replacement holiday might fall in a different month than the original (e.g., a Sunday, Jan 31st holiday replaced on Monday, Feb 1st).
    - If calculated on-the-fly, a query for "February" would need to scan January holidays to check for spill-over replacements. Storing them allows simple, indexed queries: `WHERE date BETWEEN '2026-02-01' AND '2026-02-28'`.

2.  **Stable Identifiers**:
    - The API requires specific IDs for all holidays (e.g., `mys-2024-025`).
    - Storing the record guarantees that the ID is stable and permanent. On-the-fly generation risks ID checking consistency if the generation logic changes.

3.  **Performance & Complexity**:
    - Malaysia's replacement rules are complex (cascading replacements, different weekend definitions per state).
    - Calculating this for every request (especially for features like `working-days` calculation or full-year calendars) is computationally expensive and error-prone.
    - Read-heavy workload: Holidays are read frequently but change rarely. Pre-computation is the correct optimization.

4.  **Audit & Overrides**:
    - Sometimes the government declares a specific ad-hoc change to a standard replacement rule.
    - Storing the row allows us to manually update specific replacement records without changing the global code logic.


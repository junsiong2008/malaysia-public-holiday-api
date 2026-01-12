# Cuti - Malaysian Public Holiday API

Cuti is a specialized project designed to provide an accurate, queryable API for Malaysian public holidays. It handles the complexities of Malaysia's holiday system, including replacement holiday rules, state-specific holidays, and weekend variations across different states (e.g., Friday vs. Sunday weekends).

The system ingests data from official sources (PDFs), processes it into a structured format, stores it in a PostgreSQL database with pre-calculated replacement logic, and serves it via a high-performance Go API.

## Project Structure

The project is organized into distinct components handling data extraction, storage, and API serving.

```
Cuti/
├── api/          # Go implementation of the REST API
├── data/         # Data storage for ingestion pipeline
│   ├── input/    # Raw PDF sources (e.g., HKA files)
│   └── output/   # Processed CSVs ready for ingestion
├── database/     # SQL schemas and seed data
├── docs/         # Design documentation (Schema, API specs)
├── scripts/      # Python scripts for ETL (Extract, Transform, Load)
├── README.md     # Project documentation
└── ...
```

### Component Details

- **`api/`**: The backend service written in Go. It connects to the PostgreSQL database and exposes endpoints to retrieve holiday data. It uses `chi` for routing and `pgx` for database interactions.
- **`database/`**: Contains the SQL definition files. `schema.sql` defines the tables (`states`, `holidays`, `holiday_states`), and `seed_states.sql` populates static configuration for Malaysia's 16 states/territories.
- **`scripts/`**: Python scripts used to process data.
    - `extract_pdf.py`: Extracts tabular data from official PDF sources.
    - `ingest_data.py`: The main ETL script. It reads processed CSVs, parses dates, applies replacement rules (e.g., Sunday holiday replaced on Monday), and populates the database.
- **`docs/`**: Documentation including `SCHEMA_DESIGN.md` which explains the database modeling decisions.

## Key Features

- **Complex Replacement Logic**: Automatically handles replacement holidays ("Cuti Gantian") when public holidays fall on weekends. This logic respects state-specific weekend definitions (e.g., Johor's Friday weekend vs. Kuala Lumpur's Sunday weekend).
- **State-Specific Filtering**: Holidays are linked to specific states or flagged as national, allowing precise queries for any region in Malaysia.
- **Stable Identifiers**: Holidays are assigned stable IDs (e.g., `mys-2026-001`) ensuring consistency for API consumers.
- **Bilingual Support**: Stores holiday names in both Malay (original) and English (translated).

## Getting Started

### Prerequisites

- **Go**: Version 1.23 or higher.
- **PostgreSQL**: Local or remote instance.
- **Python**: Version 3.x (for data ingestion scripts).

### Environment Setup

The project uses `.env` files for configuration.

**1. Database Credentials**

Create a `.env` file in the root or `scripts/` directory for the ingestion script, and in `api/` for the API server.

**Example `.env`**:
```bash
DATABASE_URL=postgres://user:password@localhost:5432/cuti_db
PORT=8080  # Optional, for API
```

### Database Initialization

1. Ensure your PostgreSQL database is running.
2. The database schema and seeding are handled by the ingestion script, or you can run them manually:
   ```bash
   psql -d cuti_db -f database/schema.sql
   psql -d cuti_db -f database/seed_states.sql
   ```

### Data Pipeline (ETL)

To populate the database with fresh data:

1. Place your source PDF or CSV in `data/`.
2. Run the ingestion script (ensure you have necessary Python dependencies like `pandas`, `sqlalchemy`, `psycopg2-binary`, `python-dotenv`, `deep_translator`).

   ```bash
   # From the project root (ensure paths in script are valid or move to scripts dir)
   python scripts/ingest_data.py
   ```
   *Note: The script currently expects `schema.sql` to be accessible or may need path adjustments if run from the root. Verify file paths in `scripts/ingest_data.py` before running.*

### Running the API

1. Navigate to the `api` directory:
   ```bash
   cd api
   ```
2. Start the server:
   ```bash
   go run main.go
   ```
3. The server will start on port `8080` (or the port defined in `.env`).

## API Usage

**Base URL**: `http://localhost:8080`

### Endpoints

- **`GET /holidays`**: List all holidays.
  - Query Params: `state` (e.g., `JHR`), `year` (e.g., `2026`), `month`.
- **`GET /states`**: List all supported states and their configurations.

*For full API specification, refer to [`docs/api.md`](docs/api.md) or see `api/openapi.yaml` for the OpenAPI spec.*

## Source

[Kabinet.gov.my - HKA 2026](https://www.kabinet.gov.my/storage/2025/08/HKA-2026.pdf)



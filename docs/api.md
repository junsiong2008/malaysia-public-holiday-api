# Cuti API Documentation

This document provides a detailed reference for the Cuti API, which provides Malaysian public holiday data.

**Base URL**: `https://api.malaysia-holidays.gov.my/v1` (Production)
**Local URL**: `http://localhost:8080`

## Endpoints

### 1. Get All Holidays
Retrieve public holidays with optional filtering.

- **URL**: `/holidays`
- **Method**: `GET`
- **Query Parameters**:
    - `year` (int, optional): Year for holidays (e.g., 2024).
    - `state` (string, optional): State code (e.g., `JHR`, `KUL`).
    - `type` (string, optional): Filter by type (`national`, `state`, `optional`, `replacement`, `adhoc`).
    - `month` (int, optional): Filter by month (1-12).
    - `include_replacements` (bool, default: `true`): Include replacement holidays.

**Example Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "mys-2024-001",
      "name": "Hari Tahun Baru",
      "name_en": "New Year's Day",
      "date": "2024-01-01",
      "day_of_week": "Monday",
      "type": "national",
      "states": ["ALL"]
    }
  ],
  "meta": {
    "total_count": 1,
    "year": 2024
  }
}
```

### 2. Get Specific Holiday
Retrieve details for a single holiday by its ID.

- **URL**: `/holidays/{id}`
- **Method**: `GET`
- **Path Parameters**:
    - `id` (string): Unique holiday identifier (e.g., `mys-2024-001`).

### 3. Get Upcoming Holidays
Retrieve upcoming public holidays from the current date.

- **URL**: `/holidays/upcoming`
- **Method**: `GET`
- **Query Parameters**:
    - `state` (string, optional): State code filter.
    - `limit` (int, default: 10): Number of holidays to return.
    - `include_replacements` (bool, default: `true`): Include replacement holidays.

### 4. Get All States
Retrieve a list of all Malaysian states and their configurations.

- **URL**: `/states`
- **Method**: `GET`

**Example Response**:
```json
{
  "success": true,
  "data": [
    {
      "code": "JHR",
      "name": "Johor",
      "weekend_days": ["Friday", "Saturday"],
      "weekend_pattern": "fri-sat"
    }
  ]
}
```

### 5. Get State Holidays
Retrieve all holidays for a specific state, handling state-specific replacement rules.

- **URL**: `/states/{state_code}/holidays`
- **Method**: `GET`
- **Path Parameters**:
    - `state_code` (string): State code (e.g., `JHR`).
- **Query Parameters**:
    - `year` (int, optional): Filter by year.
    - `include_replacements` (bool, default: `true`): Include replacement holidays.

### 6. Get State Weekend Configuration
Retrieve the weekend definition for a specific state.

- **URL**: `/states/{state_code}/weekend`
- **Method**: `GET`
- **Path Parameters**:
    - `state_code` (string): State code.

### 7. Check Date
Check if a specific date is a holiday given a state configuration.

- **URL**: `/holidays/check`
- **Method**: `GET`
- **Query Parameters**:
    - `date` (string, required): Date to check (YYYY-MM-DD).
    - `state` (string, optional): State code.

**Example Response**:
```json
{
  "success": true,
  "data": {
    "is_holiday": true,
    "is_weekend": false,
    "date": "2024-08-31",
    "holidays": [...]
  }
}
```

### 8. Calculate Working Days
Calculate the number of working days between two dates, excluding weekends and holidays.

- **URL**: `/holidays/working-days`
- **Method**: `GET`
- **Query Parameters**:
    - `start_date` (string, required): Start date (YYYY-MM-DD).
    - `end_date` (string, required): End date (YYYY-MM-DD).
    - `state` (string, required): State code.

### 9. Get Metadata
Retrieve API version and data freshness information.

- **URL**: `/metadata`
- **Method**: `GET`

## State Codes
| Code | State | Weekend |
|------|-------|---------|
| JHR | Johor | Fri-Sat |
| KDH | Kedah | Fri-Sat |
| KTN | Kelantan | Fri-Sat |
| TRG | Terengganu | Fri-Sat |
| KUL | Kuala Lumpur | Sat-Sun |
| ... | (Others) | Sat-Sun |

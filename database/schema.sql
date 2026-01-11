-- Enable UUID extension for generating unique IDs if needed, though we might use formatted strings
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enums for constrained string fields
CREATE TYPE holiday_type AS ENUM ('national', 'state', 'optional', 'replacement', 'adhoc');
CREATE TYPE weekend_pattern AS ENUM ('fri-sat', 'sat-sun');
CREATE TYPE saturday_rule AS ENUM ('no_replacement', 'replace_on_sunday');
CREATE TYPE religion_type AS ENUM ('islam', 'buddhism', 'hinduism', 'christianity', 'taoism', 'secular');

-- States configuration table
-- Stores static configuration about states and their weekend rules
CREATE TABLE states (
    code VARCHAR(10) PRIMARY KEY, -- ISO 3166-2:MY code (e.g., JHR, KUL)
    name VARCHAR(100) NOT NULL,
    name_ms VARCHAR(100) NOT NULL,
    weekend_days VARCHAR(20)[] NOT NULL, -- Array of days, e.g., ['Saturday', 'Sunday']
    weekend_pattern weekend_pattern NOT NULL,
    saturday_replacement_rule saturday_rule NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Holidays table
-- Core table storing holiday events
CREATE TABLE holidays (
    id VARCHAR(50) PRIMARY KEY, -- Format: mys-YYYY-XXX, e.g., mys-2024-001
    name VARCHAR(255) NOT NULL,
    name_en VARCHAR(255), -- English name might be nullable initially
    date DATE NOT NULL,
    day_of_week VARCHAR(15), -- Populated by application
    type holiday_type NOT NULL,
    
    -- Replacement Logic
    is_replacement_holiday BOOLEAN DEFAULT FALSE,
    original_date DATE, -- If this is a replacement, what was the original date?
    original_holiday_id VARCHAR(50) REFERENCES holidays(id), -- Pointer to the parent holiday
    replaced_by VARCHAR(50), -- Pointer to the replacement holiday (filled in the original holiday record)
    replacement_reason TEXT,
    
    -- Additional Metadata
    description TEXT,
    religion religion_type,
    
    -- Ad-hoc Holiday fields
    gazette_reference VARCHAR(100),
    declared_date DATE,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT check_replacement_data CHECK (
        (is_replacement_holiday = FALSE) OR 
        (is_replacement_holiday = TRUE AND original_date IS NOT NULL)
    )
);

-- Self-reference FK for replaced_by needs to be added after table creation to avoid circular issues during creation if strictly ordered, 
-- but PostgreSQL handles self-referencing FKs in CREATE TABLE fine. 
-- However, adding it explicitly:
ALTER TABLE holidays ADD CONSTRAINT fk_replaced_by FOREIGN KEY (replaced_by) REFERENCES holidays(id);


-- Join table for Holidays <-> States
-- Handles the many-to-many relationship. 
-- 'national' holidays will have entries for ALL states (or we can use a special logic, but explicit is better for querying)
-- Ideally, for 'national', we might insert all state codes.
CREATE TABLE holiday_states (
    holiday_id VARCHAR(50) REFERENCES holidays(id) ON DELETE CASCADE,
    state_code CHAR(3) REFERENCES states(code) ON DELETE CASCADE,
    PRIMARY KEY (holiday_id, state_code)
);

-- Indexes for efficient querying based on API patterns
CREATE INDEX idx_holidays_date ON holidays(date);
CREATE INDEX idx_holidays_year ON holidays(EXTRACT(YEAR FROM date));
CREATE INDEX idx_holidays_type ON holidays(type);
CREATE INDEX idx_holidays_month ON holidays(EXTRACT(MONTH FROM date));

-- Metadata table for API versioning and status
CREATE TABLE api_metadata (
    key VARCHAR(50) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

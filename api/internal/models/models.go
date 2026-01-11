package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Custom Date type for YYYY-MM-DD JSON marshaling
type Date struct {
	time.Time
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.Format("2006-01-02"))
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		return nil
	}
	// Trim quotes
	if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

// Value implements the driver Valuer interface.
func (d Date) Value() (driver.Value, error) {
	return d.Time, nil
}

// Scan implements the Scanner interface.
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		d.Time = v
	case []byte:
		t, err := time.Parse("2006-01-02", string(v))
		if err != nil {
			return err
		}
		d.Time = t
	case string:
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return err
		}
		d.Time = t
	default:
		return fmt.Errorf("cannot scan type %T into Date", value)
	}
	return nil
}

type HolidayType string
type WeekendPattern string
type SaturdayRule string
type ReligionType string

type Holiday struct {
	ID                   string       `json:"id" db:"id"`
	Name                 string       `json:"name" db:"name"`
	NameEn               *string      `json:"name_en" db:"name_en"`
	Date                 Date         `json:"date" db:"date"`
	DayOfWeek            string       `json:"day_of_week" db:"day_of_week"`
	Type                 HolidayType  `json:"type" db:"type"`
	States               []string     `json:"states" db:"states"` // Populated from join
	IsReplacementHoliday bool         `json:"is_replacement_holiday" db:"is_replacement_holiday"`
	OriginalDate         *Date        `json:"original_date" db:"original_date"`
	OriginalHolidayID    *string      `json:"original_holiday_id" db:"original_holiday_id"`
	ReplacedBy           *string      `json:"replaced_by" db:"replaced_by"`
	ReplacementReason    *string      `json:"replacement_reason" db:"replacement_reason"`
	Description          *string      `json:"description" db:"description"`
	Religion             *ReligionType `json:"religion" db:"religion"`
	GazetteReference     *string      `json:"gazette_reference" db:"gazette_reference"`
	DeclaredDate         *Date        `json:"declared_date" db:"declared_date"`
}

type State struct {
	Code                    string         `json:"code" db:"code"`
	Name                    string         `json:"name" db:"name"`
	NameMs                  string         `json:"name_ms" db:"name_ms"`
	WeekendDays             []string       `json:"weekend_days" db:"weekend_days"`
	WeekendPattern          WeekendPattern `json:"weekend_pattern" db:"weekend_pattern"`
	SaturdayReplacementRule SaturdayRule   `json:"saturday_replacement_rule" db:"saturday_replacement_rule"`
}

type Meta struct {
	TotalCount   int       `json:"total_count"`
	Year         *int      `json:"year,omitempty"`
	LastUpdated  time.Time `json:"last_updated"`
	GeneratedAt  time.Time `json:"generated_at"`
	DataVersion  string    `json:"data_version"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

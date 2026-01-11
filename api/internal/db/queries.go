package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/junsiong2008/malaysia-public-holiday-api/api/internal/models"
)

// GetStates retrieves all states
func GetStates(ctx context.Context) ([]models.State, error) {
	query := `SELECT code, name, name_ms, weekend_days, weekend_pattern, saturday_replacement_rule FROM states ORDER BY name`
	rows, err := Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []models.State
	for rows.Next() {
		var s models.State
		if err := rows.Scan(&s.Code, &s.Name, &s.NameMs, &s.WeekendDays, &s.WeekendPattern, &s.SaturdayReplacementRule); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, nil
}

// GetState retrieves a single state
func GetState(ctx context.Context, code string) (*models.State, error) {
	query := `SELECT code, name, name_ms, weekend_days, weekend_pattern, saturday_replacement_rule FROM states WHERE code = $1`
	var s models.State
	err := Pool.QueryRow(ctx, query, code).Scan(&s.Code, &s.Name, &s.NameMs, &s.WeekendDays, &s.WeekendPattern, &s.SaturdayReplacementRule)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// GetHolidays retrieves holidays with optional filters
func GetHolidays(ctx context.Context, year int, stateCode string, month int, includeReplacements bool) ([]models.Holiday, error) {
	// Base query
	sql := `
		SELECT h.id, h.name, h.name_en, h.date, h.day_of_week, h.type, h.is_replacement_holiday, 
		       h.original_date, h.original_holiday_id, h.replaced_by, h.replacement_reason, 
			   h.description, h.religion, h.gazette_reference, h.declared_date
		FROM holidays h
	`
	
	// Joins if specific state requested, or to get states later?
	// The problem is fetching 'states' array for each holiday.
	// We can do a join, but simpler for MVP: 
	// 1. Fetch holidays matching criteria.
	// 2. Fetch state mappings for those holidays (can be optimized with array_agg).
	
	// Let's use array_agg to get states in one go
	sql = `
		SELECT h.id, h.name, h.name_en, h.date, h.day_of_week, h.type, h.is_replacement_holiday, 
		       h.original_date, h.original_holiday_id, h.replaced_by, h.replacement_reason, 
			   h.description, h.religion, h.gazette_reference, h.declared_date,
			   array_agg(hs.state_code) as states
		FROM holidays h
		JOIN holiday_states hs ON h.id = hs.holiday_id
	`
	
	whereClauses := []string{}
	args := []interface{}{}
	argId := 1

	if year > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("EXTRACT(YEAR FROM h.date) = $%d", argId))
		args = append(args, year)
		argId++
	}

	if month > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("EXTRACT(MONTH FROM h.date) = $%d", argId))
		args = append(args, month)
		argId++
	}

	if !includeReplacements {
		whereClauses = append(whereClauses, "h.is_replacement_holiday = false")
	}

	// State filter logic is tricky. 
	// If stateCode is provided, we only want holidays APPLICABLE to that state.
	// The JOIN implies we initially get rows for each state-holiday pair.
	// But `array_agg` collapses them.
	// If we filter `WHERE hs.state_code = ?`, we lose other states in the agg?
	// It's acceptable for the API response `states: ["JHR"]` if filtered by JHR?
	// Usually API users expect to see ALL applicable states even if filtered by one?
	// "Get holidays FOR Johor".
	// If I filter `hs.state_code = 'JHR'`, `array_agg` will only contain `JHR`.
	// That's typically fine for "Holidays for JHR".
	
	if stateCode != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("hs.state_code = $%d", argId))
		args = append(args, stateCode)
		argId++
	}

	if len(whereClauses) > 0 {
		sql += " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			sql += " AND " + whereClauses[i]
		}
	}

	sql += " GROUP BY h.id ORDER BY h.date"

	rows, err := Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []models.Holiday
	for rows.Next() {
		var h models.Holiday
		var states []string
		err := rows.Scan(
			&h.ID, &h.Name, &h.NameEn, &h.Date, &h.DayOfWeek, &h.Type, &h.IsReplacementHoliday,
			&h.OriginalDate, &h.OriginalHolidayID, &h.ReplacedBy, &h.ReplacementReason,
			&h.Description, &h.Religion, &h.GazetteReference, &h.DeclaredDate,
			&states,
		)
		if err != nil {
			return nil, err
		}
		h.States = states
		holidays = append(holidays, h)
	}
	return holidays, nil
}

// GetHolidayByID retrieves a single holiday
func GetHolidayByID(ctx context.Context, id string) (*models.Holiday, error) {
	sql := `
		SELECT h.id, h.name, h.name_en, h.date, h.day_of_week, h.type, h.is_replacement_holiday, 
		       h.original_date, h.original_holiday_id, h.replaced_by, h.replacement_reason, 
			   h.description, h.religion, h.gazette_reference, h.declared_date,
			   array_agg(hs.state_code) as states
		FROM holidays h
		JOIN holiday_states hs ON h.id = hs.holiday_id
		WHERE h.id = $1
		GROUP BY h.id
	`
	var h models.Holiday
	var states []string
	err := Pool.QueryRow(ctx, sql, id).Scan(
		&h.ID, &h.Name, &h.NameEn, &h.Date, &h.DayOfWeek, &h.Type, &h.IsReplacementHoliday,
		&h.OriginalDate, &h.OriginalHolidayID, &h.ReplacedBy, &h.ReplacementReason,
		&h.Description, &h.Religion, &h.GazetteReference, &h.DeclaredDate,
		&states,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	h.States = states
	return &h, nil
}

// GetHolidaysForDate checks if a date is a holiday for a specific state (or all)
func GetHolidaysForDate(ctx context.Context, date time.Time, stateCode string) ([]models.Holiday, error) {
	sql := `
		SELECT h.id, h.name, h.name_en, h.date, h.day_of_week, h.type, h.is_replacement_holiday, 
		       h.original_date, h.original_holiday_id, h.replaced_by, h.replacement_reason, 
			   h.description, h.religion, h.gazette_reference, h.declared_date,
			   array_agg(hs.state_code) as states
		FROM holidays h
		JOIN holiday_states hs ON h.id = hs.holiday_id
		WHERE h.date = $1
	`
	args := []interface{}{date}
	argId := 2

	if stateCode != "" {
		sql += fmt.Sprintf(" AND hs.state_code = $%d", argId)
		args = append(args, stateCode)
	}

	sql += " GROUP BY h.id"

	rows, err := Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []models.Holiday
	for rows.Next() {
		var h models.Holiday
		var states []string
		err := rows.Scan(
			&h.ID, &h.Name, &h.NameEn, &h.Date, &h.DayOfWeek, &h.Type, &h.IsReplacementHoliday,
			&h.OriginalDate, &h.OriginalHolidayID, &h.ReplacedBy, &h.ReplacementReason,
			&h.Description, &h.Religion, &h.GazetteReference, &h.DeclaredDate,
			&states,
		)
		if err != nil {
			return nil, err
		}
		h.States = states
		holidays = append(holidays, h)
	}
	return holidays, nil
}

// GetHolidaysInRange retrieves holidays within a date range for a specific state
func GetHolidaysInRange(ctx context.Context, start, end time.Time, stateCode string) ([]models.Holiday, error) {
	// We need to filter by state, but since we also want the `states` array in the struct,
	// we have to check if the holiday IS associated with validity for that state.
	// If stateCode is provided, we filter holidays that have an entry in holiday_states for that state.
	
	sql := `
		SELECT h.id, h.name, h.name_en, h.date, h.day_of_week, h.type, h.is_replacement_holiday, 
		       h.original_date, h.original_holiday_id, h.replaced_by, h.replacement_reason, 
			   h.description, h.religion, h.gazette_reference, h.declared_date,
			   array_agg(hs.state_code) as states
		FROM holidays h
		JOIN holiday_states hs ON h.id = hs.holiday_id
		WHERE h.date >= $1 AND h.date <= $2
	`
	args := []interface{}{start, end}
	argId := 3

	// If filtering by state, we need to ensure at least one of the holiday_states matches.
	// However, the JOIN + Group By logic above aggregates ALL states for the holiday.
	// To filter, we can use HAVING or a subquery/pre-filter.
	// Simpler: Just filter in WHERE, but that might mess up aggregation if we filter the joined table?
	// If we do `AND hs.state_code = 'JHR'`, the aggregation `array_agg(hs.state_code)` will ONLY contain JHR.
	// Is that desired? For "Working days calculation", yes, we only care that it IS a holiday for JHR.
	// The `States` field in the struct might be misleading if it only shows JHR, but for internal calculation it's fine.
	// For API response to `GetHolidays`, usually we want all states.
	// But `GetHolidaysInRange` is primarily for calculation or list.
	// If we want the full list of states for the holiday, but only return holidays relevant to JHR:
	// Use EXISTS.
	
	if stateCode != "" {
		sql += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM holiday_states hs2 WHERE hs2.holiday_id = h.id AND hs2.state_code = $%d)", argId)
		args = append(args, stateCode)
	}

	sql += " GROUP BY h.id ORDER BY h.date"

	rows, err := Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []models.Holiday
	for rows.Next() {
		var h models.Holiday
		var states []string
		err := rows.Scan(
			&h.ID, &h.Name, &h.NameEn, &h.Date, &h.DayOfWeek, &h.Type, &h.IsReplacementHoliday,
			&h.OriginalDate, &h.OriginalHolidayID, &h.ReplacedBy, &h.ReplacementReason,
			&h.Description, &h.Religion, &h.GazetteReference, &h.DeclaredDate,
			&states,
		)
		if err != nil {
			return nil, err
		}
		h.States = states
		holidays = append(holidays, h)
	}
	return holidays, nil
}

// GetUpcomingHolidays retrieves upcoming holidays
func GetUpcomingHolidays(ctx context.Context, stateCode string, limit int) ([]models.Holiday, error) {
	sql := `
		SELECT h.id, h.name, h.name_en, h.date, h.day_of_week, h.type, h.is_replacement_holiday, 
		       h.original_date, h.original_holiday_id, h.replaced_by, h.replacement_reason, 
			   h.description, h.religion, h.gazette_reference, h.declared_date,
			   array_agg(hs.state_code) as states
		FROM holidays h
		JOIN holiday_states hs ON h.id = hs.holiday_id
		WHERE h.date >= CURRENT_DATE
	`
	args := []interface{}{}
	argId := 1

	if stateCode != "" {
		sql += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM holiday_states hs2 WHERE hs2.holiday_id = h.id AND hs2.state_code = $%d)", argId)
		args = append(args, stateCode)
		argId++
	}

	sql += " GROUP BY h.id ORDER BY h.date ASC"
	
	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT $%d", argId)
		args = append(args, limit)
	}

	rows, err := Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []models.Holiday
	for rows.Next() {
		var h models.Holiday
		var states []string
		err := rows.Scan(
			&h.ID, &h.Name, &h.NameEn, &h.Date, &h.DayOfWeek, &h.Type, &h.IsReplacementHoliday,
			&h.OriginalDate, &h.OriginalHolidayID, &h.ReplacedBy, &h.ReplacementReason,
			&h.Description, &h.Religion, &h.GazetteReference, &h.DeclaredDate,
			&states,
		)
		if err != nil {
			return nil, err
		}
		h.States = states
		holidays = append(holidays, h)
	}
	return holidays, nil
}

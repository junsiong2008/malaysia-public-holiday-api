package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/junsiong2008/malaysia-public-holiday-api/api/internal/db"
	"github.com/junsiong2008/malaysia-public-holiday-api/api/internal/models"
	"github.com/junsiong2008/malaysia-public-holiday-api/api/internal/utils"
)

// GetHolidays handles GET /holidays
func GetHolidays(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	stateCode := r.URL.Query().Get("state")
	monthStr := r.URL.Query().Get("month")
	includeReplacementsStr := r.URL.Query().Get("include_replacements")

	var year, month int
	var err error

	if yearStr != "" {
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid year")
			return
		}
	} else {
		// Default to current year if not specified? Spec says not required.
		// If not provided, query returns all years? Or maybe we should hint?
		// Queries.go handles 0 correctly (skips filter).
	}

	if monthStr != "" {
		month, err = strconv.Atoi(monthStr)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid month")
			return
		}
	}

	includeReplacements := true
	if includeReplacementsStr == "false" {
		includeReplacements = false
	}

	holidays, err := db.GetHolidays(r.Context(), year, stateCode, month, includeReplacements)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch holidays")
		return
	}
	
	if holidays == nil {
	    holidays = []models.Holiday{}
	}

	meta := models.Meta{
		TotalCount:  len(holidays),
		GeneratedAt: time.Now(),
		DataVersion: "2024.1.0", // TODO: Fetch from DB metadata
	}
	if year > 0 {
		meta.Year = &year
	}

	utils.RespondWithMeta(w, http.StatusOK, holidays, meta)
}

// GetHolidayByID handles GET /holidays/{id}
func GetHolidayByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	holiday, err := db.GetHolidayByID(r.Context(), id)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch holiday")
		return
	}
	if holiday == nil {
		utils.RespondError(w, http.StatusNotFound, "NOT_FOUND", "Holiday not found")
		return
	}

	utils.RespondJSON(w, http.StatusOK, holiday)
}

// GetUpcomingHolidays handles GET /holidays/upcoming
func GetUpcomingHolidays(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	stateCode := r.URL.Query().Get("state")

	limit := 10
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	holidays, err := db.GetUpcomingHolidays(r.Context(), stateCode, limit)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch upcoming holidays")
		return
	}
	
	if holidays == nil {
	    holidays = []models.Holiday{}
	}

	utils.RespondJSON(w, http.StatusOK, holidays)
}

// GetStates handles GET /states
func GetStates(w http.ResponseWriter, r *http.Request) {
	states, err := db.GetStates(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch states")
		return
	}
	
	if states == nil {
	    states = []models.State{}
	}

	utils.RespondJSON(w, http.StatusOK, states)
}

// GetStateHolidays handles GET /states/{state_code}/holidays
func GetStateHolidays(w http.ResponseWriter, r *http.Request) {
	stateCode := chi.URLParam(r, "state_code")
	yearStr := r.URL.Query().Get("year")
	
	var year int
	var err error
	if yearStr != "" {
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid year")
			return
		}
	}

	// Reuse GetHolidays logic but enforce state
	holidays, err := db.GetHolidays(r.Context(), year, stateCode, 0, true)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch holidays")
		return
	}
	
	if holidays == nil {
	    holidays = []models.Holiday{}
	}
	
	// We also need state info for metadata
	state, err := db.GetState(r.Context(), stateCode)
	if err != nil {
	    // ignore error or return
	}

	stateName := "Unknown"
	weekendPattern := ""
	if state != nil {
	    stateName = state.Name
	    weekendPattern = string(state.WeekendPattern)
	}
	
	// Custom meta for this endpoint as per spec:
	// meta: { state_code, state_name, weekend_pattern, total_count, last_updated }
	// We need a custom response struct or just use map/anonymous struct for Data?
	// The helpers assume standard Meta.
	// Let's manually construct this one.
	
	response := struct {
		Success bool        `json:"success"`
		Data    interface{} `json:"data"`
		Meta    interface{} `json:"meta"`
	}{
		Success: true,
		Data:    holidays,
		Meta: struct {
			StateCode      string `json:"state_code"`
			StateName      string `json:"state_name"`
			WeekendPattern string `json:"weekend_pattern"`
			TotalCount     int    `json:"total_count"`
			LastUpdated    string `json:"last_updated"`
		}{
			StateCode:      stateCode,
			StateName:      stateName,
			WeekendPattern: weekendPattern,
			TotalCount:     len(holidays),
			LastUpdated:    time.Now().Format(time.RFC3339),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStateWeekend handles GET /states/{state_code}/weekend
func GetStateWeekend(w http.ResponseWriter, r *http.Request) {
	stateCode := chi.URLParam(r, "state_code")
	
	state, err := db.GetState(r.Context(), stateCode)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "DATABASE ERROR")
		return
	}
	if state == nil {
		utils.RespondError(w, http.StatusNotFound, "NOT_FOUND", "State not found")
		return
	}
	
	response := struct {
	    StateCode string `json:"state_code"`
	    StateName string `json:"state_name"`
	    WeekendDays []string `json:"weekend_days"`
	    WeekendPattern string `json:"weekend_pattern"`
	}{
	    StateCode: state.Code,
	    StateName: state.Name,
	    WeekendDays: state.WeekendDays,
	    WeekendPattern: string(state.WeekendPattern),
	}
	
	utils.RespondJSON(w, http.StatusOK, response)
}

// CheckHoliday handles GET /holidays/check
func CheckHoliday(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	stateCode := r.URL.Query().Get("state")

	if dateStr == "" {
		utils.RespondError(w, http.StatusBadRequest, "MISSING_PARAMETER", "date parameter is required")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid date format (YYYY-MM-DD)")
		return
	}

	var state *models.State
	if stateCode != "" {
		state, err = db.GetState(r.Context(), stateCode)
		if err != nil || state == nil {
			utils.RespondError(w, http.StatusNotFound, "NOT_FOUND", "State not found")
			return
		}
	}

	// Get holidays on that date
	holidays, err := db.GetHolidaysForDate(r.Context(), date, stateCode)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check holidays")
		return
	}

	isHoliday := len(holidays) > 0
	
	isWeekend := false
	weekendPattern := ""
	weekendDays := []string{}

	dayOfWeek := date.Format("Monday")

	if state != nil {
		weekendPattern = string(state.WeekendPattern)
		weekendDays = state.WeekendDays
		for _, day := range state.WeekendDays {
			if day == dayOfWeek {
				isWeekend = true
				break
			}
		}
	} else {
	    // If no state provided, weekend info is ambiguous.
	    // Check if it's weekend in common states? Or just return false/null?
	    // Spec says "Check if a given date is a public holiday for a specific state, considering weekend configuration".
	    // State param is 'required: false' in spec, but logic suggests it's needed for weekend info.
	}
	
	data := struct {
		IsHoliday   bool             `json:"is_holiday"`
		IsWeekend   bool             `json:"is_weekend"`
		Date        string           `json:"date"`
		DayOfWeek   string           `json:"day_of_week"`
		Holidays    []models.Holiday `json:"holidays"`
		WeekendInfo *struct {
			WeekendPattern string   `json:"weekend_pattern"`
			WeekendDays    []string `json:"weekend_days"`
		} `json:"weekend_info,omitempty"`
	}{
		IsHoliday: isHoliday,
		IsWeekend: isWeekend,
		Date:      dateStr,
		DayOfWeek: dayOfWeek,
		Holidays:  holidays,
	}

	if state != nil {
		data.WeekendInfo = &struct {
			WeekendPattern string   `json:"weekend_pattern"`
			WeekendDays    []string `json:"weekend_days"`
		}{
			WeekendPattern: weekendPattern,
			WeekendDays:    weekendDays,
		}
	}

	utils.RespondJSON(w, http.StatusOK, data)
}

// CalculateWorkingDays handles GET /holidays/working-days
func CalculateWorkingDays(w http.ResponseWriter, r *http.Request) {
    startDateStr := r.URL.Query().Get("start_date")
    endDateStr := r.URL.Query().Get("end_date")
    stateCode := r.URL.Query().Get("state")
    
    if startDateStr == "" || endDateStr == "" || stateCode == "" {
        utils.RespondError(w, http.StatusBadRequest, "MISSING_PARAMETER", "start_date, end_date, and state are required")
        return
    }
    
    start, err := time.Parse("2006-01-02", startDateStr)
    if err != nil {
        utils.RespondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid start_date")
        return
    }
    
    end, err := time.Parse("2006-01-02", endDateStr)
    if err != nil {
        utils.RespondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid end_date")
        return
    }
    
    if end.Before(start) {
         utils.RespondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "end_date must be after start_date")
         return
    }
    
    state, err := db.GetState(r.Context(), stateCode)
    if err != nil || state == nil {
        utils.RespondError(w, http.StatusNotFound, "NOT_FOUND", "State not found")
        return
    }
    
    holidays, err := db.GetHolidaysInRange(r.Context(), start, end, stateCode)
    if err != nil {
        utils.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch holidays")
        return
    }
    
    // Map holidays by date string for easy lookup
    holidayMap := make(map[string]bool)
    for _, h := range holidays {
        holidayMap[h.Date.Time.Format("2006-01-02")] = true
    }
    
    weekendSet := make(map[string]bool)
    for _, d := range state.WeekendDays {
        weekendSet[d] = true
    }
    
    totalDays := 0
    workingDays := 0
    weekendDaysCount := 0
    holidayDaysCount := 0
    
    curr := start
    // Spec says "Calculate ... between two dates". Inclusive? 
    // Usually working days calculation is inclusive of start and end.
    // Let's assume inclusive.
    
    for !curr.After(end) {
        totalDays++
        
        dayName := curr.Format("Monday")
        dateStr := curr.Format("2006-01-02")
        
        isWeekend := weekendSet[dayName]
        isHoliday := holidayMap[dateStr]
        
        if isWeekend {
            weekendDaysCount++
        }
        
        if isHoliday {
            // Note: If a holiday falls on a weekend, is it counted as holiday day or weekend day?
            // Usually we want to know how many days are OFF.
            // "holiday_days" -> Working days lost due to holiday?
            // "weekend_days" -> Weekend days.
            // If overlap?
            // Let's count strictly:
            // If it is a holiday, we increment holidayDaysCount.
            // If it is a weekend, we increment weekendDaysCount.
            // If both, we increment both?
            // BUT workingDays = Total - (Weekend + Holiday - Overlap)?
            // Or prioritize?
            // Simpler: 
            // is_working_day = !isWeekend && !isHoliday
            holidayDaysCount++
        }
        
        if !isWeekend && !isHoliday {
            workingDays++
        }
        
        curr = curr.AddDate(0, 0, 1)
    }
    
    data := struct {
        StartDate string `json:"start_date"`
        EndDate string `json:"end_date"`
        StateCode string `json:"state_code"`
        TotalDays int `json:"total_days"`
        WorkingDays int `json:"working_days"`
        WeekendDays int `json:"weekend_days"`
        HolidayDays int `json:"holiday_days"`
        HolidaysInRange []models.Holiday `json:"holidays_in_range"`
    }{
        StartDate: startDateStr,
        EndDate: endDateStr,
        StateCode: stateCode,
        TotalDays: totalDays,
        WorkingDays: workingDays,
        WeekendDays: weekendDaysCount,
        HolidayDays: holidayDaysCount,
        HolidaysInRange: holidays,
    }
    
    utils.RespondJSON(w, http.StatusOK, data)
}

// GetMetadata handles GET /metadata
func GetMetadata(w http.ResponseWriter, r *http.Request) {
    // Basic metadata
    data := struct {
        DataVersion string `json:"data_version"`
        LastUpdated string `json:"last_updated"`
        LastGazetteCheck string `json:"last_gazette_check"`
        AdhocHolidaysCount int `json:"adhoc_holidays_count"`
        CoverageYears []int `json:"coverage_years"`
    }{
        DataVersion: "2024.1.0",
        LastUpdated: time.Now().Format(time.RFC3339),
        LastGazetteCheck: time.Now().Format(time.RFC3339),
        AdhocHolidaysCount: 0,
        CoverageYears: []int{2023, 2024, 2025, 2026},
    }
    
    utils.RespondJSON(w, http.StatusOK, data)
}

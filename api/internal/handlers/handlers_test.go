package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/junsiong2008/malaysia-public-holiday-api/api/internal/db"
	"github.com/junsiong2008/malaysia-public-holiday-api/api/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Try to load .env from parent directory (api/.env) if running from internal/handlers
	// Or assume env vars are set.
	// For local test run, we might need to load specific env.
	_ = godotenv.Load("../../.env")

	if err := db.Connect(); err != nil {
		// If DB connection fails, we might mock or skip. 
		// But for this verification request, let's assume valid DB conn is required.
		panic(err)
	}
	defer db.Close()

	os.Exit(m.Run())
}

func TestGetHolidays(t *testing.T) {
	req := httptest.NewRequest("GET", "/holidays?year=2026", nil)
	w := httptest.NewRecorder()

	// Need to context with chi URLParams if used?
	// GetHolidays uses Query params, so raw request is fine.
	
	GetHolidays(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp models.APIResponse
	err := json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.NoError(t, err)
	assert.True(t, apiResp.Success)
	
	// We assume data is returned as []interface{} or specific type when unmarshaled to interface{}
	// To check details, we can unmarshal strictly or inspect map.
    // Let's unmarshal Data to []models.Holiday
    
    // Since APIResponse.Data is interface{}, we need to marshal/unmarshal again or cast
    dataBytes, _ := json.Marshal(apiResp.Data)
    var holidays []models.Holiday
    json.Unmarshal(dataBytes, &holidays)
    
    // We expect some holidays in 2026
    assert.NotEmpty(t, holidays)
    assert.Equal(t, "Tahun Baharu 2026", holidays[0].Name)
}

func TestGetStates(t *testing.T) {
	req := httptest.NewRequest("GET", "/states", nil)
	w := httptest.NewRecorder()

	GetStates(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var apiResp models.APIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.True(t, apiResp.Success)
}

func TestCheckHoliday(t *testing.T) {
    // Test a known holiday: 2026-08-31 (National Day)
	req := httptest.NewRequest("GET", "/holidays/check?date=2026-08-31&state=KUL", nil)
	w := httptest.NewRecorder()

	CheckHoliday(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var res struct {
	    Success bool `json:"success"`
	    Data struct {
	        IsHoliday bool `json:"is_holiday"`
	        IsWeekend bool `json:"is_weekend"`
	    } `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	
	assert.True(t, res.Success)
	assert.True(t, res.Data.IsHoliday)
}

func TestCalculateWorkingDays(t *testing.T) {
    // 1st Jan 2026 (Thursday, Holiday) to 4th Jan 2026 (Sunday)
    // KUL weekend: Sat, Sun.
    // 1st (Thu) - Holiday
    // 2nd (Fri) - Work
    // 3rd (Sat) - Weekend
    // 4th (Sun) - Weekend
    // Total days: 4. Working days: 1.
    
    req := httptest.NewRequest("GET", "/holidays/working-days?start_date=2026-01-01&end_date=2026-01-04&state=KUL", nil)
    w := httptest.NewRecorder()
    
    CalculateWorkingDays(w, req)
    
    resp := w.Result()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    var res struct {
        Success bool `json:"success"`
        Data struct {
            TotalDays int `json:"total_days"`
            WorkingDays int `json:"working_days"`
            HolidayDays int `json:"holiday_days"`
            WeekendDays int `json:"weekend_days"`
        } `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&res)
    
    assert.True(t, res.Success)
    assert.Equal(t, 4, res.Data.TotalDays)
    assert.Equal(t, 1, res.Data.WorkingDays)
    assert.Equal(t, 1, res.Data.HolidayDays)
    assert.Equal(t, 2, res.Data.WeekendDays)
}

func TestGetStateHolidays(t *testing.T) {
    // Setup Chi context for URL params
    r := httptest.NewRequest("GET", "/states/JHR/holidays", nil)
    w := httptest.NewRecorder()

    // Mock chi URL param
    rctx := chi.NewRouteContext()
    rctx.URLParams.Add("state_code", "JHR")
    r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

    GetStateHolidays(w, r)
    
    resp := w.Result()
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}

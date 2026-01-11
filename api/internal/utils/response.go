package utils

import (
	"encoding/json"
	"net/http"

	"github.com/junsiong2008/malaysia-public-holiday-api/api/internal/models"
)

// RespondJSON sends a JSON response
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	response := models.APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// RespondWithMeta sends a JSON response with metadata
func RespondWithMeta(w http.ResponseWriter, status int, data interface{}, meta models.Meta) {
    response := models.APIResponse{
        Success: status >= 200 && status < 300,
        Data:    data,
        Meta:    &meta,
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(response)
}


// RespondError sends a JSON error response
func RespondError(w http.ResponseWriter, status int, code, message string) {
	response := models.ErrorResponse{
		Success: false,
	}
	response.Error.Code = code
	response.Error.Message = message

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

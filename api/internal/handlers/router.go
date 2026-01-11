package handlers

import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
)

// NewRouter returns a new HTTP router
func NewRouter() *chi.Mux {
    r := chi.NewRouter()

    // Middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.URLFormat)

    // CORS
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{"*"}, // Adjust for production
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
        ExposedHeaders:   []string{"Link"},
        AllowCredentials: true,
        MaxAge:           300,
    }))

    // Routes
    r.Route("/holidays", func(r chi.Router) {
        // Specific paths must come before wildcard {id} if they can conflict
        // But /check/ and /upcoming/ are distinct segments if structured as /holidays/check
        // Wait, the spec has /holidays/check. 
        // If I map /holidays/{id}, does /holidays/check match? 
        // Yes, "check" matches {id}.
        // Chi usually handles longest match or specific match.
        // Best practice: Register specific first.
        
        r.Get("/upcoming", GetUpcomingHolidays)
        r.Get("/check", CheckHoliday)
        r.Get("/working-days", CalculateWorkingDays)
        r.Get("/", GetHolidays) // /holidays
        r.Get("/{id}", GetHolidayByID)
    })

    r.Route("/states", func(r chi.Router) {
        r.Get("/", GetStates)
        r.Get("/{state_code}/holidays", GetStateHolidays)
        r.Get("/{state_code}/weekend", GetStateWeekend)
    })
    
    r.Get("/metadata", GetMetadata)

    return r
}

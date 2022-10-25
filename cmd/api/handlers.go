package main

import (
	"net/http"
)

func (s *server) handleHealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		env := envelope{
			"status": "available",
			"system_info": map[string]string{
				"environment": s.env,
				"version":     version,
			},
		}

		err := s.writeJSON(w, http.StatusOK, env, nil)
		if err != nil {
			s.serverErrorResponse(w, r, err)
		}
	}
}

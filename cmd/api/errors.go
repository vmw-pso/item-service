package main

import (
	"fmt"
	"net/http"
)

func (s *server) errorLog(r *http.Request, err error) {
	s.logger.PrintError(err, map[string]string{
		"request-method": r.Method,
		"request_url":    r.URL.String(),
	})
}

func (s *server) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}
	err := s.writeJSON(w, status, env, nil)
	if err != nil {
		s.errorLog(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *server) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	s.errorLog(r, err)

	message := "the server encountered a problem and could not process the request"
	s.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (s *server) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	s.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (s *server) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	s.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (s *server) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	s.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (s *server) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	s.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (s *server) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	s.errorResponse(w, r, http.StatusConflict, message)
}

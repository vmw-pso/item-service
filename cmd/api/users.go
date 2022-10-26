package main

import (
	"errors"
	"net/http"

	"github.com/vmx-pso/item-service/internal/data"
	"github.com/vmx-pso/item-service/internal/validator"
)

func (s *server) handleRegisterUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		err := s.readJSON(w, r, &requestPayload)
		if err != nil {
			s.badRequestResponse(w, r, err)
			return
		}

		user := &data.User{
			Name:      requestPayload.Name,
			Email:     requestPayload.Email,
			Activated: false,
		}

		err = user.Password.Set(requestPayload.Password)
		if err != nil {
			s.serverErrorResponse(w, r, err)
			return
		}

		v := validator.New()

		if data.ValidateUser(v, user); !v.Valid() {
			s.failedValidationResponse(w, r, v.Errors)
			return
		}

		err = s.models.Users.Insert(user)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrDuplicateEmail):
				v.AddError("email", "a user with this email already exists")
				s.failedValidationResponse(w, r, v.Errors)
			default:
				s.serverErrorResponse(w, r, err)
			}
			return
		}

		s.background(func() {
			err = s.mailer.Send(user.Email, "user_welcome.tmpl", user)
			if err != nil {
				s.logger.PrintError(err, nil)
			}
		})

		err = s.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
		if err != nil {
			s.serverErrorResponse(w, r, err)
		}
	}
}

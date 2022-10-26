package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/vmx-pso/item-service/internal/data"
	"github.com/vmx-pso/item-service/internal/validator"
)

func (app *application) handleCreateAuthenticationToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		err := app.readJSON(w, r, &requestPayload)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		v := validator.New()

		data.ValidateEmail(v, requestPayload.Email)
		data.ValidatePasswordPlaintext(v, requestPayload.Password)

		if !v.Valid() {
			app.failedValidationResponse(w, r, v.Errors)
			return
		}

		user, err := app.models.Users.GetByEmail(requestPayload.Email)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecord):
				app.invalidCredentialsResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		match, err := user.Password.Matches(requestPayload.Password)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !match {
			app.invalidCredentialsResponse(w, r)
			return
		}

		token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}
	}
}

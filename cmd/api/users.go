package main

import (
	"errors"
	"net/http"

	"github.com/vmx-pso/item-service/internal/data"
	"github.com/vmx-pso/item-service/internal/validator"
)

func (app *application) handleRegisterUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		err := app.readJSON(w, r, &requestPayload)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		user := &data.User{
			Name:      requestPayload.Name,
			Email:     requestPayload.Email,
			Activated: false,
		}

		err = user.Password.Set(requestPayload.Password)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		v := validator.New()

		if data.ValidateUser(v, user); !v.Valid() {
			app.failedValidationResponse(w, r, v.Errors)
			return
		}

		err = app.models.Users.Insert(user)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrDuplicateEmail):
				v.AddError("email", "a user with this email already exists")
				app.failedValidationResponse(w, r, v.Errors)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		app.background(func() {
			err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
			if err != nil {
				app.logger.PrintError(err, nil)
			}
		})

		err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}
	}
}

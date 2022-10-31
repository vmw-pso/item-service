package main

import (
	"errors"
	"net/http"
	"time"

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

		err = app.models.Permissions.AddForUser(user.ID, "items:read")
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		app.background(func() {
			data := map[string]interface{}{
				"activationToken": token.Plaintext,
				"userID":          user.ID,
			}

			err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
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

func (app *application) handleActivateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			TokenPlainText string `json:"token"`
		}

		err := app.readJSON(w, r, &requestPayload)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		v := validator.New()

		if data.ValidateTokenPlaintext(v, requestPayload.TokenPlainText); !v.Valid() {
			app.failedValidationResponse(w, r, v.Errors)
			return
		}

		user, err := app.models.Users.GetForToken(data.ScopeActivation, requestPayload.TokenPlainText)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecord):
				v.AddError("token", "invalid or expired activation token")
				app.failedValidationResponse(w, r, v.Errors)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		user.Activated = true

		err = app.models.Users.Update(user)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrEditConflict):
				app.editConflictResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}

		err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
		if err != nil {
			app.serverErrorResponse(w, r, err)
		}
	}
}

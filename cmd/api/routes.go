package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.handleHealthCheck())

	router.HandlerFunc(http.MethodGet, "/v1/items", app.handleListItems())
	router.HandlerFunc(http.MethodPost, "/v1/items", app.handleCreateItem())
	router.HandlerFunc(http.MethodGet, "/v1/items/:id", app.handleShowItem())
	router.HandlerFunc(http.MethodPatch, "/v1/items/:id", app.handleUpdateItem())
	router.HandlerFunc(http.MethodDelete, "/v1/items/:id", app.handleDeleteItem())

	router.HandlerFunc(http.MethodPost, "/v1/users", app.handleRegisterUser())
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.handleActivateUser())
	router.HandlerFunc(http.MethodPost, "/v1/users/authentication", app.handleCreateAuthenticationToken())

	app.recoverPanic(app.rateLimit(router))

	return router
}

package main

import (
	"expvar"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.handleHealthCheck())

	router.HandlerFunc(http.MethodGet, "/v1/items", app.requirePermission("items:read", app.handleListItems()))
	router.HandlerFunc(http.MethodPost, "/v1/items", app.requirePermission("items:write", app.handleCreateItem()))
	router.HandlerFunc(http.MethodGet, "/v1/items/:id", app.requirePermission("items:read", app.handleShowItem()))
	router.HandlerFunc(http.MethodPatch, "/v1/items/:id", app.requirePermission("items:write", app.handleUpdateItem()))
	router.HandlerFunc(http.MethodDelete, "/v1/items/:id", app.requirePermission("items:write", app.handleDeleteItem()))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.handleRegisterUser())
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.handleActivateUser())

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.handleCreateAuthenticationToken())

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}

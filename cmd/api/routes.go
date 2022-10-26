package main

import (
	"net/http"
)

func (s *server) routes() {
	s.router.HandlerFunc(http.MethodGet, "/v1/healthcheck", s.rateLimit(s.handleHealthCheck()))

	s.router.HandlerFunc(http.MethodGet, "/v1/items", s.rateLimit(s.handleListItems()))
	s.router.HandlerFunc(http.MethodPost, "/v1/items", s.rateLimit(s.handleCreateItem()))
	s.router.HandlerFunc(http.MethodGet, "/v1/items/:id", s.rateLimit(s.handleShowItem()))
	s.router.HandlerFunc(http.MethodPatch, "/v1/items/:id", s.handleUpdateItem())
	s.router.HandlerFunc(http.MethodDelete, "/v1/items/:id", s.handleDeleteItem())

	s.router.HandlerFunc(http.MethodPost, "/v1/users", s.rateLimit(s.handleRegisterUser()))
}

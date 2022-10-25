package main

import (
	"net/http"
)

func (s *server) routes() {
	s.router.HandlerFunc(http.MethodGet, "/v1/healthcheck", s.handleHealthCheck())
	s.router.HandlerFunc(http.MethodGet, "/v1/items", s.handleListItems())
	s.router.HandlerFunc(http.MethodPost, "/v1/items", s.handleCreateItem())
	s.router.HandlerFunc(http.MethodGet, "/v1/items/:id", s.handleShowItem())
	s.router.HandlerFunc(http.MethodPatch, "/v1/items/:id", s.handleUpdateItem())
	s.router.HandlerFunc(http.MethodDelete, "/v1/items/:id", s.handleDeleteItem())
}

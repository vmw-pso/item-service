package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/vmx-pso/item-service/internal/data"
	"github.com/vmx-pso/item-service/internal/validator"
)

func (s *server) handleCreateItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			Name      string     `json:"name"`
			Model     string     `json:"model"`
			Supplier  int64      `json:"supplier"`
			Price     data.Price `json:"price"`
			Currency  int64      `json:"currency"`
			ImageFile string     `json:"image"`
			Notes     string     `json:"notes"`
			Tags      []string   `json:"tags"`
		}

		err := s.readJSON(w, r, &requestPayload)
		if err != nil {
			s.badRequestResponse(w, r, err)
			return
		}

		item := &data.Item{
			Name:      requestPayload.Name,
			Model:     requestPayload.Model,
			Supplier:  requestPayload.Supplier,
			Price:     float64(requestPayload.Price),
			Currency:  requestPayload.Currency,
			ImageFile: requestPayload.ImageFile,
			Notes:     requestPayload.Notes,
			Tags:      requestPayload.Tags,
		}

		v := validator.New()

		if data.ValidateItem(v, item); !v.Valid() {
			s.failedValidationResponse(w, r, v.Errors)
			return
		}

		err = s.models.Items.Insert(item)
		if err != nil {
			s.serverErrorResponse(w, r, err)
			return
		}

		headers := make(http.Header)
		headers.Set("Location", fmt.Sprintf("/v1/items/%d", item.ID))

		err = s.writeJSON(w, http.StatusCreated, envelope{"item": item}, headers)
		if err != nil {
			s.serverErrorResponse(w, r, err)
		}
	}
}

func (s *server) handleShowItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := s.readIDParam(r)
		if err != nil || id < 1 {
			s.notFoundResponse(w, r)
			return
		}

		item, err := s.models.Items.Get(id)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecord):
				s.notFoundResponse(w, r)
			default:
				s.serverErrorResponse(w, r, err)
			}
			return
		}

		err = s.writeJSON(w, http.StatusOK, envelope{"item": item}, nil)
		if err != nil {
			s.serverErrorResponse(w, r, err)
		}
	}
}

func (s *server) handleUpdateItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := s.readIDParam(r)
		if err != nil || id < 1 {
			s.notFoundResponse(w, r)
			return
		}

		item, err := s.models.Items.Get(id)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecord):
				s.notFoundResponse(w, r)
			default:
				s.serverErrorResponse(w, r, err)
			}
			return
		}

		var requestPayload struct {
			Name      *string     `json:"name"`
			Model     *string     `json:"model"`
			Supplier  *int64      `json:"supplier"`
			Price     *data.Price `json:"price"`
			Currency  *int64      `json:"currency"`
			ImageFile *string     `json:"image"`
			Notes     *string     `json:"notes"`
			Tags      []string    `json:"tags"`
			Archived  *bool       `json:"archived"`
		}

		err = s.readJSON(w, r, &requestPayload)
		if err != nil {
			s.badRequestResponse(w, r, err)
			return
		}

		if requestPayload.Name != nil {
			item.Name = *requestPayload.Name
		}

		if requestPayload.Model != nil {
			item.Model = *requestPayload.Model
		}

		if requestPayload.Supplier != nil {
			item.Supplier = *requestPayload.Supplier
		}

		if requestPayload.Price != nil {
			item.Price = float64(*requestPayload.Price)
		}

		if requestPayload.Currency != nil {
			item.Currency = *requestPayload.Currency
		}

		if requestPayload.ImageFile != nil {
			item.ImageFile = *requestPayload.ImageFile
		}

		if requestPayload.Notes != nil {
			item.Notes = *requestPayload.Notes
		}

		if requestPayload.Tags != nil {
			item.Tags = requestPayload.Tags
		}

		if requestPayload.Archived != nil {
			item.Archived = *requestPayload.Archived
		}

		v := validator.New()

		if data.ValidateItem(v, item); !v.Valid() {
			s.failedValidationResponse(w, r, v.Errors)
			return
		}

		err = s.models.Items.Update(item)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrEditConflict):
				s.editConflictResponse(w, r)
			default:
				s.serverErrorResponse(w, r, err)
			}
			return
		}

		err = s.writeJSON(w, http.StatusOK, envelope{"item": item}, nil)
		if err != nil {
			s.serverErrorResponse(w, r, err)
		}
	}
}

func (s *server) handleDeleteItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := s.readIDParam(r)
		if err != nil || id < 1 {
			s.notFoundResponse(w, r)
			return
		}

		err = s.models.Items.Delete(id)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecord):
				s.notFoundResponse(w, r)
			default:
				s.serverErrorResponse(w, r, err)
			}
			return
		}

		err = s.writeJSON(w, http.StatusOK, envelope{"message": "successfully deleted"}, nil)
		if err != nil {
			s.serverErrorResponse(w, r, err)
		}
	}
}

func (s *server) handleListItems() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload struct {
			Name     string
			Supplier int
			Tags     []string
			data.Filters
		}

		v := validator.New()

		qs := r.URL.Query()

		requestPayload.Name = s.readString(qs, "name", "")
		requestPayload.Supplier = s.readInt(qs, "supplier", 0, v)
		requestPayload.Tags = s.readCSV(qs, "tags", []string{})
		requestPayload.Filters.Page = s.readInt(qs, "page", 1, v)
		requestPayload.Filters.PageSize = s.readInt(qs, "page_size", 20, v)
		requestPayload.Filters.Sort = s.readString(qs, "sort", "id")
		requestPayload.Filters.SortSafelist = []string{"id", "name", "model", "supplier", "price", "-id", "-name", "-model", "-price"}

		if data.ValidateFilters(v, requestPayload.Filters); !v.Valid() {
			s.failedValidationResponse(w, r, v.Errors)
			return
		}

		items, metadata, err := s.models.Items.GetAll(requestPayload.Name, requestPayload.Supplier, requestPayload.Tags, requestPayload.Filters)
		if err != nil {
			s.serverErrorResponse(w, r, err)
			return
		}

		err = s.writeJSON(w, http.StatusOK, envelope{"items": items, "metadata": metadata}, nil)
		if err != nil {
			s.serverErrorResponse(w, r, err)
		}
	}
}

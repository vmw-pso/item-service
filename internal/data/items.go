package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vmx-pso/item-service/internal/validator"

	"github.com/lib/pq"
)

type Item struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Model     string    `json:"model"`
	Supplier  int64     `json:"supplier"`
	Price     float64   `json:"price"`
	Currency  int64     `json:"currency"`
	ImageFile string    `json:"image"`
	Notes     string    `json:"notes"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Archived  bool      `json:"archived"`
}

func ValidateItem(v *validator.Validator, item *Item) {
	v.Check(item.Name != "", "name", "must be provided")
	v.Check(len(item.Name) <= 255, "name", "must not be more than 255 characters long")
	v.Check(item.Supplier != 0, "supplier", "must be provided")
	v.Check(item.Price != 0, "price", "must be provided")
	v.Check(item.Price > 0, "price", "must be a positive value")
	v.Check(item.Currency != 0, "currency", "must be provided")
	v.Check(validator.Unique(item.Tags), "tags", "must not contain duplicate values")
}

type ItemModel struct {
	DB *sql.DB
}

func (m *ItemModel) Insert(item *Item) error {
	qry := `
		INSERT INTO items(name, model, supplier, price, currency, image_file, notes, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	args := []interface{}{item.Name, item.Model, item.Supplier, item.Price, item.Currency, item.ImageFile, item.Notes, pq.Array(item.Tags)}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, qry, args...).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

func (m *ItemModel) Get(id int64) (*Item, error) {
	if id < 1 {
		return nil, ErrNoRecord
	}

	qry := `
		SELECT id, name, model, supplier, price, currency, image_file, notes, tags, created_at, updated_at, archived
		FROM items
		WHERE id = $1`

	var item Item

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, qry, id).Scan(
		&item.ID,
		&item.Name,
		&item.Model,
		&item.Supplier,
		&item.Price,
		&item.Currency,
		&item.ImageFile,
		&item.Notes,
		pq.Array(&item.Tags),
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.Archived,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecord
		default:
			return nil, err
		}
	}
	return &item, nil
}

func (m *ItemModel) Update(item *Item) error {
	qry := `
		UPDATE items
		SET name = $1, model = $2, supplier = $3, price = $4, currency = $5, image_file = $6, notes = $7, tags = $8, updated_at = $9, archived = $10
		WHERE id=$11 AND updated_at=$12
		RETURNING updated_at`

	args := []interface{}{
		item.Name,
		item.Model,
		item.Supplier,
		item.Price,
		item.Currency,
		item.ImageFile,
		item.Notes,
		pq.Array(item.Tags),
		time.Now(),
		item.Archived,
		item.ID,
		item.UpdatedAt,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, qry, args...).Scan(&item.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m *ItemModel) Delete(id int64) error {
	if id < 1 {
		return ErrNoRecord
	}

	qry := `
		DELETE FROM items
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, qry, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNoRecord
	}

	return nil
}

func (m *ItemModel) GetAll(name string, supplier int, tags []string, filters Filters) ([]*Item, Metadata, error) {
	qry := fmt.Sprintf(`
		SELECT count(*) OVER(), id, name, model, supplier, price, currency, notes, tags, created_at, updated_at, archived
		FROM items
		WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (supplier = $2 OR $2 = 0)
		AND (tags @> $3 OR $3 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $4 OFFSET $5`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{name, supplier, pq.Array(tags), filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, qry, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	items := []*Item{}

	for rows.Next() {
		var item Item
		err := rows.Scan(
			&totalRecords,
			&item.ID,
			&item.Name,
			&item.Model,
			&item.Supplier,
			&item.Price,
			&item.Currency,
			&item.Notes,
			pq.Array(&item.Tags),
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Archived,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		items = append(items, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return items, metadata, nil
}

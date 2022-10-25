package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vmx-pso/item-service/internal/data"
	"github.com/vmx-pso/item-service/internal/jsonlog"

	_ "github.com/lib/pq"
)

const version = "0.0.1"

type server struct {
	port   int
	env    string
	router *httprouter.Router
	logger *jsonlog.Logger
	db     *sql.DB
	models *data.Models
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func main() {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	if err := run(os.Args, logger); err != nil {
		logger.PrintFatal(err, nil)
	}
}

func run(args []string, logger *jsonlog.Logger) error {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var (
		port         = flags.Int("port", 80, "port to listen on")
		env          = flags.String("env", "development", "Environment (development|staging|production")
		dsn          = flags.String("dsn", "postgres://postgres:password@localhost/items?sslmode=disable", "PostreSQL DSN") // move to env variable later
		maxOpenConns = flags.Int("db-max-open-conns", 25, "PostgeSQL max open connections")
		maxIdleConns = flags.Int("db-max-idle-conns", 25, "PostgreSQL max idle connections")
		maxIdleTime  = flags.String("db-max-idle-time", "15m", "PostreSQL max idle time")
	)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	db, err := openDB(*dsn, *maxOpenConns, *maxIdleConns, *maxIdleTime)
	if err != nil {
		return err
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	router := httprouter.New()

	srv := &server{
		port:   *port,
		env:    *env,
		router: router,
		logger: logger,
		db:     db,
		models: data.NewModels(db),
	}

	router.NotFound = http.HandlerFunc(srv.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(srv.methodNotAllowedResponse)

	srv.routes()

	addr := fmt.Sprintf(":%d", srv.port)

	logger.PrintInfo("starting server", map[string]string{
		"env":  srv.env,
		"addr": addr,
	})

	return http.ListenAndServe(addr, srv)
}

func openDB(dsn string, maxOpenConns, maxIdleConns int, maxIdleTime string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	duration, err := time.ParseDuration(maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

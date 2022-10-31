package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/vmx-pso/item-service/internal/data"
	"github.com/vmx-pso/item-service/internal/jsonlog"
	"github.com/vmx-pso/item-service/internal/mailer"
	"github.com/vmx-pso/item-service/internal/vcs"

	_ "github.com/lib/pq"
)

type config struct {
	port    int
	env     string
	db      db
	limiter rateLimiter
	smtp    smtp
	cors    cors
}

type cors struct {
	trustedOrigins []string
}

type db struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type rateLimiter struct {
	enabled bool
	rps     float64
	burst   int
}

type smtp struct {
	host     string
	port     int
	username string
	password string
	sender   string
}

type application struct {
	config config
	router httprouter.Router
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

var (
	version = vcs.Version()
)

func main() {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	if err := run(os.Args, logger); err != nil {
		logger.PrintFatal(err, nil)
	}
}

func run(args []string, logger *jsonlog.Logger) error {
	var corsTrustedOrigins []string
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var (
		port           = flags.Int("port", 80, "port to listen on")
		env            = flags.String("env", "development", "Environment (development|staging|production")
		dsn            = flags.String("db-dsn", "", "PostreSQL DSN")
		maxOpenConns   = flags.Int("db-max-open-conns", 25, "PostgeSQL max open connections")
		maxIdleConns   = flags.Int("db-max-idle-conns", 25, "PostgreSQL max idle connections")
		maxIdleTime    = flags.String("db-max-idle-time", "15m", "PostreSQL max idle time")
		rps            = flags.Float64("limiter-rps", 2, "Rate limiter maximum requests per second")
		burst          = flags.Int("limiter-burst", 4, "Rate limiter maximum burst")
		enabled        = flags.Bool("limiter-enabled", true, "Enable rate limited")
		smtpHost       = flags.String("smtp-host", "smtp.mailtrap.io", "SMTP host")
		smtpPort       = flags.Int("smtp-port", 25, "SMTP port")
		smtpUsername   = flags.String("smtp-username", "5bd3436757a4cf", "SMTP username")
		smtpPassword   = flags.String("smtp-password", "68e7ccd9cc75a8", "SMTP password")
		smtpSender     = flags.String("smtp-sender", "IMS <no-reply@fakemail.com>", "SMTP sender")
		displayVersion = flags.Bool("version", false, "Display version and exit")
	)
	flags.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		corsTrustedOrigins = strings.Fields(val)
		return nil
	})

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		return nil
	}

	cfg := config{
		port: *port,
		env:  *env,
		db: db{
			dsn:          *dsn,
			maxOpenConns: *maxOpenConns,
			maxIdleConns: *maxIdleConns,
			maxIdleTime:  *maxIdleTime,
		},
		limiter: rateLimiter{
			enabled: *enabled,
			rps:     *rps,
			burst:   *burst,
		},
		smtp: smtp{
			host:     *smtpHost,
			port:     *smtpPort,
			username: *smtpUsername,
			password: *smtpPassword,
			sender:   *smtpSender,
		},
		cors: cors{
			trustedOrigins: corsTrustedOrigins,
		},
	}

	db, err := openDB(*dsn, *maxOpenConns, *maxIdleConns, *maxIdleTime)
	if err != nil {
		return err
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() interface{} {
		return db.Stats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	app := &application{
		config: cfg,
		router: *httprouter.New(),
		logger: logger,
		models: *data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	return app.serve()
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

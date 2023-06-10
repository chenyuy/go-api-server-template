package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chenyuy/go-api-server-template/api"
	"github.com/chenyuy/go-api-server-template/config"
	"github.com/chenyuy/go-api-server-template/database"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
)

type server struct {
	conf    config.PostgresConfig
	queries *database.Queries
	dbpgx   *pgxpool.Pool
}

func (s *server) connectPgx() error {
	log.Printf("connecting to database with pgx: %s %s\n", s.conf.Host, s.conf.Port)
	pool, err := pgxpool.Connect(context.Background(), s.conf.PgxConnectionInfo(50, "5m"))
	if err != nil {
		return err
	}
	s.queries = database.New(pool)
	s.dbpgx = pool

	return nil
}

func (s *server) dbMigrate() error {
	return database.Migrate(s.dbpgx)
}

func main() {
	path := flag.String("config", "", "path to config file")
	flag.Parse()
	if path == nil || *path == "" {
		flag.Usage()
		return
	}

	serv := server{}

	conf, err := config.New(*path)
	if err != nil {
		log.Printf("cannot load config file: %v\n", err)
		return
	}

	serv.conf = *conf

	if err := serv.connectPgx(); err != nil {
		log.Printf("failed to connect to the db with pgx: %v\n", err)
		return
	}
	log.Println("connect to database success")

	if err := serv.dbMigrate(); err != nil {
		log.Printf("failed to migrate db: %v\n", err)
		return
	}
	log.Println("db migration success")

	handler, err := api.New()
	if err != nil {
		log.Printf("failed to create handler: %v\n", err)
		return
	}
	log.Println("creating handler success")

	r := mux.NewRouter()

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// implement custom 404 handler
		w.WriteHeader(http.StatusNotFound)
	})

	r.Handle(
		"/",
		handler.NotImplementedHandler,
	)

	server := http.Server{
		Addr:              ":5000",
		Handler:           r,
		ReadHeaderTimeout: time.Second * 20,
		ReadTimeout:       time.Second * 20,
		WriteTimeout:      time.Second * 60,
		IdleTimeout:       time.Second * 60,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	idelConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		idelConnsClosed <- struct{}{}
		close(idelConnsClosed)
	}()
	log.Println("starting server")
	err = server.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			log.Printf("HTTP server close: %v\n", err)
		}
	}
	<-idelConnsClosed
}

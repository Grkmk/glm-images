package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	goHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/grkmk/glm-images/files"
	"github.com/grkmk/glm-images/handlers"
	hcLog "github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/env"
)

var bindAddress = env.String("BIND_ADDRESS", false, ":9091", "Bind address for the server")
var logLevel = env.String("LOG_LEVEL", false, "debug", "Log output level for the server [debug, info, trace]")
var basePath = env.String("BASE_PATH", false, "./imagestore", "Base path to save images")
var clientURL = env.String("CLIENT_URL", false, "http://localhost:3000", "Client URL")

func main() {
	env.Parse()

	logger := hcLog.New(
		&hcLog.LoggerOptions{
			Name:  "product-images",
			Level: hcLog.LevelFromString(*logLevel),
		},
	)

	standardLogger := logger.StandardLogger(&hcLog.StandardLoggerOptions{InferLevels: true})

	storage, err := files.NewLocal(*basePath, 1024*1000*5)
	if err != nil {
		logger.Error("Unable to create storage", "error", err)
		os.Exit(1)
	}

	fileHandler := handlers.NewFiles(storage, logger)

	serveMux := mux.NewRouter()
	corsHandler := goHandlers.CORS(goHandlers.AllowedOrigins([]string{*clientURL}))

	postHandler := serveMux.Methods(http.MethodPost).Subrouter()
	postHandler.HandleFunc("/images/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}", fileHandler.UploadRest)
	postHandler.HandleFunc("/", fileHandler.UploadMultipart)

	getHandler := serveMux.Methods(http.MethodGet).Subrouter()
	getHandler.Handle("/images/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}", http.StripPrefix("/images/", http.FileServer(http.Dir(*basePath))))

	gzipHandler := serveMux.Methods(http.MethodGet).Subrouter()
	gzipHandler.Handle(
		"/images/{id:[0-9]+}/{filename:[a-zA-Z]+\\.[a-z]{3}}",
		http.StripPrefix("/images/", http.FileServer(http.Dir(*basePath))),
	)
	gzipHandler.Use(handlers.GZipResponseMiddleware)

	server := http.Server{
		Addr:         *bindAddress,          // configure the bind address
		Handler:      corsHandler(serveMux), // set the default handler
		ErrorLog:     standardLogger,        // the logger for the server
		ReadTimeout:  5 * time.Second,       // max time to read request from the client
		WriteTimeout: 10 * time.Second,      // max time to write response to the client
		IdleTimeout:  120 * time.Second,     // max time for connections using TCP Keep-Alive
	}

	go func() {
		logger.Info("Starting server", "bind_address", *bindAddress)

		err := server.ListenAndServe()
		if err != nil {
			logger.Error("Unable to start server", "error", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interrupt and gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// block until a signal is received
	sig := <-c
	logger.Info("Shutting down server with", "signal", sig)

	// gracefully shutdown, waiting max 30 secs to complete current operations
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	server.Shutdown(ctx)
}

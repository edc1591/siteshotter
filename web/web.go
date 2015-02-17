package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/edc1591/siteshotter/phantom"
	"github.com/gorilla/mux"
	"github.com/timehop/golog/log"
)

var pool phantom.RenderServer

func init() {
	pool = phantom.NewRenderServerPool(4)
}

func main() {
	if err := pool.Start(); err != nil {
		log.Fatal("siteshotter/main", "Failed to start phantomjs servers", "error", err)
	}

	prepareForShutdownDown()

	r := mux.NewRouter()
	r.HandleFunc("/image.png", PageRenderHandler).Methods("GET")
	http.Handle("/", r)
	port := os.Getenv("PORT")
	log.Info("siteshotter/main", "Starting server...", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("siteshotter/main", "Failed to start server", "error", err)
	}
}

func prepareForShutdownDown() {
	// Make sure to kill all spawned processes if this proc gets killed
	sig := make(chan os.Signal, 1)
	signal.Notify(sig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		log.Info("siteshotter/main", "Received os.Signal", "signal", <-sig)
		pool.Shutdown()
		os.Exit(1)
	}()
}

func PageRenderHandler(response http.ResponseWriter, request *http.Request) {
	url := request.FormValue("url")
	body, err := pool.RenderPage(url)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte("500 Internal Server Error"))
		return
	}
	response.Header().Set("Content-Type", "image/png")
	response.WriteHeader(http.StatusOK)
	response.Write(body)
}

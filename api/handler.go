package api

import "net/http"

type Handler struct {
	NotImplementedHandler http.HandlerFunc
}

func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func New() (*Handler, error) {
	return &Handler{NotImplementedHandler: notImplementedHandler}, nil
}

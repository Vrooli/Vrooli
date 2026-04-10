package handler

import (
	"net/http"

	"tunnel-manager/domain"
)

func HandleListRoutes(svc RouteManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routes, err := svc.List()
		if err != nil {
			writeError(w, err)
			return
		}
		if routes == nil {
			routes = []domain.Route{}
		}
		writeJSON(w, http.StatusOK, routes)
	}
}

func HandleGetRoute(svc RouteManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parsePathID(w, r)
		if !ok {
			return
		}
		route, err := svc.GetByID(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, route)
	}
}

func HandleCreateRoute(svc RouteManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in domain.RouteInput
		if !decodeJSON(w, r, &in) {
			return
		}
		route, err := svc.Create(in)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, route)
	}
}

func HandleUpdateRoute(svc RouteManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parsePathID(w, r)
		if !ok {
			return
		}
		var in domain.RouteInput
		if !decodeJSON(w, r, &in) {
			return
		}
		route, err := svc.Update(id, in)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, route)
	}
}

func HandleDeleteRoute(svc RouteManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parsePathID(w, r)
		if !ok {
			return
		}
		if err := svc.Delete(id); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

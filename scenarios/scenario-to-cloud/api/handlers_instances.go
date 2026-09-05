package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"scenario-to-cloud/instance"
	"scenario-to-cloud/internal/httputil"
)

func (s *Server) handleInstancePlan(w http.ResponseWriter, r *http.Request) {
	request, err := httputil.DecodeJSON[instance.Request](r.Body, 1<<20)
	if err != nil {
		writeInstanceError(w, http.StatusBadRequest, err)
		return
	}
	plan, err := s.instanceProvider.Plan(r.Context(), request)
	if err != nil {
		writeInstanceError(w, http.StatusConflict, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, plan)
}

func (s *Server) handleInstanceCreate(w http.ResponseWriter, r *http.Request) {
	request, err := httputil.DecodeJSON[instance.Request](r.Body, 1<<20)
	if err != nil {
		writeInstanceError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.instanceProvider.Create(r.Context(), request)
	if err != nil {
		writeInstanceError(w, statusForInstanceError(err), err)
		return
	}
	value, err = s.repo.CreateInstance(r.Context(), value)
	if err != nil {
		writeInstanceError(w, http.StatusInternalServerError, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, value)
}

func (s *Server) handleInstanceAction(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	value, err := s.repo.GetInstance(r.Context(), id)
	if err != nil {
		writeInstanceError(w, http.StatusInternalServerError, err)
		return
	}
	if value == nil {
		writeInstanceError(w, http.StatusNotFound, instance.ErrInstanceNotFound)
		return
	}
	action := mux.Vars(r)["action"]
	ctx, cancel := context.WithTimeout(r.Context(), instanceActionTimeout(action))
	defer cancel()
	var operationErr error
	switch action {
	case "start":
		value.PID, operationErr = s.instanceProvider.Start(ctx, *value)
		if operationErr == nil {
			value.State = instance.StateRunning
		}
	case "stop":
		operationErr = s.instanceProvider.Stop(ctx, *value)
		if operationErr == nil {
			value.State = instance.StateStopped
			value.PID = 0
		}
	case "wait-for-ssh":
		operationErr = s.instanceProvider.WaitForSSH(ctx, *value)
	case "snapshot":
		operationErr = s.instanceProvider.Snapshot(ctx, *value, r.URL.Query().Get("name"))
	case "reset":
		operationErr = s.instanceProvider.Reset(ctx, *value, r.URL.Query().Get("name"))
	case "destroy":
		operationErr = s.instanceProvider.Destroy(ctx, *value)
		if operationErr == nil {
			operationErr = s.repo.DeleteInstance(ctx, id)
		}
	default:
		writeInstanceError(w, http.StatusBadRequest, errors.New("unsupported instance action"))
		return
	}
	if operationErr != nil {
		_ = s.repo.UpdateInstanceState(r.Context(), id, instance.StateFailed, value.PID, operationErr.Error())
		writeInstanceError(w, statusForInstanceError(operationErr), operationErr)
		return
	}
	if action != "destroy" {
		if err := s.repo.UpdateInstanceState(r.Context(), id, value.State, value.PID, ""); err != nil {
			writeInstanceError(w, http.StatusInternalServerError, err)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, value)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "destroyed", "id": id})
}

func instanceActionTimeout(action string) time.Duration {
	if action == "destroy" {
		return 2 * time.Minute
	}
	return time.Minute
}

func statusForInstanceError(err error) int {
	switch {
	case errors.Is(err, instance.ErrInvalidRequest):
		return http.StatusBadRequest
	case errors.Is(err, instance.ErrProviderUnavailable), errors.Is(err, instance.ErrUnsupportedOperation):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeInstanceError(w http.ResponseWriter, status int, err error) {
	httputil.WriteAPIError(w, status, httputil.APIError{Code: "instance_operation_failed", Message: err.Error()})
}

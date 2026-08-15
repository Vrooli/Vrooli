package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vrooli/vrooli/internal/operatorinput"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

func (s *Server) handleV2OperatorInputs(w http.ResponseWriter, _ *http.Request) {
	queue, err := operatorinput.Load()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, queue)
}

func (s *Server) handleV2OperatorInputsResolve(w http.ResponseWriter, r *http.Request) {
	var answers []operatorinput.Answer
	if err := json.NewDecoder(r.Body).Decode(&answers); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "answers must be a JSON array: " + err.Error()})
		return
	}
	if _, err := operatorinput.ResolveWith(answers, func(values map[string]string) error {
		passphrase, ok := values["credential-store-passphrase"]
		if !ok {
			return nil
		}
		description, err := securestore.DescribeStore()
		if err != nil {
			return fmt.Errorf("inspect encrypted credential store: %w", err)
		}
		if description.Initialized {
			if _, err := securestore.UnlockStore(passphrase); err != nil {
				return fmt.Errorf("unlock encrypted credential store: %w", err)
			}
			return nil
		}
		if _, err := securestore.InitializeStore(passphrase); err != nil {
			return fmt.Errorf("initialize encrypted credential store: %w", err)
		}
		return nil
	}); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "resolved", "configuration_pending": false})
}

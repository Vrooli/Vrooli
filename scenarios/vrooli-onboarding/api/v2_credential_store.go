package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

// These seams keep the onboarding surface a transport adapter. Store policy,
// encryption, wrapping, and backend selection remain owned by securestore.
var (
	credentialStoreStatusCommand = func(context.Context) (securestore.StoreStatus, error) {
		return securestore.DescribeStore()
	}
	credentialStoreReselectCommand = func(_ context.Context) (securestore.MigrationReceipt, error) {
		root, err := manifestRoot()
		if err != nil {
			return securestore.MigrationReceipt{}, err
		}
		entries, err := onboardingCredentialMigrationEntries(root)
		if err != nil {
			return securestore.MigrationReceipt{}, err
		}
		return securestore.ReselectBackend(entries)
	}
	credentialStoreSelectCommand = func(_ context.Context, backend, reason string) error {
		return securestore.SelectBackend(backend, reason)
	}
	credentialStoreInitCommand = func(_ context.Context, passphrase string) (securestore.StoreStatus, error) {
		return securestore.InitializeStore(passphrase)
	}
	credentialStoreUnlockCommand = func(_ context.Context, passphrase string) (securestore.StoreStatus, error) {
		return securestore.UnlockStore(passphrase)
	}
	credentialStoreChangePassphraseCommand = func(_ context.Context, current, next string) error {
		return securestore.ChangePassphraseStore(current, next)
	}
	credentialStoreRewrapCommand = func(_ context.Context, passphrase string) (securestore.WrapInfo, error) {
		return securestore.RewrapStore(passphrase)
	}
)

type credentialStoreSecretRequest struct {
	Passphrase string `json:"passphrase"`
}

func onboardingCredentialMigrationEntries(root string) ([]securestore.MigrationEntry, error) {
	seen := map[string]bool{}
	entries := make([]securestore.MigrationEntry, 0)
	appendItems := func(items []credentialReadiness) {
		for _, item := range items {
			logicalID := strings.TrimSpace(item.LogicalID)
			field := strings.TrimSpace(item.Field)
			if logicalID == "" || field == "" {
				continue
			}
			key := logicalID + ":" + field
			name := "vrooli.credentials.v1/" + key
			if seen[name] {
				continue
			}
			seen[name] = true
			entries = append(entries, securestore.MigrationEntry{Service: "vrooli.credentials.v1", Key: key})
		}
	}
	for _, catalog := range []struct {
		kind string
		load func(string) ([]credentialReadiness, error)
	}{
		{kind: "scenarios", load: loadScenarioCredentialReadiness},
		{kind: "resources", load: loadCredentialReadiness},
	} {
		rootEntries, err := os.ReadDir(filepath.Join(root, catalog.kind))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range rootEntries {
			if !entry.IsDir() {
				continue
			}
			items, err := catalog.load(entry.Name())
			if err != nil {
				return nil, err
			}
			appendItems(items)
		}
	}
	return entries, nil
}

type credentialStoreChangeRequest struct {
	CurrentPassphrase string `json:"current_passphrase"`
	NewPassphrase     string `json:"new_passphrase"`
}

type credentialStoreSelectRequest struct {
	Backend string `json:"backend"`
	Reason  string `json:"reason,omitempty"`
}

// handleV2CredentialStoreReselect is deliberately inventory-free at the HTTP
// boundary. The server derives the manifest-backed credential inventory so an
// operator cannot accidentally migrate arbitrary host keyring entries, and no
// credential value crosses this surface.
func (s *Server) handleV2CredentialStoreReselect(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), credentialStoreTimeout)
	defer cancel()
	receipt, err := credentialStoreReselectCommand(ctx)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":   "credential backend migration failed",
			"receipt": receipt,
		})
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (s *Server) handleV2CredentialStoreStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), credentialStoreTimeout)
	defer cancel()
	status, err := credentialStoreStatusCommand(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential store status is unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleV2CredentialStoreSelect(w http.ResponseWriter, r *http.Request) {
	var request credentialStoreSelectRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Backend) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "backend is required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), credentialStoreTimeout)
	defer cancel()
	status, statusErr := credentialStoreStatusCommand(ctx)
	if statusErr != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential store status is unavailable"})
		return
	}
	if status.Entries > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "backend selection is locked while credentials exist; use the verified reselect and migration path"})
		return
	}
	if err := credentialStoreSelectCommand(ctx, strings.TrimSpace(request.Backend), strings.TrimSpace(request.Reason)); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "selected", "backend": strings.TrimSpace(request.Backend)})
}

func (s *Server) handleV2CredentialStoreInit(w http.ResponseWriter, r *http.Request) {
	var request credentialStoreSecretRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Passphrase) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "passphrase is required"})
		return
	}
	status, err := credentialStoreInitCommand(r.Context(), request.Passphrase)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "credential store initialization failed"})
		return
	}
	writeJSON(w, http.StatusCreated, status)
}

func (s *Server) handleV2CredentialStoreUnlock(w http.ResponseWriter, r *http.Request) {
	var request credentialStoreSecretRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.Passphrase) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "passphrase is required"})
		return
	}
	status, err := credentialStoreUnlockCommand(r.Context(), request.Passphrase)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "credential store unlock failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unlocked", "active_wrap": status.ActiveWrap})
}

func (s *Server) handleV2CredentialStoreChangePassphrase(w http.ResponseWriter, r *http.Request) {
	var request credentialStoreChangeRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.CurrentPassphrase) == "" || strings.TrimSpace(request.NewPassphrase) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "current_passphrase and new_passphrase are required"})
		return
	}
	if err := credentialStoreChangePassphraseCommand(r.Context(), request.CurrentPassphrase, request.NewPassphrase); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "credential store passphrase change failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "passphrase_changed"})
}

func (s *Server) handleV2CredentialStoreRewrap(w http.ResponseWriter, r *http.Request) {
	var request credentialStoreSecretRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	wrap, err := credentialStoreRewrapCommand(r.Context(), request.Passphrase)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "credential store rewrap failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rewrapped", "provider": wrap.Provider, "key_store": wrap.KeyStore})
}

const credentialStoreTimeout = 30 * time.Second

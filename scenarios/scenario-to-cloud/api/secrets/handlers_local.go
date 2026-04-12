package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apisecrets "github.com/vrooli/api-core/secrets"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
)

const (
	localSecretsScopeWorkspace = "workspace"
	localSecretsScopeScenario  = "scenario"
)

type LocalSecretSetRequest struct {
	Value    string `json:"value,omitempty"`
	Generate string `json:"generate,omitempty"` // hex:64 | base64:32 | alnum:48 | uuid
}

type LocalSecretGetResponse struct {
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
	Masked    bool   `json:"masked"`
	Scope     string `json:"scope"`
	Scenario  string `json:"scenario_id,omitempty"`
	Path      string `json:"path"`
	Timestamp string `json:"timestamp"`
}

type LocalSecretSetResponse struct {
	OK        bool   `json:"ok"`
	Key       string `json:"key"`
	Scope     string `json:"scope"`
	Scenario  string `json:"scenario_id,omitempty"`
	Path      string `json:"path"`
	Generated bool   `json:"generated"`
	Timestamp string `json:"timestamp"`
}

// HandleGetLocalSecret returns an HTTP handler that reads a local secret from workspace/scenario scope.
//
// GET /api/v1/local-secrets/{scope}/{key}?scenario_id=<id>&reveal=true
func HandleGetLocalSecret() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := strings.TrimSpace(mux.Vars(r)["scope"])
		key := strings.TrimSpace(mux.Vars(r)["key"])
		if err := validateSecretKey(key); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_key",
				Message: err.Error(),
			})
			return
		}

		scenarioID := strings.TrimSpace(r.URL.Query().Get("scenario_id"))
		secretsPath, err := resolveLocalSecretsPath(scope, scenarioID)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_scope",
				Message: err.Error(),
			})
			return
		}

		payload, err := readLocalSecretsFile(secretsPath)
		if err != nil {
			writeLocalSecretsError(w, "read_local_secrets_failed", "Failed to read local secrets file", err)
			return
		}

		value, exists := payload[key]
		if !exists {
			httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
				Code:    "secret_not_found",
				Message: fmt.Sprintf("Secret %q not found", key),
			})
			return
		}

		reveal := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("reveal")), "true")
		resp := LocalSecretGetResponse{
			Key:       key,
			Masked:    !reveal,
			Scope:     scope,
			Scenario:  scenarioID,
			Path:      secretsPath,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		if reveal {
			resp.Value = value
		}

		httputil.WriteJSON(w, http.StatusOK, resp)
	}
}

// HandleSetLocalSecret returns an HTTP handler that writes a local secret at workspace/scenario scope.
//
// PUT /api/v1/local-secrets/{scope}/{key}?scenario_id=<id>
func HandleSetLocalSecret() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := strings.TrimSpace(mux.Vars(r)["scope"])
		key := strings.TrimSpace(mux.Vars(r)["key"])
		if err := validateSecretKey(key); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_key",
				Message: err.Error(),
			})
			return
		}

		scenarioID := strings.TrimSpace(r.URL.Query().Get("scenario_id"))
		secretsPath, err := resolveLocalSecretsPath(scope, scenarioID)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_scope",
				Message: err.Error(),
			})
			return
		}

		req, err := httputil.DecodeJSON[LocalSecretSetRequest](r.Body, 1<<20)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_json",
				Message: "Invalid request body",
				Hint:    err.Error(),
			})
			return
		}

		generated := false
		value := req.Value
		if strings.TrimSpace(req.Generate) != "" {
			generated = true
			genValue, genErr := generateSecretValue(req.Generate)
			if genErr != nil {
				httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
					Code:    "invalid_generate_spec",
					Message: genErr.Error(),
				})
				return
			}
			value = genValue
		}

		if err := validateSecretValue(value); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_value",
				Message: err.Error(),
			})
			return
		}

		store, err := newLocalSecretsStore(secretsPath)
		if err != nil {
			writeLocalSecretsError(w, "read_local_secrets_failed", "Failed to initialize local secrets store", err)
			return
		}
		if err := store.Update(func(doc *apisecrets.Document) error {
			setManagedMetadata(doc)
			doc.Secrets[key] = value
			return nil
		}); err != nil {
			writeLocalSecretsError(w, "write_local_secrets_failed", "Failed to write local secrets file", err)
			return
		}

		httputil.WriteJSON(w, http.StatusOK, LocalSecretSetResponse{
			OK:        true,
			Key:       key,
			Scope:     scope,
			Scenario:  scenarioID,
			Path:      secretsPath,
			Generated: generated,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// HandleDeleteLocalSecret returns an HTTP handler that removes a local secret at workspace/scenario scope.
//
// DELETE /api/v1/local-secrets/{scope}/{key}?scenario_id=<id>
func HandleDeleteLocalSecret() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope := strings.TrimSpace(mux.Vars(r)["scope"])
		key := strings.TrimSpace(mux.Vars(r)["key"])
		if err := validateSecretKey(key); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_key",
				Message: err.Error(),
			})
			return
		}

		scenarioID := strings.TrimSpace(r.URL.Query().Get("scenario_id"))
		secretsPath, err := resolveLocalSecretsPath(scope, scenarioID)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_scope",
				Message: err.Error(),
			})
			return
		}

		store, err := newLocalSecretsStore(secretsPath)
		if err != nil {
			writeLocalSecretsError(w, "read_local_secrets_failed", "Failed to initialize local secrets store", err)
			return
		}
		found := false
		err = store.Update(func(doc *apisecrets.Document) error {
			if _, exists := doc.Secrets[key]; !exists {
				return nil
			}
			delete(doc.Secrets, key)
			setManagedMetadata(doc)
			found = true
			return nil
		})
		if err != nil {
			writeLocalSecretsError(w, "write_local_secrets_failed", "Failed to write local secrets file", err)
			return
		}
		if !found {
			httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
				Code:    "secret_not_found",
				Message: fmt.Sprintf("Secret %q not found", key),
			})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, domain.NewSecretOperationResponse(true, key, "deleted", "Local secret deleted successfully"))
	}
}

func resolveLocalSecretsPath(scope, scenarioID string) (string, error) {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		projectStore, projectErr := apisecrets.NewProjectStoreFromEnvOrCWD(apisecrets.Config{})
		if projectErr != nil {
			return "", fmt.Errorf("repo root not found: %w", err)
		}
		repoRoot = projectStore.RepoRoot()
	}
	projectStore, err := apisecrets.NewProjectStore(apisecrets.Config{RepoRoot: repoRoot})
	if err != nil {
		return "", fmt.Errorf("repo root not found: %w", err)
	}
	repoRoot = projectStore.RepoRoot()

	switch scope {
	case localSecretsScopeWorkspace:
		return projectStore.PlaintextPath(), nil
	case localSecretsScopeScenario:
		if strings.TrimSpace(scenarioID) == "" {
			return "", fmt.Errorf("scenario_id is required for scope=%q", localSecretsScopeScenario)
		}
		if strings.Contains(scenarioID, "..") || strings.ContainsAny(scenarioID, `/\`) {
			return "", fmt.Errorf("invalid scenario_id %q", scenarioID)
		}
		scenarioPath, err := bundle.ResolveScenarioPath(repoRoot, scenarioID)
		if err != nil {
			return "", err
		}
		return filepath.Join(scenarioPath, ".vrooli", "secrets.json"), nil
	default:
		return "", fmt.Errorf("unknown scope %q (valid scopes: %s, %s)", scope, localSecretsScopeWorkspace, localSecretsScopeScenario)
	}
}

func readLocalSecretsFile(path string) (map[string]string, error) {
	return apisecrets.LoadFile(path)
}

func newLocalSecretsStore(path string) (*apisecrets.Store, error) {
	return apisecrets.NewFileStore(path)
}

func setManagedMetadata(doc *apisecrets.Document) {
	if doc.Metadata == nil {
		doc.Metadata = map[string]json.RawMessage{}
	}
	doc.Metadata["_metadata"] = json.RawMessage(fmt.Sprintf(`{"managed_by":"scenario-to-cloud","last_updated":%q}`, time.Now().UTC().Format(time.RFC3339)))
}

func writeLocalSecretsError(w http.ResponseWriter, code, message string, err error) {
	status := http.StatusInternalServerError
	var secretsErr *apisecrets.Error
	if errors.As(err, &secretsErr) {
		switch secretsErr.Kind {
		case apisecrets.ErrInvalidInput:
			status = http.StatusBadRequest
		case apisecrets.ErrInvalidData, apisecrets.ErrInsecurePermissions, apisecrets.ErrSymlinkPath:
			status = http.StatusConflict
		}
	}
	httputil.WriteAPIError(w, status, httputil.APIError{
		Code:    code,
		Message: message,
		Hint:    err.Error(),
	})
}

func generateSecretValue(spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", fmt.Errorf("generate spec is required")
	}
	if strings.EqualFold(spec, "uuid") {
		return generateUUID()
	}

	parts := strings.Split(spec, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("unsupported generate spec %q (expected hex:<n>, base64:<n>, alnum:<n>, or uuid)", spec)
	}
	method := strings.ToLower(strings.TrimSpace(parts[0]))
	n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || n <= 0 || n > 4096 {
		return "", fmt.Errorf("invalid generate length %q", parts[1])
	}

	switch method {
	case "hex":
		bytesNeeded := (n + 1) / 2
		buf := make([]byte, bytesNeeded)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate hex secret: %w", err)
		}
		out := hex.EncodeToString(buf)
		if len(out) > n {
			out = out[:n]
		}
		return out, nil
	case "base64":
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate base64 secret: %w", err)
		}
		enc := base64.RawStdEncoding.EncodeToString(buf)
		if len(enc) > n {
			enc = enc[:n]
		}
		return enc, nil
	case "alnum":
		const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate alnum secret: %w", err)
		}
		out := make([]byte, n)
		for i := range buf {
			out[i] = alphabet[int(buf[i])%len(alphabet)]
		}
		return string(out), nil
	default:
		return "", fmt.Errorf("unsupported generate method %q", method)
	}
}

func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

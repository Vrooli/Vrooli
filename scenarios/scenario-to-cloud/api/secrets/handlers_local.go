package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

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
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "read_local_secrets_failed",
				Message: "Failed to read local secrets file",
				Hint:    err.Error(),
			})
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

		payload, err := readLocalSecretsFile(secretsPath)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "read_local_secrets_failed",
				Message: "Failed to read local secrets file",
				Hint:    err.Error(),
			})
			return
		}

		payload[key] = value
		if err := writeLocalSecretsFile(secretsPath, payload); err != nil {
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "write_local_secrets_failed",
				Message: "Failed to write local secrets file",
				Hint:    err.Error(),
			})
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

		payload, err := readLocalSecretsFile(secretsPath)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "read_local_secrets_failed",
				Message: "Failed to read local secrets file",
				Hint:    err.Error(),
			})
			return
		}

		if _, exists := payload[key]; !exists {
			httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
				Code:    "secret_not_found",
				Message: fmt.Sprintf("Secret %q not found", key),
			})
			return
		}
		delete(payload, key)

		if err := writeLocalSecretsFile(secretsPath, payload); err != nil {
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "write_local_secrets_failed",
				Message: "Failed to write local secrets file",
				Hint:    err.Error(),
			})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, domain.NewSecretOperationResponse(true, key, "deleted", "Local secret deleted successfully"))
	}
}

func resolveLocalSecretsPath(scope, scenarioID string) (string, error) {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return "", fmt.Errorf("repo root not found: %w", err)
	}

	switch scope {
	case localSecretsScopeWorkspace:
		return filepath.Join(repoRoot, ".vrooli", "secrets.json"), nil
	case localSecretsScopeScenario:
		if strings.TrimSpace(scenarioID) == "" {
			return "", fmt.Errorf("scenario_id is required for scope=%q", localSecretsScopeScenario)
		}
		if strings.Contains(scenarioID, "..") || strings.ContainsAny(scenarioID, `/\`) {
			return "", fmt.Errorf("invalid scenario_id %q", scenarioID)
		}
		return filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "secrets.json"), nil
	default:
		return "", fmt.Errorf("unknown scope %q (valid scopes: %s, %s)", scope, localSecretsScopeWorkspace, localSecretsScopeScenario)
	}
}

func readLocalSecretsFile(path string) (map[string]string, error) {
	out := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return out, nil
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		if str, ok := v.(string); ok {
			out[k] = str
		}
	}
	return out, nil
}

func writeLocalSecretsFile(path string, payload map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	doc := map[string]interface{}{
		"_metadata": map[string]interface{}{
			"managed_by":   "scenario-to-cloud",
			"last_updated": time.Now().UTC().Format(time.RFC3339),
		},
	}
	for k, v := range payload {
		doc[k] = v
	}

	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o600)
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

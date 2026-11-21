package ensure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
)

const ollamaManagedLifecycleLockKey = "resource-lifecycle:ollama"

// acquireManagedOllamaLifecycleLock uses the exact cross-process lock key held
// by Vrooli's managed-service start/restart/stop owner. Keep this coupled to
// internal/resources/lifecycle_lock.go; changing either key would reopen the
// replacement-listener race.
func acquireManagedOllamaLifecycleLock(ctx context.Context) (func(), error) {
	return artifactlock.AcquireContext(ctx, ollamaManagedLifecycleLockKey)
}

func acquireManagedOllamaLifecycleLockWithPath(ctx context.Context, pathFn func(string) (string, error)) (func(), error) {
	return artifactlock.AcquireWithPathContext(ctx, ollamaManagedLifecycleLockKey, pathFn)
}

const managedServiceTokenEnv = "VROOLI_MANAGED_SERVICE_INSTANCE_TOKEN"

type managedOllamaIdentity struct {
	PID         int
	Incarnation string
	ModelRoot   string
	BaseURL     string
}

// resolveManagedOllamaClient binds a missing-model pull to the process Vrooli
// owns for resource-ollama. A loopback address alone is not enough: another
// local Ollama can use a different model store while receiving the same API
// request.
func resolveManagedOllamaClient(ctx context.Context, expectedRoot string) (*Client, managedOllamaIdentity, error) {
	identity, err := inspectManagedOllama(expectedRoot)
	if err != nil {
		return nil, managedOllamaIdentity{}, err
	}
	client, err := newLoopbackClient(identity.BaseURL, net.DefaultResolver.LookupIP)
	if err != nil {
		return nil, managedOllamaIdentity{}, fmt.Errorf("managed Ollama listener is not local: %w", err)
	}
	identity.BaseURL = client.BaseURL
	if err := verifyManagedOllama(ctx, identity); err != nil {
		return nil, managedOllamaIdentity{}, err
	}
	return client, identity, nil
}

func inspectManagedOllama(expectedRoot string) (managedOllamaIdentity, error) {
	if !filepath.IsAbs(expectedRoot) {
		return managedOllamaIdentity{}, errors.New("expected Ollama model root is not absolute")
	}
	records, err := resources.ReadManagedServiceRecords()
	if err != nil {
		return managedOllamaIdentity{}, fmt.Errorf("read managed Ollama owner record: %w", err)
	}
	var state *resources.ManagedServiceState
	for i := range records {
		if records[i].Resource == "ollama" {
			state = &records[i].State
			break
		}
	}
	if state == nil || state.PID <= 0 {
		return managedOllamaIdentity{}, errors.New("managed Ollama process identity is unavailable; refusing model pull")
	}
	incarnation, err := process.Incarnation(state.PID)
	if err != nil {
		return managedOllamaIdentity{}, fmt.Errorf("read managed Ollama process incarnation: %w", err)
	}
	env, err := process.ReadEnvironment(state.PID)
	if err != nil {
		return managedOllamaIdentity{}, fmt.Errorf("read managed Ollama process environment: %w", err)
	}
	return managedOllamaIdentityFromOwner(*state, expectedRoot, process.IsPIDRunning(state.PID), incarnation, env)
}

func managedOllamaIdentityFromOwner(state resources.ManagedServiceState, expectedRoot string, processAlive bool, actualIncarnation string, env map[string]string) (managedOllamaIdentity, error) {
	var err error
	if state.PID <= 0 || state.Incarnation == "" || state.OwnershipTokenHash == "" || !processAlive {
		return managedOllamaIdentity{}, errors.New("managed Ollama process identity is unavailable; refusing model pull")
	}
	if actualIncarnation != state.Incarnation {
		return managedOllamaIdentity{}, errors.New("managed Ollama process incarnation does not match its owner record")
	}
	token := env[managedServiceTokenEnv]
	tokenHash := sha256.Sum256([]byte(token))
	if token == "" || hex.EncodeToString(tokenHash[:]) != state.OwnershipTokenHash {
		return managedOllamaIdentity{}, errors.New("managed Ollama process ownership token does not match its owner record")
	}
	modelRoot := strings.TrimSpace(env["OLLAMA_MODELS"])
	if !filepath.IsAbs(modelRoot) {
		return managedOllamaIdentity{}, errors.New("managed Ollama process has no absolute OLLAMA_MODELS path")
	}
	modelRoot, err = filepath.EvalSymlinks(modelRoot)
	if err != nil {
		return managedOllamaIdentity{}, fmt.Errorf("resolve managed Ollama OLLAMA_MODELS: %w", err)
	}
	expectedRoot, err = filepath.EvalSymlinks(expectedRoot)
	if err != nil || filepath.Clean(modelRoot) != filepath.Clean(expectedRoot) {
		return managedOllamaIdentity{}, errors.New("managed Ollama OLLAMA_MODELS does not match the admitted model-store path")
	}
	baseURL, err := baseURLFromOllamaHost(env["OLLAMA_HOST"])
	if err != nil {
		return managedOllamaIdentity{}, err
	}
	localClient, err := newLoopbackClient(baseURL, net.DefaultResolver.LookupIP)
	if err != nil {
		return managedOllamaIdentity{}, fmt.Errorf("managed Ollama listener is not local: %w", err)
	}
	baseURL = localClient.BaseURL
	return managedOllamaIdentity{PID: state.PID, Incarnation: state.Incarnation, ModelRoot: modelRoot, BaseURL: baseURL}, nil
}

func verifyManagedOllama(ctx context.Context, expected managedOllamaIdentity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := inspectManagedOllama(expected.ModelRoot)
	if err != nil {
		return err
	}
	if current.PID != expected.PID || current.Incarnation != expected.Incarnation || current.ModelRoot != expected.ModelRoot || current.BaseURL != expected.BaseURL {
		return errors.New("managed Ollama process or model-store configuration changed during acquisition")
	}
	return nil
}

func baseURLFromOllamaHost(raw string) (string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil || host == "" || port == "" {
		return "", errors.New("managed Ollama process has an invalid OLLAMA_HOST listener")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", errors.New("managed Ollama process has an invalid OLLAMA_HOST port")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsLoopback() {
			return "", errors.New("managed Ollama OLLAMA_HOST is not loopback")
		}
	} else if !strings.EqualFold(host, "localhost") {
		return "", errors.New("managed Ollama OLLAMA_HOST must be a loopback address")
	}
	u := &url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}
	return u.String(), nil
}

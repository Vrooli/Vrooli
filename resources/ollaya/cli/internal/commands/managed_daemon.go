package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

const ollayaManagedLifecycleLockKey = "resource-lifecycle:ollaya"
const managedServiceTokenEnv = "VROOLI_MANAGED_SERVICE_INSTANCE_TOKEN"

type managedOllayaIdentity struct {
	PID         int
	Incarnation string
	ModelRoot   string
	BaseURL     string
	PublicURL   string
}

// prepareManagedModelPull holds the exact control-plane lifecycle lock for
// this resource from owner inspection through the last pull receipt. The
// private daemon endpoint comes from the authenticated owner process, never
// from the caller's OLLAYA_URL.
func prepareManagedModelPull(ctx context.Context, getenv func(string) string) (string, *upstream.Client, func(context.Context) error, func(), error) {
	release, err := artifactlock.AcquireContext(ctx, ollayaManagedLifecycleLockKey)
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("acquire managed Ollaya lifecycle lock: %w", err)
	}
	failed := true
	defer func() {
		if failed {
			release()
		}
	}()
	root, err := ollayaModelRoot(getenv)
	if err != nil {
		return "", nil, nil, nil, err
	}
	identity, err := inspectManagedOllaya(root, getenv)
	if err != nil {
		return "", nil, nil, nil, err
	}
	verify := func(checkCtx context.Context) error { return verifyManagedOllaya(checkCtx, identity, getenv) }
	if err := verify(ctx); err != nil {
		return "", nil, nil, nil, err
	}
	httpClient := &http.Client{
		Transport:     &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	bound := &upstream.Client{
		HTTP: httpClient,
		GetEnv: func(key string) string {
			if key == "OLLAYA_URL" {
				return identity.BaseURL
			}
			if getenv == nil {
				return ""
			}
			return getenv(key)
		},
		Now: time.Now,
	}
	failed = false
	return root, bound, verify, release, nil
}

func inspectManagedOllaya(expectedRoot string, getenv func(string) string) (managedOllayaIdentity, error) {
	if !filepath.IsAbs(expectedRoot) {
		return managedOllayaIdentity{}, errors.New("expected Ollaya model root is not absolute")
	}
	records, err := resources.ReadManagedServiceRecords()
	if err != nil {
		return managedOllayaIdentity{}, fmt.Errorf("read managed Ollaya owner record: %w", err)
	}
	var state *resources.ManagedServiceState
	for i := range records {
		if records[i].Resource == "ollaya" {
			state = &records[i].State
			break
		}
	}
	if state == nil || state.PID <= 0 {
		return managedOllayaIdentity{}, errors.New("managed Ollaya process identity is unavailable; refusing model pull")
	}
	incarnation, err := process.Incarnation(state.PID)
	if err != nil {
		return managedOllayaIdentity{}, fmt.Errorf("read managed Ollaya process incarnation: %w", err)
	}
	env, err := process.ReadEnvironment(state.PID)
	if err != nil {
		return managedOllayaIdentity{}, fmt.Errorf("read managed Ollaya process environment: %w", err)
	}
	return managedOllayaIdentityFromOwner(*state, expectedRoot, process.IsPIDRunning(state.PID), incarnation, env, resolvedOllayaURL(getenv))
}

func managedOllayaIdentityFromOwner(state resources.ManagedServiceState, expectedRoot string, processAlive bool, actualIncarnation string, env map[string]string, requestedURL string) (managedOllayaIdentity, error) {
	if state.PID <= 0 || state.Incarnation == "" || state.OwnershipTokenHash == "" || !processAlive {
		return managedOllayaIdentity{}, errors.New("managed Ollaya process identity is unavailable; refusing model pull")
	}
	if actualIncarnation != state.Incarnation {
		return managedOllayaIdentity{}, errors.New("managed Ollaya process incarnation does not match its owner record")
	}
	token := env[managedServiceTokenEnv]
	tokenHash := sha256.Sum256([]byte(token))
	if token == "" || hex.EncodeToString(tokenHash[:]) != state.OwnershipTokenHash {
		return managedOllayaIdentity{}, errors.New("managed Ollaya process ownership token does not match its owner record")
	}
	modelRoot := strings.TrimSpace(env["OLLAYA_MODELS"])
	if !filepath.IsAbs(modelRoot) {
		return managedOllayaIdentity{}, errors.New("managed Ollaya process has no absolute OLLAYA_MODELS path")
	}
	modelRoot, err := filepath.EvalSymlinks(modelRoot)
	if err != nil {
		return managedOllayaIdentity{}, fmt.Errorf("resolve managed Ollaya OLLAYA_MODELS: %w", err)
	}
	expectedRoot, err = filepath.EvalSymlinks(expectedRoot)
	if err != nil || filepath.Clean(modelRoot) != filepath.Clean(expectedRoot) {
		return managedOllayaIdentity{}, errors.New("managed Ollaya OLLAYA_MODELS does not match the admitted model-store path")
	}
	baseURL, err := baseURLFromOllayaHost(env["OLLAYA_HOST"])
	if err != nil {
		return managedOllayaIdentity{}, err
	}
	publicURL, err := publicOllayaURLFromOwnerEnv(env)
	if err != nil {
		return managedOllayaIdentity{}, err
	}
	if requestedURL == "" || !sameLoopbackRootURL(requestedURL, publicURL) {
		return managedOllayaIdentity{}, errors.New("configured OLLAYA_URL does not match the managed Ollaya loopback endpoint")
	}
	return managedOllayaIdentity{PID: state.PID, Incarnation: state.Incarnation, ModelRoot: modelRoot, BaseURL: baseURL, PublicURL: publicURL}, nil
}

func verifyManagedOllaya(ctx context.Context, expected managedOllayaIdentity, getenv func(string) string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := inspectManagedOllaya(expected.ModelRoot, getenv)
	if err != nil {
		return err
	}
	if current.PID != expected.PID || current.Incarnation != expected.Incarnation || current.ModelRoot != expected.ModelRoot || current.BaseURL != expected.BaseURL || current.PublicURL != expected.PublicURL {
		return errors.New("managed Ollaya process, endpoint, or model-store configuration changed during acquisition")
	}
	return nil
}

func baseURLFromOllayaHost(raw string) (string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil || host == "" || port == "" {
		return "", errors.New("managed Ollaya process has an invalid OLLAYA_HOST listener")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", errors.New("managed Ollaya process has an invalid OLLAYA_HOST port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", errors.New("managed Ollaya OLLAYA_HOST must be a literal loopback address")
	}
	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(ip.String(), port)}).String(), nil
}

func publicOllayaURLFromOwnerEnv(env map[string]string) (string, error) {
	port := strings.TrimSpace(env["RESOURCE_PORT_HTTP"])
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", errors.New("managed Ollaya owner record lacks a valid public loopback port")
	}
	return (&url.URL{Scheme: "http", Host: net.JoinHostPort("localhost", port)}).String(), nil
}

func resolvedOllayaURL(getenv func(string) string) string {
	if getenv == nil {
		return upstream.LocalURL(func(string) string { return "" })
	}
	return upstream.LocalURL(getenv)
}

func sameLoopbackRootURL(candidate, expected string) bool {
	a, errA := url.Parse(strings.TrimSpace(candidate))
	b, errB := url.Parse(strings.TrimSpace(expected))
	if errA != nil || errB != nil || a.Scheme != "http" || b.Scheme != "http" || a.User != nil || b.User != nil || a.Path != "" && a.Path != "/" || a.RawQuery != "" || a.Fragment != "" {
		return false
	}
	if b.Path != "" || b.RawQuery != "" || b.Fragment != "" {
		return false
	}
	host := a.Hostname()
	if !strings.EqualFold(host, "localhost") && host != "127.0.0.1" {
		return false
	}
	return a.Port() != "" && a.Port() == b.Port()
}

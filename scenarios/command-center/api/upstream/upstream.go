// Package upstream provides the descriptor-backed read clients used by
// command-center. Producer-specific generated clients do not belong here:
// source behavior is described in config and interpreted by descriptor.go.
package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

var ErrNotAvailable = errors.New("upstream not available")

// Client is the stable raw projection boundary between connectors and the
// domain registry. The connector runtime, not a producer SDK, implements it.
type Client interface {
	Name() string
	Fetch(context.Context, string) (json.RawMessage, error)
}

type FeatureProbe interface {
	ProbeFeatures(context.Context) (map[string]string, map[string]string)
}

// NewRESTResolved creates a descriptor-backed JSON client for a lifecycle-
// resolved REST origin. bearerToken is retained as a call-site convenience;
// production credentials are supplied through the descriptor auth callback.
func NewRESTResolved(name string, resolve func() string, bearerToken string) Client {
	return NewDescriptorClient(Descriptor{
		ID:          name,
		Transport:   TransportHTTPJSON,
		ResolveBase: resolve,
		Auth: func(req *http.Request) {
			if bearerToken != "" {
				req.Header.Set("Authorization", "Bearer "+bearerToken)
			}
		},
	})
}

// NewJSONConnectResolvedPaths creates a descriptor-backed Connect client.
func NewJSONConnectResolvedPaths(name string, resolve func() string, paths []string, features ...map[string]string) Client {
	return NewDescriptorClient(Descriptor{
		ID:          name,
		Transport:   TransportConnect,
		ResolveBase: resolve,
		Paths:       paths,
		Features:    optionalFeatureSet(features),
	})
}

func NewJSONConnectResolved(name string, resolve func() string, path string, features ...map[string]string) Client {
	return NewJSONConnectResolvedPaths(name, resolve, []string{path}, features...)
}

func NewSwarm(baseURL string) Client {
	return NewRESTResolved("swarm", func() string { return baseURL }, "")
}

func NewSwarmResolved(resolve func() string) Client {
	return NewRESTResolved("swarm", resolve, "")
}

func NewVrooli(baseURL string) Client {
	return NewRESTResolved("vrooli", func() string { return baseURL }, "")
}

func NewLPBS(baseURL, bearerToken string) Client {
	return NewRESTResolved("lpbs", func() string { return baseURL }, bearerToken)
}

func NewLPBSResolved(resolve func() string, bearerToken string) Client {
	return NewRESTResolved("lpbs", resolve, bearerToken)
}

// NewLPBSRemoteProfileResolved routes production reads through the local LPBS
// remote-profile broker. The broker owns the deployed service credential; the
// caller only supplies the local LPBS service credential and a profile tag.
func NewLPBSRemoteProfileResolved(resolveBase, resolveServiceToken func() string, profileTag string) Client {
	return &remoteProfileClient{
		resolveBase:         resolveBase,
		resolveServiceToken: resolveServiceToken,
		profileTag:          profileTag,
		http:                &http.Client{Timeout: 5 * time.Second},
	}
}

func optionalFeatureSet(features []map[string]string) map[string]string {
	if len(features) == 0 || features[0] == nil {
		return nil
	}
	return features[0]
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

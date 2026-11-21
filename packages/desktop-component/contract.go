// Package desktopcomponent owns the signed, engine-independent optional desktop
// component contract shared by release production, LPBS delivery and runtime admission.
package desktopcomponent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/platform-go/versionrange"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type Source struct {
	CatalogURL   string `json:"catalog_url"`
	PublicKeyPEM string `json:"public_key_pem"`
}

type Entry struct {
	SchemaVersion int             `json:"schema_version"`
	ID            string          `json:"id"`
	Owner         string          `json:"owner"`
	Slice         string          `json:"slice,omitempty"`
	Version       string          `json:"version"`
	OS            string          `json:"os"`
	Arch          string          `json:"arch"`
	HostRange     string          `json:"host_range"`
	Provides      []Feature       `json:"provides"`
	StateClass    string          `json:"state_class"`
	StandaloneURL string          `json:"standalone_url,omitempty"`
	License       string          `json:"license"`
	ArtifactURL   string          `json:"artifact_url"`
	SHA256        string          `json:"sha256"`
	TreeSHA256    string          `json:"tree_sha256"`
	Size          int64           `json:"size"`
	Manifest      json.RawMessage `json:"manifest"`
}

type Feature struct {
	ID      string `json:"id"`
	Service string `json:"service"`
	Port    string `json:"port"`
}

type SignedEntry struct {
	Entry     Entry                               `json:"entry"`
	Signature resourcedeployment.ReleaseSignature `json:"signature"`
}

var (
	identifier = regexp.MustCompile(`^[a-z][a-z0-9.-]*$`)
	digest     = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

func ValidID(value string) bool {
	return identifier.MatchString(value) && filepath.IsLocal(value) && !strings.Contains(value, "..")
}

// CanonicalBytes is the producer/consumer signing contract. Object member order
// and whitespace in the nested service manifest are not release facts: LPBS's
// JSONB catalog may normalize both. Sort those objects recursively, preserving
// array order and JSON numbers, before encoding the fixed entry fields.
func (e Entry) CanonicalBytes() ([]byte, error) {
	if e.SchemaVersion != 1 || !ValidID(e.ID) || !ValidID(e.Owner) || (e.Slice != "" && !ValidID(e.Slice)) {
		return nil, fmt.Errorf("invalid component identity")
	}
	if e.Version == "" || strings.TrimSpace(e.HostRange) == "" || (!digest.MatchString(e.SHA256) || !digest.MatchString(e.TreeSHA256)) || e.Size <= 0 || e.Size > 2<<30 {
		return nil, fmt.Errorf("invalid component release bounds")
	}
	if e.OS != "linux" && e.OS != "darwin" && e.OS != "windows" {
		return nil, fmt.Errorf("unsupported component OS")
	}
	if e.Arch != "amd64" && e.Arch != "arm64" {
		return nil, fmt.Errorf("unsupported component architecture")
	}
	if e.StateClass != "stateless" {
		return nil, fmt.Errorf("stateful standalone components require their own governed data contract")
	}
	for _, raw := range []string{e.ArtifactURL, e.StandaloneURL} {
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1"))) {
			return nil, fmt.Errorf("invalid component URL")
		}
	}
	if e.ArtifactURL == "" || e.License == "" || len(e.Provides) == 0 || !json.Valid(e.Manifest) {
		return nil, fmt.Errorf("incomplete component metadata")
	}
	if err := e.ValidateFeatures(); err != nil {
		return nil, err
	}
	var serviceManifest any
	decoder := json.NewDecoder(bytes.NewReader(e.Manifest))
	decoder.UseNumber()
	if err := decoder.Decode(&serviceManifest); err != nil {
		return nil, fmt.Errorf("invalid component service manifest: %w", err)
	}
	var err error
	e.Manifest, err = json.Marshal(serviceManifest)
	if err != nil {
		return nil, err
	}
	return json.Marshal(e)
}

// ValidateFeatures binds each exported feature to this release's own projected
// service and declared port. Producers, signature admission, and stored-install
// admission share this check; it must run before fetching or merging a component.
// Decode only the fields used here so unknown manifest fields remain signed facts.
func (e Entry) ValidateFeatures() error {
	var manifest struct {
		App struct {
			Scenario string `json:"scenario"`
			Version  string `json:"version"`
		} `json:"app"`
		Services []struct {
			ID    string `json:"id"`
			Ports struct {
				Requested []struct {
					Name string `json:"name"`
				} `json:"requested"`
			} `json:"ports"`
			Health struct {
				Type     string   `json:"type"`
				PortName string   `json:"port_name"`
				Path     string   `json:"path"`
				Command  []string `json:"command"`
			} `json:"health"`
			Readiness struct {
				Type     string `json:"type"`
				PortName string `json:"port_name"`
				Pattern  string `json:"pattern"`
			} `json:"readiness"`
			LogDir string `json:"log_dir"`
		} `json:"services"`
	}
	if err := json.Unmarshal(e.Manifest, &manifest); err != nil {
		return fmt.Errorf("invalid component service manifest: %w", err)
	}
	if !ValidID(e.Owner) || manifest.App.Scenario != e.Owner || manifest.App.Version != e.Version {
		return fmt.Errorf("component owner/version does not match its service manifest")
	}
	services := map[string]int{}
	for i, service := range manifest.Services {
		if !ValidID(service.ID) || !strings.HasPrefix(service.ID, e.Owner+"--") {
			return fmt.Errorf("component service %q is outside its owner's projected graph", service.ID)
		}
		if _, found := services[service.ID]; found {
			return fmt.Errorf("ambiguous component service %q", service.ID)
		}
		services[service.ID] = i
	}
	seen := map[string]bool{}
	if len(e.Provides) == 0 {
		return fmt.Errorf("component requires exported features")
	}
	for _, feature := range e.Provides {
		if !ValidID(feature.ID) || !ValidID(feature.Service) || !ValidID(feature.Port) || seen[feature.ID] {
			return fmt.Errorf("invalid or ambiguous component feature")
		}
		seen[feature.ID] = true
		i, found := services[feature.Service]
		if !found {
			return fmt.Errorf("component feature %q selects a service outside its projected graph", feature.ID)
		}
		service := manifest.Services[i]
		ports := map[string]bool{}
		for _, port := range service.Ports.Requested {
			if !ValidID(port.Name) || ports[port.Name] {
				return fmt.Errorf("invalid or ambiguous component service port")
			}
			ports[port.Name] = true
		}
		if !ports[feature.Port] {
			return fmt.Errorf("component feature %q selects an undeclared service port", feature.ID)
		}
		usablePattern := func(pattern string) bool {
			_, err := regexp.Compile(pattern)
			return pattern != "" && service.LogDir != "" && err == nil
		}
		healthOK := false
		switch service.Health.Type {
		case "http", "tcp":
			healthOK = ports[service.Health.PortName]
		case "command":
			healthOK = len(service.Health.Command) > 0 && strings.TrimSpace(service.Health.Command[0]) != ""
		case "log_match":
			healthOK = usablePattern(service.Health.Path)
		}
		readinessOK := false
		switch service.Readiness.Type {
		case "health_success", "dependencies_ready":
			readinessOK = true
		case "port_open":
			readinessOK = ports[service.Readiness.PortName]
		case "log_match":
			readinessOK = usablePattern(service.Readiness.Pattern)
		}
		if !healthOK || !readinessOK {
			return fmt.Errorf("component feature %q requires usable service health and readiness", feature.ID)
		}
	}
	return nil
}

func (e SignedEntry) Verify(publicKeyPEM string) error {
	canonical, err := e.Entry.CanonicalBytes()
	if err != nil {
		return err
	}
	return resourcedeployment.VerifyReleaseSignature(canonical, e.Signature, []byte(publicKeyPEM))
}

func (e Entry) Compatible(version, osName, arch string) bool {
	return e.OS == osName && e.Arch == arch && versionrange.Satisfies(version, e.HostRange)
}

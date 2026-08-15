// Package capabilityledger generates the capability readout directly from
// tool and safeguard manifests. It intentionally has no authored fleet table.
package capabilityledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/deployability"
)

var operatingSystems = []deployability.HostOS{
	deployability.HostOSLinux,
	deployability.HostOSMacOS,
	deployability.HostOSWindows,
}

type manifest struct {
	Name                 string                         `json:"name"`
	Capability           string                         `json:"capability"`
	Role                 string                         `json:"capability_role"`
	Platforms            []string                       `json:"platforms"`
	PlatformDeclarations map[string]platformDeclaration `json:"platform_declarations,omitempty"`
	Packages             map[string]interface{}         `json:"packages"`
	Source               json.RawMessage                `json:"source"`
	Handler              json.RawMessage                `json:"handler"`
	Manual               bool                           `json:"manual"`
}

type platformDeclaration struct {
	Status    string `json:"status"`
	Mechanism string `json:"mechanism"`
}

type Vocabulary struct {
	Capabilities     []string                     `json:"capabilities"`
	PlatformPolicies map[string]map[string]string `json:"platform_policies,omitempty"`
}

type CapabilitySituation string

const (
	SituationBuiltEverywhere  CapabilitySituation = "built_everywhere"
	SituationNoWorkRequired   CapabilitySituation = "no_work_required"
	SituationNoEquivalentEver CapabilitySituation = "no_equivalent_ever"
	SituationPeerNobodyWired  CapabilitySituation = "real_peer_nobody_wired"
)

type PlatformEntry struct {
	Status      deployability.CapabilityResolutionStatus `json:"status"`
	Implementer string                                   `json:"implementer,omitempty"`
	Mechanism   string                                   `json:"mechanism,omitempty"`
	Reason      string                                   `json:"reason"`
}

type Entry struct {
	Capability      string                   `json:"capability"`
	Situation       CapabilitySituation      `json:"situation"`
	SituationReason string                   `json:"situation_reason"`
	Platforms       map[string]PlatformEntry `json:"platforms"`
}

type Ledger struct {
	Capabilities []Entry `json:"capabilities"`
}

func Generate(root string) (Ledger, error) {
	vocabulary, err := readVocabulary(filepath.Join(root, ".vrooli", "capability-vocabulary.json"))
	if err != nil {
		return Ledger{}, err
	}
	manifests, err := readManifests(root)
	if err != nil {
		return Ledger{}, err
	}
	if err := validateManifestDeclarations(manifests, vocabulary.Capabilities); err != nil {
		return Ledger{}, err
	}
	implementations := make([]deployability.CapabilityImplementation, 0, len(manifests))
	for _, item := range manifests {
		platforms := make(map[deployability.HostOS]deployability.PlatformDeclaration, len(operatingSystems))
		for _, hostOS := range operatingSystems {
			platforms[hostOS] = deployability.PlatformDeclaration{Status: "unsupported", Mechanism: mechanism(item, hostOS)}
		}
		for _, declared := range item.Platforms {
			hostOS := deployability.HostOS(strings.TrimSpace(declared))
			if _, ok := platforms[hostOS]; ok {
				platforms[hostOS] = deployability.PlatformDeclaration{Status: "supported"}
			}
		}
		for declaredOS, declaration := range item.PlatformDeclarations {
			if hostOS, ok := normalizeHostOS(declaredOS); ok {
				platforms[hostOS] = deployability.PlatformDeclaration{Status: declaration.Status, Mechanism: declaration.Mechanism}
			}
		}
		implementations = append(implementations, deployability.CapabilityImplementation{
			Name: item.Name, Capability: item.Capability, Role: item.Role, Platforms: platforms,
		})
	}

	ledger := Ledger{Capabilities: make([]Entry, 0, len(vocabulary.Capabilities))}
	for _, capability := range vocabulary.Capabilities {
		entry := Entry{Capability: capability, Platforms: make(map[string]PlatformEntry, len(operatingSystems))}
		statuses := make(map[deployability.HostOS]deployability.CapabilityResolutionStatus, len(operatingSystems))
		for _, hostOS := range operatingSystems {
			resolution := deployability.ResolveCapability(implementations, capability, hostOS)
			statuses[hostOS] = resolution.Status
			entry.Platforms[string(hostOS)] = PlatformEntry{
				Status: resolution.Status, Implementer: resolution.Implementer,
				Mechanism: resolution.Mechanism, Reason: resolution.Reason,
			}
		}
		entry.Situation, entry.SituationReason = classifySituation(capability, statuses, vocabulary.PlatformPolicies)
		ledger.Capabilities = append(ledger.Capabilities, entry)
	}
	return ledger, nil
}

func validateManifestDeclarations(manifests []manifest, vocabulary []string) error {
	allowed := make(map[string]struct{}, len(vocabulary))
	for _, capability := range vocabulary {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			return fmt.Errorf("capability vocabulary contains an empty entry")
		}
		if _, duplicate := allowed[capability]; duplicate {
			return fmt.Errorf("capability vocabulary duplicates %q", capability)
		}
		allowed[capability] = struct{}{}
	}
	for _, item := range manifests {
		capability := strings.TrimSpace(item.Capability)
		if capability == "" {
			return fmt.Errorf("capability manifest %q has no capability", item.Name)
		}
		if _, ok := allowed[capability]; !ok {
			return fmt.Errorf("capability manifest %q declares unknown capability %q", item.Name, capability)
		}
		switch strings.TrimSpace(item.Role) {
		case "primary", "peer":
		default:
			return fmt.Errorf("capability manifest %q has invalid capability_role %q", item.Name, item.Role)
		}
	}
	return nil
}

func classifySituation(capability string, statuses map[deployability.HostOS]deployability.CapabilityResolutionStatus, policies map[string]map[string]string) (CapabilitySituation, string) {
	if policy := policies[capability]; len(policy) > 0 {
		noEquivalent := false
		noWork := false
		for hostOS, value := range policy {
			switch value {
			case string(SituationNoWorkRequired):
				noWork = true
			case string(SituationNoEquivalentEver):
				noEquivalent = true
			default:
				return SituationNoEquivalentEver, "the capability policy contains an unsupported value for " + hostOS
			}
		}
		if noEquivalent {
			return SituationNoEquivalentEver, "the capability policy declares no equivalent for at least one OS or architecture"
		}
		if noWork {
			return SituationNoWorkRequired, "the capability policy declares the host OS mechanism native and requiring no setup"
		}
	}
	allImplemented := true
	for _, status := range statuses {
		if status != deployability.CapabilityImplemented {
			allImplemented = false
			break
		}
	}
	if allImplemented {
		return SituationBuiltEverywhere, "a declared implementation resolves on linux, macos, and windows"
	}
	for _, status := range statuses {
		if status == deployability.CapabilityUnwired {
			return SituationPeerNobodyWired, "a mechanism is named for an unsupported host OS, but no peer implementation is wired"
		}
	}
	return SituationNoEquivalentEver, "no implementation or mechanism resolves on at least one host OS"
}

func readVocabulary(path string) (Vocabulary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Vocabulary{}, fmt.Errorf("read capability vocabulary: %w", err)
	}
	var vocabulary Vocabulary
	if err := json.Unmarshal(data, &vocabulary); err != nil {
		return Vocabulary{}, fmt.Errorf("decode capability vocabulary: %w", err)
	}
	sort.Strings(vocabulary.Capabilities)
	return vocabulary, nil
}

func readManifests(root string) ([]manifest, error) {
	paths := make([]string, 0)
	for _, pattern := range []string{"internal/tools/*/tool.json", "internal/safeguards/*/safeguard.json"} {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	manifests := make([]manifest, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read capability manifest %s: %w", path, err)
		}
		var item manifest
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, fmt.Errorf("decode capability manifest %s: %w", path, err)
		}
		if strings.TrimSpace(item.Name) == "" {
			item.Name = filepath.Base(filepath.Dir(path))
		}
		manifests = append(manifests, item)
	}
	servicePaths, err := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range servicePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read service capability manifest %s: %w", path, err)
		}
		var service struct {
			Service struct {
				Name                 string                                    `json:"name"`
				Capabilities         []string                                  `json:"capabilities"`
				PlatformCapabilities map[string]map[string]platformDeclaration `json:"platform_capabilities"`
			} `json:"service"`
		}
		if err := json.Unmarshal(data, &service); err != nil {
			return nil, fmt.Errorf("decode service capability manifest %s: %w", path, err)
		}
		for capability, declarations := range service.Service.PlatformCapabilities {
			if strings.TrimSpace(capability) == "" {
				continue
			}
			name := strings.TrimSpace(service.Service.Name)
			if name == "" {
				name = filepath.Base(filepath.Dir(filepath.Dir(path)))
			}
			manifests = append(manifests, manifest{
				Name: name + "/" + capability, Capability: capability, Role: "primary",
				PlatformDeclarations: declarations,
			})
		}
	}
	sort.Slice(manifests, func(i, j int) bool {
		if manifests[i].Capability != manifests[j].Capability {
			return manifests[i].Capability < manifests[j].Capability
		}
		return manifests[i].Name < manifests[j].Name
	})
	return manifests, nil
}

func mechanism(item manifest, hostOS deployability.HostOS) string {
	if hostOS == deployability.HostOSMacOS && item.Packages["brew"] != nil {
		return "brew"
	}
	if rawValuePresent(item.Source) {
		return "source"
	}
	if rawValuePresent(item.Handler) {
		return "handler"
	}
	if item.Manual {
		return "manual"
	}
	return ""
}

func rawValuePresent(value json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(value))
	return trimmed != "" && trimmed != "null" && trimmed != `""` && trimmed != "{}"
}

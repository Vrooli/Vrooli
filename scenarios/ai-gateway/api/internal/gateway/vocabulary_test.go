package gateway_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	measureshandler "ai-gateway/handlers/measures"
	"ai-gateway/internal/gateway"
	"ai-gateway/internal/gatewaycatalog"
)

// vocabScenarioRoot resolves scenarios/ai-gateway from this test package.
func vocabScenarioRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(dir, "..", "..", ".."))
}

func normalizeProfile(protoName string) string {
	name := strings.ToLower(strings.TrimPrefix(protoName, "PROFILE_"))
	return strings.ReplaceAll(name, "_", "-")
}

func acceptedProfileSet(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, profile := range gateway.AcceptedProfiles() {
		if out[profile] {
			t.Fatalf("AcceptedProfiles declares %q more than once", profile)
		}
		out[profile] = true
	}
	return out
}

// AcceptedProfiles is the runtime vocabulary. It must equal the selectable
// proto Profile enum values, not drift from them.
func TestAcceptedProfilesMatchProtoEnum(t *testing.T) {
	accepted := acceptedProfileSet(t)
	expected := map[string]bool{}
	for _, name := range sharedv1.Profile_name {
		if name == sharedv1.Profile_PROFILE_UNSPECIFIED.String() {
			continue
		}
		expected[normalizeProfile(name)] = true
	}
	if len(accepted) != len(expected) {
		t.Fatalf("AcceptedProfiles has %d values, proto has %d selectable values", len(accepted), len(expected))
	}
	for profile := range expected {
		if !accepted[profile] {
			t.Errorf("AcceptedProfiles is missing proto profile %q", profile)
		}
	}
	for profile := range accepted {
		if !expected[profile] {
			t.Errorf("AcceptedProfiles contains %q, which has no proto Profile value", profile)
		}
	}
}

// The data catalog is the operator-vocabulary source of truth. Every accepted
// profile must have exactly one catalog row, and the catalog must not invent
// profiles the gateway cannot accept.
func TestAcceptedProfilesHaveCatalogRows(t *testing.T) {
	profiles, err := gatewaycatalog.LoadProfiles(filepath.Join(vocabScenarioRoot(t), "config", "profile-catalog.json"))
	if err != nil {
		t.Fatalf("load profile catalog: %v", err)
	}
	accepted := acceptedProfileSet(t)
	declared := map[string]bool{}
	for _, row := range profiles.Profiles {
		if declared[row.Key] {
			t.Fatalf("profile catalog declares %q more than once", row.Key)
		}
		declared[row.Key] = true
	}
	for profile := range accepted {
		if !declared[profile] {
			t.Errorf("profile catalog has no row for accepted profile %q", profile)
		}
	}
	for profile := range declared {
		if !accepted[profile] {
			t.Errorf("profile catalog declares %q, but the gateway does not accept it", profile)
		}
	}
}

type cliManifest struct {
	Groups []struct {
		Name     string `json:"name"`
		Commands []struct {
			Name    string `json:"name"`
			Flags   []cliFlag
			Measure *struct {
				Domain string `json:"domain"`
			} `json:"measure"`
			Binding struct {
				Kind   string `json:"kind"`
				Method string `json:"method"`
			} `json:"binding"`
		} `json:"commands"`
	} `json:"groups"`
	Measures struct {
		Domains []struct {
			Domain   string `json:"domain"`
			Stateful bool   `json:"stateful"`
			Reason   string `json:"reason"`
		} `json:"domains"`
		Omitted []struct {
			Domain string `json:"domain"`
			Reason string `json:"reason"`
		} `json:"omitted"`
	} `json:"measures"`
}

type cliFlag struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Default     string   `json:"default"`
	Values      []string `json:"values"`
}

func loadCLIManifest(t *testing.T) (cliManifest, string) {
	t.Helper()
	path := filepath.Join(vocabScenarioRoot(t), "cli", "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var manifest cliManifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return manifest, path
}

// The routing preview CLI flag is the operator-facing profile enum. The exact
// JSON path read here is:
//
//	groups[name="routing"].commands[name="preview"].flags[name="profile"].values
//
// A flag that declares values must declare exactly the accepted set; a future
// profile cannot land in the proto and gateway without updating this surface.
func TestCLIRoutingPreviewProfilesMatch(t *testing.T) {
	manifest, path := loadCLIManifest(t)
	var preview *cliFlag
	for _, group := range manifest.Groups {
		if group.Name != "routing" {
			continue
		}
		for _, command := range group.Commands {
			if command.Name != "preview" {
				continue
			}
			for index := range command.Flags {
				if command.Flags[index].Name == "profile" {
					preview = &command.Flags[index]
				}
			}
		}
	}
	if preview == nil {
		t.Fatalf(`%s: missing groups[name="routing"].commands[name="preview"].flags[name="profile"]`, path)
	}
	if len(preview.Values) == 0 {
		t.Fatalf("routing preview profile flag declares no values; add the closed enum at groups[name=\"routing\"].commands[name=\"preview\"].flags[name=\"profile\"].values")
	}
	accepted := acceptedProfileSet(t)
	declared := map[string]bool{}
	for _, value := range preview.Values {
		if declared[value] {
			t.Errorf("routing preview profile flag declares %q more than once", value)
		}
		declared[value] = true
		if !accepted[value] {
			t.Errorf("routing preview profile flag declares %q, but the gateway does not accept it", value)
		}
	}
	for profile := range accepted {
		if !declared[profile] {
			t.Errorf("routing preview profile flag is missing accepted profile %q", profile)
		}
	}
	if !declared[preview.Default] {
		t.Errorf("routing preview profile default %q is not an accepted profile", preview.Default)
	}

	// Every CLI flag that accepts a closed profile or privacy vocabulary must
	// publish its values, including operator commands outside routing preview.
	privacyValues := []string{"public", "internal", "confidential", "secret"}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			for _, flag := range command.Flags {
				if flag.Name != "profile" && flag.Name != "privacy-class" {
					continue
				}
				if len(flag.Values) == 0 {
					t.Errorf("groups[name=%q].commands[name=%q].flags[name=%q] omits its closed values", group.Name, command.Name, flag.Name)
					continue
				}
				got := append([]string(nil), flag.Values...)
				want := append([]string(nil), preview.Values...)
				if flag.Name == "privacy-class" {
					want = privacyValues
				}
				// Embedding cannot use a remotely routable profile.
				if flag.Name == "profile" && group.Name == "inference" && command.Name == "embed" {
					want = []string{"local-only"}
				}
				sort.Strings(got)
				sort.Strings(want)
				if strings.Join(got, ",") != strings.Join(want, ",") {
					t.Errorf("groups[name=%q].commands[name=%q].flags[name=%q] declares values %v; want %v", group.Name, command.Name, flag.Name, got, want)
				}
				if flag.Default != "" && !containsValue(want, flag.Default) {
					t.Errorf("groups[name=%q].commands[name=%q].flags[name=%q] default %q is outside its closed values", group.Name, command.Name, flag.Name, flag.Default)
				}
			}
		}
	}
}

func containsValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Measure declarations are projected across handler code, the CLI manifest,
// proto RPCs, and the manifest's stateful-domain inventory. A stateful domain
// must either have measures or name its explicit omission reason.
func TestMeasureDomainsMatchCLIAndProto(t *testing.T) {
	manifest, path := loadCLIManifest(t)
	type commandBinding struct {
		name   string
		method string
	}
	cliByName := map[string]commandBinding{}
	cliByMethod := map[string]string{}
	for _, group := range manifest.Groups {
		if group.Name != "measures" {
			continue
		}
		for _, command := range group.Commands {
			if command.Measure == nil || strings.TrimSpace(command.Measure.Domain) == "" {
				t.Errorf("%s: measures command %q has no measure domain", path, command.Name)
				continue
			}
			fullName := command.Measure.Domain + "." + command.Name
			if _, duplicate := cliByName[fullName]; duplicate {
				t.Fatalf("%s: measure %q is declared more than once", path, fullName)
			}
			binding := commandBinding{name: fullName, method: command.Binding.Method}
			cliByName[fullName] = binding
			if command.Binding.Method != "" {
				if previous, duplicate := cliByMethod[command.Binding.Method]; duplicate {
					t.Errorf("%s: measures commands %q and %q bind the same RPC %q", path, previous, fullName, command.Binding.Method)
				}
				cliByMethod[command.Binding.Method] = fullName
			}
		}
	}
	if len(cliByName) == 0 {
		t.Fatalf("%s: measures group declares no commands", path)
	}

	service := measuresv1.File_ai_gateway_v1_measures_measures_proto.Services().ByName("MeasuresService")
	if service == nil {
		t.Fatal("measures proto defines no MeasuresService")
	}
	protoMethods := map[string]bool{}
	methods := service.Methods()
	for index := 0; index < methods.Len(); index++ {
		protoMethods[string(methods.Get(index).Name())] = true
	}

	handlerNames := map[string]bool{}
	handlerDomains := map[string]bool{}
	for _, declaration := range measureshandler.Declarations() {
		if declaration.Name == "" || declaration.Domain == "" {
			t.Errorf("handler measure declaration must have a name and domain: %+v", declaration)
			continue
		}
		fullName := declaration.Name
		if !strings.HasPrefix(fullName, declaration.Domain+".") {
			t.Errorf("handler measure name %q does not use declared domain %q", fullName, declaration.Domain)
		}
		if handlerNames[fullName] {
			t.Errorf("handler declares measure %q more than once", fullName)
		}
		handlerNames[fullName] = true
		handlerDomains[declaration.Domain] = true
		binding, ok := cliByName[fullName]
		if !ok {
			t.Errorf("CLI manifest has no measures command for handler measure %q", fullName)
			continue
		}
		if binding.method == "" {
			t.Errorf("CLI measures command %q has no bound RPC method", fullName)
			continue
		}
		if !protoMethods[binding.method] {
			t.Errorf("CLI measures command %q binds %q, which MeasuresService does not define", fullName, binding.method)
		}
	}
	for fullName, binding := range cliByName {
		if !handlerNames[fullName] {
			t.Errorf("CLI measures command %q has no handler declaration", fullName)
		}
		if binding.method != "" && !protoMethods[binding.method] {
			t.Errorf("CLI measures command %q binds %q, which MeasuresService does not define", fullName, binding.method)
		}
	}
	for method := range protoMethods {
		if _, found := cliByMethod[method]; !found {
			t.Errorf("proto MeasuresService.%s has no CLI measures command", method)
		}
	}

	declaredDomains := map[string]bool{}
	statefulDomains := map[string]bool{}
	for _, domain := range manifest.Measures.Domains {
		if domain.Domain == "" || declaredDomains[domain.Domain] {
			t.Errorf("%s: measure domains must have unique non-empty names; got %q", path, domain.Domain)
			continue
		}
		declaredDomains[domain.Domain] = true
		if domain.Stateful {
			statefulDomains[domain.Domain] = true
		}
	}
	omitted := map[string]string{}
	for _, omission := range manifest.Measures.Omitted {
		if omission.Domain == "" || strings.TrimSpace(omission.Reason) == "" {
			t.Errorf("%s: measure omissions require a domain and reason", path)
			continue
		}
		if _, duplicate := omitted[omission.Domain]; duplicate {
			t.Errorf("%s: measure domain %q has multiple omission declarations", path, omission.Domain)
		}
		omitted[omission.Domain] = omission.Reason
		if !statefulDomains[omission.Domain] {
			t.Errorf("%s: omitted measure domain %q is not declared stateful", path, omission.Domain)
		}
	}
	for domain := range handlerDomains {
		if !statefulDomains[domain] {
			t.Errorf("%s: handler measure domain %q is not declared stateful", path, domain)
		}
		if _, explicitlyOmitted := omitted[domain]; explicitlyOmitted {
			t.Errorf("%s: measure domain %q both has handler measures and is declared omitted", path, domain)
		}
	}
	for domain := range statefulDomains {
		if !handlerDomains[domain] {
			if _, explicitlyOmitted := omitted[domain]; !explicitlyOmitted {
				t.Errorf("%s: stateful domain %q has no measures and no explicit omission reason", path, domain)
			}
		}
	}
	for domain := range persistedMeasureDomains(t) {
		if !statefulDomains[domain] {
			t.Errorf("%s: persisted entity %q is not declared as a stateful measures domain", path, domain)
		}
	}
}

func persistedMeasureDomains(t *testing.T) map[string]bool {
	t.Helper()
	root := filepath.Join(vocabScenarioRoot(t), "api", "internal")
	createTable := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-z][a-z0-9_]*)\s*\(`)
	createdAt := regexp.MustCompile(`(?i)\bcreated_at\b`)
	domains := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !(strings.HasSuffix(path, ".sql") || strings.HasSuffix(path, ".go")) || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		matches := createTable.FindAllStringSubmatchIndex(content, -1)
		for index, match := range matches {
			end := len(content)
			if index+1 < len(matches) {
				end = matches[index+1][0]
			}
			if createdAt.MatchString(content[match[0]:end]) {
				domains[content[match[2]:match[3]]] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan persisted entity schemas: %v", err)
	}
	return domains
}

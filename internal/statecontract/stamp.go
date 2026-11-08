package statecontract

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
)

// CodecStamp is set by governed native builders with
// buildinfo.PersistentStateBuildFlag. It carries, per stored named type, a
// digest of the source the compiled shape cannot show. An unstamped executable
// is unqualified for automatic rollback.
var CodecStamp string

// CodecPackages is the source inventory the builder scans for enum constants
// and custom codecs, as repository-relative directories. A stored named scalar
// or custom-codec type outside these packages makes the executable unqualified
// until its package is added here.
var CodecPackages = []string{
	"internal/capacity",
	"internal/cliinstall",
	"internal/coreactivation",
	"internal/credentialspec",
	"internal/hostreqspec",
	"internal/operatorcapability",
	"internal/process",
	"internal/recovery",
	"internal/resources",
	"internal/resources/manifest",
	"internal/runtimesupervisor",
	"internal/safeguards/emergency-watchdog",
	"internal/scenario",
	"internal/scenarioruntime",
	"internal/setpoint",
	"internal/agentsession",
	"internal/operatorstate",
	"packages/binaryfetch",
	"packages/cli-core/agentcatalog",
	"packages/cli-core/agentcatalog/harness",
	"packages/cli-core/usagemode",
	"packages/platform-go",
	"packages/resource-deployment",
}

// WholeSources maps each whole-source owner to the repository-relative
// sources the builder hashes completely (non-test Go files; "dir/prefix*"
// selects files by name prefix). Comments and layout do not participate;
// every declaration and function body does.
var WholeSources = map[string][]string{
	"core-activation":             {"internal/coreactivation"},
	"watchdog-publication":        {"internal/safeguards/emergency-watchdog"},
	"coding-agent-recovery":       {"internal/recovery"},
	"native-operation-identities": {"packages/platform-go/operation*", "packages/platform-go/owned_service*", "packages/platform-go/containment*"},
	// Acquisition declarations resources stores (manifest vocabulary that
	// binaryfetch validates inline) and the artifact identity digests the
	// prior recomputes to verify what it launches.
	"acquisition": {"packages/binaryfetch"},
	// Sealing, key wrapping and the unlock cache; the rest of securestore is
	// precise.
	// The resource storage root layout: where every resource's state lives.
	"resource-storage": {"internal/resources/runtime/storage"},
	"secure-store":     {"internal/credentialpolicy", "internal/securestore/sealedfile*", "internal/securestore/encryptedstore*", "internal/securestore/keyprovider*", "internal/securestore/unlock*", "internal/securestore/native_wrap*", "internal/securestore/device_identity*"},
}

// ConstantSources maps each constants-stamped owner to the packages whose
// package-level constants (typed and untyped, names and values) participate.
// The SQLite owners store untyped status, kind and reason strings in TEXT
// columns that an older binary queries by value, so every constant of those
// packages is part of their stored vocabulary; function bodies are not.
var ConstantSources = map[string][]string{
	"runtime-registry": {"internal/scenarioruntime"},
	"capacity":         {"internal/capacity"},
}

// PreciseSources maps every other persistence owner to its source packages.
// The builder classifies and checks them:
//
//   - every package-level untyped string constant is a stored value unless
//     NotStoredConstants lists it; stored values participate by name and
//     value, so a new or changed one changes the contract;
//   - format versions (fields tagged schema_version, *_version, or an integer
//     version) may only be written or compared with a named package constant,
//     and every such constant must be a fact of the owner;
//   - the time layouts its code formats and parses participate as a set;
//   - a string literal written to, compared with or switched against a
//     vocabulary field (Stamp.Literals) makes the executable unqualified
//     when the field is a string field of a declared stored type.
//
// ConstantSources packages get the same version, layout and literal checks.
var PreciseSources = map[string][]string{
	"process-records":         {"internal/process"},
	"host-session":            {"internal/hostsession"},
	"agent-sessions":          {"internal/agentsession"},
	"sustain-observations":    {"internal/setpoint"},
	"resources":               {"internal/resources"},
	"resource-artifact-cache": {"packages/resource-deployment"},
	"installation-inventory":  {"internal/cliinstall"},
	"supervisor-owner":        {"internal/runtimesupervisor"},
	"operator-state":          {"internal/operatorstate"},
	"operator-capability":     {"internal/operatorcapability"},
	"agent-catalogs":          {"packages/cli-core/agentcatalog", "packages/cli-core/agentcatalog/harness", "packages/cli-core/usagemode"},
	"artifact-leases":         {"internal/artifactlease"},
	"resource-manifest":       {"internal/resources/manifest"},
	"scenario-contract":       {"internal/scenario"},
	"secure-store":            {"internal/securestore"},
}

// StoredValueCalls name the functions and methods whose argument (by index)
// is written to stored state as a vocabulary value. A string literal there
// would hide the value from the constant stamp, so it must be a named
// constant of the owner. The builder enforces this inside owner packages; the
// owner's own tests enforce it for callers anywhere in the repository.
var StoredValueCalls = map[string]int{
	"UpdateProcessRefStatus": 2,
	"auditDemandLeaseTx":     2,
	"auditDemandInstanceTx":  2,
}

const stampFormat = "v2"

var (
	stampKeyPattern     = regexp.MustCompile(`^[A-Za-z0-9_./-]+\.[A-Za-z_][A-Za-z0-9_]*$`)
	stampPkgPattern     = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
	stampDigestPattern  = regexp.MustCompile(`^[a-f0-9]{32}$`)
	stampOwnerPattern   = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	stampValuePattern   = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}/[A-Za-z_][A-Za-z0-9_]*$`)
	stampViolationShape = regexp.MustCompile(`^[a-z-]+@[A-Za-z0-9_./:-]+$`)
	stampLiteralShape   = regexp.MustCompile(`^[A-Za-z0-9_./-]+\.(\*|[A-Za-z_][A-Za-z0-9_]*)\.[A-Za-z_][A-Za-z0-9_]*@[A-Za-z0-9_./:-]+$`)
)

// Stamp is the builder's scan result.
type Stamp struct {
	// Packages are the scanned codec packages (import paths).
	Packages []string
	// Entries map a type key to its codec-source digest.
	Entries map[string]string
	// Sources map a whole-source owner to its source digest.
	Sources map[string]string
	// Consts map a constants-stamped owner to its constants digest.
	Consts map[string]string
	// Values map "owner/Name" to the digest of a stored untyped constant.
	Values map[string]string
	// Layouts map an owner to the digest of its time-layout set.
	Layouts map[string]string
	// Versions are "owner/Const" format-version constants in use.
	Versions []string
	// Literals are "importpath.Type.Field@file:line" string literals written
	// to, compared with or switched against a vocabulary field ("*" when the
	// receiver type is not syntactically known). The executable turns those
	// whose field is a string field of a declared stored type into
	// stored-literal violations; the rest are not stored and are ignored.
	Literals []string
	// Violations are "kind@file:line" source checks that failed; any one
	// makes the executable unqualified.
	Violations []string
}

type stampCodecs struct {
	packages   map[string]bool
	entries    map[string]string
	sources    map[string]string
	consts     map[string]string
	values     map[string]string
	layouts    map[string]string
	versions   []string
	literals   []string
	violations []string
}

// EncodeStamp renders the builder's scan result as one linker value.
func EncodeStamp(stamp Stamp) (string, error) {
	pkgs := append([]string(nil), stamp.Packages...)
	sort.Strings(pkgs)
	for i, pkg := range pkgs {
		if !stampPkgPattern.MatchString(pkg) || i > 0 && pkgs[i-1] == pkg {
			return "", fmt.Errorf("codec inventory package %q is invalid or duplicated", pkg)
		}
	}
	pairs := func(kind string, digests map[string]string, key *regexp.Regexp) (string, error) {
		items := make([]string, 0, len(digests))
		for name, digest := range digests {
			if !key.MatchString(name) || !stampDigestPattern.MatchString(digest) {
				return "", fmt.Errorf("%s stamp entry %q is invalid", kind, name)
			}
			items = append(items, name+":"+digest)
		}
		sort.Strings(items)
		return strings.Join(items, ","), nil
	}
	list := func(kind string, items []string, shape *regexp.Regexp) (string, error) {
		out := append([]string(nil), items...)
		sort.Strings(out)
		for i, item := range out {
			if !shape.MatchString(item) || i > 0 && out[i-1] == item {
				return "", fmt.Errorf("%s stamp item %q is invalid or duplicated", kind, item)
			}
		}
		return strings.Join(out, ","), nil
	}
	parts := []string{stampFormat, "pkgs=" + strings.Join(pkgs, ",")}
	for _, section := range []struct {
		name string
		text func() (string, error)
	}{
		{"srcs", func() (string, error) { return pairs("whole-source", stamp.Sources, stampOwnerPattern) }},
		{"consts", func() (string, error) { return pairs("constants", stamp.Consts, stampOwnerPattern) }},
		{"vals", func() (string, error) { return pairs("stored value", stamp.Values, stampValuePattern) }},
		{"lays", func() (string, error) { return pairs("time layout", stamp.Layouts, stampOwnerPattern) }},
		{"vers", func() (string, error) { return list("format version", stamp.Versions, stampValuePattern) }},
		{"lits", func() (string, error) { return list("stored-field literal", stamp.Literals, stampLiteralShape) }},
		{"viol", func() (string, error) { return list("violation", stamp.Violations, stampViolationShape) }},
	} {
		text, err := section.text()
		if err != nil {
			return "", err
		}
		parts = append(parts, section.name+"="+text)
	}
	keys := make([]string, 0, len(stamp.Entries))
	for key := range stamp.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !stampKeyPattern.MatchString(key) || !stampDigestPattern.MatchString(stamp.Entries[key]) {
			return "", fmt.Errorf("codec stamp entry %q is invalid", key)
		}
		parts = append(parts, key+"="+stamp.Entries[key])
	}
	return strings.Join(parts, ";"), nil
}

var stampSections = []string{"pkgs", "srcs", "consts", "vals", "lays", "vers", "lits", "viol"}

func parseStamp(stamp string) (stampCodecs, error) {
	codecs := stampCodecs{packages: map[string]bool{}, entries: map[string]string{}, sources: map[string]string{}, consts: map[string]string{}, values: map[string]string{}, layouts: map[string]string{}}
	if stamp == "" {
		return codecs, fmt.Errorf("executable has no persistence codec stamp (not built by a governed builder)")
	}
	parts := strings.Split(stamp, ";")
	if len(parts) < 1+len(stampSections) || parts[0] != stampFormat {
		return codecs, fmt.Errorf("persistence codec stamp has an unsupported format")
	}
	section := map[string]string{}
	for i, name := range stampSections {
		value, ok := strings.CutPrefix(parts[1+i], name+"=")
		if !ok {
			return codecs, fmt.Errorf("persistence codec stamp has an unsupported format")
		}
		section[name] = value
	}
	for _, pkg := range strings.Split(section["pkgs"], ",") {
		if !stampPkgPattern.MatchString(pkg) || codecs.packages[pkg] {
			return codecs, fmt.Errorf("persistence codec stamp package inventory is invalid")
		}
		codecs.packages[pkg] = true
	}
	for _, pairSection := range []struct {
		name   string
		key    *regexp.Regexp
		target map[string]string
	}{{"srcs", stampOwnerPattern, codecs.sources}, {"consts", stampOwnerPattern, codecs.consts}, {"vals", stampValuePattern, codecs.values}, {"lays", stampOwnerPattern, codecs.layouts}} {
		if section[pairSection.name] == "" {
			continue
		}
		for _, item := range strings.Split(section[pairSection.name], ",") {
			name, digest, ok := strings.Cut(item, ":")
			if _, dup := pairSection.target[name]; !ok || dup || !pairSection.key.MatchString(name) || !stampDigestPattern.MatchString(digest) {
				return codecs, fmt.Errorf("persistence codec stamp %s inventory is invalid", pairSection.name)
			}
			pairSection.target[name] = digest
		}
	}
	for _, listSection := range []struct {
		name   string
		shape  *regexp.Regexp
		target *[]string
	}{{"vers", stampValuePattern, &codecs.versions}, {"lits", stampLiteralShape, &codecs.literals}, {"viol", stampViolationShape, &codecs.violations}} {
		if section[listSection.name] == "" {
			continue
		}
		for _, item := range strings.Split(section[listSection.name], ",") {
			if !listSection.shape.MatchString(item) {
				return codecs, fmt.Errorf("persistence codec stamp %s list is invalid", listSection.name)
			}
			*listSection.target = append(*listSection.target, item)
		}
	}
	for _, part := range parts[1+len(stampSections):] {
		key, digest, ok := strings.Cut(part, "=")
		if !ok || !stampKeyPattern.MatchString(key) || !stampDigestPattern.MatchString(digest) {
			return codecs, fmt.Errorf("persistence codec stamp entry is invalid")
		}
		if _, dup := codecs.entries[key]; dup {
			return codecs, fmt.Errorf("persistence codec stamp entry %s is duplicated", key)
		}
		pkg := key[:strings.LastIndex(key, ".")]
		if !codecs.packages[pkg] {
			return codecs, fmt.Errorf("persistence codec stamp entry %s is outside its package inventory", key)
		}
		codecs.entries[key] = digest
	}
	return codecs, nil
}

func (c stampCodecs) lookup(t reflect.Type, codec bool) (string, error) {
	if !c.packages[t.PkgPath()] {
		// A dependency pinned by version and checksum fixes its codec source.
		if module := pinnedModule(t.PkgPath()); module != "" {
			return "module " + module, nil
		}
		return "", fmt.Errorf("stored type %s has enum constants or a custom codec but package %s is neither in statecontract.CodecPackages nor a checksum-pinned dependency", TypeKey(t), t.PkgPath())
	}
	// Instantiated generic types share their declaration's source entry.
	key, _, _ := strings.Cut(TypeKey(t), "[")
	entry, ok := c.entries[key]
	if codec && !ok {
		// The scanner always records a codec type's declaration; a missing
		// entry means it could not see the codec, so fail closed.
		return "", fmt.Errorf("stored type %s has a custom codec the build stamp does not cover", TypeKey(t))
	}
	return entry, nil
}

// readBuildInfo is replaced in tests.
var readBuildInfo = debug.ReadBuildInfo

// pinnedModule returns path@version+sum of the dependency module providing
// pkg when the build pinned it by checksum. Replaced or local modules (the
// repository's own packages/ modules) are not pinned and must be scanned.
func pinnedModule(pkg string) string {
	info, ok := readBuildInfo()
	if !ok {
		return ""
	}
	var best *debug.Module
	for _, dep := range info.Deps {
		if dep == nil || (pkg != dep.Path && !strings.HasPrefix(pkg, dep.Path+"/")) {
			continue
		}
		if best == nil || len(dep.Path) > len(best.Path) {
			best = dep
		}
	}
	if best == nil || best.Replace != nil || best.Sum == "" || best.Version == "" || best.Version == "(devel)" {
		return ""
	}
	return best.Path + "@" + best.Version + " " + best.Sum
}

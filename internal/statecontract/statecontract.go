package statecontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
)

// ManifestVersion identifies this contract algorithm. Executables built
// before it reported a whole-source digest that cannot be compared per owner.
const ManifestVersion = 2

// Owner is one persistence owner's declaration of what it stores.
//
// Roots are the compiled envelopes the owner writes (JSON documents, JSON
// columns, or Go row mirrors of SQL tables). Facts are format values the prior
// release relies on to locate and accept stored data: schema versions, DDL,
// witness bytes and fingerprints of derived stored identities. Fact values are
// compiled values, never paths or runtime data.
//
// WholeSource owners are the rollback machinery and native identity: small,
// rarely edited, full of inline format literals behind strict readers. Their
// complete non-test source (WholeSources) participates, so any edit there
// needs a rebaseline; their roots still name shape changes precisely.
//
// StampConstants owners (ConstantSources) contribute every package-level
// constant of their packages: their stored vocabulary is untyped strings.
type Owner struct {
	Name           string
	Roots          []reflect.Type
	Facts          map[string]any
	WholeSource    bool
	StampConstants bool
}

// Facts carrying build-stamped digests.
const (
	wholeSourceFact      = "whole_source"
	packageConstantsFact = "package_constants"
	timeLayoutsFact      = "time_layouts"
	storedConstantPrefix = "constant:"
)

// TypeShape is the JSON-relevant local shape of one stored type. Named field
// and element types are referenced by key and described by their own entries.
type TypeShape struct {
	Kind   string   `json:"kind"`
	Elem   string   `json:"elem,omitempty"`
	Fields []string `json:"fields,omitempty"`
	Codec  string   `json:"codec,omitempty"`
}

// OwnerManifest is one owner's stored contract. Digest covers every other
// field, so equal digests mean equal roots, shapes and facts.
type OwnerManifest struct {
	Owner  string               `json:"owner"`
	Digest string               `json:"sha256"`
	Roots  []string             `json:"roots"`
	Types  map[string]TypeShape `json:"types"`
	Facts  map[string]string    `json:"facts"`
}

// Manifest is the complete per-owner contract of one executable.
type Manifest struct {
	Version   int             `json:"version"`
	Contract  string          `json:"contract_sha256"`
	Toolchain string          `json:"toolchain"`
	Owners    []OwnerManifest `json:"owners"`
}

var ownerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

// Compute derives the manifest from compiled declarations and the build's
// codec stamp. It fails (the executable is unqualified for automatic rollback)
// when the stamp is absent or does not cover a stored type that needs it.
func Compute(owners []Owner) (Manifest, error) {
	return compute(owners, CodecStamp, runtime.Version())
}

func compute(owners []Owner, stamp, goVersion string) (Manifest, error) {
	manifest := Manifest{Version: ManifestVersion, Toolchain: toolchainRelease(goVersion)}
	codecs, err := parseStamp(stamp)
	if err != nil {
		return manifest, err
	}
	if len(codecs.violations) > 0 {
		return manifest, fmt.Errorf("persistence source checks failed when this executable was built: %s", DescribeViolations(codecs.violations))
	}
	if len(owners) == 0 {
		return manifest, fmt.Errorf("no persistence owners are declared")
	}
	seen := map[string]bool{}
	for _, owner := range owners {
		if !ownerNamePattern.MatchString(owner.Name) || seen[owner.Name] {
			return manifest, fmt.Errorf("persistence owner name %q is invalid or duplicated", owner.Name)
		}
		seen[owner.Name] = true
		entry, err := ownerManifest(owner, codecs)
		if err != nil {
			return manifest, fmt.Errorf("persistence owner %s: %w", owner.Name, err)
		}
		manifest.Owners = append(manifest.Owners, entry)
	}
	for name := range codecs.sources {
		if !declaredOwner(owners, name, func(o Owner) bool { return o.WholeSource }) {
			return manifest, fmt.Errorf("build stamped whole source for %s, which no owner declares as whole-source", name)
		}
	}
	for name := range codecs.consts {
		if !declaredOwner(owners, name, func(o Owner) bool { return o.StampConstants }) {
			return manifest, fmt.Errorf("build stamped package constants for %s, which no owner declares as constants-stamped", name)
		}
	}
	for _, version := range codecs.versions {
		owner, constant, _ := strings.Cut(version, "/")
		if !declaredOwner(owners, owner, func(o Owner) bool { _, ok := o.Facts[constant]; return ok }) {
			return manifest, fmt.Errorf("format version constant %s of owner %s is not declared as one of its facts", constant, owner)
		}
	}
	if violations := LiteralViolations(owners, codecs.literals); len(violations) > 0 {
		return manifest, fmt.Errorf("persistence source checks failed when this executable was built: %s", DescribeViolations(violations))
	}
	sort.Slice(manifest.Owners, func(i, j int) bool { return manifest.Owners[i].Owner < manifest.Owners[j].Owner })
	manifest.Contract = contractDigest(manifest)
	return manifest, nil
}

// LiteralViolations keeps the stored-field literal candidates
// ("importpath.Type.Field@file:line" or "importpath.*.Field@…") whose field
// is a string field of a type reachable from the owners' stored envelopes, as
// stored-literal violations. Such a value is a stored vocabulary word an
// older binary reads by value; written inline it would change without the
// contract noticing, so it must be a named constant, which constant stamping
// covers.
func LiteralViolations(owners []Owner, candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}
	fields := map[string]bool{}
	seen := map[reflect.Type]bool{}
	for _, owner := range owners {
		for _, root := range owner.Roots {
			storedStringFields(root, fields, seen)
		}
	}
	var violations []string
	for _, candidate := range candidates {
		field, location, ok := strings.Cut(candidate, "@")
		if ok && fields[field] {
			violations = append(violations, "stored-literal@"+location)
		}
	}
	sort.Strings(violations)
	return slices.Compact(violations)
}

// storedStringFields records "importpath.Type.Field" and
// "importpath.*.Field" for every stored string-kinded field reachable from t.
func storedStringFields(t reflect.Type, fields map[string]bool, seen map[reflect.Type]bool) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		storedStringFields(t.Elem(), fields, seen)
	case reflect.Map:
		storedStringFields(t.Key(), fields, seen)
		storedStringFields(t.Elem(), fields, seen)
	case reflect.Struct:
		key, _, _ := strings.Cut(TypeKey(t), "[")
		for _, field := range reflect.VisibleFields(t) {
			if !field.IsExported() || field.Tag.Get("json") == "-" {
				continue
			}
			// Promoted fields are recorded on the outer type too, since a
			// selector names them through it.
			if field.Type.Kind() == reflect.String && t.Name() != "" {
				fields[key+"."+field.Name] = true
				fields[t.PkgPath()+".*."+field.Name] = true
			}
			if len(field.Index) == 1 {
				storedStringFields(field.Type, fields, seen)
			}
		}
	}
}

func contractDigest(manifest Manifest) string {
	type ownerDigest struct {
		Owner  string `json:"owner"`
		Digest string `json:"sha256"`
	}
	summary := struct {
		Version   int           `json:"version"`
		Toolchain string        `json:"toolchain"`
		Owners    []ownerDigest `json:"owners"`
	}{Version: manifest.Version, Toolchain: manifest.Toolchain}
	for _, owner := range manifest.Owners {
		summary.Owners = append(summary.Owners, ownerDigest{owner.Owner, owner.Digest})
	}
	return digestJSON(summary)
}

// Verify checks that a manifest received from another executable is
// internally consistent: every owner digest covers its shapes and facts, and
// the contract covers every owner. A consistent manifest can be diffed
// against another and bound to an executable's reported contract digest.
func Verify(manifest Manifest) error {
	if manifest.Version != ManifestVersion {
		return fmt.Errorf("state manifest format v%d is not v%d", manifest.Version, ManifestVersion)
	}
	for i, owner := range manifest.Owners {
		if i > 0 && manifest.Owners[i-1].Owner >= owner.Owner {
			return fmt.Errorf("state manifest owners are not canonical")
		}
		if owner.Roots == nil || owner.Types == nil || owner.Facts == nil || ownerDigest(owner) != owner.Digest {
			return fmt.Errorf("state manifest owner %s does not match its digest", owner.Owner)
		}
	}
	if len(manifest.Owners) == 0 || contractDigest(manifest) != manifest.Contract {
		return fmt.Errorf("state manifest does not match its contract digest")
	}
	return nil
}

// Contract returns the contract digest, or "" when the executable is
// unqualified. Use Compute to retain the reason.
func Contract(owners []Owner) string {
	manifest, err := Compute(owners)
	if err != nil {
		return ""
	}
	return manifest.Contract
}

func declaredOwner(owners []Owner, name string, kind func(Owner) bool) bool {
	for _, owner := range owners {
		if owner.Name == name && kind(owner) {
			return true
		}
	}
	return false
}

func ownerManifest(owner Owner, codecs stampCodecs) (OwnerManifest, error) {
	entry := OwnerManifest{Owner: owner.Name, Roots: []string{}, Types: map[string]TypeShape{}, Facts: map[string]string{}}
	_, precise := PreciseSources[owner.Name]
	_, constants := ConstantSources[owner.Name]
	if !owner.WholeSource && !precise && !constants {
		return entry, fmt.Errorf("has no source inventory (add it to statecontract.PreciseSources, ConstantSources or WholeSources)")
	}
	if precise || constants {
		layouts, ok := codecs.layouts[owner.Name]
		if !ok {
			return entry, fmt.Errorf("its sources were not analysed by the build (stale builder or missing statecontract.PreciseSources entry)")
		}
		entry.Facts[timeLayoutsFact] = layouts
	}
	for key, digest := range codecs.values {
		if name, ok := strings.CutPrefix(key, owner.Name+"/"); ok {
			entry.Facts[storedConstantPrefix+name] = digest
		}
	}
	if len(owner.Roots) == 0 && len(owner.Facts) == 0 && !owner.WholeSource {
		return entry, fmt.Errorf("declares neither stored envelopes nor format facts")
	}
	if owner.WholeSource {
		digest, ok := codecs.sources[owner.Name]
		if !ok {
			return entry, fmt.Errorf("whole-source owner has no source digest in the build stamp (add it to statecontract.WholeSources)")
		}
		entry.Facts[wholeSourceFact] = digest
	}
	if owner.StampConstants {
		digest, ok := codecs.consts[owner.Name]
		if !ok {
			return entry, fmt.Errorf("constants-stamped owner has no constants digest in the build stamp (add it to statecontract.ConstantSources)")
		}
		entry.Facts[packageConstantsFact] = digest
	}
	walker := shapeWalker{codecs: codecs, types: entry.Types}
	for _, root := range owner.Roots {
		if root == nil {
			return entry, fmt.Errorf("stored envelope type is nil")
		}
		entry.Roots = append(entry.Roots, walker.ref(root))
		if walker.err != nil {
			return entry, walker.err
		}
	}
	sort.Strings(entry.Roots)
	for name, value := range owner.Facts {
		if strings.TrimSpace(name) == "" || name == wholeSourceFact || name == packageConstantsFact || name == timeLayoutsFact || strings.HasPrefix(name, storedConstantPrefix) {
			return entry, fmt.Errorf("format fact name is empty")
		}
		data, err := json.Marshal(value)
		if err != nil {
			return entry, fmt.Errorf("format fact %s is not canonical JSON: %w", name, err)
		}
		sum := sha256.Sum256(data)
		entry.Facts[name] = hex.EncodeToString(sum[:])
	}
	entry.Digest = ownerDigest(entry)
	return entry, nil
}

func ownerDigest(entry OwnerManifest) string {
	return digestJSON(struct {
		Owner string               `json:"owner"`
		Roots []string             `json:"roots"`
		Types map[string]TypeShape `json:"types"`
		Facts map[string]string    `json:"facts"`
	}{entry.Owner, entry.Roots, entry.Types, entry.Facts})
}

// shapeWalker records the local shape of every named type reachable through
// the JSON codec's view of a root. Unnamed composites are spelled inline.
type shapeWalker struct {
	codecs stampCodecs
	types  map[string]TypeShape
	err    error
}

const maxStoredTypes = 4096

// ref returns the spelling used to reference t from a field or element.
func (w *shapeWalker) ref(t reflect.Type) string {
	if t.Name() == "" {
		return w.spell(t)
	}
	key := TypeKey(t)
	if _, ok := w.types[key]; ok {
		return key
	}
	if len(w.types) >= maxStoredTypes {
		w.fail(fmt.Errorf("stored type graph exceeds its bound"))
		return key
	}
	w.types[key] = TypeShape{} // reserve before recursion for cyclic types
	shape := TypeShape{Kind: t.Kind().String()}
	switch t.Kind() {
	case reflect.Struct:
		shape.Fields = w.fields(t)
	case reflect.Array, reflect.Pointer, reflect.Slice, reflect.Map:
		shape.Elem = w.spell(t)
	}
	if needsCodecCoverage(t) {
		codec, err := w.codecs.lookup(t, ownCodec(t))
		if err != nil {
			w.fail(err)
		}
		shape.Codec = codec
	}
	w.types[key] = shape
	return key
}

// spell renders an unnamed type structurally, referencing named children.
func (w *shapeWalker) spell(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + w.ref(t.Elem())
	case reflect.Slice:
		return "[]" + w.ref(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), w.ref(t.Elem()))
	case reflect.Map:
		return "map[" + w.ref(t.Key()) + "]" + w.ref(t.Elem())
	case reflect.Struct:
		if t.Name() != "" {
			return w.ref(t)
		}
		return "struct{" + strings.Join(w.fields(t), "; ") + "}"
	default:
		if t.Name() != "" {
			return w.ref(t)
		}
		return t.String()
	}
}

// fields lists the struct fields the JSON codec reads and writes as
// "key|json-tag|type|embedded" in declaration order. The key is the stored
// JSON name (the tag name, else the Go name), so renaming a Go field behind an
// unchanged tag keeps the contract. Order stays significant: several
// envelopes are self-hashed by re-encoding them (managed-update contract
// digest, credential MAC, frozen layout, override inventory).
func (w *shapeWalker) fields(t reflect.Type) []string {
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if field.PkgPath != "" && !field.Anonymous || tag == "-" {
			continue
		}
		key := field.Name
		if name, _, _ := strings.Cut(tag, ","); name != "" {
			key = name
		}
		fields = append(fields, fmt.Sprintf("%s|%s|%s|%t", key, tag, w.ref(field.Type), field.Anonymous))
	}
	return fields
}

func (w *shapeWalker) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

// TypeKey is the stable identity of a named type: its import path and name.
func TypeKey(t reflect.Type) string {
	if t.PkgPath() == "" {
		return t.String()
	}
	return t.PkgPath() + "." + t.Name()
}

var codecMethods = []string{"MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText"}

// needsCodecCoverage reports stored types whose stored values depend on
// source the compiled shape cannot show: enum constants of named scalar
// types and custom codecs. Standard-library types are bound by the toolchain.
func needsCodecCoverage(t reflect.Type) bool {
	if standardLibrary(t.PkgPath()) {
		return false
	}
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	return ownCodec(t)
}

// ownCodec reports a custom JSON/text codec declared on t itself rather than
// promoted from an embedded field (the embedded type is covered on its own).
func ownCodec(t reflect.Type) bool {
	for _, method := range codecMethods {
		_, value := t.MethodByName(method)
		_, pointer := reflect.PointerTo(t).MethodByName(method)
		if (value || pointer) && !promotedMethod(t, method) {
			return true
		}
	}
	return false
}

func promotedMethod(t reflect.Type, method string) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.Anonymous {
			continue
		}
		if _, ok := field.Type.MethodByName(method); ok {
			return true
		}
		if field.Type.Kind() != reflect.Pointer {
			if _, ok := reflect.PointerTo(field.Type).MethodByName(method); ok {
				return true
			}
		}
	}
	return false
}

func standardLibrary(pkg string) bool {
	first, _, _ := strings.Cut(pkg, "/")
	return pkg == "" || !strings.Contains(first, ".")
}

// toolchainRelease keeps the Go major.minor release and any experiment
// suffix. Patch releases do not change encoding/json output.
func toolchainRelease(version string) string {
	release, suffix, _ := strings.Cut(version, " ")
	if strings.HasPrefix(release, "go") {
		parts := strings.SplitN(strings.TrimPrefix(release, "go"), ".", 3)
		if len(parts) >= 2 {
			release = "go" + parts[0] + "." + parts[1]
		}
	}
	if suffix != "" {
		return release + " " + suffix
	}
	return release
}

func digestJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("statecontract: canonical digest input failed to encode: %v", err))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

var violationKinds = map[string]string{
	"version-value":       "a format version field is written or compared with something other than a named package constant",
	"codec-cross-package": "a custom codec calls into another module package the stamp cannot follow",
	"stored-literal":      "a stored vocabulary value is written as a string literal instead of a named constant of its owner",
}

// DescribeViolations renders stamp violations ("kind@file:line") as text.
func DescribeViolations(violations []string) string {
	parts := make([]string, 0, len(violations))
	for _, violation := range violations {
		kind, location, _ := strings.Cut(violation, "@")
		text := violationKinds[kind]
		if text == "" {
			text = kind
		}
		parts = append(parts, location+": "+text)
	}
	return strings.Join(parts, "; ")
}

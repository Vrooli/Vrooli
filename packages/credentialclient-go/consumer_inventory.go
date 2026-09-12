package credentialclient

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

// ConsumerKind describes how a runtime obtains an input. It is deliberately
// small and closed: adding a new kind must be an explicit inventory decision.
type ConsumerKind string

const (
	ConsumerAuthority       ConsumerKind = "authority"
	ConsumerEnvironment     ConsumerKind = "environment"
	ConsumerGenerated       ConsumerKind = "generated"
	ConsumerProviderDefault ConsumerKind = "provider_default"
	ConsumerDynamic         ConsumerKind = "dynamic"
	ConsumerDelegated       ConsumerKind = "delegated"
	ConsumerExternal        ConsumerKind = "external"
	ConsumerPublic          ConsumerKind = "public"
	ConsumerBackup          ConsumerKind = "backup"
)

// ConsumerBinding is an observed runtime use of one declared input. SourceRef
// is absolute and points to the observed call site, never to a secret value.
type ConsumerBinding struct {
	ID        string       `json:"id"`
	Owner     string       `json:"owner"`
	Kind      ConsumerKind `json:"kind"`
	Consumer  string       `json:"consumer,omitempty"`
	SourceRef string       `json:"source_ref"`
}

// ConsumerInventory is the machine-readable declaration-to-consumer view used
// by drift gates. Rows preserve declaration sites; a shared logical address is
// not silently collapsed before ownership and provenance have been compared.
type ConsumerInventory struct {
	Rows                    []ConsumerInventoryRow `json:"rows"`
	Gaps                    []ConsumerInventoryGap `json:"gaps"`
	DeclarationSiteCount    int                    `json:"declaration_site_count"`
	DistinctAddressCount    int                    `json:"distinct_address_count"`
	UnresolvedConsumerCount int                    `json:"unresolved_consumer_count"`
}

type ConsumerInventoryRow struct {
	CredentialRef
	Owner       string            `json:"owner"`
	SourceRef   string            `json:"source_ref"`
	Kind        string            `json:"kind"`
	Consumers   []ConsumerBinding `json:"consumers,omitempty"`
	Disposition string            `json:"disposition"`
	Declared    bool              `json:"declared"`
}

type ConsumerInventoryGap struct {
	Address     string `json:"address"`
	Owner       string `json:"owner,omitempty"`
	Consumer    string `json:"consumer,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
	Kind        string `json:"kind"`
	Required    bool   `json:"required"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation"`
}

type inventoryDeclaration struct {
	Owner       string
	Root        string
	SourceRef   string
	Descriptor  credentialspec.Descriptor
	Registrants []credentialspec.Consumer
	Declared    bool
}

// literalCredentialUse finds the safe, statically attributable forms used by
// authored consumers. Dynamic and provider-default chains are not guessed from
// a missing literal; they remain explicit gaps until an owner registers them.
var literalCredentialUse = regexp.MustCompile(`\b(?:Resolve|Require|Status|Delete|Put|Provision|ResolveOrMint|ResolveOrMintWithCredentialLossOverride)\s*\(\s*(?:[^,()\n]+\s*,\s*)?(?:"([a-z0-9][a-z0-9._-]+/[a-z0-9][a-z0-9._/-]+)"|([A-Za-z_][A-Za-z0-9_]*))\s*,\s*"([a-zA-Z0-9._-]+)"\s*\)`)
var identityAssignment = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=)\s*(?:credentialauthority\.)?(?:ParseIdentity|Identity)\s*\(\s*"([a-z0-9][a-z0-9._-]+/[a-z0-9][a-z0-9._/-]+)"\s*\)`)
var nestedLiteralCredentialUse = regexp.MustCompile(`\b(?:Resolve|Require|Status|Delete|Put|Provision|ResolveOrMint|ResolveOrMintWithCredentialLossOverride)\s*\(\s*(?:[^,()\n]+\s*,\s*)?(?:credentialauthority\.)?(?:ParseIdentity|Identity)\s*\(\s*"([a-z0-9][a-z0-9._-]+/[a-z0-9][a-z0-9._/-]+)"\s*\)\s*,\s*"([a-zA-Z0-9._-]+)"\s*\)`)
var literalEnvUse = regexp.MustCompile(`\b(?:Getenv|LookupEnv)\s*\(\s*"([A-Z][A-Z0-9_]*)"\s*\)`)

// DiscoverConsumerInventory walks the selected manifest population and the
// owning source trees. It is intentionally a bounded diagnostic gate, not a
// replacement for a language compiler: exact literal bindings are attributed;
// dynamic/provider-default paths must be registered by their owner.
func DiscoverConsumerInventory(root string, scope Scope) (ConsumerInventory, error) {
	return DiscoverConsumerInventoryContext(context.Background(), root, scope)
}

// DiscoverConsumerInventoryContext is the cancellable form used by readiness
// callers. Consumer inventory is a bounded source scan, but repository-sized
// roots can still contain enough authored files that a request should be able
// to stop waiting when its deadline expires.
func DiscoverConsumerInventoryContext(ctx context.Context, root string, scope Scope) (ConsumerInventory, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return ConsumerInventory{}, nil
	}
	root, err := filepath.Abs(trimmedRoot)
	if err != nil {
		return ConsumerInventory{}, fmt.Errorf("resolve inventory root: %w", err)
	}
	declarations, err := discoverInventoryDeclarations(root, scope)
	if err != nil {
		return ConsumerInventory{}, err
	}

	uses := make(map[string][]ConsumerBinding)
	unknown := make(map[string]ConsumerInventoryGap)
	knownAddresses := make(map[string]struct{})
	envAddresses := make(map[string]map[string][]string)
	ownerRoots := make(map[string]string)
	for _, declaration := range declarations {
		if err := ctx.Err(); err != nil {
			return ConsumerInventory{}, err
		}
		address := declaration.Descriptor.LogicalID + ":" + declaration.Descriptor.ResolvedField()
		if declaration.Declared && strings.TrimSpace(declaration.Descriptor.LogicalID) != "" {
			knownAddresses[address] = struct{}{}
		}
		// The repository-root manifest describes host-owned inputs. Optional
		// host inputs are intentionally allowed to have no authored consumer,
		// and scanning the entire repository to rediscover that fact makes a
		// readiness request pay for unrelated shared packages. Required project
		// inputs and explicit registrations still opt into source tracing.
		if declaration.Owner != ProjectScopeOwner || declaration.Descriptor.Required || len(declaration.Registrants) > 0 {
			ownerRoots[declaration.Owner] = declaration.Root
		}
		if env := strings.TrimSpace(declaration.Descriptor.Env); env != "" {
			if envAddresses[declaration.Owner] == nil {
				envAddresses[declaration.Owner] = make(map[string][]string)
			}
			envAddresses[declaration.Owner][env] = append(envAddresses[declaration.Owner][env], address)
		}
	}

	// Resolve registrations only after every declaration has contributed its
	// address. A manifest may list a consumer for a descriptor that appears
	// later in the discovered population; checking during the first pass would
	// incorrectly classify that valid forward reference as undeclared.
	for _, declaration := range declarations {
		if err := ctx.Err(); err != nil {
			return ConsumerInventory{}, err
		}
		for _, registrant := range declaration.Registrants {
			if scope.Tier != "" && !tierAppliesToConsumer(registrant, scope.Tier) {
				continue
			}
			binding := registeredBinding(declaration.Root, declaration.Owner, registrant)
			if binding.ID == "" {
				continue
			}
			if pattern := strings.TrimSpace(registrant.AddressPattern); pattern != "" {
				matched := false
				for address := range knownAddresses {
					if !matchesAddressPattern(pattern, address) {
						continue
					}
					uses[address] = appendUniqueBinding(uses[address], binding)
					matched = true
				}
				if matched {
					continue
				}
			} else if _, known := knownAddresses[binding.ID]; known {
				uses[binding.ID] = appendUniqueBinding(uses[binding.ID], binding)
				continue
			}
			addUnknownConsumer(unknown, binding, registrantRequired(registrant), "registered runtime consumer has no matching declaration", "declare the authority address in the owning manifest or remove the consumer registration")
		}
	}

	// Scan each owner once. Literal bindings are useful drift evidence, while
	// registered bindings cover dynamic/generated/provider-default paths that
	// source text cannot attribute safely.
	for owner, sourceRoot := range ownerRoots {
		files, walkErr := sourceFilesContext(ctx, sourceRoot, "")
		if walkErr != nil {
			return ConsumerInventory{}, walkErr
		}
		for _, file := range files {
			bindings, scanErr := scanConsumerFileContext(ctx, file, owner)
			if scanErr != nil {
				return ConsumerInventory{}, scanErr
			}
			for _, binding := range bindings {
				if err := ctx.Err(); err != nil {
					return ConsumerInventory{}, err
				}
				if binding.Kind == ConsumerEnvironment {
					for _, address := range envAddresses[owner][binding.ID] {
						uses[address] = appendUniqueBinding(uses[address], binding)
					}
					continue
				}
				if _, known := knownAddresses[binding.ID]; known {
					uses[binding.ID] = appendUniqueBinding(uses[binding.ID], binding)
					continue
				}
				if binding.Kind == ConsumerAuthority || binding.Kind == ConsumerDynamic {
					addUnknownConsumer(unknown, binding, true, "runtime consumer has no matching declaration", "declare the authority address in the owning manifest or remove the consumer binding")
				}
			}
		}
	}

	rows := make([]ConsumerInventoryRow, 0, len(declarations))
	seenAddresses := make(map[string]struct{})
	gaps := make([]ConsumerInventoryGap, 0)
	for _, declaration := range declarations {
		if err := ctx.Err(); err != nil {
			return ConsumerInventory{}, err
		}
		descriptor := declaration.Descriptor
		if strings.TrimSpace(descriptor.LogicalID) == "" {
			continue
		}
		address := descriptor.LogicalID + ":" + descriptor.ResolvedField()
		consumers := append([]ConsumerBinding(nil), uses[address]...)
		disposition := "bound"
		if !declaration.Declared {
			disposition = "declaration-missing"
		} else if len(consumers) == 0 {
			disposition = "unbound-optional"
			if descriptor.Required {
				disposition = "required-binding-missing"
				gaps = append(gaps, ConsumerInventoryGap{
					Address: address, Owner: declaration.Owner, SourceRef: declaration.SourceRef, Required: true,
					Kind: descriptorKind(descriptor), Reason: "required declaration has no observed runtime consumer",
					Remediation: "register the owner runtime binding, including a dynamic or provider-default binding when applicable",
				})
			}
		}
		rows = append(rows, ConsumerInventoryRow{
			CredentialRef: CredentialRef{
				Resource: declaration.Owner, Env: descriptor.Env, LogicalID: descriptor.LogicalID,
				Field: descriptor.ResolvedField(), Owner: declaration.Owner, SourceRef: declaration.SourceRef,
				Kind: descriptorKind(descriptor), ConsumerRefs: bindingIDs(consumers), Label: descriptor.Label,
				Description: descriptor.Description, ObtainURL: descriptor.ObtainURL,
				Provisioning: descriptor.Provisioning, DerivedFrom: descriptor.DerivedFrom, Required: descriptor.Required,
			},
			Owner: declaration.Owner, SourceRef: declaration.SourceRef, Kind: descriptorKind(descriptor),
			Consumers: consumers, Disposition: disposition, Declared: declaration.Declared,
		})
		seenAddresses[address] = struct{}{}
	}
	for _, gap := range unknown {
		gaps = append(gaps, gap)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LogicalID+":"+rows[i].Field == rows[j].LogicalID+":"+rows[j].Field {
			return rows[i].SourceRef < rows[j].SourceRef
		}
		return rows[i].LogicalID+":"+rows[i].Field < rows[j].LogicalID+":"+rows[j].Field
	})
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Address == gaps[j].Address {
			return gaps[i].SourceRef < gaps[j].SourceRef
		}
		return gaps[i].Address < gaps[j].Address
	})
	return ConsumerInventory{
		Rows: rows, Gaps: gaps, DeclarationSiteCount: len(rows), DistinctAddressCount: len(seenAddresses),
		UnresolvedConsumerCount: len(unknown),
	}, nil
}

func registeredBinding(ownerRoot, owner string, consumer credentialspec.Consumer) ConsumerBinding {
	field := strings.TrimSpace(consumer.Field)
	if field == "" {
		field = credentialspec.DefaultField
	}
	kind := ConsumerKind(strings.TrimSpace(consumer.Kind))
	sourceRef := strings.TrimSpace(consumer.SourceRef)
	if sourceRef != "" && !filepath.IsAbs(sourceRef) {
		sourceRef = filepath.Join(ownerRoot, sourceRef)
	}
	if sourceRef != "" {
		sourceRef = filepath.Clean(sourceRef)
	}
	id := strings.TrimSpace(consumer.LogicalID) + ":" + field
	if pattern := strings.TrimSpace(consumer.AddressPattern); pattern != "" {
		id = pattern
	}
	return ConsumerBinding{ID: id, Owner: owner, Kind: kind, Consumer: strings.TrimSpace(consumer.Consumer), SourceRef: sourceRef}
}

func tierAppliesToConsumer(consumer credentialspec.Consumer, tier string) bool {
	if len(consumer.Tiers) == 0 {
		return true
	}
	for _, candidate := range consumer.Tiers {
		if strings.TrimSpace(candidate) == tier {
			return true
		}
	}
	return false
}

func addUnknownConsumer(unknown map[string]ConsumerInventoryGap, binding ConsumerBinding, required bool, reason, remediation string) {
	key := string(binding.Kind) + "\x00" + binding.ID + "\x00" + binding.SourceRef
	unknown[key] = ConsumerInventoryGap{Address: binding.ID, Owner: binding.Owner, Consumer: binding.Consumer, SourceRef: binding.SourceRef, Kind: string(binding.Kind), Required: required, Reason: reason, Remediation: remediation}
}

func registrantRequired(consumer credentialspec.Consumer) bool { return consumer.Required }

func discoverInventoryDeclarations(root string, scope Scope) ([]inventoryDeclaration, error) {
	declarations := make([]inventoryDeclaration, 0)
	add := func(owner, sourceRef, sourceRoot string, declaration credentialspec.Declaration) {
		declaredAddresses := make(map[string]struct{})
		for _, descriptor := range declaration.All() {
			if scope.Tier != "" && !tierApplies(descriptor, scope.Tier) {
				continue
			}
			address := descriptor.LogicalID + ":" + descriptor.ResolvedField()
			declaredAddresses[address] = struct{}{}
			declarations = append(declarations, inventoryDeclaration{Owner: owner, Root: sourceRoot, SourceRef: sourceRef, Descriptor: descriptor, Registrants: declaration.Consumers, Declared: true})
		}
		for _, consumer := range declaration.Consumers {
			if scope.Tier != "" && !tierAppliesToConsumer(consumer, scope.Tier) {
				continue
			}
			field := strings.TrimSpace(consumer.Field)
			if field == "" {
				field = credentialspec.DefaultField
			}
			address := strings.TrimSpace(consumer.LogicalID) + ":" + field
			if _, declared := declaredAddresses[address]; declared {
				continue
			}
			if strings.TrimSpace(consumer.LogicalID) == "" {
				// Address-pattern registrations are represented as explicit gaps;
				// they are not fabricated into a concrete declaration row, but
				// retain the owner/root so the registration is still projected.
				declarations = append(declarations, inventoryDeclaration{Owner: owner, Root: sourceRoot, SourceRef: sourceRef, Registrants: []credentialspec.Consumer{consumer}})
				continue
			}
			provisioning := ""
			if strings.TrimSpace(consumer.Kind) == string(ConsumerGenerated) {
				provisioning = credentialspec.ProvisioningGenerated
			}
			declarations = append(declarations, inventoryDeclaration{
				Owner: owner, Root: sourceRoot, SourceRef: sourceRef,
				Descriptor:  credentialspec.Descriptor{LogicalID: strings.TrimSpace(consumer.LogicalID), Field: field, Required: consumer.Required, Provisioning: provisioning},
				Registrants: []credentialspec.Consumer{consumer}, Declared: false,
			})
		}
	}
	if scope.IncludeProject {
		path := filepath.Join(root, ".vrooli", "service.json")
		if manifest, readErr := scenario.ReadService(path); readErr == nil {
			add(ProjectScopeOwner, path, root, manifest.Credentials)
		} else if !os.IsNotExist(readErr) {
			return nil, fmt.Errorf("read project service manifest: %w", readErr)
		}
	}
	if scope.Resources == nil || len(scope.Resources) > 0 {
		if scope.Resources != nil {
			for _, name := range scope.Resources {
				path := manifestpkg.DefaultPath(root, name)
				manifest, loadErr := manifestpkg.Load(path)
				if loadErr == nil {
					add(name, path, filepath.Dir(path), manifest.Credentials)
				}
			}
		} else {
			names, err := catalog.New(root).ManifestNames()
			if err != nil {
				return nil, fmt.Errorf("discover resource manifests: %w", err)
			}
			for _, name := range names {
				path := manifestpkg.DefaultPath(root, name)
				manifest, loadErr := manifestpkg.Load(path)
				if loadErr == nil {
					add(name, path, filepath.Dir(path), manifest.Credentials)
				}
			}
		}
	}
	if scope.Scenarios == nil || len(scope.Scenarios) > 0 {
		if scope.Scenarios != nil {
			for _, name := range scope.Scenarios {
				path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
				manifest, readErr := scenario.ReadService(path)
				if readErr != nil {
					return nil, fmt.Errorf("read selected scenario manifest %s: %w", name, readErr)
				}
				owner := strings.TrimSpace(manifest.Service.Name)
				if owner == "" {
					owner = name
				}
				add(owner, path, filepath.Join(root, "scenarios", name), manifest.Credentials)
			}
		} else {
			found, err := scenario.Discover(root, scenario.SandboxEnvFromEnv())
			if err != nil {
				return nil, fmt.Errorf("discover scenario manifests: %w", err)
			}
			for _, item := range found {
				add(item.Slug, item.ServicePath, item.Path, item.Manifest.Credentials)
			}
		}
	}
	if scope.IncludeManaged {
		for _, managed := range credentialinventory.ManagedSystemEntries(root) {
			declaration := credentialspec.Declaration{
				Descriptors: []credentialspec.Descriptor{{
					LogicalID:    managed.LogicalID,
					Field:        managed.Field,
					Required:     len(managed.Consumers) > 0,
					Kind:         credentialspec.KindDelegated,
					Provisioning: credentialspec.ProvisioningGenerated,
				}},
				Consumers: managed.Consumers,
			}
			add(managed.Owner, filepath.Join(root, "internal", "credentialinventory", "inventory.go"), root, declaration)
		}
	}
	return declarations, nil
}

func tierApplies(descriptor credentialspec.Descriptor, tier string) bool {
	if len(descriptor.Tiers) == 0 {
		return true
	}
	for _, candidate := range descriptor.Tiers {
		if strings.TrimSpace(candidate) == tier {
			return true
		}
	}
	return false
}

func descriptorKind(descriptor credentialspec.Descriptor) string {
	if strings.TrimSpace(descriptor.Kind) != "" {
		return descriptor.EffectiveKind()
	}
	switch strings.TrimSpace(descriptor.Provisioning) {
	case credentialspec.ProvisioningGenerated:
		return string(ConsumerGenerated)
	case credentialspec.ProvisioningDerived:
		return string(ConsumerDynamic)
	default:
		if descriptor.Injectable() {
			return string(ConsumerEnvironment)
		}
		return string(ConsumerAuthority)
	}
}

func sourceFiles(root, excluded string) ([]string, error) {
	return sourceFilesContext(context.Background(), root, excluded)
}

func sourceFilesContext(ctx context.Context, root, excluded string) ([]string, error) {
	files := make([]string, 0)
	root = filepath.Clean(root)
	generatedProtoRoot := filepath.Join(root, "packages", "proto", "gen")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == "vendor" || name == ".vrooli" || name == "test" || name == "tests" || name == "__tests__" {
				if filepath.Clean(path) != root {
					return filepath.SkipDir
				}
			}
			// A project-root owner is scanned alongside the selected scenario
			// and resource owners. Do not rescan those trees, and do not spend
			// readiness time on generated protobuf output.
			if filepath.Dir(path) == root && (name == "packages" || name == "scenarios" || name == "resources") {
				return filepath.SkipDir
			}
			if filepath.Clean(path) == generatedProtoRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Clean(path) == filepath.Clean(excluded) || filepath.Clean(path) == filepath.Join(filepath.Clean(root), ".vrooli", "service.json") {
			return nil
		}
		base := strings.ToLower(entry.Name())
		if strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".py", ".sh", ".ps1":
			if info, infoErr := entry.Info(); infoErr == nil && info.Size() <= 2<<20 {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan consumer source %s: %w", root, err)
	}
	sort.Strings(files)
	return files, nil
}

func scanConsumerFile(path, owner string) ([]ConsumerBinding, error) {
	return scanConsumerFileContext(context.Background(), path, owner)
}

func scanConsumerFileContext(ctx context.Context, path, owner string) ([]ConsumerBinding, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open consumer source %s: %w", path, err)
	}
	defer file.Close()
	bindings := make([]ConsumerBinding, 0)
	identities := make(map[string]string)
	scanner := bufio.NewScanner(file)
	// Bundled frontend assets can contain a single minified line close to the
	// source-file size limit. Keep the bounded scanner, but allow those lines
	// to be inspected instead of turning a valid owner tree into a gate error.
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	lineNumber := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lineNumber++
		line := scanner.Text()
		location := fmt.Sprintf("%s:%d", path, lineNumber)
		for _, match := range identityAssignment.FindAllStringSubmatch(line, -1) {
			identities[match[1]] = match[2]
		}
		for _, match := range literalCredentialUse.FindAllStringSubmatch(line, -1) {
			identity := match[1]
			if identity == "" {
				identity = identities[match[2]]
			}
			if identity == "" {
				continue
			}
			bindings = appendUniqueBinding(bindings, ConsumerBinding{ID: identity + ":" + match[3], Owner: owner, Kind: ConsumerAuthority, Consumer: path, SourceRef: location})
		}
		for _, match := range nestedLiteralCredentialUse.FindAllStringSubmatch(line, -1) {
			bindings = appendUniqueBinding(bindings, ConsumerBinding{ID: match[1] + ":" + match[2], Owner: owner, Kind: ConsumerAuthority, Consumer: path, SourceRef: location})
		}
		for _, match := range literalEnvUse.FindAllStringSubmatch(line, -1) {
			bindings = appendUniqueBinding(bindings, ConsumerBinding{ID: match[1], Owner: owner, Kind: ConsumerEnvironment, Consumer: path, SourceRef: location})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan consumer source %s: %w", path, err)
	}
	return bindings, nil
}

func appendUniqueBinding(bindings []ConsumerBinding, candidate ConsumerBinding) []ConsumerBinding {
	for _, binding := range bindings {
		if binding.ID == candidate.ID && binding.Kind == candidate.Kind && binding.SourceRef == candidate.SourceRef {
			return bindings
		}
	}
	return append(bindings, candidate)
}

func bindingIDs(bindings []ConsumerBinding) []string {
	ids := make([]string, 0, len(bindings))
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		id := binding.Consumer
		if id == "" {
			id = binding.ID
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// matchesAddressPattern supports the deliberately small pattern language used
// by credentialspec.Consumer: an exact address or a {name} placeholder for a
// dynamic logical-id segment. Patterns never become regular expressions, so a
// manifest cannot accidentally broaden a consumer registration through regex
// syntax.
func matchesAddressPattern(pattern, address string) bool {
	pattern = strings.TrimSpace(pattern)
	address = strings.TrimSpace(address)
	if pattern == "" || address == "" {
		return false
	}
	if pattern == address {
		return true
	}
	const placeholder = "{name}"
	index := strings.Index(pattern, placeholder)
	if index < 0 {
		return false
	}
	prefix, suffix := pattern[:index], pattern[index+len(placeholder):]
	if !strings.HasPrefix(address, prefix) || !strings.HasSuffix(address, suffix) {
		return false
	}
	value := strings.TrimSuffix(strings.TrimPrefix(address, prefix), suffix)
	return value != "" && !strings.ContainsAny(value, "{}")
}

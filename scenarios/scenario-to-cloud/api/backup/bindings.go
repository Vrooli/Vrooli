package backup

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// BindingInputs is what the resolver needs beside the closure: where the
// workload's data lives on the target and which hooks components declare.
type BindingInputs struct {
	// Workdir is the deployment workdir on the target; file bindings resolve
	// beneath <workdir>/scenarios/<owner>/<mount>.
	Workdir string
	// Hooks maps a component id to its declared deployment.backup hook.
	Hooks map[string]domain.DataHook
	// ReleaseHooks maps a component id to its declared release hook.
	ReleaseHooks map[string]domain.DataHook
	// ProviderRefs maps a binding id to the backup owner's target reference.
	ProviderRefs map[string]string
}

// ResolveBindings turns the closure's declared persistent data into engine
// bindings. Binding syntax (declared, never a host path for a database):
//
//	database:<name>   -> sql, provider postgres (the resource owner's tooling)
//	schema:<name>     -> sql, provider postgres
//	sqlite:<abs path> -> sql, provider sqlite (embedded database file)
//	dir:<mount>       -> files, provider object_store, beneath the owner's dir
//	bucket:<name>     -> files, provider object_store, absolute locator required
//	application:<id> -> application, provider application_hooks (hooks required)
//
// An undeclared or unresolvable binding is an invalid_request naming it; the
// resolver never guesses a provider.
func ResolveBindings(closure *domain.Closure, in BindingInputs) ([]domain.DataBinding, error) {
	if closure == nil {
		return nil, apierrors.New(apierrors.CodeClosureUnavailable, "closure is required to resolve data bindings")
	}
	out := make([]domain.DataBinding, 0, len(closure.PersistentData))
	for _, data := range closure.PersistentData {
		binding, err := resolveOne(data, in)
		if err != nil {
			return nil, err
		}
		out = append(out, binding)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func resolveOne(data domain.ClosurePersistentData, in BindingInputs) (domain.DataBinding, error) {
	kind, locator, ok := strings.Cut(strings.TrimSpace(data.Binding), ":")
	if !ok || strings.TrimSpace(locator) == "" {
		return domain.DataBinding{}, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: binding %q is not <kind>:<locator>", data.ID, data.Binding).WithDetail("persistent_data", data.ID)
	}
	binding := domain.DataBinding{ID: data.ID, Owner: data.Owner, MigrationOwner: data.MigrationOwner, ProviderRef: in.ProviderRefs[data.ID]}
	switch kind {
	case "database", "schema":
		binding.Kind, binding.Provider, binding.Locator = domain.DataBindingKindSQL, domain.BackupProviderPostgres, locator
	case "sqlite":
		if !path.IsAbs(locator) {
			return domain.DataBinding{}, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: sqlite locator must be absolute", data.ID)
		}
		binding.Kind, binding.Provider, binding.Locator = domain.DataBindingKindSQL, domain.BackupProviderSQLite, locator
	case "dir":
		if strings.TrimSpace(in.Workdir) == "" || strings.TrimSpace(data.Owner) == "" {
			return domain.DataBinding{}, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: dir binding needs a workdir and an owner", data.ID)
		}
		clean := path.Clean("/" + locator)
		if clean == "/" || strings.Contains(locator, "..") {
			return domain.DataBinding{}, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: dir binding %q escapes the owner directory", data.ID, locator)
		}
		binding.Kind, binding.Provider = domain.DataBindingKindFiles, domain.BackupProviderObjectStore
		binding.Locator = path.Join(in.Workdir, "scenarios", data.Owner, clean)
	case "bucket":
		if !path.IsAbs(locator) {
			return domain.DataBinding{}, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: bucket locator must be an absolute mount", data.ID)
		}
		binding.Kind, binding.Provider, binding.Locator = domain.DataBindingKindFiles, domain.BackupProviderObjectStore, locator
	case "application":
		binding.Kind, binding.Provider, binding.Locator = domain.DataBindingKindApplication, domain.BackupProviderApplicationHooks, locator
	default:
		return domain.DataBinding{}, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: binding kind %q is not database, schema, sqlite, dir, bucket or application", data.ID, kind)
	}
	if data.MigrationOwner == "scenario" || binding.Kind == domain.DataBindingKindApplication {
		owner := data.DeclaredBy
		if idx := strings.Index(owner, ":"); idx >= 0 {
			owner = owner[idx+1:]
		}
		if hook, ok := in.Hooks[owner]; ok {
			h := hook
			binding.Quiesce = &h
		}
		if hook, ok := in.ReleaseHooks[owner]; ok {
			h := hook
			binding.Release = &h
		}
	}
	if binding.Kind == domain.DataBindingKindApplication && binding.Quiesce == nil {
		return domain.DataBinding{}, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: application binding declares no quiesce hook", data.ID)
	}
	return binding, nil
}

// EngineBindings converts domain bindings into the engine's shape.
func EngineBindings(bindings []domain.DataBinding) []recoverypoint.Binding {
	out := make([]recoverypoint.Binding, 0, len(bindings))
	for _, b := range bindings {
		e := recoverypoint.Binding{ID: b.ID, Owner: b.Owner, Kind: b.Kind, Provider: b.Provider, Locator: b.Locator, MigrationOwner: b.MigrationOwner}
		if b.Quiesce != nil {
			e.Quiesce = &recoverypoint.Hook{Tool: b.Quiesce.Tool, Argv: append([]string(nil), b.Quiesce.Argv...)}
		}
		if b.Release != nil {
			e.Release = &recoverypoint.Hook{Tool: b.Release.Tool, Argv: append([]string(nil), b.Release.Argv...)}
		}
		out = append(out, e)
	}
	return out
}

// DomainBindings converts engine bindings back into the domain shape.
func DomainBindings(bindings []recoverypoint.Binding) []domain.DataBinding {
	out := make([]domain.DataBinding, 0, len(bindings))
	for _, b := range bindings {
		d := domain.DataBinding{ID: b.ID, Owner: b.Owner, Kind: b.Kind, Provider: b.Provider, Locator: b.Locator, MigrationOwner: b.MigrationOwner}
		if b.Quiesce != nil {
			d.Quiesce = &domain.DataHook{Tool: b.Quiesce.Tool, Argv: append([]string(nil), b.Quiesce.Argv...)}
		}
		if b.Release != nil {
			d.Release = &domain.DataHook{Tool: b.Release.Tool, Argv: append([]string(nil), b.Release.Argv...)}
		}
		out = append(out, d)
	}
	return out
}

// ValidateBindings refuses bindings that would let a database be captured by
// anything but its owner's native tooling.
func ValidateBindings(bindings []domain.DataBinding) error {
	if len(bindings) == 0 {
		return apierrors.New(apierrors.CodeInvalidRequest, "at least one data binding is required")
	}
	seen := map[string]bool{}
	for _, b := range bindings {
		if seen[b.ID] {
			return apierrors.Newf(apierrors.CodeInvalidRequest, "binding %s is declared twice", b.ID)
		}
		seen[b.ID] = true
		if b.Kind == domain.DataBindingKindSQL && b.Provider == domain.BackupProviderObjectStore {
			return apierrors.New(apierrors.CodeInvalidRequest, fmt.Sprintf("binding %s: a database is never captured by a generic file copy", b.ID)).WithDetail("binding", b.ID)
		}
		if err := (recoverypoint.Binding{ID: b.ID, Kind: b.Kind, Provider: b.Provider, Locator: b.Locator}).Validate(); err != nil {
			return FromEngine(err, recoverypoint.CodeInvalidArgument)
		}
	}
	return nil
}

package archtest

import (
	"go/ast"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// sshImporters lists the packages allowed to import scenario-to-cloud/ssh
// while the package exists. The bounded reach adapter is the single cloud
// owner of the ssh transport (DL-02, P09): it turns a typed reach.Command
// into an argv-only remote invocation. Nothing else may hold an SSH runner,
// because a second holder is a second transport authority with its own
// policy (host keys, key custody, shell composition).
var sshImporters = map[string]string{
	"reach/sshadapter": "the bounded ssh transport adapter behind reach.Reach (P09); the only argv-to-ssh seam",
}

// TestOnlyTheReachAdapterImportsSSH [REQ:STC-P0-043] fails when any package
// outside the allowlist imports the ssh package. When the package is deleted
// (DL-02 closes) the test still guards against a new package at that path.
func TestOnlyTheReachAdapterImportsSSH(t *testing.T) {
	_, src := loadSources(t)
	for _, pkg := range packagesImporting(src, modulePath+"/ssh") {
		if _, ok := sshImporters[pkg]; ok {
			continue
		}
		t.Errorf("package %s imports %s/ssh; only %v may hold an ssh runner (deletion ledger DL-02)", packageName(pkg), modulePath, sortedKeys(toSet(sshImporters)))
	}
}

// execUsers lists the packages allowed to import os/exec and why. Every
// entry is a local process the cloud side must start itself; none of them
// reaches a target. A target effect is a typed verb through reach, never a
// local exec.
var execUsers = map[string]string{
	"reach/sshadapter": "starts the local ssh/scp client for the bounded adapter (P09)",
	"release":          "reproducible native CLI build (go), git inputs and release-authority signing on the build host (P10)",
	"backup":           "runs the declared consistency provider argv (pg_dump, data-backup-manager) on the operator side (P12)",
	"instance":         "local QEMU/cloud-localds provider for the disposable qualification instance (P20)",
	"sshidentity":      "ssh-keygen fingerprint of a locally held public key (identity display, no target reach)",
}

// TestOSExecIsConfinedToLocalProcessOwners [REQ:STC-P0-043] fails when a
// package outside the allowlist imports os/exec, and when an allowlisted
// package no longer uses it (a stale allowlist is a boundary nobody checks).
func TestOSExecIsConfinedToLocalProcessOwners(t *testing.T) {
	_, src := loadSources(t)
	users := packagesImporting(src, "os/exec")
	seen := map[string]struct{}{}
	for _, pkg := range users {
		seen[pkg] = struct{}{}
		if _, ok := execUsers[pkg]; ok {
			continue
		}
		t.Errorf("package %s imports os/exec; target effects are reach verbs and local processes need an allowlist entry with a reason", packageName(pkg))
	}
	for pkg := range execUsers {
		if _, ok := seen[pkg]; !ok {
			t.Errorf("allowlist entry %q no longer imports os/exec; remove the entry", pkg)
		}
	}
}

// productNameLiterals are the only places a hosted-consumer product name may
// appear in non-test cloud source: documented examples shown to an operator.
// Generic cloud code paths carry no product-name conditional (brief rule 9).
var productNameLiterals = map[string]string{
	"manifest/validator.go": "the manifest doctor's example hint for scenario.id; text shown to an operator, never compared",
}

var productNamePattern = regexp.MustCompile(`(?i)landing-page-business-suite|\blpbs\b`)

// TestNoProductNameConditionalsInCloudCode [REQ:STC-P0-043] fails when a
// non-test source file mentions the representative hosted consumer outside
// the documented-example allowlist.
func TestNoProductNameConditionalsInCloudCode(t *testing.T) {
	_, src := loadSources(t)
	used := map[string]struct{}{}
	for _, f := range src {
		f.stringLiterals(func(lit *ast.BasicLit, value string) {
			if !productNamePattern.MatchString(value) {
				return
			}
			if _, ok := productNameLiterals[f.Rel]; ok {
				used[f.Rel] = struct{}{}
				return
			}
			t.Errorf("%s names the hosted consumer in a string literal; generic cloud code paths carry no product name", f.position(lit))
		})
		for _, group := range f.File.Comments {
			if productNamePattern.MatchString(group.Text()) {
				if _, ok := productNameLiterals[f.Rel]; ok {
					used[f.Rel] = struct{}{}
					continue
				}
				t.Errorf("%s names the hosted consumer in a comment; document the example in the allowlist or drop it", f.position(group))
			}
		}
	}
	for rel := range productNameLiterals {
		if _, ok := used[rel]; !ok {
			t.Errorf("allowlist entry %q no longer names the product; remove the entry", rel)
		}
	}
}

// cloudOperationsWrite matches SQL that changes cloud_operations rows. DDL
// (CREATE/ALTER) is schema ownership, held by persistence itself.
var cloudOperationsWrite = regexp.MustCompile(`(?is)\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+cloud_operations\b`)

// operationWriters lists the packages allowed to call a persistence method
// that writes cloud_operations. The durable operation owner (P07) admits,
// acquires, heartbeats, commits receipts and transitions state; every other
// package reads standing or hands a reviewed plan to operations.Admit.
var operationWriters = map[string]string{
	"operations": "the durable operation owner: admission, fence acquisition, receipts, state transitions (DL-05)",
}

// TestOnlyTheOperationsOwnerWritesOperationRows [REQ:STC-P0-043] derives the
// set of persistence methods whose SQL writes cloud_operations (directly or
// through another persistence method) and fails when any package outside
// api/operations calls one of them.
func TestOnlyTheOperationsOwnerWritesOperationRows(t *testing.T) {
	_, src := loadSources(t)
	writers := map[string]struct{}{}
	// Pass 1: methods whose body carries a write statement.
	for _, f := range src {
		if f.Pkg != "persistence" {
			continue
		}
		f.funcBodies(func(decl *ast.FuncDecl, receiver string) {
			if receiver != "Repository" || !decl.Name.IsExported() {
				return
			}
			ast.Inspect(decl.Body, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if ok && cloudOperationsWrite.MatchString(unquoted(lit)) {
					writers[decl.Name.Name] = struct{}{}
				}
				return true
			})
		})
	}
	if _, ok := writers["CreateOperation"]; !ok {
		t.Fatalf("writer derivation is broken: CreateOperation not found; derived %v", sortedKeys(writers))
	}
	// Pass 2: persistence methods that delegate to a writer are writers too
	// (iterate to a fixpoint; the graph is tiny).
	for changed := true; changed; {
		changed = false
		for _, f := range src {
			if f.Pkg != "persistence" {
				continue
			}
			f.funcBodies(func(decl *ast.FuncDecl, receiver string) {
				if receiver != "Repository" || !decl.Name.IsExported() {
					return
				}
				if _, already := writers[decl.Name.Name]; already {
					return
				}
				f.selectorCallsIn(decl.Body, writers, func(*ast.CallExpr, string) {
					writers[decl.Name.Name] = struct{}{}
					changed = true
				})
			})
		}
	}
	// Pass 3: callers outside persistence.
	for _, f := range src {
		if f.Pkg == "persistence" {
			continue
		}
		f.selectorCalls(writers, func(call *ast.CallExpr, name string) {
			if _, ok := operationWriters[f.Pkg]; ok {
				return
			}
			t.Errorf("%s: package %s calls persistence.%s, which writes cloud_operations; only %v may (use operations.Admit for admission)", f.position(call), packageName(f.Pkg), name, sortedKeys(toSet(operationWriters)))
		})
	}
}

// credentialLifecycleOwner is the only package that composes a target-side
// credential verb (`vrooli cloud-target credential ingest|acknowledge|revoke`).
// Distribution, acknowledgement and revocation are the credential lifecycle
// (P13, DL-14): a second composer would be a second lifecycle authority.
const credentialLifecycleOwner = "credentials"

var credentialVerbs = map[string]struct{}{"ingest": {}, "acknowledge": {}, "revoke": {}}

// TestOnlyTheCredentialsOwnerComposesCredentialVerbs [REQ:STC-P0-043] fails
// when a composite literal outside api/credentials spells a cloud-target
// credential verb.
func TestOnlyTheCredentialsOwnerComposesCredentialVerbs(t *testing.T) {
	_, src := loadSources(t)
	found := 0
	for _, f := range src {
		ast.Inspect(f.File, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			hasCredential, hasVerb := false, false
			for _, elt := range lit.Elts {
				basic, ok := elt.(*ast.BasicLit)
				if !ok {
					continue
				}
				value := unquoted(basic)
				if value == "credential" {
					hasCredential = true
				}
				if _, ok := credentialVerbs[value]; ok {
					hasVerb = true
				}
			}
			if !hasCredential || !hasVerb {
				return true
			}
			found++
			if f.Pkg != credentialLifecycleOwner {
				t.Errorf("%s composes a cloud-target credential verb; only api/%s owns the credential lifecycle (DL-14)", f.position(lit), credentialLifecycleOwner)
			}
			return true
		})
	}
	if found == 0 {
		t.Fatal("no credential verb composition found; the derivation is broken")
	}
}

// authorityLibraryUsers lists the packages allowed to open the local
// credential-authority library client (credentialauthority.Default) and why.
// Each resolves one operator-side value for a declared purpose; none manages
// credential state, which is the credentials package's through the target
// verbs above.
var authorityLibraryUsers = map[string]string{
	"backup":  "resolves the recovery key reference and the managed postgres password for the local consistency provider (P12); operator-side by design, never on the target",
	"secrets": "reads the secrets-manager service token for the deployment-tier client; a bearer for another owner, not a managed binding",
	"vps":     "resolves declared bundle secrets from the onboarding identity at deploy time (credentials.provision); values travel only on the ingest verb's stdin",
	"":        "handlers_publication.go binds the Ed25519 receipt signer to the authority's trust root (P17); a signing key store, not a lifecycle call",
}

// TestAuthorityLibraryIsOpenedByDeclaredOwnersOnly [REQ:STC-P0-043] fails
// when a package outside the allowlist opens the local authority client, and
// when an allowlisted package no longer does.
func TestAuthorityLibraryIsOpenedByDeclaredOwnersOnly(t *testing.T) {
	_, src := loadSources(t)
	const authorityImport = "github.com/vrooli/vrooli/packages/credential-authority-go"
	seen := map[string]struct{}{}
	for _, f := range src {
		if !f.importsPackage(authorityImport) {
			continue
		}
		f.selectorCalls(map[string]struct{}{"Default": {}}, func(call *ast.CallExpr, _ string) {
			sel := call.Fun.(*ast.SelectorExpr)
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "credentialauthority" {
				return
			}
			seen[f.Pkg] = struct{}{}
			if _, ok := authorityLibraryUsers[f.Pkg]; ok {
				return
			}
			t.Errorf("%s: package %s opens the credential authority library; every use needs an allowlist reason", f.position(call), packageName(f.Pkg))
		})
	}
	for pkg := range authorityLibraryUsers {
		if _, ok := seen[pkg]; !ok {
			t.Errorf("allowlist entry %q no longer opens the credential authority; remove the entry", packageName(pkg))
		}
	}
}

func unquoted(lit *ast.BasicLit) string {
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return strings.Trim(lit.Value, "`")
	}
	return value
}

func packageName(pkg string) string {
	if pkg == "" {
		return "main"
	}
	return pkg
}

func toSet(m map[string]string) map[string]struct{} {
	out := make(map[string]struct{}, len(m))
	for k := range m {
		out[k] = struct{}{}
	}
	return out
}

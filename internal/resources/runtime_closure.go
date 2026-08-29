package resources

import (
	"debug/elf"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// A digest proves an artifact is the right bytes. It does not prove the host
// can start it. The reranker CPU target passed its digest check and could not
// load libiomp5.so, so the fallback the resolver picked in the fact-blind
// window was never runnable either — and nothing said so until the service log
// was read by hand.
//
// This file answers the missing question: can the loader satisfy this
// executable's dynamic dependencies on this host? It answers it by reading the
// binary's own headers. It never executes the artifact: running an untrusted
// staged binary to discover what it needs is the wrong trade.

// ClosureState is the verdict on one artifact's runtime closure.
type ClosureState string

const (
	// ClosureOK means every declared dependency resolved.
	ClosureOK ClosureState = "ok"
	// ClosureUnresolved means at least one declared dependency did not resolve.
	ClosureUnresolved ClosureState = "unresolved"
	// ClosureUnknown means the check could not run: an unsupported binary
	// format, an unsupported platform, or an unreadable file. It never blocks
	// an install, because an unanswerable question is not a failed one.
	ClosureUnknown ClosureState = "unknown"
)

// ClosureVerdict is what the check found.
type ClosureVerdict struct {
	State ClosureState `json:"state"`
	// Needed are the dynamic dependencies the binary declares.
	Needed []string `json:"needed,omitempty"`
	// Unresolved are the dependencies no searched directory provided.
	Unresolved []string `json:"unresolved,omitempty"`
	// Searched are the directories that were searched, in the order the loader
	// would search them.
	Searched []string `json:"searched,omitempty"`
	// Reason explains an unknown verdict.
	Reason string `json:"reason,omitempty"`
}

// ErrRuntimeClosure is returned when a staged artifact declares a dependency
// the host cannot provide.
var ErrRuntimeClosure = errors.New("runtime closure is not satisfiable")

// RuntimeClosureError names every unresolved library and every path searched,
// so the operator does not have to reconstruct the loader's view by hand.
type RuntimeClosureError struct {
	Resource string
	Artifact string
	Verdict  ClosureVerdict
}

func (e *RuntimeClosureError) Error() string {
	return fmt.Sprintf(
		"resource %s artifact %s declares shared libraries this host cannot provide: %s (searched %s). The digest is correct; the artifact simply cannot start here",
		e.Resource, filepath.Base(e.Artifact), strings.Join(e.Verdict.Unresolved, ", "), strings.Join(e.Verdict.Searched, ", "))
}

// Unwrap lets errors.Is(err, ErrRuntimeClosure) succeed.
func (e *RuntimeClosureError) Unwrap() error { return ErrRuntimeClosure }

// defaultLibraryDirs are the loader search directories per platform, in the
// order the loader consults them. They are a floor, not the whole story: a
// host with a custom ld.so.conf has more, which is why an unresolved verdict
// names what it searched rather than asserting the library is absent from the
// machine.
var defaultLibraryDirs = map[string][]string{
	"linux": {
		"/lib/x86_64-linux-gnu", "/usr/lib/x86_64-linux-gnu",
		"/lib/aarch64-linux-gnu", "/usr/lib/aarch64-linux-gnu",
		"/lib64", "/usr/lib64", "/lib", "/usr/lib", "/usr/local/lib",
	},
	"darwin": {
		"/usr/lib", "/usr/local/lib", "/opt/homebrew/lib", "/System/Library/Frameworks",
	},
}

// VerifyRuntimeClosure reads an executable's declared dynamic dependencies and
// reports whether each one resolves.
//
// extraDirs are directories the artifact ships or declares, searched ahead of
// the host's own: a target that bundles its libraries beside the executable
// must not be rejected because the host lacks them system-wide.
func VerifyRuntimeClosure(artifactPath string, extraDirs []string) ClosureVerdict {
	needed, verdict := declaredDependencies(artifactPath)
	if verdict.State == ClosureUnknown {
		return verdict
	}
	if len(needed) == 0 {
		return ClosureVerdict{State: ClosureOK, Reason: "the artifact declares no dynamic dependencies"}
	}

	searched := closureSearchDirs(artifactPath, extraDirs)
	result := ClosureVerdict{State: ClosureOK, Needed: needed, Searched: searched}
	for _, library := range needed {
		if !libraryResolves(library, searched) {
			result.Unresolved = append(result.Unresolved, library)
		}
	}
	if len(result.Unresolved) > 0 {
		result.State = ClosureUnresolved
		slices.Sort(result.Unresolved)
	}
	return result
}

// closureSearchDirs builds the loader's search order: the artifact's own
// directory and its conventional lib subdirectories first, then any declared
// library paths, then the host defaults.
func closureSearchDirs(artifactPath string, extraDirs []string) []string {
	base := filepath.Dir(artifactPath)
	dirs := []string{base, filepath.Join(base, "lib"), filepath.Join(base, "lib64")}
	for _, dir := range extraDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(base, dir)
		}
		dirs = append(dirs, dir)
	}
	dirs = append(dirs, defaultLibraryDirs[runtime.GOOS]...)

	seen := make(map[string]struct{}, len(dirs))
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		clean := filepath.Clean(dir)
		if _, duplicate := seen[clean]; duplicate {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

// libraryResolves reports whether any searched directory provides the library.
// A Mach-O load command carries an absolute path or an @rpath-relative one; an
// absolute path is checked where it points.
func libraryResolves(library string, searched []string) bool {
	if strings.HasPrefix(library, "/") {
		if _, err := os.Stat(library); err == nil {
			return true
		}
		// Fall through: an absolute path that is missing may still be provided
		// by the same basename in a searched directory, which is what a
		// relocated bundle looks like.
		library = filepath.Base(library)
	}
	library = strings.TrimPrefix(library, "@rpath/")
	library = strings.TrimPrefix(library, "@loader_path/")
	library = strings.TrimPrefix(library, "@executable_path/")
	for _, dir := range searched {
		if _, err := os.Stat(filepath.Join(dir, library)); err == nil {
			return true
		}
	}
	return false
}

// declaredDependencies reads the binary's own headers. Nothing is executed.
func declaredDependencies(artifactPath string) ([]string, ClosureVerdict) {
	info, err := os.Stat(artifactPath)
	if err != nil {
		return nil, ClosureVerdict{State: ClosureUnknown, Reason: fmt.Sprintf("cannot read the staged artifact: %v", err)}
	}
	if info.IsDir() {
		return nil, ClosureVerdict{State: ClosureUnknown, Reason: "the staged artifact is a directory; the closure check inspects a single executable"}
	}

	if file, err := elf.Open(artifactPath); err == nil {
		defer file.Close()
		needed, err := file.DynString(elf.DT_NEEDED)
		if err != nil {
			// A static binary has no dynamic section at all, which is a
			// satisfied closure rather than an unreadable one.
			return nil, ClosureVerdict{State: ClosureOK, Reason: "the ELF artifact has no dynamic section, so it is statically linked"}
		}
		return needed, ClosureVerdict{State: ClosureOK}
	}

	if file, err := macho.Open(artifactPath); err == nil {
		defer file.Close()
		libraries, err := file.ImportedLibraries()
		if err != nil {
			return nil, ClosureVerdict{State: ClosureUnknown, Reason: fmt.Sprintf("cannot read Mach-O load commands: %v", err)}
		}
		return libraries, ClosureVerdict{State: ClosureOK}
	}

	// Windows PE import tables name DLLs resolved through a search order this
	// check does not model. Reporting unknown is honest; blocking a Windows
	// install on an unimplemented check would not be.
	return nil, ClosureVerdict{
		State:  ClosureUnknown,
		Reason: fmt.Sprintf("the staged artifact is not an ELF or Mach-O executable, so its runtime closure cannot be read on %s", runtime.GOOS),
	}
}

// StagedArtifactClosure reports the runtime-closure verdict for the artifact
// currently staged for a resource. ok is false when the resource stages no
// artifact or none is present, which is not a failure — there is simply nothing
// to inspect yet.
func (c *Controller) StagedArtifactClosure(manifest ResourceManifest) (ClosureVerdict, bool) {
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return ClosureVerdict{}, false
	}
	path, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		return ClosureVerdict{}, false
	}
	if _, err := os.Stat(path); err != nil {
		return ClosureVerdict{}, false
	}
	return verifyManagedServiceRuntimeClosure(manifest, path), true
}

package adoptions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/testutil/mocks"
)

// libBody is a realistic multi-line component body. A verbatim header-less copy
// of it should score ~1.0; a re-implemented sibling primitive scores far lower.
const libBody = `import { forwardRef, type InputHTMLAttributes } from "react";

export type InputProps = InputHTMLAttributes<HTMLInputElement>;

export const Input = forwardRef<HTMLInputElement, InputProps>(
  function Input({ className, type, ...props }, ref) {
    return (
      <input
        ref={ref}
        type={type}
        className={className}
        {...props}
      />
    );
  },
);
`

func discoverLibrary() *fakeLibrary {
	return &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-input": {ID: "cmp-input", LibraryID: "rcl:Input", DisplayName: "Input", Version: "1.1.0", LatestVersion: "1.1.0"},
		},
		versions: map[string]components.ComponentVersion{
			"cmp-input@1.1.0": {
				ComponentID: "cmp-input", LibraryID: "rcl:Input", Version: "1.1.0",
				Status:        components.VersionStatusReleased,
				Content:       libBody,
				ContentSHA256: sha(libBody),
				Files:         []components.ComponentVersionFile{{Path: "Input.tsx", Content: libBody, ContentSHA256: sha(libBody), IsEntry: true}},
			},
		},
	}
}

func TestService_Discover_FindsPlantedHeaderlessCopy(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := discoverLibrary()
	// A verbatim vendored copy that lost its provenance header.
	files := &fakeFiles{
		bytes:    map[string][]byte{"experience-manager::ui/src/components/ui/input.tsx": []byte(libBody)},
		untagged: []adoptions.CandidateFile{{Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/input.tsx", Content: []byte(libBody)}},
	}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	out, err := svc.Discover(context.Background(), adoptions.DiscoverInput{})
	require.NoError(t, err)
	require.Equal(t, 1, out.Scanned)
	require.Len(t, out.Candidates, 1, "the planted header-less copy must surface")
	c := out.Candidates[0]
	require.Equal(t, "experience-manager", c.Scenario)
	require.Equal(t, "ui/src/components/ui/input.tsx", c.AdoptedPath)
	require.Equal(t, "cmp-input", c.ComponentID)
	require.Equal(t, "rcl:Input", c.LibraryID)
	require.Equal(t, "1.1.0", c.Version)
	require.InDelta(t, 1.0, c.Similarity, 0.001, "a verbatim copy scores ~1.0")
	require.True(t, c.BasenameMatch)
	require.NotEmpty(t, c.Evidence)
}

func TestService_Discover_SkipsRecordedAndBelowThreshold(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	// An already-tracked copy: header stripped on disk but a record exists.
	repo.Seed(adoptions.Adoption{
		ID: "rec-1", ComponentID: "cmp-input", LibraryID: "rcl:Input",
		Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/input.tsx",
		Files: []adoptions.AdoptionFile{{AdoptedPath: "ui/src/components/ui/input.tsx"}},
	})
	lib := discoverLibrary()
	files := &fakeFiles{untagged: []adoptions.CandidateFile{
		// Recorded path — must be skipped (not drift-blind).
		{Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/input.tsx", Content: []byte(libBody)},
		// A genuinely unrelated file — must fall below threshold.
		{Scenario: "other", AdoptedPath: "ui/src/pages/Dashboard.tsx", Content: []byte("export const Dashboard = () => <main>totally unrelated content here</main>;\nconst x = 1;\nconst y = 2;\n")},
	}}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	out, err := svc.Discover(context.Background(), adoptions.DiscoverInput{})
	require.NoError(t, err)
	require.Equal(t, 2, out.Scanned)
	require.Empty(t, out.Candidates, "recorded paths and dissimilar files must not surface")
}

func TestService_ConfirmDiscovery_InjectsHeaderAndRecords(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := discoverLibrary()
	files := &fakeFiles{
		bytes:    map[string][]byte{"experience-manager::ui/src/components/ui/input.tsx": []byte(libBody)},
		untagged: []adoptions.CandidateFile{{Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/input.tsx", Content: []byte(libBody)}},
	}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	res, err := svc.ConfirmDiscovery(context.Background(), adoptions.ConfirmDiscoveryInput{
		Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/input.tsx",
		ComponentID: "cmp-input", Version: "1.1.0",
	})
	require.NoError(t, err)
	require.InDelta(t, 1.0, res.Similarity, 0.001)
	require.Equal(t, "1.1.0", res.Adoption.AdoptedVersion)
	require.Equal(t, adoptions.LibraryVersionStatusCurrent, res.Adoption.LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, res.Adoption.LocalStatus)
	require.Len(t, res.Adoption.Files, 1)
	require.Equal(t, sha(libBody), res.Adoption.Files[0].SourceSHA256)

	// The header must now be on disk, above the original body.
	written := string(files.bytes["experience-manager::ui/src/components/ui/input.tsx"])
	require.Contains(t, written, "@vrooliComponentSource rcl:Input")
	require.Contains(t, written, "@vrooliComponentVersion 1.1.0")
	require.Contains(t, written, "export const Input")

	// It is no longer a candidate: recorded + would now be header-tagged.
	files.untagged = nil // FS scanner would drop the now-tagged file
	out, err := svc.Discover(context.Background(), adoptions.DiscoverInput{})
	require.NoError(t, err)
	require.Empty(t, out.Candidates)

	// Confirming an already-tagged file is refused.
	_, err = svc.ConfirmDiscovery(context.Background(), adoptions.ConfirmDiscoveryInput{
		Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/input.tsx",
		ComponentID: "cmp-input", Version: "1.1.0",
	})
	require.Error(t, err)
}

// TestFSScanner_PartitionsTaggedAndUntagged proves the single filesystem walk
// routes each file to exactly one of ScanProvenance (header-tagged) or
// ScanUntagged (header-less) — so a tagged copy is never double-reported as a
// discovery candidate.
func TestFSScanner_PartitionsTaggedAndUntagged(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "experience-manager", "ui", "src", "components", "ui")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	tagged := "/**\n * @vrooliComponentSource rcl:Button\n * @vrooliComponentVersion 1.2.0\n */\nexport const Button = () => null;\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "button.tsx"), []byte(tagged), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "input.tsx"), []byte(libBody), 0o644))

	reader := adoptions.NewFSScenarioFileReader(root)

	prov, err := reader.ScanProvenance(context.Background())
	require.NoError(t, err)
	require.Len(t, prov, 1)
	require.Equal(t, "ui/src/components/ui/button.tsx", prov[0].AdoptedPath)

	untagged, err := reader.ScanUntagged(context.Background())
	require.NoError(t, err)
	require.Len(t, untagged, 1)
	require.Equal(t, "ui/src/components/ui/input.tsx", untagged[0].AdoptedPath)

	// No path appears in both partitions.
	require.NotEqual(t, prov[0].AdoptedPath, untagged[0].AdoptedPath)
}

func TestDiscover_RejectsDivergentSiblingPrimitive(t *testing.T) {
	// A shadcn-style cva button shares only a couple of Tailwind strings with a
	// clsx/twMerge library button — it must stay below the default threshold,
	// mirroring the real experience-manager button.tsx disposition.
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{"cmp-btn": {ID: "cmp-btn", LibraryID: "rcl:Button", DisplayName: "Button", Version: "1.2.0", LatestVersion: "1.2.0"}},
		versions: map[string]components.ComponentVersion{
			"cmp-btn@1.2.0": {
				ComponentID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.2.0", Status: components.VersionStatusReleased,
				Content: libBody, ContentSHA256: sha(libBody),
				Files: []components.ComponentVersionFile{{Path: "Button.tsx", Content: "import { clsx } from \"clsx\";\nimport { twMerge } from \"tailwind-merge\";\nexport function Button({ children, variant }) {\n  return <button className={twMerge(clsx(variant))}>{children}</button>;\n}\n", ContentSHA256: sha("b"), IsEntry: true}},
			},
		},
	}
	divergent := "import { cva } from \"class-variance-authority\";\nimport { Slot } from \"@radix-ui/react-slot\";\nconst buttonVariants = cva(\"inline-flex\");\nexport function Button({ asChild }) {\n  const Comp = asChild ? Slot : \"button\";\n  return <Comp className={buttonVariants()} />;\n}\n"
	files := &fakeFiles{untagged: []adoptions.CandidateFile{{Scenario: "experience-manager", AdoptedPath: "ui/src/components/ui/button.tsx", Content: []byte(divergent)}}}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	out, err := svc.Discover(context.Background(), adoptions.DiscoverInput{})
	require.NoError(t, err)
	require.Empty(t, out.Candidates, "a re-implemented sibling primitive must not be auto-surfaced as a copy")

	// Lowering the threshold surfaces it for review, but it is clearly weak.
	lowered, err := svc.Discover(context.Background(), adoptions.DiscoverInput{MinSimilarity: 0.05})
	require.NoError(t, err)
	if len(lowered.Candidates) > 0 {
		require.Less(t, lowered.Candidates[0].Similarity, 0.5)
	}
}

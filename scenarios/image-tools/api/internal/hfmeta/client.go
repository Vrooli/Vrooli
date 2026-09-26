package hfmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/envkit-go"
)

// HFClient is the production Fetcher. It dispatches by source shape:
//
//   - a local path that exists → inspected directly off disk (no network);
//   - an http(s):// URL → a single-file checkpoint with minimal metadata
//     (license/NSFW unknown ⇒ the import gate treats it as unverified);
//   - otherwise an "org/name" HuggingFace repo id → probed via huggingface_hub.
//
// The HF probe shells the same governed huggingface_hub dependency the snapshot
// fetcher uses (internal/fetch). The Runner seam lets a recorded-fixture test
// replay captured HF JSON without a network.
type HFClient struct {
	// Python is the interpreter for the HF probe (defaults to "python3").
	Python string
	// Env, when set, replaces the child environment (else os.Environ is used).
	Env []string
	// Runner, when set, overrides the default python probe and returns the raw HF
	// JSON for a repo id. Used by recorded-fixture tests.
	Runner func(ctx context.Context, repoID string) ([]byte, error)
}

// Inspect resolves Metadata for source. It never downloads weights.
func (c *HFClient) Inspect(ctx context.Context, source string) (Metadata, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return Metadata{}, fmt.Errorf("hfmeta: empty source")
	}
	if pathExists(source) {
		return inspectLocal(source)
	}
	if hasURLScheme(source) {
		return inspectURL(source), nil
	}
	return c.inspectRepo(ctx, source)
}

// rawHF is the JSON shape the python probe emits (a flattened subset of
// HfApi.model_info + a model_index.json peek).
type rawHF struct {
	RepoID        string     `json:"repo_id"`
	Revision      string     `json:"revision"`
	PipelineClass string     `json:"pipeline_class"`
	Tags          []string   `json:"tags"`
	BaseModel     string     `json:"base_model"`
	License       string     `json:"license"`
	NSFW          bool       `json:"nsfw"`
	Files         []FileInfo `json:"files"`
}

func (c *HFClient) inspectRepo(ctx context.Context, repoID string) (Metadata, error) {
	raw, err := c.runProbe(ctx, repoID)
	if err != nil {
		return Metadata{}, err
	}
	var r rawHF
	if err := json.Unmarshal(raw, &r); err != nil {
		return Metadata{}, fmt.Errorf("hfmeta: parse probe output for %q: %w", repoID, err)
	}
	m := Metadata{
		Source:        repoID,
		RepoID:        firstNonEmpty(r.RepoID, repoID),
		Revision:      r.Revision,
		PipelineClass: r.PipelineClass,
		Tags:          r.Tags,
		BaseModel:     r.BaseModel,
		Files:         r.Files,
		License:       r.License,
		NSFW:          r.NSFW,
	}
	m.Layout = DetectLayout(m.Files)
	m.SizeBytes = m.TotalSize()
	return m, nil
}

func (c *HFClient) runProbe(ctx context.Context, repoID string) ([]byte, error) {
	if c.Runner != nil {
		return c.Runner(ctx, repoID)
	}
	python := c.Python
	if python == "" {
		python = "python3"
	}
	cmd := exec.CommandContext(ctx, python, "-c", probeScript, repoID)
	if c.Env != nil {
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env(c.Env))
	} else {
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("hfmeta: probe %q: %w", repoID, err)
	}
	return out, nil
}

// probeScript fetches repo metadata + a model_index.json peek and prints the
// rawHF JSON. Argument-free (input via argv) so there is no shell-quoting surface.
const probeScript = `import json, sys
from huggingface_hub import HfApi, hf_hub_download
repo = sys.argv[1]
api = HfApi()
info = api.model_info(repo, files_metadata=True)
card = getattr(info, "card_data", None) or {}
def cget(k):
    try:
        return card.get(k)
    except AttributeError:
        return getattr(card, k, None)
files = []
for s in (info.siblings or []):
    files.append({"path": s.rfilename, "size": int(s.size or 0)})
pipeline_class = ""
if any(f["path"] == "model_index.json" for f in files):
    try:
        p = hf_hub_download(repo, "model_index.json", revision=info.sha)
        with open(p) as fh:
            pipeline_class = json.load(fh).get("_class_name", "") or ""
    except Exception:
        pipeline_class = ""
tags = list(info.tags or [])
nsfw = ("not-for-all-audiences" in tags) or bool(cget("not-for-all-audiences"))
lic = cget("license") or ""
if isinstance(lic, list):
    lic = lic[0] if lic else ""
base = cget("base_model") or ""
if isinstance(base, list):
    base = base[0] if base else ""
print(json.dumps({
    "repo_id": repo,
    "revision": info.sha or "",
    "pipeline_class": pipeline_class,
    "tags": tags,
    "base_model": base,
    "license": str(lic),
    "nsfw": bool(nsfw),
    "files": files,
}))
`

// inspectLocal inspects a model source already on disk: a directory is listed
// (layout + size + model_index.json peek); a single file is a single-file
// checkpoint. License/NSFW are unknown for local sources.
func inspectLocal(p string) (Metadata, error) {
	info, err := os.Stat(p)
	if err != nil {
		return Metadata{}, fmt.Errorf("hfmeta: stat %q: %w", p, err)
	}
	m := Metadata{Source: p}
	if !info.IsDir() {
		m.Layout = LayoutSingleFile
		m.Files = []FileInfo{{Path: filepath.Base(p), Size: info.Size()}}
		m.SizeBytes = info.Size()
		return m, nil
	}
	walkErr := filepath.WalkDir(p, func(fp string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, statErr := d.Info()
		if statErr != nil {
			return statErr
		}
		rel, relErr := filepath.Rel(p, fp)
		if relErr != nil {
			return relErr
		}
		m.Files = append(m.Files, FileInfo{Path: filepath.ToSlash(rel), Size: fi.Size()})
		return nil
	})
	if walkErr != nil {
		return Metadata{}, fmt.Errorf("hfmeta: list %q: %w", p, walkErr)
	}
	m.Layout = DetectLayout(m.Files)
	m.PipelineClass = peekLocalPipelineClass(p, m.Files)
	m.SizeBytes = m.TotalSize()
	return m, nil
}

// peekLocalPipelineClass reads _class_name from a local model_index.json if one
// is present (diffusers-repo layout), else returns "".
func peekLocalPipelineClass(root string, files []FileInfo) string {
	for _, f := range files {
		if filepath.ToSlash(f.Path) != "model_index.json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "model_index.json")) //nolint:gosec // root is a user-supplied inspect path
		if err != nil {
			return ""
		}
		var idx struct {
			ClassName string `json:"_class_name"`
		}
		if json.Unmarshal(data, &idx) == nil {
			return idx.ClassName
		}
		return ""
	}
	return ""
}

// inspectURL records a direct weight URL as a single-file checkpoint. The file
// name drives the eventual Asset.Filename; license/NSFW are unknown.
func inspectURL(rawURL string) Metadata {
	name := filepath.Base(strings.SplitN(rawURL, "?", 2)[0])
	return Metadata{
		Source: rawURL,
		Layout: LayoutSingleFile,
		Files:  []FileInfo{{Path: name}},
	}
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func hasURLScheme(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

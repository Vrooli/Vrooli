package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"
	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters/adapters_v1connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

type inventoryArtifact struct {
	Kind               string   `json:"kind"`
	ID                 string   `json:"id"`
	Path               string   `json:"path"`
	FilesystemIdentity string   `json:"filesystem_identity,omitempty"`
	Installed          bool     `json:"installed"`
	Enabled            bool     `json:"enabled,omitempty"`
	Readiness          string   `json:"readiness,omitempty"`
	InstallIdentity    string   `json:"install_identity,omitempty"`
	InstalledAt        string   `json:"installed_at,omitempty"`
	RecordedBytes      uint64   `json:"recorded_bytes"`
	ApparentBytes      uint64   `json:"apparent_bytes"`
	AllocatedBytes     uint64   `json:"allocated_bytes"`
	AllocatedKnown     bool     `json:"allocated_known"`
	UniqueFileCount    int      `json:"unique_file_count"`
	SharedInodeRefs    int      `json:"shared_inode_refs"`
	SymlinkFiles       int      `json:"symlink_files"`
	DirectorySymlinks  int      `json:"directory_symlink_trees_excluded"`
	ExternalLinkBytes  uint64   `json:"external_symlink_target_bytes"`
	ExternalLinkPaths  []string `json:"external_symlink_targets,omitempty"`
	LastSuccessfulUse  string   `json:"last_successful_use,omitempty"`
	SuccessfulUseState string   `json:"successful_use_state"`
	SuccessfulJobs     int64    `json:"successful_jobs,omitempty"`
	LastOperation      string   `json:"last_operation,omitempty"`
	LastTier           string   `json:"last_tier,omitempty"`
	Finding            string   `json:"finding,omitempty"`
}

type inventoryTotals struct {
	ArtifactCount            int    `json:"artifact_count"`
	InstalledCount           int    `json:"installed_count"`
	RecordedBytes            uint64 `json:"recorded_bytes"`
	PerArtifactApparentBytes uint64 `json:"per_artifact_apparent_bytes"`
	UniqueInodeAllocated     uint64 `json:"unique_inode_allocated_bytes"`
	AllocatedKnown           bool   `json:"allocated_known"`
	SharedInodeReferences    int    `json:"shared_inode_references"`
}

type inventoryReport struct {
	Scope                  string              `json:"scope"`
	RecordedBytesBasis     string              `json:"recorded_bytes_basis"`
	Totals                 inventoryTotals     `json:"totals"`
	Artifacts              []inventoryArtifact `json:"artifacts"`
	DuplicateContentGroups [][]string          `json:"duplicate_content_identity_groups,omitempty"`
	Findings               []string            `json:"findings,omitempty"`
}

type inventoryModelUsage struct {
	SuccessfulJobs   int64  `json:"successful_jobs"`
	ModelID          string `json:"model_id"`
	LastSuccessfulAt string `json:"last_successful_at"`
	LastOperation    string `json:"last_operation"`
	LastTier         string `json:"last_tier"`
}

type inodeInfo struct {
	key       string
	allocated uint64
	known     bool
	info      os.FileInfo
}

func (h *handlers) inventory(ctx cliapp.RunContext) error {
	ctxRun := context.Background()
	modelsResp, err := h.client.ListModels(ctxRun, connect.NewRequest(&modelsv1.ListModelsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list models for storage inventory", err, nil)
	}
	if modelsResp == nil || modelsResp.Msg == nil {
		return fmt.Errorf("server returned no model inventory")
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	adapterClient := adaptersconnect.NewAdaptersServiceClient(httpClient, baseURL)
	adaptersResp, err := adapterClient.ListAdapters(ctxRun, connect.NewRequest(&adaptersv1.ListAdaptersRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list adapters for storage inventory", err, nil)
	}
	if adaptersResp == nil || adaptersResp.Msg == nil {
		return fmt.Errorf("server returned no adapter inventory")
	}
	usage, err := queryInventoryUsage(ctxRun, httpClient, baseURL)
	if err != nil {
		return err
	}
	usageByModel := make(map[string]inventoryModelUsage, len(usage))
	for _, record := range usage {
		usageByModel[record.ModelID] = record
	}
	doctor, err := h.client.DoctorCatalog(ctxRun, connect.NewRequest(&modelsv1.DoctorCatalogRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("read model lifecycle for storage inventory", err, nil)
	}
	if doctor == nil || doctor.Msg == nil {
		return fmt.Errorf("server returned no model lifecycle inventory")
	}
	readiness := map[string]string{}
	for _, item := range doctor.Msg.Models {
		readiness[item.ModelId] = item.State.String()
	}

	report := inventoryReport{Scope: "installed Image Tools model/adapter roots plus detected Hugging Face caches; byte totals include regular files and file-symlink targets (not directory metadata or directory-symlink trees); artifact identity uses the install receipt checksum, and shared physical files are deduplicated by device/inode where available", RecordedBytesBasis: "InstallRecord.SizeBytes captured from the acquisition-time file tree; not a live size or safe reclaim amount", Artifacts: []inventoryArtifact{}}
	seen := map[string]inodeInfo{}
	identityRefs := map[string][]string{}
	for _, model := range modelsResp.Msg.Models {
		if model == nil || model.Install == nil || !model.Install.Installed {
			continue
		}
		item := inventoryArtifact{Kind: "model", ID: model.Id, Path: model.Install.Path, Installed: true, Enabled: model.Enabled, Readiness: readiness[model.Id], InstallIdentity: model.Install.Checksum, InstalledAt: model.Install.InstalledAt, RecordedBytes: model.Install.SizeBytes}
		if item.Readiness == "" || item.Readiness == "MODEL_LIFECYCLE_STATE_UNSPECIFIED" {
			item.Readiness = "unknown"
			report.Findings = append(report.Findings, fmt.Sprintf("model %s has no lifecycle receipt", model.Id))
		}
		if use, ok := usageByModel[model.Id]; ok {
			item.LastSuccessfulUse, item.LastOperation, item.LastTier = use.LastSuccessfulAt, use.LastOperation, use.LastTier
			item.SuccessfulUseState = "recorded"
			item.SuccessfulJobs = use.SuccessfulJobs
		} else {
			item.SuccessfulUseState = "unknown"
		}
		if err := inspectInventoryPath(&item, seen); err != nil {
			item.Finding = err.Error()
		}
		appendInventoryFindings(&report, item)
		if item.RecordedBytes != item.ApparentBytes {
			report.Findings = append(report.Findings, fmt.Sprintf("model %s recorded bytes %d differ from apparent bytes %d", item.ID, item.RecordedBytes, item.ApparentBytes))
		}
		if item.InstallIdentity != "" {
			identityRefs[item.InstallIdentity] = append(identityRefs[item.InstallIdentity], "model:"+item.ID)
		}
		report.Artifacts = append(report.Artifacts, item)
	}
	for _, adapter := range adaptersResp.Msg.Adapters {
		if adapter == nil || adapter.Install == nil || !adapter.Install.Installed {
			continue
		}
		item := inventoryArtifact{Kind: "adapter", ID: adapter.Id, Path: adapter.Install.Path, Installed: true, Enabled: adapter.Enabled, Readiness: fmt.Sprintf("ready=%t", adapter.Ready), InstallIdentity: adapter.Install.Checksum, InstalledAt: adapter.Install.InstalledAt, RecordedBytes: adapter.Install.SizeBytes}
		if err := inspectInventoryPath(&item, seen); err != nil {
			item.Finding = err.Error()
		}
		appendInventoryFindings(&report, item)
		if item.RecordedBytes != item.ApparentBytes {
			report.Findings = append(report.Findings, fmt.Sprintf("adapter %s recorded bytes %d differ from apparent bytes %d", item.ID, item.RecordedBytes, item.ApparentBytes))
		}
		if item.InstallIdentity != "" {
			identityRefs[item.InstallIdentity] = append(identityRefs[item.InstallIdentity], "adapter:"+item.ID)
		}
		report.Artifacts = append(report.Artifacts, item)
	}
	identities := make([]string, 0, len(identityRefs))
	for checksum := range identityRefs {
		identities = append(identities, checksum)
	}
	sort.Strings(identities)
	for _, checksum := range identities {
		refs := identityRefs[checksum]
		if len(refs) > 1 {
			sort.Strings(refs)
			report.DuplicateContentGroups = append(report.DuplicateContentGroups, refs)
			report.Findings = append(report.Findings, fmt.Sprintf("install identity %s appears in %d artifact receipts; physical files are still counted by inode", checksum, len(refs)))
		}
	}
	for _, cachePath := range cacheRoots() {
		_, statErr := os.Lstat(cachePath)
		if os.IsNotExist(statErr) {
			continue
		}
		item := inventoryArtifact{Kind: "shared-cache", ID: "external-model-cache", Path: cachePath, Readiness: "shared-owner-attribution-unknown", Finding: "shared cache bytes are visible but not attributed solely to Image Tools"}
		if statErr != nil {
			item.Finding = statErr.Error()
		} else if err := inspectInventoryPath(&item, seen); err != nil {
			item.Finding = err.Error()
		}
		report.Artifacts = append(report.Artifacts, item)
	}
	for _, artifact := range report.Artifacts {
		report.Totals.ArtifactCount++
		if artifact.Installed {
			report.Totals.InstalledCount++
		}
		report.Totals.RecordedBytes += artifact.RecordedBytes
		report.Totals.PerArtifactApparentBytes += artifact.ApparentBytes
		report.Totals.SharedInodeReferences += artifact.SharedInodeRefs
	}
	allocatedKnown := len(report.Artifacts) > 0
	for _, artifact := range report.Artifacts {
		if !artifact.AllocatedKnown {
			allocatedKnown = false
		}
	}
	for _, file := range seen {
		if file.known {
			report.Totals.UniqueInodeAllocated += file.allocated
		} else {
			allocatedKnown = false
		}
	}
	report.Totals.AllocatedKnown = allocatedKnown
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(report)
	}
	rows := make([]string, 0, len(report.Artifacts)+1)
	rows = append(rows, fmt.Sprintf("install-receipt=%d B, per-artifact apparent=%d B, cross-root unique allocated=%d B (known=%t), shared inode references=%d", report.Totals.RecordedBytes, report.Totals.PerArtifactApparentBytes, report.Totals.UniqueInodeAllocated, report.Totals.AllocatedKnown, report.Totals.SharedInodeReferences))
	for _, item := range report.Artifacts {
		use := "use unknown"
		if item.LastSuccessfulUse != "" {
			use = fmt.Sprintf("%d successful jobs; last %s (%s)", item.SuccessfulJobs, item.LastSuccessfulUse, item.LastOperation)
		}
		rows = append(rows, fmt.Sprintf("%s %s: lifecycle=%s install-receipt=%d B apparent=%d B allocated=%d B files=%d symlink-files=%d directory-symlink-trees-excluded=%d external-target-bytes=%d identity=%s; %s", item.Kind, item.ID, item.Readiness, item.RecordedBytes, item.ApparentBytes, item.AllocatedBytes, item.UniqueFileCount, item.SymlinkFiles, item.DirectorySymlinks, item.ExternalLinkBytes, identityLabel(item.InstallIdentity), use))
	}
	rows = append(rows, report.Findings...)
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("%d installed artifact(s) inventoried; absent usage remains unknown", report.Totals.InstalledCount)}, ResultsHeading: "Storage inventory", Results: rows})
}

func appendInventoryFindings(report *inventoryReport, item inventoryArtifact) {
	if item.Finding != "" {
		report.Findings = append(report.Findings, fmt.Sprintf("%s %s: %s", item.Kind, item.ID, item.Finding))
	}
	if item.ExternalLinkBytes > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("%s %s references %d B through external file symlinks; these bytes are shared-owner-attribution unknown and deduplicated by inode", item.Kind, item.ID, item.ExternalLinkBytes))
	}
	if item.DirectorySymlinks > 0 {
		report.Findings = append(report.Findings, fmt.Sprintf("%s %s has %d directory symlink tree(s) excluded from byte totals; allocated size is incomplete", item.Kind, item.ID, item.DirectorySymlinks))
	}
}

func inspectInventoryPath(item *inventoryArtifact, seen map[string]inodeInfo) error {
	if strings.TrimSpace(item.Path) == "" {
		return fmt.Errorf("install receipt has no path")
	}
	root, err := filepath.Abs(item.Path)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", item.Path, err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat %s: %w", root, err)
	}
	item.FilesystemIdentity = filesystemIdentity(info)
	if !info.IsDir() && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file or directory", root)
	}
	local := map[string]inodeInfo{}
	visit := func(path string, fileInfo os.FileInfo, viaSymlink bool) error {
		if !fileInfo.Mode().IsRegular() {
			return nil
		}
		if viaSymlink {
			item.SymlinkFiles++
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				item.AllocatedKnown = false
				return fmt.Errorf("resolve file symlink %s: %w", path, err)
			}
			rel, err := filepath.Rel(root, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
				item.ExternalLinkBytes += uint64(max(int64(0), fileInfo.Size()))
				item.ExternalLinkPaths = append(item.ExternalLinkPaths, resolved)
			}
		}
		inode := inodeFor(path, fileInfo)
		if _, ok := local[inode.key]; !ok {
			if same := sameInodeKey(local, fileInfo); same != "" {
				inode.key = same
			}
		}
		if _, ok := seen[inode.key]; !ok {
			if same := sameInodeKey(seen, fileInfo); same != "" {
				inode.key = same
			}
		}
		if _, ok := local[inode.key]; ok {
			return nil
		}
		local[inode.key] = inode
		item.ApparentBytes += uint64(max(int64(0), fileInfo.Size()))
		item.UniqueFileCount++
		if prior, ok := seen[inode.key]; ok {
			item.SharedInodeRefs++
			if prior.known {
				item.AllocatedBytes += prior.allocated
			}
		} else {
			seen[inode.key] = inode
			if inode.known {
				item.AllocatedBytes += inode.allocated
			}
		}
		if !inode.known {
			item.AllocatedKnown = false
		}
		return nil
	}
	item.AllocatedKnown = true
	if info.IsDir() {
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				targetInfo, err := os.Stat(path)
				if err != nil {
					item.AllocatedKnown = false
					item.Finding = fmt.Sprintf("unreadable symlink target %s: %v", path, err)
					return nil
				}
				if targetInfo.IsDir() {
					item.DirectorySymlinks++
					item.AllocatedKnown = false
					return nil
				}
				if !targetInfo.Mode().IsRegular() {
					return nil
				}
				return visit(path, targetInfo, true)
			}
			fileInfo, err := entry.Info()
			if err != nil {
				return err
			}
			return visit(path, fileInfo, false)
		})
	} else {
		err = visit(root, info, false)
	}
	if err != nil {
		item.AllocatedKnown = false
		return fmt.Errorf("walk %s: %w", root, err)
	}
	sort.Strings(item.ExternalLinkPaths)
	item.ExternalLinkPaths = uniqueStrings(item.ExternalLinkPaths)
	return nil
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func filesystemIdentity(info os.FileInfo) string {
	value := reflect.ValueOf(info.Sys())
	if value.IsValid() && value.Kind() == reflect.Pointer && !value.IsNil() {
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Struct {
		dev := value.FieldByName("Dev")
		if dev.IsValid() && dev.CanUint() {
			return fmt.Sprintf("dev:%d", dev.Uint())
		}
	}
	return "unknown"
}

func inodeFor(path string, info os.FileInfo) inodeInfo {
	identity := filepath.Clean(path)
	value := reflect.ValueOf(info.Sys())
	if value.IsValid() && value.Kind() == reflect.Pointer && !value.IsNil() {
		value = value.Elem()
	}
	if value.IsValid() && value.Kind() == reflect.Struct {
		dev, ino := value.FieldByName("Dev"), value.FieldByName("Ino")
		if dev.IsValid() && ino.IsValid() && dev.CanUint() && ino.CanUint() {
			identity = fmt.Sprintf("dev:%d:ino:%d", dev.Uint(), ino.Uint())
		}
		if blocks := value.FieldByName("Blocks"); blocks.IsValid() && blocks.CanInt() && blocks.Int() >= 0 {
			return inodeInfo{key: identity, allocated: uint64(blocks.Int()) * 512, known: true, info: info}
		}
	}
	return inodeInfo{key: identity, info: info}
}

func sameInodeKey(priorFiles map[string]inodeInfo, current os.FileInfo) string {
	for key, prior := range priorFiles {
		if prior.info != nil && os.SameFile(current, prior.info) {
			return key
		}
	}
	return ""
}

func identityLabel(identity string) string {
	if identity == "" {
		return "unknown"
	}
	return identity
}

func cacheRoots() []string {
	paths := []string{os.Getenv("HF_HOME"), os.Getenv("HF_HUB_CACHE"), os.Getenv("HUGGINGFACE_HUB_CACHE"), os.Getenv("TRANSFORMERS_CACHE")}
	if strings.TrimSpace(paths[0]) == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			paths[0] = filepath.Join(home, ".cache", "huggingface")
		}
	}
	if strings.TrimSpace(paths[1]) == "" && strings.TrimSpace(paths[2]) == "" && strings.TrimSpace(paths[0]) != "" {
		paths[1] = filepath.Join(paths[0], "hub")
	}
	seen := map[string]bool{}
	var out []string
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		abs, err := filepath.Abs(path)
		if err != nil || seen[abs] {
			continue
		}
		seen[abs] = true
		out = append(out, abs)
	}
	return out
}

func queryInventoryUsage(ctx context.Context, client interface {
	Do(*http.Request) (*http.Response, error)
}, baseURL string) ([]inventoryModelUsage, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/api/v1/measures/usage"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("models: build inventory usage query: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("models: build inventory usage query: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("models: query inventory usage: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20+1))
	if err != nil {
		return nil, fmt.Errorf("models: read inventory usage response: %w", err)
	}
	if len(body) > 4<<20 {
		return nil, fmt.Errorf("models: inventory usage response exceeds 4 MiB")
	}
	var usage []inventoryModelUsage
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models: usage API returned HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, &usage); err != nil {
		return nil, fmt.Errorf("models: decode inventory usage: %w", err)
	}
	return usage, nil
}

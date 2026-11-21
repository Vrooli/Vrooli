package assistantmigration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	shared "github.com/vrooli/vrooli/scenarios/portal/assistantmigration"
)

const GroupName = "assistant-migration"

func Register(_ *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"inventory": cliapp.ExternalDelegation(runInventory),
		"export":    cliapp.ExternalDelegation(runExport),
		"reconcile": cliapp.ExternalDelegation(runReconcile),
		"review":    cliapp.ExternalDelegation(runReview),
		"capture":   cliapp.ExternalDelegation(runCapture),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("assistant migration: load manifest: %w", err)
	}
	return group, nil
}

func runInventory(ctx cliapp.RunContext) error {
	manifest, err := shared.Inventory(ctx.Flag("source"))
	if err != nil {
		return fmt.Errorf("inventory legacy Assistant: %w", err)
	}
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), manifest)
	}
	rows := make([]string, 0, len(manifest.Records))
	for _, record := range manifest.Records {
		rows = append(rows, fmt.Sprintf("%s %s %d bytes sha256=%s", record.Kind, record.RelativePath, record.Bytes, record.SHA256))
	}
	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d legacy Assistant record(s) discovered; source remains unchanged.", len(manifest.Records)), "Review and export before retiring the old runtime."},
		ResultsHeading: "Records",
		Results:        rows,
		ResultCount:    len(rows),
		ListShaped:     true,
	})
}

func runExport(ctx cliapp.RunContext) error {
	source, destination := ctx.Flag("source"), ctx.Flag("destination")
	manifest, err := shared.Export(source, destination)
	if err != nil {
		return fmt.Errorf("export legacy Assistant: %w", err)
	}
	payload := struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Records     int    `json:"records"`
		Manifest    string `json:"manifest"`
	}{source, filepath.Clean(destination), len(manifest.Records), filepath.Join(filepath.Clean(destination), "manifest.json")}
	return cliapp.RenderAction(ctx, payload, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Exported %d legacy Assistant record(s) for review.", payload.Records)},
		Changes:     []string{fmt.Sprintf("Wrote a private copy under %s.", payload.Destination), "The source corpus was not deleted or started."},
		NextCommand: []string{"Review manifest.json, then run assistant-migration reconcile before import."},
	})
}

func runReconcile(ctx cliapp.RunContext) error {
	source := filepath.Clean(ctx.Flag("source"))
	manifestPath := ctx.Flag("manifest")
	data, err := privateFile(manifestPath, 8<<20)
	if err != nil {
		return fmt.Errorf("read migration manifest: %w", err)
	}
	var manifest shared.Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return errors.New("migration manifest is invalid JSON")
	}
	if filepath.Clean(manifest.SourceRoot) != source {
		return errors.New("migration manifest belongs to a different source root")
	}
	if err := shared.Reconcile(source, manifest); err != nil {
		return fmt.Errorf("reconcile legacy Assistant: %w", err)
	}
	payload := struct {
		Source  string `json:"source"`
		Records int    `json:"records"`
		State   string `json:"state"`
	}{source, len(manifest.Records), "unchanged"}
	return cliapp.RenderAction(ctx, payload, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Reconciled %d legacy Assistant record(s); source is unchanged.", payload.Records)},
		NextCommand: []string{"Proceed only after an operator reviews the exported records."},
	})
}

func samePathOrChild(root, candidate string) bool {
	rootAbs, rootErr := filepath.Abs(root)
	candidateAbs, candidateErr := filepath.Abs(candidate)
	if rootErr != nil || candidateErr != nil {
		return true
	}
	rel, err := filepath.Rel(rootAbs, candidateAbs)
	return err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func runReview(ctx cliapp.RunContext) error {
	source := filepath.Clean(ctx.Flag("source"))
	manifestPath := ctx.Flag("manifest")
	data, err := privateFile(manifestPath, 8<<20)
	if err != nil {
		return fmt.Errorf("read migration manifest: %w", err)
	}
	var manifest shared.Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return errors.New("migration manifest is invalid JSON")
	}
	if filepath.Clean(manifest.SourceRoot) != source {
		return errors.New("migration manifest belongs to a different source root")
	}
	review, err := shared.BuildReview(source, manifest)
	if err != nil {
		return fmt.Errorf("build migration review: %w", err)
	}
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), review)
	}
	rows := make([]string, 0, len(review.Requests))
	for _, request := range review.Requests {
		rows = append(rows, fmt.Sprintf("%s owner=%s links=%d key=%s", request.LegacyID, request.Owner, len(request.EvidenceLinks), request.CaptureKey))
	}
	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d issue capture handoff(s) prepared; source remains unchanged.", len(rows)), "Review the owner and evidence links before routing."},
		ResultsHeading: "Current-owner requests",
		Results:        rows,
		ResultCount:    len(rows),
		ListShaped:     true,
	})
}

func runCapture(ctx cliapp.RunContext) error {
	source := filepath.Clean(ctx.Flag("source"))
	manifestPath := ctx.Flag("manifest")
	destination := filepath.Clean(ctx.Flag("destination"))
	data, err := privateFile(manifestPath, 8<<20)
	if err != nil {
		return fmt.Errorf("read migration manifest: %w", err)
	}
	var manifest shared.Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return errors.New("migration manifest is invalid JSON")
	}
	if filepath.Clean(manifest.SourceRoot) != source {
		return errors.New("migration manifest belongs to a different source root")
	}
	if samePathOrChild(source, destination) {
		return errors.New("capture destination must be outside the legacy source")
	}
	review, err := shared.BuildReview(source, manifest)
	if err != nil {
		return fmt.Errorf("build migration review: %w", err)
	}
	router, err := shared.NewFileRouter(destination, source)
	if err != nil {
		return err
	}
	receipts, err := shared.Route(context.Background(), review, router)
	if err != nil {
		return fmt.Errorf("route current-owner captures: %w", err)
	}
	payload := struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Requests    int    `json:"requests"`
		Receipts    int    `json:"receipts"`
	}{source, destination, len(review.Requests), len(receipts)}
	return cliapp.RenderAction(ctx, payload, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Routed %d current-owner capture(s).", len(receipts))},
		Changes:     []string{fmt.Sprintf("Wrote private handoffs under %s.", destination), "The legacy source corpus was not changed or deleted."},
		NextCommand: []string{"Review owner handoffs before retiring the legacy runtime."},
	})
}

func privateFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("manifest must be a private regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("manifest is unavailable")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || opened.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("manifest changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("manifest exceeds limit")
	}
	return data, nil
}

package reactvitest

import (
	"connectrpc.com/connect"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	apiDiscovery "github.com/vrooli/api-core/discovery"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit/audit_v1connect"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

var syntaxRules = []string{"focused-test", "malformed-expectation", "async-assertion"}

func (Analyzer) CollectSyntax(ctx context.Context, input adapters.QualityInput) ([]testquality.Result, testquality.Reason) {
	files, err := syntaxFiles(input.Root)
	if err != nil {
		return nil, testquality.ResolutionLimit
	}
	if len(files) == 0 {
		return nil, testquality.MissingInput
	}
	if !input.Executed {
		return unknownSyntax(input, files, testquality.NotExecuted), testquality.ReasonNone
	}
	ctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()
	resolver := apiDiscovery.NewResolver(apiDiscovery.ResolverConfig{})
	url, err := resolver.ResolveScenarioURLDefault(ctx, "quality-health")
	if err != nil {
		return unknownSyntax(input, files, testquality.OwnerUnavailable), testquality.ReasonNone
	}
	client := auditconnect.NewAuditServiceClient(http.DefaultClient, url)
	response, err := client.ObserveTestSyntax(ctx, connect.NewRequest(&auditv1.ObserveTestSyntaxRequest{RootPath: input.Root, Files: files}))
	if err != nil {
		return unknownSyntax(input, files, testquality.OwnerUnavailable), testquality.ReasonNone
	}
	return convertSyntax(input, files, response.Msg), testquality.ReasonNone
}

func syntaxFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if isTSTestFile(path) {
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink test source requires resolved owner discovery")
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
			if len(files) > 100 {
				return fmt.Errorf("syntax batch limit exceeded")
			}
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}
func unknownSyntax(input adapters.QualityInput, files []string, reason testquality.Reason) []testquality.Result {
	var rows []testquality.Result
	for _, file := range files {
		for _, rule := range syntaxRules {
			rows = append(rows, testquality.Result{RuleID: rule, RuleVersion: "1", Target: testquality.Target{Workspace: input.Workspace, File: file, Scope: "file"}, TestKind: input.TestKind, SupportProfile: "vitest-syntax-1.6.9", Status: testquality.Unknown, Reason: reason, EvidenceKind: testquality.Static, Enforcement: testquality.Advisory, Severity: testquality.Warning})
		}
	}
	return rows
}

func convertSyntax(input adapters.QualityInput, files []string, response *auditv1.ObserveTestSyntaxResponse) []testquality.Result {
	rows := unknownSyntax(input, files, testquality.MissingInput)
	if response == nil {
		return rows
	}
	if response.GetSchemaVersion() != "vitest-lint/v1" {
		return unknownSyntax(input, files, testquality.UnsupportedVersion)
	}
	if response.GetUnavailableReason() != "" {
		return unknownSyntax(input, files, testquality.NormalizeReason(testquality.Reason(response.GetUnavailableReason())))
	}
	byFile := map[string]*auditv1.TestSyntaxObservation{}
	digests := map[string]string{}
	for _, file := range files {
		f, err := os.Open(filepath.Join(input.Root, file))
		if err != nil {
			continue
		}
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			f.Close()
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, 1024*1024+1))
		f.Close()
		if err != nil || len(data) > 1024*1024 {
			continue
		}
		digest := sha256.Sum256(data)
		digests[file] = hex.EncodeToString(digest[:])
	}
	duplicates := map[string]bool{}
	for _, observation := range response.GetObservations() {
		file := observation.GetFile()
		if byFile[file] != nil {
			duplicates[file] = true
		}
		byFile[file] = observation
	}
	for i := range rows {
		row := &rows[i]
		observation := byFile[row.Target.File]
		if observation == nil || duplicates[row.Target.File] {
			continue
		}
		projection := syntaxProjection(observation)
		if !projection.Pass {
			row.Reason = testquality.UnsupportedVersion
			row.Limitations = []string{projection.Remediation, "Observed native configuration: " + projection.NativeValue}
			continue
		}
		digest := digests[row.Target.File]
		if digest == "" {
			continue
		}
		if digest != observation.GetSourceDigest() {
			row.Reason = testquality.StaleEvidence
			continue
		}
		row.Limitations = append([]string(nil), observation.GetLimitations()...)
		row.EvidenceRefs = []string{"quality-health:ObserveTestSyntax/" + observation.GetSourceDigest() + "/" + row.Target.File}
		if observation.GetStatus() != "observed" {
			row.Reason = testquality.NormalizeReason(testquality.Reason(observation.GetReason()))
			continue
		}
		var selected *auditv1.TestSyntaxCheck
		count := 0
		for _, check := range observation.GetChecks() {
			if check.GetRule() == row.RuleID {
				selected = check
				count++
			}
		}
		if count != 1 {
			continue
		}
		for _, d := range observation.GetDiagnostics() {
			if d.GetCanonicalRule() == row.RuleID {
				row.Diagnostics = append(row.Diagnostics, testquality.Diagnostic{RuleID: d.GetNativeRuleId(), MessageID: d.GetMessageId(), Message: d.GetMessage(), Severity: int(d.GetNativeSeverity()), Location: testquality.Location{Line: int(d.GetLine()), Column: int(d.GetColumn()), EndLine: int(d.GetEndLine()), EndColumn: int(d.GetEndColumn())}})
			}
		}
		status := testquality.NormalizeStatus(testquality.Status(selected.GetStatus()))
		reason := testquality.NormalizeReason(testquality.Reason(selected.GetReason()))
		if status == testquality.CheckedClean && len(row.Diagnostics) > 0 {
			continue
		}
		if status == testquality.Violation && len(row.Diagnostics) == 0 {
			continue
		}
		if status == testquality.NotApplicable {
			continue
		}
		if status != testquality.Unknown && reason != testquality.ReasonNone {
			continue
		}
		row.Status, row.Reason = status, reason
		if len(row.Diagnostics) > 0 {
			row.Location = row.Diagnostics[0].Location
		}
		*row = row.Normalized()
	}
	return rows
}

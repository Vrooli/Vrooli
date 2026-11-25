package certification

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	apiV1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainV1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

const dmLaneGapOrigin = "scenario-to-cloud:dm-lane"

// SwarmManagerDMLaneGapIntake creates a candidate in Swarm Manager's
// supported backlog intake and verifies it through that system's GET route.
// Its route is recorded as tags and evidence; Swarm Manager has no assignee
// field, so this is owner-labelled triage rather than machine assignment.
type SwarmManagerDMLaneGapIntake struct {
	HTTPClient *http.Client
	ResolveURL func(context.Context) (string, error)
}

type DMLaneIntakeReceipt struct {
	SchemaVersion   int    `json:"schema_version"`
	ReceiptID       string `json:"receipt_id"`
	GapID           string `json:"gap_id"`
	BacklogRef      string `json:"backlog_ref"`
	BacklogName     string `json:"backlog_name"`
	ScenarioOwner   string `json:"scenario_owner"`
	WorkstreamID    string `json:"workstream_id"`
	WorkstreamOwner string `json:"workstream_owner"`
	WorkstreamRef   string `json:"workstream_ref"`
}

func NewSwarmManagerDMLaneGapIntake(timeout time.Duration) *SwarmManagerDMLaneGapIntake {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &SwarmManagerDMLaneGapIntake{
		HTTPClient: &http.Client{Timeout: timeout},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "swarm-manager")
		},
	}
}

func (i *SwarmManagerDMLaneGapIntake) Submit(ctx context.Context, gap DMLaneGap) (DMLaneIntakeReceipt, error) {
	if i == nil || i.ResolveURL == nil {
		return DMLaneIntakeReceipt{}, fmt.Errorf("Swarm Manager owner intake discovery is unavailable")
	}
	if err := validateDMLaneGapOwnerRoute(gap); err != nil {
		return DMLaneIntakeReceipt{}, err
	}
	base, err := i.ResolveURL(ctx)
	if err != nil {
		return DMLaneIntakeReceipt{}, fmt.Errorf("resolve Swarm Manager intake: %w", err)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return DMLaneIntakeReceipt{}, fmt.Errorf("Swarm Manager intake URL is empty")
	}
	client := i.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	name := "dm-lane-gap-" + strings.TrimPrefix(gap.ID, "sha256:")
	endpoint := base + "/api/v1/backlog/idea/" + url.PathEscape(name)
	item, status, err := readDMLaneBacklogItem(ctx, client, endpoint)
	if err != nil {
		return DMLaneIntakeReceipt{}, err
	}
	if status == http.StatusNotFound {
		description, err := dmlaneGapDescription(gap)
		if err != nil {
			return DMLaneIntakeReceipt{}, err
		}
		tags := []string{"capability-work", "origin:" + dmLaneGapOrigin, "source:" + dmLaneGapOrigin,
			"dm-lane-gap:" + strings.TrimPrefix(gap.ID, "sha256:"), "owner-scenario:" + gap.Scenario,
			"owner-workstream:" + gap.WorkstreamID, "owner-workstream-ref:" + gap.WorkstreamRef,
			"profile:" + gap.ProfileID, "stage:" + gap.Stage, "status:" + string(gap.Status)}
		priority := int32(5)
		request := &apiV1.CreateBacklogItemRequest{
			Name: name, Title: fmt.Sprintf("DM lane %s gap for %s [%s]", gap.Stage, gap.Scenario, gap.ID),
			Description: &description, Kind: "idea", Priority: &priority, Tags: tags,
			AcceptanceAllow: []string{"scenarios/" + gap.WorkstreamOwner + "/**"},
		}
		body, err := protojson.Marshal(request)
		if err != nil {
			return DMLaneIntakeReceipt{}, fmt.Errorf("encode Swarm Manager gap candidate: %w", err)
		}
		post, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/backlog", bytes.NewReader(body))
		if err != nil {
			return DMLaneIntakeReceipt{}, err
		}
		post.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(post)
		if err != nil {
			return DMLaneIntakeReceipt{}, fmt.Errorf("create Swarm Manager gap candidate: %w", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// A timeout/duplicate race is resolved by exact identity readback.
			item, status, err = readDMLaneBacklogItem(ctx, client, endpoint)
			if err != nil || status != http.StatusOK {
				return DMLaneIntakeReceipt{}, fmt.Errorf("Swarm Manager create returned %d and exact readback did not resolve it", resp.StatusCode)
			}
		} else {
			item, status, err = readDMLaneBacklogItem(ctx, client, endpoint)
			if err != nil || status != http.StatusOK {
				return DMLaneIntakeReceipt{}, fmt.Errorf("Swarm Manager created gap candidate but exact readback failed")
			}
		}
	}
	if status != http.StatusOK || !matchesDMLaneGapBacklogItem(item, gap, name) {
		return DMLaneIntakeReceipt{}, fmt.Errorf("existing Swarm Manager item %s conflicts with the persisted DM lane gap identity, evidence, or owner route", name)
	}
	receipt := DMLaneIntakeReceipt{SchemaVersion: 1, GapID: gap.ID, BacklogRef: "swarm-manager:backlog:idea/" + name,
		BacklogName: name, ScenarioOwner: gap.ScenarioOwner, WorkstreamID: gap.WorkstreamID,
		WorkstreamOwner: gap.WorkstreamOwner, WorkstreamRef: gap.WorkstreamRef}
	receipt.ReceiptID, err = dmlaneIntakeReceiptID(receipt)
	return receipt, err
}

func validateDMLaneGapOwnerRoute(gap DMLaneGap) error {
	routes, err := loadDMLaneOwnerRoutes()
	if err != nil {
		return err
	}
	workstream, err := resolveDMLaneWorkstreamOwner(gap.Stage, routes)
	if err != nil || gap.SchemaVersion != 1 || !dmlaneOwnerScenarioPattern.MatchString(gap.Scenario) || strings.TrimSpace(gap.ProfileID) == "" || strings.TrimSpace(gap.Reason) == "" || (gap.Status != DMLaneBlocked && gap.Status != DMLaneFailed && gap.Status != DMLaneNotRun) || gap.ScenarioOwner != "scenario:"+gap.Scenario || gap.WorkstreamID != workstream.ID || gap.WorkstreamOwner != workstream.OwnerScenario || gap.WorkstreamRef != workstream.WorkstreamRef {
		return fmt.Errorf("DM lane gap has a missing, unknown, or ambiguous owner route")
	}
	id, err := dmlaneGapID(gap)
	if err != nil || gap.ID != id {
		return fmt.Errorf("DM lane gap has an invalid content identity")
	}
	return nil
}

func dmlaneGapDescription(gap DMLaneGap) (string, error) {
	data := struct {
		Kind            string    `json:"kind"`
		OwnerAssignment string    `json:"owner_assignment"`
		Gap             DMLaneGap `json:"gap"`
	}{Kind: "capability-work", OwnerAssignment: "Owner labels are for steward triage; Swarm Manager does not enforce assignee routing.", Gap: gap}
	raw, err := json.MarshalIndent(data, "", "  ")
	return string(raw), err
}

func readDMLaneBacklogItem(ctx context.Context, client *http.Client, endpoint string) (*domainV1.BacklogItem, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("read Swarm Manager gap candidate: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, resp.StatusCode, nil
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, resp.StatusCode, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	var response apiV1.BacklogItemResponse
	if err := protojson.Unmarshal(raw, &response); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode Swarm Manager gap readback: %w", err)
	}
	return response.GetItem(), resp.StatusCode, nil
}

func matchesDMLaneGapBacklogItem(item *domainV1.BacklogItem, gap DMLaneGap, name string) bool {
	if item == nil {
		return false
	}
	description, err := dmlaneGapDescription(gap)
	if err != nil {
		return false
	}
	title := fmt.Sprintf("DM lane %s gap for %s [%s]", gap.Stage, gap.Scenario, gap.ID)
	if item.GetName() != name || item.GetKind() != "idea" || item.GetTitle() != title || item.GetDescription() != description || !hasAllDMLaneTags(item.GetTags(), gap) {
		return false
	}
	return containsDMLaneString(item.GetAcceptanceAllow(), "scenarios/"+gap.WorkstreamOwner+"/**")
}

func hasAllDMLaneTags(tags []string, gap DMLaneGap) bool {
	for _, tag := range []string{"capability-work", "origin:" + dmLaneGapOrigin, "source:" + dmLaneGapOrigin,
		"dm-lane-gap:" + strings.TrimPrefix(gap.ID, "sha256:"), "owner-scenario:" + gap.Scenario,
		"owner-workstream:" + gap.WorkstreamID, "owner-workstream-ref:" + gap.WorkstreamRef,
		"profile:" + gap.ProfileID, "stage:" + gap.Stage, "status:" + string(gap.Status)} {
		if !containsDMLaneTag(tags, tag) {
			return false
		}
	}
	return true
}

func containsDMLaneString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func dmlaneIntakeReceiptID(receipt DMLaneIntakeReceipt) (string, error) {
	receipt.ReceiptID = ""
	raw, err := json.Marshal(receipt)
	if err != nil {
		return "", fmt.Errorf("encode DM lane intake receipt identity: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func writeDMLaneIntakeReceipt(dir string, receipt DMLaneIntakeReceipt) (string, error) {
	var err error
	receipt.ReceiptID, err = dmlaneIntakeReceiptID(receipt)
	if err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "dm-intake-"+strings.TrimPrefix(receipt.ReceiptID, "sha256:")+".json")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		old, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Equal(old, raw) {
			return path, nil
		}
		return "", fmt.Errorf("DM lane intake receipt collision")
	}
	if err != nil {
		return "", err
	}
	if _, err := f.Write(raw); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", closeErr
	}
	return path, nil
}

// ReadPersistAndRouteDMLaneVerdicts makes durable verdicts and gap records the
// source of every Swarm Manager candidate, then persists owner-intake receipts.
func ReadPersistAndRouteDMLaneVerdicts(ctx context.Context, profileReader DMLaneProfileReader, operationReader DMLaneOperationReader, evidenceDir string, intake *SwarmManagerDMLaneGapIntake) ([]DMLaneVerdict, []DMLaneIntakeReceipt, []string, error) {
	verdicts, paths, err := ReadAndPersistDMLaneVerdicts(ctx, profileReader, operationReader, evidenceDir)
	if err != nil {
		return nil, nil, nil, err
	}
	gapPaths := make([]string, 0)
	for _, path := range paths {
		if strings.Contains(path, string(filepath.Separator)+"gaps"+string(filepath.Separator)) {
			gapPaths = append(gapPaths, path)
		}
	}
	gaps, err := ReadDMLaneGaps(gapPaths)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(gaps) > 0 && intake == nil {
		return nil, nil, nil, fmt.Errorf("Swarm Manager gap intake is required for non-passed stages")
	}
	receipts := make([]DMLaneIntakeReceipt, 0, len(gaps))
	for _, gap := range gaps {
		receipt, err := intake.Submit(ctx, gap)
		if err != nil {
			return nil, nil, nil, err
		}
		receiptPath, err := writeDMLaneIntakeReceipt(filepath.Join(evidenceDir, "owner-intake"), receipt)
		if err != nil {
			return nil, nil, nil, err
		}
		receipts = append(receipts, receipt)
		paths = append(paths, receiptPath)
	}
	return verdicts, receipts, paths, nil
}

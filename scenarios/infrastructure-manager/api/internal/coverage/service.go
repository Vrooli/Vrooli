// Package coverage owns the stateless join of owner-authored spaces and the
// operator-authored setpoint. It deliberately has no database schema and no
// mutating operation: both halves are read from their sources on every call.
package coverage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/spacedoc"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/setpoint"
)

// The setpoint types are internal/setpoint's: one parser for the bar file,
// one shape served by this scenario. The aliases keep the coverage surface's
// names for its handlers and tests.
type (
	Bar          = setpoint.Bar
	Unmapped     = setpoint.Unmapped
	setpointFile = setpoint.Document
)

type SpaceReader interface {
	Read(context.Context, spacedoc.Projection) (*spacedoc.SpaceDefinition, error)
}

type FileSpaceReader struct {
	Root string
}

func (r FileSpaceReader) Read(_ context.Context, projection spacedoc.Projection) (*spacedoc.SpaceDefinition, error) {
	owner, ok := projectionOwners[projection]
	if !ok {
		return nil, fmt.Errorf("coverage: no owner configured for projection %q", projection)
	}
	path := filepath.Join(r.Root, "scenarios", owner, "docs", "spaces", string(projection)+"-space.md")
	md, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	def, err := spacedoc.Parse(projection, md)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	def.Source = relSource(r.Root, path)
	return def, nil
}

var projectionOwners = map[spacedoc.Projection]string{
	spacedoc.ProjectionSupervision:     "vrooli-autoheal",
	spacedoc.ProjectionAvailability:    "vrooli-autoheal",
	spacedoc.ProjectionRecovery:        "vrooli-autoheal",
	spacedoc.ProjectionCapacity:        "infrastructure-manager",
	spacedoc.ProjectionHeadroom:        "storage-manager",
	spacedoc.ProjectionDurability:      "data-backup-manager",
	spacedoc.ProjectionAttribution:     "system-monitor",
	spacedoc.ProjectionValidationCost:  "test-genie",
	spacedoc.ProjectionAgentThroughput: "agent-manager",
	spacedoc.ProjectionCommissioning:   "infrastructure-manager",
	spacedoc.ProjectionSubstrate:       "vrooli-autoheal",
}

func NewFileSpaceReader() (FileSpaceReader, error) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return FileSpaceReader{}, err
	}
	return FileSpaceReader{Root: root}, nil
}

// LoadSetpoint reads the bar file through internal/setpoint, the only parser
// of reliability-setpoint.json; every invariant the file must hold (unique
// cells, reasons on ungradeable bars, units and thresholds on gradeable
// ones) is enforced there.
func LoadSetpoint(root string) (setpointFile, error) {
	sp, err := setpoint.Load(filepath.Join(root, filepath.FromSlash(setpoint.RelativePath)))
	if err != nil {
		return setpointFile{}, err
	}
	return sp.Document, nil
}

// RetentionFloor derives the minimum history window from the operator-authored
// setpoint. The longest explicit day/hour sustain plus the configured margin
// determines when a trend is measurable; prose values such as "one read" or
// "pending telemetry" do not fabricate a duration.
func RetentionFloor(root string) (time.Duration, error) {
	setpoint, err := LoadSetpoint(root)
	if err != nil {
		return 0, err
	}
	maxSustain := time.Duration(0)
	for _, bar := range setpoint.Bars {
		for _, match := range sustainDurationPattern.FindAllStringSubmatch(strings.ToLower(bar.Sustain), -1) {
			value, parseErr := strconv.Atoi(match[1])
			if parseErr != nil {
				continue
			}
			candidate := time.Duration(value) * time.Hour
			if match[2] == "d" {
				candidate *= 24
			}
			if candidate > maxSustain {
				maxSustain = candidate
			}
		}
	}
	if maxSustain == 0 {
		return 0, fmt.Errorf("setpoint: no explicit sustain duration is available for retention")
	}
	if setpoint.Constants.RetentionMarginDays < 0 {
		return 0, fmt.Errorf("setpoint: retention_margin_days cannot be negative")
	}
	return maxSustain + time.Duration(setpoint.Constants.RetentionMarginDays)*24*time.Hour, nil
}

var sustainDurationPattern = regexp.MustCompile(`(\d+)\s*([dh])`)

type Service struct {
	Root   string
	Reader SpaceReader
	Now    func() time.Time
}

func NewService(root string, reader SpaceReader) *Service {
	if reader == nil {
		reader = FileSpaceReader{Root: root}
	}
	return &Service{Root: root, Reader: reader, Now: func() time.Time { return time.Now().UTC() }}
}

type ProjectionSnapshot struct {
	Definition *spacedoc.SpaceDefinition
	Bars       []Bar
}

type Snapshot struct {
	Projections map[spacedoc.Projection]ProjectionSnapshot
	Findings    []IntegrityFinding
	ComputedAt  time.Time
}

type IntegrityFinding struct {
	Code     string
	Message  string
	Location string
	Severity string
}

func (s *Service) Snapshot(ctx context.Context, requested []spacedoc.Projection) (Snapshot, error) {
	setpoint, err := LoadSetpoint(s.Root)
	if err != nil {
		return Snapshot{}, err
	}
	projections := requested
	if len(projections) == 0 {
		projections = make([]spacedoc.Projection, 0, len(projectionOwners))
		for projection := range projectionOwners {
			projections = append(projections, projection)
		}
		sort.Slice(projections, func(i, j int) bool { return projections[i] < projections[j] })
	}
	byProjection := make(map[spacedoc.Projection]ProjectionSnapshot, len(projections))
	findings := make([]IntegrityFinding, 0)
	for _, projection := range projections {
		def, readErr := s.Reader.Read(ctx, projection)
		if readErr != nil {
			findings = append(findings, IntegrityFinding{Code: "SPACE_UNAVAILABLE", Message: readErr.Error(), Location: string(projection), Severity: "warning"})
			continue
		}
		bars := barsFor(setpoint.Bars, projection)
		byProjection[projection] = ProjectionSnapshot{Definition: def, Bars: bars}
		findings = append(findings, validateProjection(def, bars)...)
	}
	return Snapshot{Projections: byProjection, Findings: findings, ComputedAt: s.Now().UTC()}, nil
}

func barsFor(bars []Bar, projection spacedoc.Projection) []Bar {
	out := make([]Bar, 0)
	for _, bar := range bars {
		if bar.Projection == string(projection) {
			out = append(out, bar)
		}
	}
	return out
}

func validateProjection(def *spacedoc.SpaceDefinition, bars []Bar) []IntegrityFinding {
	barByCell := make(map[string]Bar, len(bars))
	for _, bar := range bars {
		barByCell[bar.CellRef] = bar
	}
	findings := make([]IntegrityFinding, 0)
	for _, cell := range def.Cells {
		ref := string(def.Projection) + "/" + cell.ID
		if cell.Status == spacedoc.StatusNow {
			if _, ok := barByCell[ref]; !ok {
				findings = append(findings, IntegrityFinding{Code: "NOW_WITHOUT_BAR", Message: "NOW cell has no setpoint bar", Location: ref, Severity: "error"})
			}
		}
		if cell.Status == spacedoc.StatusMissing && strings.TrimSpace(cell.GapOpenedOn) == "" {
			findings = append(findings, IntegrityFinding{Code: "MISSING_UNDATED", Message: "MISSING cell has no gap_opened_on date", Location: ref, Severity: "error"})
		}
	}
	return findings
}

func relSource(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

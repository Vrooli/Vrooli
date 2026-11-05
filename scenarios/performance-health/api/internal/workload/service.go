package workload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
)

type repository interface {
	Insert(context.Context, *sweepv1.WorkloadReading) error
	Complete(context.Context, *sweepv1.WorkloadReading) error
	Latest(context.Context, string, string) (*sweepv1.WorkloadReading, error)
}

type Service struct {
	repoRoot, outputRoot string
	store                repository
	resolve              func(context.Context, string, string) (string, error)
	runCommand           func(context.Context, string, []string, string) error
	environment          func(context.Context) (*commonv1.CaptureEnvironment, error)
	// One workload capture at a time per owner. A busy request fails explicitly;
	// this is synchronous bounded work, not a new queued job system.
	running sync.Mutex
}

func NewService(repoRoot, outputRoot string, store repository) *Service {
	r := discovery.NewResolver(discovery.ResolverConfig{})
	return &Service{repoRoot: repoRoot, outputRoot: outputRoot, store: store,
		resolve: r.ResolveScenarioURL, runCommand: runCommand, environment: vroolicli.New().HostCaptureEnvironment}
}

func (s *Service) Run(ctx context.Context, scenario, name string) (*sweepv1.WorkloadReading, error) {
	d, err := s.declaration(scenario, name)
	if err != nil {
		return nil, err
	}
	if !s.running.TryLock() {
		return nil, errors.New("a performance workload is already running")
	}
	defer s.running.Unlock()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.TimeoutSeconds)*time.Second)
	defer cancel()
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	started := time.Now().UTC()
	r := &sweepv1.WorkloadReading{Scenario: scenario, Workload: name, OperationId: hex.EncodeToString(id),
		Outcome: sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_UNAVAILABLE, Reason: "workload admitted; terminal receipt pending", CapturedAt: started.Format(time.RFC3339Nano),
		ProducerDigest: d.producerDigest, ContractDigest: d.contractDigest, ConfigurationDigest: d.configDigest, BudgetMs: d.P95MaxMS}
	root := filepath.Join(s.outputRoot, r.OperationId)
	if err := s.store.Insert(ctx, r); err != nil {
		return nil, fmt.Errorf("admit workload %s: %w", r.OperationId, err)
	}
	output := filepath.Join(root, "producer")
	r.ReceiptPath = filepath.Join(output, "receipt.json")
	err = os.MkdirAll(root, 0700)
	if err == nil {
		err = s.measure(ctx, d, scenario, output, root, started, r)
	}
	if err != nil {
		unavailable(r, sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_FAILED, err.Error())
	}
	persistCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	if err := s.store.Complete(persistCtx, r); err != nil {
		return nil, fmt.Errorf("retain workload owner receipt %s: %w", r.OperationId, err)
	}
	return r, nil
}

func (s *Service) measure(ctx context.Context, d declaration, scenario, output, root string, started time.Time, r *sweepv1.WorkloadReading) error {
	endpoints, expected, err := s.currentCandidate(ctx, scenario, d)
	if err != nil {
		return err
	}
	environment, err := s.environment(ctx)
	if err != nil {
		return fmt.Errorf("workload host inventory unavailable: %w", err)
	}
	if environment == nil || environment.GetNumCpu() < d.MinCPUCores || environment.GetTotalMemBytes() < d.MinMemoryBytes {
		return errors.New("workload host does not meet declared reference cohort")
	}
	r.BuildIdentity = expected.BuildIdentity
	endpoints["output"] = output
	args := append([]string(nil), d.Command...)
	for i, arg := range args {
		if strings.HasPrefix(arg, "{") && strings.HasSuffix(arg, "}") {
			value, ok := endpoints[strings.Trim(arg, "{}")]
			if !ok {
				return fmt.Errorf("undeclared workload argument %s", arg)
			}
			args[i] = value
		}
	}
	invocation, _ := json.MarshalIndent(map[string]any{"command": args, "directory": d.Directory, "definition": d.Definition, "started_at": started, "environment": environment}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "invocation.json"), invocation, 0600); err != nil {
		return err
	}
	dir, err := confinedPath(d.root, d.Directory)
	if err != nil {
		return err
	}
	commandErr := s.runCommand(ctx, dir, args, root)
	finished := time.Now().UTC()
	raw, err := readBounded(r.ReceiptPath, 16<<20)
	if err != nil {
		return errors.Join(commandErr, fmt.Errorf("producer receipt unavailable: %w", err))
	}
	r.ReceiptSha256 = digest(raw)
	if err := errors.Join(commandErr, evaluate(raw, d, expected, started, finished, r)); err != nil {
		return err
	}
	r.Reason = ""
	current, err := s.declaration(scenario, d.name)
	if err != nil || current.configDigest != d.configDigest || current.producerDigest != d.producerDigest || current.contractDigest != d.contractDigest {
		return errors.New("workload declaration changed during invocation")
	}
	_, after, err := s.currentCandidate(ctx, scenario, d)
	if err != nil || after != expected {
		return errors.New("deployed candidate changed during invocation")
	}
	if err := os.Mkdir(filepath.Join(output, "artifacts"), 0700); err != nil {
		return err
	}
	return retainArtifacts(raw, output)
}

// Get only reads the newest attempt. Applicability failure cannot uncover an
// older success. It rechecks retained bytes, not mutable producer-owned files.
func (s *Service) Get(ctx context.Context, scenario, name string) (*sweepv1.WorkloadReading, error) {
	d, err := s.declaration(scenario, name)
	if err != nil {
		return nil, err
	}
	r, err := s.store.Latest(ctx, scenario, name)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return &sweepv1.WorkloadReading{Scenario: scenario, Workload: name, Outcome: sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_UNAVAILABLE, Reason: "no owner workload receipt"}, nil
	}
	if err := s.applicable(ctx, d, scenario, r); err != nil {
		unavailable(r, sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_UNAVAILABLE, err.Error())
	}
	return r, nil
}

func (s *Service) applicable(ctx context.Context, d declaration, scenario string, r *sweepv1.WorkloadReading) error {
	if r.ConfigurationDigest != d.configDigest || r.ProducerDigest != d.producerDigest || r.ContractDigest != d.contractDigest {
		return errors.New("workload configuration, producer or contract changed")
	}
	started, err := time.Parse(time.RFC3339Nano, r.CapturedAt)
	now := time.Now().UTC()
	if err != nil || started.After(now) || now.Sub(started) > time.Duration(d.MaxAgeSeconds)*time.Second {
		return errors.New("workload receipt expired or has an invalid timestamp")
	}
	if r.Outcome == sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_UNAVAILABLE {
		return nil // admitted, but no terminal measurement was durably committed
	}
	_, expected, err := s.currentCandidate(ctx, scenario, d)
	if err != nil {
		return err
	}
	if expected.BuildIdentity != r.BuildIdentity {
		return errors.New("workload receipt describes a different deployed build")
	}
	if r.Outcome != sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED {
		return nil
	}
	raw, err := readBounded(r.ReceiptPath, 16<<20)
	if err != nil || digest(raw) != r.ReceiptSha256 {
		return errors.New("owner producer receipt missing or changed")
	}
	if err := evaluate(raw, d, expected, started, now, r); err != nil {
		return err
	}
	return verifyRetained(raw, filepath.Dir(r.ReceiptPath))
}

func (s *Service) ReadAll(ctx context.Context, scenario string) ([]*sweepv1.WorkloadReading, error) {
	declared, err := declarations(s.repoRoot, scenario)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)
	var readings []*sweepv1.WorkloadReading
	for _, name := range names {
		r, err := s.Get(ctx, scenario, name)
		if err != nil {
			return nil, err
		}
		readings = append(readings, r)
	}
	return readings, nil
}

func (s *Service) declaration(scenario, name string) (declaration, error) {
	if s == nil || s.store == nil {
		return declaration{}, errors.New("workload owner is unavailable")
	}
	if !slug.MatchString(name) {
		return declaration{}, errors.New("invalid workload name")
	}
	all, err := declarations(s.repoRoot, scenario)
	if err != nil {
		return declaration{}, err
	}
	d, ok := all[name]
	if !ok {
		return declaration{}, errors.New("workload is not declared by the scenario")
	}
	return d, nil
}

func (s *Service) currentCandidate(ctx context.Context, scenario string, d declaration) (map[string]string, candidate, error) {
	endpoints := make(map[string]string, len(d.Endpoints))
	for name, port := range d.Endpoints {
		url, err := s.resolve(ctx, scenario, port)
		if err != nil {
			return nil, candidate{}, err
		}
		endpoints[name] = url
	}
	var c candidate
	if err := health(ctx, endpoints["api_url"], &c); err != nil {
		return nil, c, err
	}
	digestValue := strings.TrimPrefix(c.BuildIdentity, "sha256:")
	if b, err := hex.DecodeString(digestValue); err != nil || len(b) != 32 || !strings.HasPrefix(c.BuildIdentity, "sha256:") {
		return nil, c, errors.New("target health lacks a managed build identity")
	}
	if url := endpoints["driver_url"]; url != "" {
		var driver struct {
			Ready   bool   `json:"ready"`
			Version string `json:"version"`
			Browser struct {
				Version string `json:"version"`
			} `json:"browser"`
		}
		if err := health(ctx, url, &driver); err != nil {
			return nil, c, err
		}
		if !driver.Ready || driver.Version == "" || driver.Browser.Version == "" {
			return nil, c, errors.New("workload driver is not ready or versioned")
		}
		c.DriverVersion, c.BrowserVersion = driver.Version, driver.Browser.Version
	}
	return endpoints, c, nil
}

func health(ctx context.Context, base string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/health", nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("workload target health HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
}

func unavailable(r *sweepv1.WorkloadReading, outcome sweepv1.WorkloadOutcome, reason string) {
	r.Outcome, r.Reason, r.WithinBudget = outcome, reason, false
	r.P95Ms, r.WallP95Ms = 0, 0
}

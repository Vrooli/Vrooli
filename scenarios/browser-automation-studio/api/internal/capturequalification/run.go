package capturequalification

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const SampleCount = 100

type Config struct {
	Output, ScenarioRoot, APIURL, DriverURL string
}

type candidate struct {
	BuildIdentity  string `json:"build_identity"`
	DriverVersion  string `json:"driver_version"`
	BrowserVersion string `json:"browser_version"`
	DriverSessions int    `json:"driver_sessions"`
}

type attempt struct {
	Index        int           `json:"index"`
	Warmup       bool          `json:"warmup"`
	URL          string        `json:"url"`
	Status       string        `json:"status"`
	Error        string        `json:"error,omitempty"`
	OperationID  string        `json:"operation_id,omitempty"`
	OwnerMS      float64       `json:"owner_duration_ms,omitempty"`
	WallMS       float64       `json:"wall_ms"`
	Observations []observation `json:"observations"`
	Artifacts    []artifact    `json:"artifacts"`
	ResponseSHA  string        `json:"response_sha256"`
	ResponseFile string        `json:"response_file"`
	StderrFile   string        `json:"stderr_file"`
}

// Receipt contains every declared attempt and raw observation. A consumer must
// validate applicability and completeness; this producer does not certify a row.
type Receipt struct {
	SchemaVersion   int            `json:"schema_version"`
	OperationID     string         `json:"operation_id"`
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      time.Time      `json:"finished_at"`
	FixtureRevision string         `json:"fixture_revision"`
	ProducerDigest  string         `json:"producer_digest"`
	ContractDigest  string         `json:"contract_digest"`
	SampleCount     int            `json:"sample_count"`
	DeclaredWarmups int            `json:"declared_warmups"`
	Before          candidate      `json:"before"`
	After           candidate      `json:"after"`
	Environment     map[string]any `json:"environment"`
	Attempts        []attempt      `json:"attempts"`
	Errors          []string       `json:"errors"`
}

type captureFunc func(context.Context, string, string) ([]byte, []byte, error)
type identityFunc func(context.Context) (candidate, error)

// Run never restarts the target, retries a capture, or removes saved artifacts.
// Output must be a new directory. An incomplete run retains a receipt and fails.
func Run(ctx context.Context, cfg Config) (*Receipt, error) {
	return run(ctx, cfg, func(ctx context.Context, url, label string) ([]byte, []byte, error) {
		cmd := exec.CommandContext(ctx, "browser-automation-studio", "--api-base", cfg.APIURL,
			"capture", "--url", url, "--capture", "screenshot,dom-tree", "--inline-dom-tree",
			"--width", "1280", "--height", "720", "--device-scale-factor", "1",
			"--wait-for", "#ready", "--label", label, "--json")
		var stdout, stderr boundedBuffer
		stdout.remaining, stderr.remaining = 16<<20, 64<<10
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		cmd.WaitDelay = time.Second
		err := cmd.Run()
		return stdout.Bytes(), stderr.Bytes(), err
	}, func(ctx context.Context) (candidate, error) { return readIdentity(ctx, cfg) })
}

func run(ctx context.Context, cfg Config, capture captureFunc, identity identityFunc) (receipt *Receipt, resultErr error) {
	if cfg.Output == "" || cfg.ScenarioRoot == "" || cfg.APIURL == "" || cfg.DriverURL == "" {
		return nil, errors.New("output, scenario root, API URL and driver URL are required")
	}
	if err := os.Mkdir(cfg.Output, 0700); err != nil {
		return nil, fmt.Errorf("create new output directory: %w", err)
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, err
	}
	nonce := hex.EncodeToString(nonceBytes)
	receipt = &Receipt{SchemaVersion: 1, OperationID: nonce, StartedAt: time.Now().UTC(),
		FixtureRevision: digest([]byte(fixtureHTML)), SampleCount: SampleCount, DeclaredWarmups: 1,
		Environment: map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "go": runtime.Version(),
			"logical_cpus": runtime.NumCPU(), "concurrency": 1, "network": "loopback", "viewport": []int{1280, 720},
			"dpr": 1, "retries": 0, "cache_state": "warm process, fresh context/navigation per capture; no-store fixture",
			"limitations": []string{"shared host; no native platform matrix or load isolation", "not a release qualification receipt"}}}
	defer func() {
		receipt.FinishedAt = time.Now().UTC()
		if resultErr != nil {
			receipt.Errors = append(receipt.Errors, resultErr.Error())
		}
		resultErr = errors.Join(resultErr, writeJSON(filepath.Join(cfg.Output, "receipt.json"), receipt))
	}()
	producer, contract, err := sourceIdentities(cfg.ScenarioRoot)
	if err != nil {
		return receipt, err
	}
	receipt.ProducerDigest, receipt.ContractDigest = producer, contract
	receipt.Before, err = identity(ctx)
	if err != nil {
		return receipt, err
	}
	f := newFixture(nonce)
	defer f.Close()
	seenOperations := make(map[string]bool)
	for index := -1; index < SampleCount; index++ {
		trial := fmt.Sprintf("%03d", index)
		if index == -1 {
			trial = "warmup"
		}
		a := captureAttempt(ctx, cfg.Output, f, capture, index, trial)
		if a.OperationID != "" && seenOperations[a.OperationID] {
			a.Status, a.Error = "failed", "capture returned a previously used operation ID"
		}
		seenOperations[a.OperationID] = true
		receipt.Attempts = append(receipt.Attempts, a)
		if err := writeJSON(filepath.Join(cfg.Output, trial+"-receipt.json"), a); err != nil {
			return receipt, err
		}
	}
	receipt.After, err = identity(ctx)
	if err != nil {
		return receipt, err
	}
	if receipt.Before.BuildIdentity != receipt.After.BuildIdentity || receipt.Before.BrowserVersion != receipt.After.BrowserVersion || receipt.Before.DriverVersion != receipt.After.DriverVersion {
		return receipt, errors.New("candidate changed during capture cohort")
	}
	afterProducer, afterContract, err := sourceIdentities(cfg.ScenarioRoot)
	if err != nil || producer != afterProducer || contract != afterContract {
		return receipt, errors.Join(errors.New("producer or contract changed during cohort"), err)
	}
	for _, a := range receipt.Attempts {
		if a.Status != "verified" {
			return receipt, errors.New("capture cohort has failed or unattempted samples; see retained attempts")
		}
	}
	return receipt, nil
}

func captureAttempt(ctx context.Context, output string, f *fixture, capture captureFunc, index int, trial string) attempt {
	a := attempt{Index: index, Warmup: index < 0, URL: f.URL + "/fixture?trial=" + trial, Status: "failed",
		ResponseFile: trial + "-response.json", StderrFile: trial + "-stderr.txt"}
	if err := ctx.Err(); err != nil {
		a.Status, a.Error = "not_attempted", err.Error()
		return a
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	start := time.Now()
	stdout, stderr, commandErr := capture(ctx, a.URL, "capture-cohort-"+f.nonce+"-"+trial)
	a.WallMS = float64(time.Since(start)) / float64(time.Millisecond)
	a.ResponseSHA = digest(stdout)
	err := errors.Join(os.WriteFile(filepath.Join(output, a.ResponseFile), stdout, 0600),
		os.WriteFile(filepath.Join(output, a.StderrFile), stderr, 0600))
	var response captureResponse
	parseErr := json.Unmarshal(stdout, &response)
	a.OperationID, a.OwnerMS = response.ExecutionID, response.DurationMS
	a.Observations = f.observations(a.URL)
	if err = errors.Join(err, commandErr, parseErr); err == nil {
		a.Artifacts, err = verifyCapture(response, a.Observations, f.nonce, trial, a.URL)
	}
	if err != nil {
		a.Error = err.Error()
	} else {
		a.Status = "verified"
	}
	return a
}

func sourceIdentities(root string) (string, string, error) {
	paths, err := filepath.Glob(filepath.Join(root, "api/internal/capturequalification/*.go"))
	if err != nil || len(paths) == 0 {
		return "", "", errors.New("maintained producer source is missing")
	}
	paths = append(paths, filepath.Join(root, "api/cmd/capture-cohort/main.go"))
	sort.Strings(paths)
	var source bytes.Buffer
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		rel, _ := filepath.Rel(root, path)
		fmt.Fprintf(&source, "%s\x00%s\x00", filepath.ToSlash(rel), b)
	}
	contract, err := os.ReadFile(filepath.Join(root, "docs/internal/REFRACTOR_CONTRACT.json"))
	return digest(source.Bytes()), digest(contract), err
}

func readIdentity(ctx context.Context, cfg Config) (candidate, error) {
	var api struct {
		BuildIdentity string `json:"build_identity"`
	}
	var driver struct {
		Ready    bool   `json:"ready"`
		Version  string `json:"version"`
		Sessions int    `json:"sessions"`
		Browser  struct {
			Version string `json:"version"`
		} `json:"browser"`
	}
	if err := readHealth(ctx, cfg.APIURL, &api); err != nil {
		return candidate{}, err
	}
	if err := readHealth(ctx, cfg.DriverURL, &driver); err != nil {
		return candidate{}, err
	}
	if !strings.HasPrefix(api.BuildIdentity, "sha256:") || len(api.BuildIdentity) != 71 || !driver.Ready || driver.Version == "" || driver.Browser.Version == "" {
		return candidate{}, errors.New("health lacks managed build identity or ready versioned browser")
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(api.BuildIdentity, "sha256:")); err != nil {
		return candidate{}, fmt.Errorf("invalid managed build digest: %w", err)
	}
	return candidate{api.BuildIdentity, driver.Version, driver.Browser.Version, driver.Sessions}, nil
}

func readHealth(ctx context.Context, base string, into any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
		return fmt.Errorf("health returned HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(into)
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	remaining int
}

func (b *boundedBuffer) Bytes() []byte { return b.buffer.Bytes() }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		n, _ := b.buffer.Write(p[:b.remaining])
		b.remaining = 0
		return n, errors.New("capture command exceeded output limit")
	}
	b.remaining -= len(p)
	return b.buffer.Write(p)
}

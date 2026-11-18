// Package setupprivilege owns the narrowly typed privileged executor used by
// the main setup flow. Requests never contain an executable path or argv.
package setupprivilege

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/internal/apparmorprofiles"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	hosthardening "github.com/vrooli/vrooli/internal/safeguards/host-hardening"
	"github.com/vrooli/vrooli/internal/safeguards/observabilityinstall"
	onboardingapplyprivileges "github.com/vrooli/vrooli/internal/safeguards/onboarding-apply-privileges"
	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	protocolVersion = 1
	maxOutputBytes  = 512 << 10
	maxRequestBytes = 1 << 20
)

var packageToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+_.:@-]*$`)
var aptCodename = regexp.MustCompile(`^[a-z][a-z0-9.-]*$`)

type request struct {
	Version             int                             `json:"version"`
	Kind                string                          `json:"kind"`
	Manager             string                          `json:"manager,omitempty"`
	Package             string                          `json:"package,omitempty"`
	Repository          string                          `json:"repository,omitempty"`
	KeyData             []byte                          `json:"key_data,omitempty"`
	SourceLine          string                          `json:"source_line,omitempty"`
	RepoRoot            string                          `json:"repo_root,omitempty"`
	Tools               []string                        `json:"tools,omitempty"`
	Safeguards          []string                        `json:"safeguards,omitempty"`
	ProfileID           string                          `json:"profile_id,omitempty"`
	HostHardening       *hostreqkit.HostHardeningPolicy `json:"host_hardening,omitempty"`
	ObservabilityTarget observabilityinstall.Target     `json:"observability_target,omitempty"`
	TargetUser          string                          `json:"target_user,omitempty"`
	Retention           int                             `json:"retain_vmcores,omitempty"`
}

type response struct {
	OK              bool                         `json:"ok"`
	Error           string                       `json:"error,omitempty"`
	Stdout          string                       `json:"stdout,omitempty"`
	Stderr          string                       `json:"stderr,omitempty"`
	OutputTruncated bool                         `json:"output_truncated,omitempty"`
	BrokerStatus    *privilegebroker.SetupStatus `json:"broker_status,omitempty"`
}

// Session starts sudo only when its first supported host change is requested.
// Closing the setup session closes the protocol stream and waits for its child.
type Session struct {
	ctx    context.Context
	bin    string
	stdout io.Writer
	stderr io.Writer

	mu             sync.Mutex
	cmd            *exec.Cmd
	input          io.WriteCloser
	output         *bufio.Reader
	started        bool
	closed         bool
	broken         error
	commandFactory func(context.Context, string) *exec.Cmd
}

func NewSession(ctx context.Context, executable string, stdout, stderr io.Writer) *Session {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Session{ctx: ctx, bin: executable, stdout: stdout, stderr: stderr}
}

func (s *Session) InstallPackage(manager, packageName string) error {
	if _, _, err := packageCommand(manager, packageName); err != nil {
		return err
	}
	_, err := s.call(s.ctx, request{Version: protocolVersion, Kind: "install-package", Manager: manager, Package: packageName}, "privileged package install failed")
	return err
}

func (s *Session) InstallAptRepository(repository string, keyData []byte, sourceLine string) error {
	if err := validateAptRepository(repository, keyData, sourceLine); err != nil {
		return err
	}
	_, err := s.call(s.ctx, request{
		Version: protocolVersion, Kind: "install-apt-repository", Repository: repository,
		KeyData: append([]byte(nil), keyData...), SourceLine: sourceLine,
	}, "install APT repository")
	return err
}

func (s *Session) InstallPrivilegeBroker(ctx context.Context, repoRoot string) (privilegebroker.SetupStatus, error) {
	result, err := s.call(ctx, request{Version: protocolVersion, Kind: "install-privilege-broker", RepoRoot: repoRoot}, "privileged broker installation failed")
	if err != nil {
		return privilegebroker.SetupStatus{}, err
	}
	if result.BrokerStatus == nil || !result.BrokerStatus.Available {
		return privilegebroker.SetupStatus{}, errors.New("privileged broker installation returned no available status")
	}
	return *result.BrokerStatus, nil
}

func (s *Session) InstallOnboardingApplyGrant(tools, safeguards []string) error {
	_, err := s.call(s.ctx, request{
		Version: protocolVersion, Kind: "install-onboarding-apply-grant",
		Tools: append([]string(nil), tools...), Safeguards: append([]string(nil), safeguards...),
	}, "install onboarding apply grant")
	return err
}

func (s *Session) InstallAppArmorProfile(profileID string) error {
	if _, ok := apparmorprofiles.Lookup(profileID); !ok {
		return fmt.Errorf("AppArmor profile %q is not allowed in setup privilege session", profileID)
	}
	_, err := s.call(s.ctx, request{Version: protocolVersion, Kind: "install-apparmor-profile", ProfileID: profileID}, "install AppArmor profile")
	return err
}

func (s *Session) ApplyHostHardening(policy hostreqkit.HostHardeningPolicy) error {
	if err := hosthardening.ValidatePolicy(policy); err != nil {
		return err
	}
	_, err := s.call(s.ctx, request{Version: protocolVersion, Kind: "apply-host-hardening", HostHardening: &policy}, "apply host hardening")
	return err
}

func (s *Session) InstallObservability(target observabilityinstall.Target, retention int) error {
	if err := observabilityinstall.ValidateSetupRequest(target, retention); err != nil {
		return err
	}
	user := hostreqkit.InvokingUser()
	if user == "" {
		return errors.New("cannot resolve the invoking setup user for observability install")
	}
	_, err := s.call(s.ctx, request{
		Version: protocolVersion, Kind: "install-observability",
		ObservabilityTarget: target, TargetUser: user, Retention: retention,
	}, "install observability service")
	return err
}

func (s *Session) call(ctx context.Context, req request, action string) (response, error) {
	if runtime.GOOS != "linux" {
		return response{}, fmt.Errorf("setup privilege session is unsupported on %s", runtime.GOOS)
	}
	if ctx == nil {
		ctx = s.ctx
	}
	if err := ctx.Err(); err != nil {
		return response{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return response{}, err
	}
	if s.closed {
		return response{}, errors.New("setup privilege session is closed")
	}
	if s.broken != nil {
		return response{}, fmt.Errorf("setup privilege session is unavailable: %w", s.broken)
	}
	if err := s.startLocked(); err != nil {
		s.broken = err
		return response{}, err
	}
	if err := ctx.Err(); err != nil {
		return response{}, s.abortLocked(err)
	}
	stopSignal := context.AfterFunc(ctx, func() {
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Signal(os.Interrupt)
		}
	})
	// This check is the dispatch boundary. Once the request is sent, cancellation
	// interrupts the worker but cannot roll back host changes already applied.
	if err := ctx.Err(); err != nil {
		stopSignal()
		return response{}, s.abortLocked(err)
	}
	result, err := s.exchangeLocked(req)
	stopSignal()
	if ctx.Err() != nil {
		return response{}, s.abortLocked(ctx.Err())
	}
	if err != nil {
		return response{}, err
	}
	if !result.OK {
		return response{}, fmt.Errorf("%s: %s", action, result.Error)
	}
	return result, nil
}

func (s *Session) abortLocked(cause error) error {
	s.broken, s.closed = cause, true
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(os.Interrupt)
	}
	if s.input != nil {
		_ = s.input.Close()
	}
	if s.cmd != nil {
		_ = s.cmd.Wait()
	}
	return cause
}

func (s *Session) exchangeLocked(req request) (response, error) {
	if err := json.NewEncoder(s.input).Encode(req); err != nil {
		s.broken = fmt.Errorf("send setup privilege request: %w", err)
		return response{}, s.broken
	}
	var result response
	if err := json.NewDecoder(s.output).Decode(&result); err != nil {
		s.broken = fmt.Errorf("read setup privilege response: %w", err)
		return response{}, s.broken
	}
	if result.Stdout != "" && s.stdout != nil {
		_, _ = io.WriteString(s.stdout, result.Stdout)
	}
	if result.Stderr != "" && s.stderr != nil {
		_, _ = io.WriteString(s.stderr, result.Stderr)
	}
	if result.OutputTruncated && s.stderr != nil {
		_, _ = io.WriteString(s.stderr, "\n[setup privilege output truncated at 512 KiB]\n")
	}
	return result, nil
}

func (s *Session) startLocked() error {
	if s.started {
		return nil
	}
	if strings.TrimSpace(s.bin) == "" {
		return errors.New("setup privilege session requires the Vrooli executable path")
	}
	var cmd *exec.Cmd
	if s.commandFactory != nil {
		cmd = s.commandFactory(s.ctx, s.bin)
	} else {
		cmd = exec.CommandContext(s.ctx, "sudo", "--", s.bin, "__setup-privilege-session")
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open setup privilege input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("open setup privilege output: %w", err)
	}
	cmd.Stderr = s.stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("start setup privilege session: %w", err)
	}
	s.cmd, s.input, s.output, s.started = cmd, stdin, bufio.NewReader(stdout), true
	return nil
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if !s.started {
		return nil
	}
	_ = s.input.Close()
	if err := s.cmd.Wait(); err != nil {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}
		return fmt.Errorf("close setup privilege session: %w", err)
	}
	return nil
}

// RunWorker is the hidden root-side protocol endpoint. It accepts only fixed
// operation schemas and allowlists; it is not a command proxy. Operation
// callbacks are injectable so protocol behavior can be tested without
// elevation or host mutation.
func RunWorker(input io.Reader, output io.Writer, runPackage func(context.Context, string, string, io.Writer, io.Writer) error, runRepository func(context.Context, string, []byte, string) error, runBroker func(context.Context, string) (privilegebroker.SetupStatus, error), runOnboardingGrant func(context.Context, []string, []string) error, runAppArmor func(context.Context, string) error, runHostHardening func(context.Context, hostreqkit.HostHardeningPolicy) error, runObservability func(context.Context, observabilityinstall.Target, string, int) error) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("setup privilege session is unsupported on %s", runtime.GOOS)
	}
	if os.Geteuid() != 0 {
		return errors.New("setup privilege session must run with an elevated token")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return runWorkerContext(ctx, input, output, runPackage, runRepository, runBroker, runOnboardingGrant, runAppArmor, runHostHardening, runObservability)
}

func runWorker(input io.Reader, output io.Writer, runPackage func(context.Context, string, string, io.Writer, io.Writer) error, runRepository func(context.Context, string, []byte, string) error, runBroker func(context.Context, string) (privilegebroker.SetupStatus, error), runOnboardingGrant func(context.Context, []string, []string) error, runAppArmor func(context.Context, string) error, runHostHardening func(context.Context, hostreqkit.HostHardeningPolicy) error, runObservability func(context.Context, observabilityinstall.Target, string, int) error) error {
	return runWorkerContext(context.Background(), input, output, runPackage, runRepository, runBroker, runOnboardingGrant, runAppArmor, runHostHardening, runObservability)
}

func runWorkerContext(ctx context.Context, input io.Reader, output io.Writer, runPackage func(context.Context, string, string, io.Writer, io.Writer) error, runRepository func(context.Context, string, []byte, string) error, runBroker func(context.Context, string) (privilegebroker.SetupStatus, error), runOnboardingGrant func(context.Context, []string, []string) error, runAppArmor func(context.Context, string) error, runHostHardening func(context.Context, hostreqkit.HostHardeningPolicy) error, runObservability func(context.Context, observabilityinstall.Target, string, int) error) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maxRequestBytes)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		var req request
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			return fmt.Errorf("decode setup privilege request: %w", err)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			return errors.New("setup privilege request must contain exactly one JSON object per line")
		}
		res := response{}
		if req.Version != protocolVersion {
			res.Error = "unsupported setup privilege request"
		} else if err := ctx.Err(); err != nil {
			res.Error = err.Error()
		} else if req.Kind == "install-package" {
			if _, _, err := packageCommand(req.Manager, req.Package); err != nil {
				res.Error = err.Error()
			} else {
				out, errOut := &boundedBuffer{}, &boundedBuffer{}
				err := runPackage(ctx, req.Manager, req.Package, out, errOut)
				res.Stdout, res.Stderr = out.String(), errOut.String()
				res.OutputTruncated = out.truncated || errOut.truncated
				if err != nil {
					res.Error = err.Error()
				} else {
					res.OK = true
				}
			}
		} else if req.Kind == "install-apt-repository" && runRepository != nil {
			if err := validateAptRepository(req.Repository, req.KeyData, req.SourceLine); err != nil {
				res.Error = err.Error()
			} else if err := runRepository(ctx, req.Repository, req.KeyData, req.SourceLine); err != nil {
				res.Error = err.Error()
			} else {
				res.OK = true
			}
		} else if req.Kind == "install-privilege-broker" && strings.TrimSpace(req.RepoRoot) != "" && runBroker != nil {
			status, err := runBroker(ctx, req.RepoRoot)
			if err != nil {
				res.Error = err.Error()
			} else {
				res.OK, res.BrokerStatus = status.Available, &status
				if !status.Available {
					res.Error = strings.TrimSpace(status.Reason)
					if res.Error == "" {
						res.Error = "broker installation did not verify availability"
					}
				}
			}
		} else if req.Kind == "install-onboarding-apply-grant" && runOnboardingGrant != nil {
			if err := runOnboardingGrant(ctx, req.Tools, req.Safeguards); err != nil {
				res.Error = err.Error()
			} else {
				res.OK = true
			}
		} else if req.Kind == "install-apparmor-profile" && runAppArmor != nil {
			if !appArmorProfileRequestIsClosed(scanner.Bytes()) {
				res.Error = "AppArmor profile request accepts only profile_id"
			} else if _, ok := apparmorprofiles.Lookup(req.ProfileID); !ok {
				res.Error = "unsupported AppArmor profile"
			} else if err := runAppArmor(ctx, req.ProfileID); err != nil {
				res.Error = err.Error()
			} else {
				res.OK = true
			}
		} else if req.Kind == "apply-host-hardening" && runHostHardening != nil {
			if !requestHasOnlyFields(scanner.Bytes(), "version", "kind", "host_hardening") || req.HostHardening == nil {
				res.Error = "host hardening request accepts only host_hardening"
			} else if err := hosthardening.ValidatePolicy(*req.HostHardening); err != nil {
				res.Error = err.Error()
			} else if err := runHostHardening(ctx, *req.HostHardening); err != nil {
				res.Error = err.Error()
			} else {
				res.OK = true
			}
		} else if req.Kind == "install-observability" && runObservability != nil {
			if !requestHasOnlyFields(scanner.Bytes(), "version", "kind", "observability_target", "target_user", "retain_vmcores") {
				res.Error = "observability install request contains unsupported fields"
			} else if err := observabilityinstall.ValidateSetupRequest(req.ObservabilityTarget, req.Retention); err != nil {
				res.Error = err.Error()
			} else if req.TargetUser == "" || req.TargetUser != hostreqkit.InvokingUser() {
				res.Error = "observability install target must match the invoking setup user"
			} else if err := runObservability(ctx, req.ObservabilityTarget, req.TargetUser, req.Retention); err != nil {
				res.Error = err.Error()
			} else {
				res.OK = true
			}
		} else {
			res.Error = "unsupported setup privilege request"
		}
		if err := encoder.Encode(res); err != nil {
			return fmt.Errorf("write setup privilege response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read setup privilege request: %w", err)
	}
	return nil
}

func appArmorProfileRequestIsClosed(payload []byte) bool {
	return requestHasOnlyFields(payload, "version", "kind", "profile_id")
}

func requestHasOnlyFields(payload []byte, allowed ...string) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return false
	}
	allow := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allow[field] = struct{}{}
	}
	for field := range fields {
		if _, ok := allow[field]; !ok {
			return false
		}
	}
	return true
}

type aptRepositorySpec struct {
	keyPath            string
	sourcePath         string
	primaryFingerprint string
}

func validateAptRepository(repository string, keyData []byte, sourceLine string) error {
	spec, ok := aptRepositories[repository]
	if !ok {
		return fmt.Errorf("APT repository %q is not allowed in setup privilege session", repository)
	}
	if len(keyData) == 0 || len(keyData) > 256<<10 {
		return errors.New("APT repository key data is empty or exceeds 256 KiB")
	}
	if strings.ContainsAny(sourceLine, "\r\n\x00") || len(sourceLine) > 2048 {
		return errors.New("APT repository source does not match its fixed repository identity")
	}
	fields := strings.Fields(sourceLine)
	switch repository {
	case "stripe":
		if sourceLine != "deb [signed-by="+spec.keyPath+"] https://packages.stripe.dev/stripe-cli-debian-local stable main" {
			return errors.New("APT repository source does not match its fixed repository identity")
		}
	case "hashicorp":
		if len(fields) != 6 || fields[0] != "deb" || fields[1] != "[signed-by="+spec.keyPath ||
			(fields[2] != "arch=amd64]" && fields[2] != "arch=arm64]") ||
			fields[3] != "https://apt.releases.hashicorp.com" || !aptCodename.MatchString(fields[4]) || fields[5] != "main" {
			return errors.New("APT repository source does not match its fixed repository identity")
		}
	}
	return nil
}

var aptRepositories = map[string]aptRepositorySpec{
	"hashicorp": {
		keyPath: "/usr/share/keyrings/hashicorp-archive-keyring.gpg", sourcePath: "/etc/apt/sources.list.d/hashicorp.list",
		primaryFingerprint: "D55C0D1AC78A8D8126CB631CFC9CA96ACA026560",
	},
	"stripe": {
		keyPath: "/usr/share/keyrings/stripe.gpg", sourcePath: "/etc/apt/sources.list.d/stripe.list",
		primaryFingerprint: "6681D7C3D103DAC65D79C25EDEEBD57F917C83E3",
	},
}

func RunSystemAptRepository(ctx context.Context, repository string, keyData []byte, sourceLine string) error {
	if err := validateAptRepository(repository, keyData, sourceLine); err != nil {
		return err
	}
	commandCtx, cancel := context.WithTimeout(ctx, tuning.SetupExtendedOperationTimeout())
	defer cancel()
	if err := verifyAptSigningKey(commandCtx, repository, keyData); err != nil {
		return err
	}
	spec := aptRepositories[repository]
	return applyAptRepository(commandCtx, spec, keyData, sourceLine, writeAptFile, updateAptIndexes)
}

func verifyAptSigningKey(ctx context.Context, repository string, keyData []byte) error {
	spec, ok := aptRepositories[repository]
	if !ok {
		return fmt.Errorf("APT repository %q is not allowed in setup privilege session", repository)
	}
	file, err := os.CreateTemp("", "vrooli-apt-key-check-*")
	if err != nil {
		return fmt.Errorf("prepare APT signing-key verification: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.Write(keyData); err != nil {
		_ = file.Close()
		return fmt.Errorf("write APT signing-key verification input: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close APT signing-key verification input: %w", err)
	}
	stdout, stderr := &boundedBuffer{}, &boundedBuffer{}
	cmd := exec.CommandContext(ctx, "gpg", "--batch", "--no-options", "--show-keys", "--with-colons", "--fingerprint", path)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("inspect APT signing key: %w: %s", err, strings.TrimSpace(stdout.String()+"\n"+stderr.String()))
	}
	fingerprint, err := primaryFingerprint(stdout.String())
	if err != nil {
		return err
	}
	if fingerprint != spec.primaryFingerprint {
		return fmt.Errorf("APT repository %s signing-key fingerprint mismatch: got %s", repository, fingerprint)
	}
	return nil
}

func primaryFingerprint(output string) (string, error) {
	primaryKeySeen := false
	waitingForPrimaryFingerprint := false
	fingerprint := ""
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 10 {
			continue
		}
		if fields[0] == "pub" {
			if primaryKeySeen {
				return "", errors.New("APT signing key contains more than one primary key")
			}
			primaryKeySeen = true
			waitingForPrimaryFingerprint = true
			continue
		}
		if waitingForPrimaryFingerprint && fields[0] == "fpr" {
			fingerprint = strings.ToUpper(fields[9])
			waitingForPrimaryFingerprint = false
		}
	}
	if !primaryKeySeen || fingerprint == "" {
		return "", errors.New("APT signing key has no primary fingerprint")
	}
	return fingerprint, nil
}

// Repository file updates are convergent rather than transactional. Setup
// stops before package installation on any error; a retry writes the same
// validated files and reruns apt-get update to complete the state.
func applyAptRepository(ctx context.Context, spec aptRepositorySpec, keyData []byte, sourceLine string, writeFile func(string, []byte) error, update func(context.Context) error) error {
	if err := writeFile(spec.keyPath, keyData); err != nil {
		return fmt.Errorf("write APT signing key: %w", err)
	}
	if err := writeFile(spec.sourcePath, []byte(sourceLine+"\n")); err != nil {
		return fmt.Errorf("write APT source: %w", err)
	}
	if err := update(ctx); err != nil {
		return fmt.Errorf("update APT package indexes: %w", err)
	}
	return nil
}

func updateAptIndexes(ctx context.Context) error {
	return runBoundedCommand(ctx, "apt-get", "update", "-qq")
}

func writeAptFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("refusing unsafe APT directory %s", dir)
	}
	if !isRootOwnedDirectory(info) {
		return fmt.Errorf("APT directory %s is not owned by root", dir)
	}
	tmp, err := os.CreateTemp(dir, ".vrooli-apt-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func RunSystemBrokerInstall(ctx context.Context, repoRoot string) (privilegebroker.SetupStatus, error) {
	executable, err := os.Executable()
	if err != nil {
		return privilegebroker.SetupStatus{}, fmt.Errorf("resolve setup executable: %w", err)
	}
	return privilegebroker.DefaultInstallerForRepo(executable, repoRoot).Install(ctx)
}

func RunSystemOnboardingApplyGrant(ctx context.Context, tools, safeguards []string) error {
	content, err := onboardingapplyprivileges.PolicyContentForCurrentProcess(tools, safeguards)
	if err != nil {
		return err
	}
	return installOnboardingGrant(ctx, content)
}

func installOnboardingGrant(ctx context.Context, content string) error {
	const grantPath = "/etc/sudoers.d/vrooli-onboarding-apply"
	if err := ensureSudoersDirectory(); err != nil {
		return fmt.Errorf("prepare sudoers directory: %w", err)
	}
	tmp, err := os.CreateTemp("", "vrooli-onboarding-apply-*.sudoers")
	if err != nil {
		return fmt.Errorf("prepare onboarding grant: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.WriteString(tmp, content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write onboarding grant: %w", err)
	}
	if err := tmp.Chmod(0o440); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set onboarding grant mode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close onboarding grant: %w", err)
	}
	if err := runBoundedCommand(ctx, "visudo", "-c", "-f", tmpPath); err != nil {
		return fmt.Errorf("validate onboarding grant: %w", err)
	}
	if err := runBoundedCommand(ctx, "install", "-m", "440", tmpPath, grantPath); err != nil {
		return fmt.Errorf("install onboarding grant: %w", err)
	}
	return nil
}

func runBoundedCommand(ctx context.Context, program string, args ...string) error {
	stdout, stderr := &boundedBuffer{}, &boundedBuffer{}
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w: %s", program, err, strings.TrimSpace(stdout.String()+"\n"+stderr.String()))
	}
	return nil
}

func RunSystemPackageInstall(ctx context.Context, manager, packageName string, stdout, stderr io.Writer) error {
	program, args, err := packageCommand(manager, packageName)
	if err != nil {
		return err
	}
	commandCtx, cancel := context.WithTimeout(ctx, tuning.SetupExtendedOperationTimeout())
	defer cancel()
	cmd := exec.CommandContext(commandCtx, program, args...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd.Run()
}

func packageCommand(manager, packageName string) (string, []string, error) {
	manager = strings.ToLower(strings.TrimSpace(manager))
	packageName = strings.TrimSpace(packageName)
	if !packageToken.MatchString(packageName) {
		return "", nil, errors.New("invalid package identifier for setup privilege session")
	}
	switch manager {
	case "apt", "apt-get":
		return "apt-get", []string{"install", "-y", packageName}, nil
	case "dnf":
		return "dnf", []string{"install", "-y", packageName}, nil
	case "yum":
		return "yum", []string{"install", "-y", packageName}, nil
	case "pacman":
		return "pacman", []string{"-S", "--noconfirm", packageName}, nil
	case "apk":
		return "apk", []string{"add", packageName}, nil
	default:
		return "", nil, fmt.Errorf("package manager %q is not allowed in the setup privilege session", manager)
	}
}

type boundedBuffer struct {
	data      []byte
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	remaining := maxOutputBytes - len(b.data)
	if remaining > 0 {
		if len(p) > remaining {
			b.data = append(b.data, p[:remaining]...)
		} else {
			b.data = append(b.data, p...)
		}
	}
	if len(p) > remaining {
		b.truncated = true
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string { return string(b.data) }

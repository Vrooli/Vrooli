package emergencywatchdog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const watchdogImageLimit = 512 << 20

var (
	watchdogStampPattern  = regexp.MustCompile(`^managed:[a-f0-9]{64}$`)
	watchdogDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// The receipt describes executable publication and its --version canary, not
// completion of a scheduled pressure check or healthy native timer activation.
// It lives beside the two bounded executable slots. Paths are derived by the
// owner, never taken from a decoded receipt.
type watchdogUpdateReceipt struct {
	SchemaVersion  int    `json:"schema_version"`
	Phase          string `json:"phase"`
	CandidateSHA   string `json:"candidate_sha256"`
	CandidateStamp string `json:"candidate_stamp"`
	PreviousSHA    string `json:"previous_sha256"`
	PreviousStamp  string `json:"previous_stamp"`
}

var publishWatchdogImage = copyWatchdogImage

// ReconcileBinaryPublication lets the independent native supervisor finish an
// already-admitted executable publication after setup/caller loss. It cannot
// build, enroll a release, replace scheduler definitions or start a new update.
func ReconcileBinaryPublication(ctx context.Context, home string) error {
	if ctx == nil {
		return fmt.Errorf("watchdog reconciliation context is required")
	}
	if !filepath.IsAbs(home) {
		return fmt.Errorf("watchdog reconciliation requires an absolute invoking-user home")
	}
	p := resolvePaths(home)
	if _, err := os.Lstat(p.Binary + ".update.json"); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	lockCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	release, err := artifactlock.AcquireWithPathContext(lockCtx, "watchdog-binary:"+p.Binary, func(string) (string, error) { return p.Binary + ".lock", nil })
	if err != nil {
		return err
	}
	defer release()
	if err := guardBaselineRestoration(p); err != nil {
		return err
	}
	r, err := readWatchdogUpdateReceipt(p)
	if err != nil {
		return err
	}
	if r == nil || r.Phase != "publishing" && r.Phase != "restoring" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := guardRootUpdateMutation(ctx, home); err != nil {
		return err
	}
	return reconcileWatchdogPublication(p, *r)
}

func updateWatchdogBinary(p paths) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	release, err := artifactlock.AcquireWithPathContext(ctx, "watchdog-binary:"+p.Binary, func(string) (string, error) { return p.Binary + ".lock", nil })
	if err != nil {
		return fmt.Errorf("lock watchdog update: %w", err)
	}
	defer release()
	return updateWatchdogBinaryLocked(ctx, p, watchdogSource{})
}

// watchdogSource selects what a publication installs. The zero value builds
// from the resolved source checkout (setup). An operator transition passes its
// explicit checkout and the image it already built and qualified against the
// candidate root, so the published watchdog is exactly the one it checked.
type watchdogSource struct {
	root     string
	prebuilt string
	// digest binds a prebuilt image to the bytes its caller probed.
	digest string
}

func guardRootUpdateMutation(ctx context.Context, home string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return err
	}
	return store.GuardOrdinaryMutation(ctx)
}

// Caller retains the shared publication lock through native scheduler writes.
func updateWatchdogBinaryLocked(ctx context.Context, p paths, source watchdogSource) error {
	if err := guardBaselineRestoration(p); err != nil {
		return err
	}
	if err := guardRootUpdateMutation(ctx, p.Home); err != nil {
		return err
	}
	previous, err := readWatchdogUpdateReceipt(p)
	if err != nil {
		return err
	}
	if previous != nil && (previous.Phase == "publishing" || previous.Phase == "restoring") {
		if err := reconcileWatchdogPublication(p, *previous); err != nil {
			return err
		}
		previous, err = readWatchdogUpdateReceipt(p)
		if err != nil {
			return err
		}
	}
	root := source.root
	var want string
	if source.prebuilt != "" {
		// A prebuilt image is published as the exact bytes its caller
		// qualified, with the stamp it was built with; the live tree may
		// have moved on since and is not consulted.
		want = installedWatchdogVersion(source.prebuilt)
		if !watchdogStampPattern.MatchString(want) || !watchdogDigestPattern.MatchString(source.digest) {
			return fmt.Errorf("prebuilt watchdog lacks its build stamp or bound digest")
		}
	} else {
		if root == "" {
			resolved, err := resolveWatchdogRootFn()
			if err != nil || strings.TrimSpace(resolved) == "" {
				return fmt.Errorf("resolve watchdog source root: %w", errors.Join(err, errors.New("a source root is required")))
			}
			root = resolved
		}
		want, err = expectedWatchdogVersion(root)
		if err != nil {
			return fmt.Errorf("fingerprint watchdog source: %w", err)
		}
	}
	if err := pruneAbandonedWatchdogStages(filepath.Dir(p.Binary)); err != nil {
		return err
	}
	// A verified release-built image already present needs no local compiler.
	if watchdogImageCurrent(p, want, previous) {
		return nil
	}
	stage, err := os.CreateTemp(filepath.Dir(p.Binary), ".vrooli-watchdog-stage-*")
	if err != nil {
		return err
	}
	stagePath := stage.Name()
	if err := stage.Close(); err != nil {
		_ = os.Remove(stagePath)
		return err
	}
	defer os.Remove(stagePath)
	// Give the invoking user ownership before its compiler opens the output.
	if err := config.WriteOwnedFileAtomic(stagePath, nil, 0o700); err != nil {
		return err
	}
	if source.prebuilt != "" {
		if err := copyPrebuiltWatchdog(source.prebuilt, stagePath); err != nil {
			return err
		}
	} else if err := buildWatchdogFn(root, stagePath); err != nil {
		return err
	}
	stagedFile, err := openWatchdogImage(stagePath)
	if err != nil {
		return fmt.Errorf("qualify staged watchdog file: %w", err)
	}
	chmodErr := stagedFile.Chmod(0o700)
	closeErr := stagedFile.Close()
	if err := errors.Join(chmodErr, closeErr); err != nil {
		return err
	}
	sha, err := watchdogImageDigest(stagePath)
	if err != nil {
		return fmt.Errorf("qualify staged watchdog: %w", err)
	}
	if source.prebuilt != "" && sha != source.digest {
		return fmt.Errorf("staged watchdog is not the image its caller qualified; installed image unchanged")
	}
	if installedWatchdogVersion(stagePath) != want {
		return fmt.Errorf("staged watchdog did not report the expected source stamp; installed image unchanged")
	}
	if previous != nil && (previous.Phase == "failed_restored" || previous.Phase == "failed_unrecovered") && previous.CandidateSHA == sha {
		return fmt.Errorf("watchdog release previously failed publication qualification; a changed release is required")
	}
	// The embedded stamp was selected before the compiler read its inputs.
	// A staged canary alone cannot detect edits made during compilation. Use
	// the same complete source/input owner again before touching either live
	// executable slot or its durable publication receipt.
	if source.prebuilt == "" {
		current, err := expectedWatchdogVersion(root)
		if err != nil {
			return fmt.Errorf("recheck watchdog source after build: %w", err)
		}
		if current != want {
			return fmt.Errorf("watchdog source changed during build; installed image unchanged")
		}
	}
	r := watchdogUpdateReceipt{SchemaVersion: 1, Phase: "publishing", CandidateSHA: sha, CandidateStamp: want}
	oldSHA, err := watchdogImageDigest(p.Binary)
	if err == nil {
		if previous != nil && previous.Phase == "verified" && oldSHA != previous.CandidateSHA {
			return fmt.Errorf("installed watchdog differs from its verified publication; refusing to overwrite retained recovery material")
		}
		r.PreviousSHA = oldSHA
		if err := copyWatchdogImage(p.Binary, p.Binary+".previous", oldSHA); err != nil {
			return fmt.Errorf("retain prior watchdog before publication: %w", err)
		}
		r.PreviousStamp = installedWatchdogVersion(p.Binary + ".previous")
		if !watchdogStampPattern.MatchString(r.PreviousStamp) {
			r.PreviousStamp = ""
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("observe prior watchdog: %w", err)
	} else if previous != nil && previous.PreviousSHA != "" {
		// A missing current image must not erase an existing recovery copy's
		// identity when the next admitted publication is recorded.
		backupSHA, backupErr := watchdogImageDigest(p.Binary + ".previous")
		if backupErr != nil || backupSHA != previous.PreviousSHA {
			return fmt.Errorf("prior retained watchdog cannot be verified; publication refused")
		}
		r.PreviousSHA = backupSHA
		r.PreviousStamp = previous.PreviousStamp
		if r.PreviousStamp == "" || installedWatchdogVersion(p.Binary+".previous") != r.PreviousStamp {
			return fmt.Errorf("prior retained watchdog canary is unavailable; publication refused")
		}
	}
	if err := writeWatchdogUpdateReceipt(p, r); err != nil {
		return fmt.Errorf("persist watchdog publication intent: %w", err)
	}
	// An error may follow a visible rename. Reconcile bytes and the canary
	// rather than claiming the old executable was necessarily preserved.
	publishErr := publishWatchdogImage(stagePath, p.Binary, sha)
	if err := reconcileWatchdogPublication(p, r); err != nil {
		return errors.Join(publishErr, err)
	}
	return nil
}

func pruneAbandonedWatchdogStages(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	entries, err := f.ReadDir(1025)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > 1024 {
		return fmt.Errorf("watchdog artifact directory exceeds bounded cleanup inventory")
	}
	removed := 0
	for _, entry := range entries {
		if removed == 16 {
			break
		}
		if !strings.HasPrefix(entry.Name(), ".vrooli-watchdog-stage-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
		removed++
	}
	return nil
}

func watchdogImageCurrent(p paths, want string, r *watchdogUpdateReceipt) bool {
	if installedWatchdogVersion(p.Binary) != want {
		return false
	}
	sha, err := watchdogImageDigest(p.Binary)
	if err != nil {
		return false
	}
	if r != nil && r.Phase == "verified" && r.CandidateStamp == want {
		return sha == r.CandidateSHA
	}
	return true
}

func reconcileWatchdogPublication(p paths, r watchdogUpdateReceipt) error {
	current, err := watchdogImageDigest(p.Binary)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("observe interrupted watchdog publication: %w", err)
	}
	if r.Phase == "publishing" && current == r.CandidateSHA && installedWatchdogVersion(p.Binary) == r.CandidateStamp {
		r.Phase = "verified"
		return writeWatchdogUpdateReceipt(p, r)
	}
	if r.PreviousSHA != "" && current == r.PreviousSHA && r.Phase == "publishing" {
		// A crash before the rename cannot have replaced this exact old image.
		r.Phase = "failed_restored"
		if r.PreviousStamp == "" || installedWatchdogVersion(p.Binary) != r.PreviousStamp {
			r.Phase = "failed_unrecovered"
		}
		if err := writeWatchdogUpdateReceipt(p, r); err != nil {
			return err
		}
		return fmt.Errorf("watchdog publication did not replace its prior image; retained receipt records %s", r.Phase)
	}
	if current != "" && current != r.CandidateSHA && current != r.PreviousSHA {
		return fmt.Errorf("watchdog executable changed outside the admitted publication; refusing restoration")
	}
	if r.PreviousSHA == "" {
		r.Phase = "failed_unrecovered"
		if err := writeWatchdogUpdateReceipt(p, r); err != nil {
			return err
		}
		return fmt.Errorf("watchdog publication canary failed and no prior executable was retained")
	}
	if digest, err := watchdogImageDigest(p.Binary + ".previous"); err != nil || digest != r.PreviousSHA {
		return fmt.Errorf("retained watchdog image cannot be verified; restoration refused")
	}
	r.Phase = "restoring"
	if err := writeWatchdogUpdateReceipt(p, r); err != nil {
		return err
	}
	if current != r.PreviousSHA {
		if err := copyWatchdogImage(p.Binary+".previous", p.Binary, r.PreviousSHA); err != nil {
			return fmt.Errorf("restore watchdog image: %w", err)
		}
	}
	got, err := watchdogImageDigest(p.Binary)
	if err != nil || got != r.PreviousSHA {
		return fmt.Errorf("restored watchdog bytes are unverified")
	}
	r.Phase = "failed_restored"
	if r.PreviousStamp == "" || installedWatchdogVersion(p.Binary) != r.PreviousStamp {
		r.Phase = "failed_unrecovered"
	}
	if err := writeWatchdogUpdateReceipt(p, r); err != nil {
		return err
	}
	return fmt.Errorf("watchdog publication qualification failed; retained receipt records %s", r.Phase)
}

func openWatchdogImage(path string) (*os.File, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > watchdogImageLimit {
		return nil, fmt.Errorf("watchdog image must be a bounded nonempty regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = f.Close()
		return nil, fmt.Errorf("watchdog image identity changed while opening")
	}
	return f, nil
}

func watchdogImageDigest(path string) (string, error) {
	f, err := openWatchdogImage(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, watchdogImageLimit+1))
	if err != nil || n <= 0 || n > watchdogImageLimit {
		return "", fmt.Errorf("watchdog image exceeded its read bounds")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyWatchdogImage(source, destination, expected string) error {
	f, err := openWatchdogImage(source)
	if err != nil {
		return err
	}
	defer f.Close()
	return config.WriteOwnedFileDurableStream(destination, 0o700, func(w io.Writer) error {
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(w, h), io.LimitReader(f, watchdogImageLimit+1))
		if err != nil {
			return err
		}
		if n <= 0 || n > watchdogImageLimit || hex.EncodeToString(h.Sum(nil)) != expected {
			return fmt.Errorf("watchdog image changed before publication")
		}
		return nil
	})
}

func writeWatchdogUpdateReceipt(p paths, r watchdogUpdateReceipt) error {
	if err := validateWatchdogUpdateReceipt(r); err != nil {
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileDurable(p.Binary+".update.json", data, 0o600)
}

func readWatchdogUpdateReceipt(p paths) (*watchdogUpdateReceipt, error) {
	path := p.Binary + ".update.json"
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 8192 {
		return nil, fmt.Errorf("watchdog publication receipt is not a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("watchdog receipt changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil || len(data) > 8192 {
		return nil, fmt.Errorf("watchdog receipt read exceeded bounds")
	}
	// Decode fields individually so duplicate and case-alias fields cannot
	// acquire last-field-wins meaning in a recovery decision.
	allowed := map[string]bool{"schema_version": true, "phase": true, "candidate_sha256": true, "candidate_stamp": true, "previous_sha256": true, "previous_stamp": true}
	d := json.NewDecoder(bytes.NewReader(data))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return nil, fmt.Errorf("invalid watchdog publication receipt")
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid watchdog receipt field")
		}
		name, ok := key.(string)
		if !ok || !allowed[name] || seen[name] {
			return nil, fmt.Errorf("invalid or duplicate watchdog receipt field")
		}
		seen[name] = true
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, fmt.Errorf("invalid watchdog receipt value")
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("null watchdog receipt value")
		}
	}
	if _, err := d.Token(); err != nil {
		return nil, fmt.Errorf("invalid watchdog receipt object")
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing watchdog receipt data")
	}
	var r watchdogUpdateReceipt
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("invalid watchdog receipt types")
	}
	if len(seen) != len(allowed) {
		return nil, fmt.Errorf("invalid watchdog receipt identity")
	}
	if err := validateWatchdogUpdateReceipt(r); err != nil {
		return nil, err
	}
	return &r, nil
}

func validateWatchdogUpdateReceipt(r watchdogUpdateReceipt) error {
	if r.SchemaVersion != 1 || !watchdogDigestPattern.MatchString(r.CandidateSHA) || !watchdogStampPattern.MatchString(r.CandidateStamp) || r.PreviousSHA != "" && !watchdogDigestPattern.MatchString(r.PreviousSHA) || r.PreviousStamp != "" && !watchdogStampPattern.MatchString(r.PreviousStamp) || r.PreviousSHA == "" && r.PreviousStamp != "" {
		return fmt.Errorf("invalid watchdog receipt identity")
	}
	if r.Phase == "restoring" && r.PreviousSHA == "" || r.Phase == "failed_restored" && (r.PreviousSHA == "" || r.PreviousStamp == "") {
		return fmt.Errorf("watchdog restoration receipt lacks retained identity")
	}
	switch r.Phase {
	case "publishing", "restoring", "verified", "failed_restored", "failed_unrecovered":
	default:
		return fmt.Errorf("invalid watchdog receipt phase")
	}
	return nil
}

// versionCapture discards overflow while reporting it, so an invalid candidate
// cannot turn a setup version probe into an unbounded allocation.
type watchdogVersionCapture struct {
	buffer   bytes.Buffer
	overflow bool
}

func (w *watchdogVersionCapture) Write(p []byte) (int, error) {
	n := len(p)
	space := 256 - w.buffer.Len()
	if len(p) > space {
		w.overflow = true
		p = p[:space]
	}
	_, _ = w.buffer.Write(p)
	return n, nil
}

func probeWatchdogVersion(binary string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	name, args := hostreqkit.InvokingUserCommand(binary, "--version")
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	var capture watchdogVersionCapture
	cmd.Stdout = &capture
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil || capture.overflow {
		return ""
	}
	version := strings.TrimSpace(capture.buffer.String())
	if !watchdogStampPattern.MatchString(version) {
		return ""
	}
	return version
}

// RetainedWatchdogDefinition binds one previously qualified owner definition.
// Name is one of the four watchdog/root-recovery units, never an arbitrary unit.
type RetainedWatchdogDefinition struct {
	Name               string `json:"name"`
	ExpectedCurrentSHA string `json:"expected_current_sha256"`
	RetainedPath       string `json:"retained_path"`
	RetainedSHA        string `json:"retained_sha256"`
}

// RetainedWatchdogBaseline is explicit operator maintenance, not an automatic
// cross-contract compatibility exception. All inputs must be retained before
// the source checkout or compiler becomes unavailable.
type RetainedWatchdogBaseline struct {
	Home                      string                       `json:"home"`
	ExpectedCurrentSHA        string                       `json:"expected_current_sha256"`
	ExpectedCurrentReceiptSHA string                       `json:"expected_current_receipt_sha256"`
	ExpectedCurrentStamp      string                       `json:"expected_current_stamp"`
	RetainedImage             string                       `json:"retained_image"`
	RetainedSHA               string                       `json:"retained_sha256"`
	RetainedStamp             string                       `json:"retained_stamp"`
	Definitions               []RetainedWatchdogDefinition `json:"definitions"`
}

// RestoreRetainedBaseline publishes reviewed retained bytes, not a source build.
func RestoreRetainedBaseline(ctx context.Context, spec RetainedWatchdogBaseline) error {
	if ctx == nil || runtime.GOOS != "linux" || !filepath.IsAbs(spec.Home) || filepath.Clean(spec.Home) != spec.Home {
		return fmt.Errorf("retained baseline requires a Linux owner context and canonical home")
	}
	release, err := AcquirePublicationLock(ctx, spec.Home)
	if err != nil {
		return err
	}
	defer release()
	p := resolvePaths(spec.Home)
	if err := guardRootUpdateMutation(ctx, spec.Home); err != nil {
		return err
	}
	files, err := qualifyRetainedBaseline(spec, p)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(encoded)
	requestSHA := hex.EncodeToString(digest[:])
	journalPath := p.Binary + ".baseline-restore.json"
	historyPath := p.Binary + ".baseline-restore-" + requestSHA + ".previous.json"
	terminalPath := p.Binary + ".baseline-restore-" + requestSHA + ".verified.json"
	active, err := readBaselineJournal(journalPath)
	if err != nil {
		return err
	}
	terminal, err := readBaselineJournal(terminalPath)
	if err != nil {
		return err
	}
	if active != nil && active.RequestSHA != requestSHA {
		return fmt.Errorf("another retained-baseline request owns restoration")
	}
	current, err := readWatchdogUpdateReceipt(p)
	if err != nil || current == nil {
		return fmt.Errorf("qualified current publication required: %w", err)
	}
	final := watchdogUpdateReceipt{SchemaVersion: 1, Phase: "verified", CandidateSHA: spec.RetainedSHA, CandidateStamp: spec.RetainedStamp, PreviousSHA: spec.ExpectedCurrentSHA, PreviousStamp: spec.ExpectedCurrentStamp}
	publishing := final
	publishing.Phase = "publishing"
	if terminal != nil {
		if terminal.OriginalReceiptSHA != spec.ExpectedCurrentReceiptSHA || active != nil && *active != *terminal {
			return fmt.Errorf("terminal restoration provenance differs from exact active request")
		}
		history, err := readBoundedBaselineFile(historyPath, 8192)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(history)
		if hex.EncodeToString(sum[:]) != terminal.OriginalReceiptSHA {
			return fmt.Errorf("immutable restoration history changed")
		}
		if terminal.RequestSHA != requestSHA || terminal.Phase != "verified" || *current != final {
			return fmt.Errorf("completed restoration no longer owns current publication")
		}
		if err := verifyBaselineImageAndDefinitions(p, spec, files); err != nil {
			return err
		}
		if err := qualifyBaselineNative(ctx, p, files, false); err != nil {
			return err
		}
		if err := verifyBaselinePrevious(p, spec); err != nil {
			return err
		}
		if active != nil {
			if active.Phase != "verified" {
				return fmt.Errorf("terminal/active restoration phase differs")
			}
			return retireBaselineJournal(journalPath)
		}
		return nil
	}
	if active == nil {
		if current.Phase != "verified" || current.CandidateSHA != spec.ExpectedCurrentSHA || current.CandidateStamp != spec.ExpectedCurrentStamp {
			return fmt.Errorf("current verified publication differs from reviewed baseline")
		}
		image, err := watchdogImageDigest(p.Binary)
		if err != nil || image != spec.ExpectedCurrentSHA {
			return fmt.Errorf("current watchdog bytes differ from reviewed publication")
		}
		if err := qualifyBaselineBackupSlot(p, *current, spec, false); err != nil {
			return err
		}
		for _, f := range files {
			if err := baselineDefinitionMatches(f.path, f.definition.ExpectedCurrentSHA); err != nil {
				return err
			}
		}
		if err := qualifyBaselineAdmission(ctx, p, files, false); err != nil {
			return err
		}
		old, err := readBoundedBaselineFile(p.Binary+".update.json", 8192)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(old)
		if hex.EncodeToString(sum[:]) != spec.ExpectedCurrentReceiptSHA {
			return fmt.Errorf("current publication receipt differs from reviewed bytes")
		}
		if err := writeBaselineEvidenceOnce(historyPath, old); err != nil {
			return err
		}
		oldSHA := sha256.Sum256(old)
		active = &baselineRestoreJournal{Version: 1, RequestSHA: requestSHA, OriginalReceiptSHA: hex.EncodeToString(oldSHA[:]), Phase: "prepared"}
		if err := writeBaselineJournal(journalPath, *active); err != nil {
			return err
		}
	}
	if active.RequestSHA != requestSHA || active.Phase != "prepared" && active.Phase != "publishing" && active.Phase != "verified" {
		return fmt.Errorf("invalid restoration ownership")
	}
	old, err := readBoundedBaselineFile(historyPath, 8192)
	if err != nil {
		return err
	}
	oldSHA := sha256.Sum256(old)
	if hex.EncodeToString(oldSHA[:]) != active.OriginalReceiptSHA || active.OriginalReceiptSHA != spec.ExpectedCurrentReceiptSHA {
		return fmt.Errorf("restoration history changed")
	}
	oldReceipt := watchdogUpdateReceipt{}
	if err := json.Unmarshal(old, &oldReceipt); err != nil {
		return err
	}
	// The active journal fences every other publication entrypoint. A crash
	// between a receipt write and journal acknowledgement may expose either
	// exact receipt, but never a different admission.
	if *current != publishing && *current != final && !(active.Phase == "prepared" && *current == oldReceipt) {
		return fmt.Errorf("foreign publication replaced the retained-baseline request")
	}
	if err := qualifyBaselineBackupSlot(p, oldReceipt, spec, true); err != nil {
		return err
	}
	image, err := watchdogImageDigest(p.Binary)
	if err != nil || image != spec.ExpectedCurrentSHA && image != spec.RetainedSHA {
		return fmt.Errorf("foreign watchdog image; restoration refused")
	}
	for _, f := range files {
		if err := baselineDefinitionMatchesEither(f.path, f.definition.ExpectedCurrentSHA, f.definition.RetainedSHA); err != nil {
			return err
		}
	}
	if err := qualifyBaselineAdmission(ctx, p, files, true); err != nil {
		return err
	}
	{
		if err := verifyBaselinePrevious(p, spec); err != nil {
			if image != spec.ExpectedCurrentSHA {
				return fmt.Errorf("current image unavailable for exact rollback retention")
			}
			copyErr := baselineRetainCurrent(p.Binary, p.Binary+".previous", spec.ExpectedCurrentSHA)
			if err := verifyBaselinePrevious(p, spec); err != nil {
				return fmt.Errorf("retain actual current watchdog before restoration: %w", errors.Join(copyErr, err))
			}
		}
		if image != spec.RetainedSHA {
			copyErr := publishWatchdogImage(spec.RetainedImage, p.Binary, spec.RetainedSHA)
			got, readErr := watchdogImageDigest(p.Binary)
			if readErr != nil || got != spec.RetainedSHA {
				return fmt.Errorf("retained image publication incomplete: %w", errors.Join(copyErr, readErr))
			}
		}
		if installedWatchdogVersion(p.Binary) != spec.RetainedStamp {
			return fmt.Errorf("retained watchdog canary differs")
		}
		// Old journal-unaware reconcilers must never observe a publishing receipt
		// before its target is visible. A prior final acknowledgement stays final;
		// the separate journal still owns unfinished native completion.
		if *current != final {
			if err := writeWatchdogUpdateReceipt(p, publishing); err != nil {
				return err
			}
			active.Phase = "publishing"
			if err := writeBaselineJournal(journalPath, *active); err != nil {
				return err
			}
		}
		for _, f := range files {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := baselineDefinitionMatchesEither(f.path, f.definition.ExpectedCurrentSHA, f.definition.RetainedSHA); err != nil {
				return err
			}
			if baselineDefinitionMatches(f.path, f.definition.RetainedSHA) == nil {
				continue
			}
			writeErr := baselineDefinitionPublish(f.path, f.content)
			if err := baselineDefinitionMatches(f.path, f.definition.RetainedSHA); err != nil {
				return fmt.Errorf("retained definition publication incomplete: %w", errors.Join(writeErr, err))
			}
		}
		for _, args := range [][]string{{"--user", "daemon-reload"}, {"--user", "enable", "--now", timerName}, {"--user", "enable", "--now", baselineRecoveryTimer()}} {
			if err := baselineNativeCommand(ctx, args); err != nil {
				return err
			}
		}
		if err := verifyBaselineImageAndDefinitions(p, spec, files); err != nil {
			return err
		}
		if err := qualifyBaselineNative(ctx, p, files, false); err != nil {
			return err
		}
		if err := writeWatchdogUpdateReceipt(p, final); err != nil {
			return err
		}
	}
	active.Phase = "verified"
	if err := writeBaselineJournal(journalPath, *active); err != nil {
		return err
	}
	data, err := json.Marshal(active)
	if err != nil {
		return err
	}
	if err := writeBaselineEvidenceOnce(terminalPath, data); err != nil {
		return err
	}
	// Terminal evidence remains immutable; retiring only this owned active
	// journal releases the normal update path without changing old codecs.
	return retireBaselineJournal(journalPath)
}

type baselineRestoreJournal struct {
	Version            int    `json:"version"`
	RequestSHA         string `json:"request_sha256"`
	OriginalReceiptSHA string `json:"original_receipt_sha256"`
	Phase              string `json:"phase"`
}
type retainedBaselineFile struct {
	definition RetainedWatchdogDefinition
	path       string
	content    []byte
}

var (
	baselineRetainCurrent     = copyWatchdogImage
	baselineDefinitionPublish = func(path string, content []byte) error { return config.WriteOwnedFileDurable(path, content, 0o644) }
	baselineNativeProperties  = recoveryNativeProperties
	baselineNativeCommand     = func(ctx context.Context, args []string) error {
		name, argv := hostreqkit.InvokingUserCommand("systemctl", args...)
		cmd := exec.CommandContext(ctx, name, argv...)
		cmd.WaitDelay = time.Second
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("restore native scheduler %v: %w: %s", args, err, strings.TrimSpace(string(out)))
		}
		return nil
	}
)

func baselineRecoveryTimer() string {
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRootUpdateRecoveryTimer)
	return unit.Systemd
}

func baselineUnitNames() []string {
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRootUpdateRecovery)
	return []string{serviceName, timerName, unit.Systemd, baselineRecoveryTimer()}
}

func qualifyRetainedBaseline(spec RetainedWatchdogBaseline, p paths) ([]retainedBaselineFile, error) {
	if !watchdogDigestPattern.MatchString(spec.ExpectedCurrentReceiptSHA) || !watchdogDigestPattern.MatchString(spec.ExpectedCurrentSHA) || !watchdogDigestPattern.MatchString(spec.RetainedSHA) || !watchdogStampPattern.MatchString(spec.ExpectedCurrentStamp) || !watchdogStampPattern.MatchString(spec.RetainedStamp) || spec.ExpectedCurrentSHA == spec.RetainedSHA {
		return nil, fmt.Errorf("exact distinct image identities required")
	}
	if spec.RetainedImage == p.Binary+".previous" || spec.RetainedImage == p.Binary {
		return nil, fmt.Errorf("retained image must be outside rotating current/previous slots")
	}
	if !filepath.IsAbs(spec.RetainedImage) || filepath.Clean(spec.RetainedImage) != spec.RetainedImage {
		return nil, fmt.Errorf("canonical retained image path required")
	}
	got, err := watchdogImageDigest(spec.RetainedImage)
	if err != nil || got != spec.RetainedSHA || installedWatchdogVersion(spec.RetainedImage) != spec.RetainedStamp {
		return nil, fmt.Errorf("retained image identity/canary differs")
	}
	names := map[string]bool{}
	for _, name := range baselineUnitNames() {
		names[name] = true
	}
	if len(spec.Definitions) != len(names) {
		return nil, fmt.Errorf("all four owner definitions required")
	}
	files := []retainedBaselineFile{}
	artifact := platform.RenderedArtifact{Target: "linux"}
	for _, d := range spec.Definitions {
		if !names[d.Name] || !watchdogDigestPattern.MatchString(d.ExpectedCurrentSHA) || !watchdogDigestPattern.MatchString(d.RetainedSHA) || !filepath.IsAbs(d.RetainedPath) || filepath.Clean(d.RetainedPath) != d.RetainedPath {
			return nil, fmt.Errorf("unqualified retained native definition")
		}
		delete(names, d.Name)
		content, err := readBaselineDefinition(d.RetainedPath)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != d.RetainedSHA {
			return nil, fmt.Errorf("retained definition changed")
		}
		files = append(files, retainedBaselineFile{d, filepath.Join(filepath.Dir(p.ServiceUnit), d.Name), content})
		artifact.Files = append(artifact.Files, platform.RenderedFile{Name: d.Name, Content: string(content)})
	}
	verdict := platform.ValidateArtifact(artifact, platform.ScopeUser)
	if verdict.State != platform.VerdictAccepted {
		return nil, fmt.Errorf("retained native definitions unqualified: %s", verdict.String())
	}
	sort.Slice(files, func(i, j int) bool { return files[i].definition.Name < files[j].definition.Name })
	return files, nil
}
func readBaselineDefinition(path string) ([]byte, error) { return readBoundedBaselineFile(path, 1<<20) }
func readBoundedBaselineFile(path string, limit int64) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() > limit {
		return nil, fmt.Errorf("native definition is not a bounded regular file: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, fmt.Errorf("native definition changed while opening")
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, fmt.Errorf("native definition exceeded bound")
	}
	return b, nil
}

func baselineDefinitionMatches(path, want string) error {
	b, err := readBaselineDefinition(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != want {
		return fmt.Errorf("native definition differs from reviewed bytes: %s", path)
	}
	return nil
}

func baselineDefinitionMatchesEither(path, a, b string) error {
	if baselineDefinitionMatches(path, a) == nil {
		return nil
	}
	return baselineDefinitionMatches(path, b)
}

func verifyBaselineImageAndDefinitions(p paths, spec RetainedWatchdogBaseline, files []retainedBaselineFile) error {
	sum, err := watchdogImageDigest(p.Binary)
	if err != nil || sum != spec.RetainedSHA || installedWatchdogVersion(p.Binary) != spec.RetainedStamp {
		return fmt.Errorf("restored image unverified")
	}
	for _, f := range files {
		if err := baselineDefinitionMatches(f.path, f.definition.RetainedSHA); err != nil {
			return err
		}
	}
	return nil
}

func qualifyBaselineNative(ctx context.Context, p paths, files []retainedBaselineFile, resuming bool) error {
	return qualifyBaselineNativeState(ctx, p, files, resuming, true)
}

func qualifyBaselineAdmission(ctx context.Context, p paths, files []retainedBaselineFile, resuming bool) error {
	return qualifyBaselineNativeState(ctx, p, files, resuming, false)
}

func qualifyBaselineNativeState(ctx context.Context, p paths, files []retainedBaselineFile, resuming, healthy bool) error {
	for _, f := range files {
		props, err := baselineNativeProperties(ctx, f.definition.Name)
		if err != nil {
			return err
		}
		if props["LoadState"] != "loaded" || props["FragmentPath"] != f.path || props["DropInPaths"] != "" || props["NeedDaemonReload"] != "no" && !(resuming && props["NeedDaemonReload"] == "yes") {
			return fmt.Errorf("unknown native override/state for %s", f.definition.Name)
		}
		if props["ActiveState"] != "active" && props["ActiveState"] != "inactive" && props["ActiveState"] != "failed" {
			return fmt.Errorf("unknown native activity state for %s", f.definition.Name)
		}
		if props["UnitFileState"] != "enabled" && props["UnitFileState"] != "disabled" && props["UnitFileState"] != "static" && props["UnitFileState"] != "indirect" {
			return fmt.Errorf("unknown native enablement for %s", f.definition.Name)
		}
		if healthy && strings.HasSuffix(f.definition.Name, ".timer") && (props["ActiveState"] != "active" || props["UnitFileState"] != "enabled") {
			return fmt.Errorf("native timer not enabled/active: %s", f.definition.Name)
		}
	}
	return nil
}

func readBaselineJournal(path string) (*baselineRestoreJournal, error) {
	data, err := readBaselineDefinition(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		if _, e := os.Lstat(path); errors.Is(e, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) > 4096 {
		return nil, fmt.Errorf("restoration journal exceeds bound")
	}
	allowed := map[string]bool{"version": true, "request_sha256": true, "original_receipt_sha256": true, "phase": true}
	seen := map[string]bool{}
	d := json.NewDecoder(bytes.NewReader(data))
	first, err := d.Token()
	if err != nil || first != json.Delim('{') {
		return nil, fmt.Errorf("invalid restoration journal")
	}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, err
		}
		name, ok := key.(string)
		if !ok || !allowed[name] || seen[name] {
			return nil, fmt.Errorf("invalid or duplicate restoration journal field")
		}
		seen[name] = true
		var v json.RawMessage
		if err := d.Decode(&v); err != nil {
			return nil, err
		}
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return nil, fmt.Errorf("null restoration journal field")
		}
	}
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	if err := d.Decode(new(any)); err != io.EOF || len(seen) != len(allowed) {
		return nil, fmt.Errorf("ambiguous restoration journal")
	}
	var r baselineRestoreJournal
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.Version != 1 || !watchdogDigestPattern.MatchString(r.RequestSHA) || !watchdogDigestPattern.MatchString(r.OriginalReceiptSHA) || r.Phase != "prepared" && r.Phase != "publishing" && r.Phase != "verified" {
		return nil, fmt.Errorf("invalid restoration journal")
	}
	return &r, nil
}

func writeBaselineJournal(path string, r baselineRestoreJournal) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileDurable(path, b, 0o600)
}

var baselineEvidenceLink = os.Link

func writeBaselineEvidenceOnce(path string, data []byte) error {
	// Write/fsync a complete private stage first, then publish without replacing
	// any existing immutable evidence. Caller loss can leave an unlinked stage,
	// never a truncated final history or terminal receipt.
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".stage-")
	if err != nil {
		return err
	}
	stage := f.Name()
	defer os.Remove(stage)
	if err := config.ChownFileToInvokingUser(f); err != nil {
		f.Close()
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err := errors.Join(err, closeErr); err != nil {
		return err
	}
	if err := baselineEvidenceLink(stage, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			old, readErr := readBoundedBaselineFile(path, 8192)
			if readErr == nil && bytes.Equal(old, data) {
				return syncBaselineDirectory(filepath.Dir(path))
			}
			return fmt.Errorf("retained baseline evidence differs")
		}
		return err
	}
	return syncBaselineDirectory(filepath.Dir(path))
}

func syncBaselineDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func retireBaselineJournal(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncBaselineDirectory(filepath.Dir(path))
}

func guardBaselineRestoration(p paths) error {
	r, err := readBaselineJournal(p.Binary + ".baseline-restore.json")
	if err != nil {
		return err
	}
	if r != nil {
		return fmt.Errorf("pending retained-baseline restoration; resume its exact owned request")
	}
	return nil
}

func verifyBaselinePrevious(p paths, spec RetainedWatchdogBaseline) error {
	sum, err := watchdogImageDigest(p.Binary + ".previous")
	if err != nil || sum != spec.ExpectedCurrentSHA || installedWatchdogVersion(p.Binary+".previous") != spec.ExpectedCurrentStamp {
		return fmt.Errorf("canonical previous slot does not retain reviewed current publication")
	}
	return nil
}

func qualifyBaselineBackupSlot(p paths, original watchdogUpdateReceipt, spec RetainedWatchdogBaseline, resuming bool) error {
	sum, err := watchdogImageDigest(p.Binary + ".previous")
	if errors.Is(err, os.ErrNotExist) && original.PreviousSHA == "" {
		return nil
	}
	if err != nil {
		return err
	}
	if sum == original.PreviousSHA {
		return nil
	}
	if resuming && sum == spec.ExpectedCurrentSHA {
		return nil
	}
	return fmt.Errorf("foreign canonical previous slot; refused overwrite")
}

// copyPrebuiltWatchdog stages an already built image; the staged canary and
// source-stamp checks that follow still qualify it before publication.
func copyPrebuiltWatchdog(source, stage string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 512<<20 {
		return fmt.Errorf("prebuilt watchdog is not a bounded regular image")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(stage, data, 0o700)
}

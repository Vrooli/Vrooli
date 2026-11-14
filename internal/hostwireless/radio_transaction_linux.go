//go:build linux

package hostwireless

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

type wirelessRadioRecord struct {
	Version          int                      `json:"version"`
	Transaction      string                   `json:"transaction"`
	IntentSHA256     string                   `json:"intent_sha256"`
	Device           string                   `json:"device"`
	Phase            string                   `json:"phase"`
	Before           []wirelessModuleSnapshot `json:"before"`
	PreservedModules []string                 `json:"preserved_modules"`
	Dependencies     []string                 `json:"dependencies"`
	BootID           string                   `json:"activation_boot_id"`
	RestoreKind      string                   `json:"restore_kind,omitempty"`
	RecoveryBootID   string                   `json:"recovery_boot_id,omitempty"`
}

const radioPhaseUnloading = "unloading"
const radioPhaseLoading = "loading"
const radioPhaseObserved = "observed"
const radioPhaseRestoring = "restoring_stock"
const radioPhaseRestored = "stock_restored"

func nativeWirelessRadioSystem() *wirelessRadioSystem {
	s := &wirelessRadioSystem{command: nativeWirelessRadioCommand, identity: nativeRadioModuleIdentity, aliases: nativeRadioAliases}
	s.unload = func(ctx context.Context, module string) error {
		_, err := nativeWirelessRadioCommand(ctx, "/usr/sbin/rmmod", "--", module)
		return err
	}
	return s
}

func nativeWirelessRadioCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	switch name {
	case "/usr/sbin/modprobe", "/usr/sbin/rmmod", "/usr/sbin/iw", "/usr/bin/busctl":
	default:
		return nil, errors.New("radio command is outside the fixed owner")
	}
	if name == "/usr/bin/busctl" && (!slices.Contains(args, "--auto-start=no") || !slices.Contains(args, "call") || !slices.Contains(args, "--allow-interactive-authorization=no")) {
		return nil, errors.New("passive radio DBus observation must not start services or prompt")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return hostreqkit.RunPrivilegedCommandWithOutput("error", name, args, func(command string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
		cmd.WaitDelay = time.Second
		var output radioCommandOutput
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if output.exceeded {
			return nil, errors.New("radio command output exceeds bound")
		}
		return output.Bytes(), err
	})
}

type radioCommandOutput struct {
	bytes.Buffer
	exceeded bool
}

func (b *radioCommandOutput) Write(data []byte) (int, error) {
	if len(data) > 64<<10-b.Len() {
		b.exceeded = true
		return 0, errors.New("radio output exceeds bound")
	}
	return b.Buffer.Write(data)
}

func (t *durableWireless) readRadioRecord() (wirelessRadioRecord, error) {
	var r wirelessRadioRecord
	data, err := t.readState(t.statePath("radio.json"), 128<<10)
	if err != nil {
		return r, err
	}
	if err = strictWirelessJSON(data, &r); err != nil {
		return r, err
	}
	if r.Version != 1 || r.Transaction != t.intent.ID || r.IntentSHA256 != t.intentSHA || t.intent.Context == nil || r.Device != t.intent.Context.Device || !slices.Contains([]string{radioPhaseUnloading, radioPhaseLoading, radioPhaseObserved, radioPhaseRestoring, radioPhaseRestored}, r.Phase) {
		return r, errors.New("radio journal lineage or phase differs")
	}
	if !recoveryUUID.MatchString(r.BootID) {
		return r, errors.New("invalid radio activation boot identity")
	}
	cohort := radioModuleNames(t.intent.Plan.Profile)
	if len(r.Before) != len(cohort) || len(r.PreservedModules) > 4096 || len(r.Dependencies) > 128 {
		return r, errors.New("radio journal module closure differs")
	}
	for i, module := range r.Before {
		if module.Name != cohort[i] || len(module.Parameters) > 64 {
			return r, errors.New("radio journal module identity differs")
		}
	}
	for _, module := range append(slices.Clone(r.Dependencies), r.PreservedModules...) {
		if !nativeName.MatchString(module) {
			return r, errors.New("invalid radio dependency identity")
		}
	}
	return r, nil
}

func (t *durableWireless) saveRadioRecord(r wirelessRadioRecord) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(data) > 128<<10 {
		return errors.New("radio journal exceeds bound")
	}
	return t.writeState(t.statePath("radio.json"), data)
}
func wirelessRadioPending(t *durableWireless) bool {
	r, err := t.readRadioRecord()
	return err != nil && !errors.Is(err, fs.ErrNotExist) || err == nil && r.Phase != radioPhaseRestored
}

func plannedRadioDependencies(ctx context.Context, t *durableWireless) ([]string, error) {
	out, err := t.radioSystem().command(ctx, "/usr/sbin/modprobe", "-C", "/dev/null", "-S", nobleABI, "--show-depends", "--ignore-install", "--", radioLeaf(t.intent.Plan.Profile))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 || fields[0] != "insmod" || !strings.HasPrefix(fields[1], "/lib/modules/"+nobleABI+"/") || strings.Contains(fields[1], "..") {
			return nil, errors.New("unsupported or overridden radio dependency command")
		}
		base := path.Base(fields[1])
		for _, suffix := range []string{".zst", ".xz", ".gz"} {
			base = strings.TrimSuffix(base, suffix)
		}
		if !strings.HasSuffix(base, ".ko") {
			return nil, errors.New("radio dependency is not a module")
		}
		name := radioModuleName(base)
		if name == "atlantic" || name == "nvidia" || strings.HasPrefix(name, "nvidia_") {
			return nil, errors.New("radio dependency overlaps a protected provider")
		}
		names = append(names, name)
	}
	if len(names) == 0 || len(names) > 128 {
		return nil, errors.New("radio dependency closure unavailable")
	}
	return names, nil
}

func unloadRadioCohort(ctx context.Context, t *durableWireless) error {
	if err := assertWirelessConsumers(ctx, t); err != nil {
		return err
	}
	names := radioModuleNames(t.intent.Plan.Profile)
	for i := len(names) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		loaded, candidate, stock, err := t.radioSystem().identity(ctx, t, names[i])
		if err != nil {
			return err
		}
		if !loaded {
			continue
		}
		if !candidate && !stock {
			return errors.New("refusing to unload an unreviewed radio module")
		}
		// rmmod names exactly one module and never recursively unloads stock
		// dependencies. No force/wait flags or unrelated provider names exist.
		if err = t.radioSystem().unload(ctx, names[i]); err != nil {
			return fmt.Errorf("radio module %s remains busy: %w", names[i], err)
		}
	}
	return nil
}

func activateWirelessRadio(ctx context.Context, t *durableWireless) (resultErr error) {
	if t.readOnly {
		return errors.New("radio preview cannot activate")
	}
	if t.phase != "published" || t.intent.Context == nil {
		return errors.New("complete admitted publication required")
	}
	if err := t.validate(ctx); err != nil {
		return err
	}
	record, err := t.readRadioRecord()
	if err == nil {
		boot, e := radioBootID(t)
		if e != nil || boot != record.BootID {
			return errors.New("radio journal belongs to a different boot; use verified fallback recovery")
		}
	}
	if err == nil && record.Phase == radioPhaseObserved {
		observation, e := observeWirelessRadio(ctx, t)
		if e == nil && observation.CandidateModulesVerified && observation.BindingObserved {
			return nil
		}
		return errors.New("previous activation drifted; recover before retry")
	}
	if err == nil && record.Phase == radioPhaseRestored {
		return errors.New("rolled-back radio lineage needs a fresh stage")
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if errors.Is(err, fs.ErrNotExist) {
		boot, e := radioBootID(t)
		if e != nil {
			return e
		}
		if boot != t.intent.Context.BootID {
			return errors.New("boot changed since stage; restore stock on the independent kernel and stage a fresh qualification")
		}
		before, e := moduleSnapshots(t.root, t.intent.Plan.Profile)
		if e != nil {
			return e
		}
		for _, module := range before {
			loaded, _, stock, e := t.radioSystem().identity(ctx, t, module.Name)
			if e != nil {
				return e
			}
			if loaded && !stock {
				return errors.New("candidate or unknown code loaded before the owned activation; use explicit rollback")
			}
		}
		afterBoot, e := radioBootID(t)
		if e != nil || afterBoot != boot {
			return errors.New("boot changed during activation snapshot")
		}
		preserved, e := loadedModuleNames(t.root)
		if e != nil {
			return e
		}
		deps, e := plannedRadioDependencies(ctx, t)
		if e != nil {
			return e
		}
		record = wirelessRadioRecord{Version: 1, Transaction: t.intent.ID, IntentSHA256: t.intentSHA, Device: t.intent.Context.Device, Phase: radioPhaseUnloading, Before: before, PreservedModules: preserved, Dependencies: deps, BootID: boot}
		if err = t.saveRadioRecord(record); err != nil {
			return err
		}
	}
	boot, e := radioBootID(t)
	if e != nil || boot != record.BootID {
		return errors.New("activation journal belongs to a different boot; use independent-kernel recovery")
	}
	if record.Phase == radioPhaseRestoring {
		return errors.New("interrupted radio rollback must finish before activation")
	}
	defer func() {
		if resultErr != nil {
			cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			resultErr = errors.Join(resultErr, rollbackWirelessRadio(cleanup, t))
		}
	}()
	if err = unloadRadioCohort(ctx, t); err != nil {
		return err
	}
	record.Phase = radioPhaseLoading
	if err = t.saveRadioRecord(record); err != nil {
		return err
	}
	if _, err = t.radioSystem().command(ctx, "/usr/sbin/modprobe", "-C", "/dev/null", "-S", nobleABI, "--ignore-install", "--", radioLeaf(t.intent.Plan.Profile)); err != nil {
		return err
	}
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		observation, e := observeWirelessRadio(ctx, t)
		if e != nil {
			return e
		}
		if observation.CandidateModulesVerified && observation.BindingObserved && (len(observation.Interfaces) > 0 || len(observation.Controllers) > 0) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("candidate interface/controller did not initialize within the bounded qualification window")
		case <-ticker.C:
		}
	}
	if err = t.verifyProtected(ctx); err != nil {
		return err
	}
	record.Phase = radioPhaseObserved
	return t.saveRadioRecord(record)
}

// Current wired/GPU/fallback proof is verified by the caller. This path only
// restores exact files for the non-running target kernel and never loads or
// unloads any module of the independently running fallback.
func recoverWirelessFilesOnFallback(ctx context.Context, t *durableWireless) error {
	if !t.independentRecovery || t.readOnly {
		return errors.New("independently verified writable recovery context required")
	}
	record, err := t.readRadioRecord()
	present := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	t.deferParent = true
	if err = t.rollback(ctx); err != nil {
		return err
	}
	if present {
		boot, e := radioBootID(t)
		if e != nil {
			return e
		}
		record.Phase = radioPhaseRestored
		record.RestoreKind = "independent_kernel_files_only"
		record.RecoveryBootID = boot
		if err = t.saveRadioRecord(record); err != nil {
			return err
		}
	}
	return t.completeParentLineage()
}

func rollbackWirelessRadio(ctx context.Context, t *durableWireless) error {
	if t.readOnly {
		return errors.New("radio preview cannot roll back")
	}
	record, err := t.readRadioRecord()
	if errors.Is(err, fs.ErrNotExist) {
		observation, e := observeWirelessRadio(ctx, t)
		if e != nil {
			return e
		}
		if len(observation.CandidateModulesLoaded) == 0 {
			return t.rollback(ctx)
		}
		boot, e := radioBootID(t)
		if e != nil {
			return e
		}
		if boot != t.intent.Context.BootID {
			return errors.New("candidate was loaded on another boot; use independent-kernel file recovery")
		}
		preserved, e := loadedModuleNames(t.root)
		if e != nil {
			return e
		}
		dependencies, e := plannedRadioDependencies(ctx, t)
		if e != nil {
			return e
		}
		record = wirelessRadioRecord{Version: 1, Transaction: t.intent.ID, IntentSHA256: t.intentSHA, Device: t.intent.Context.Device, Phase: radioPhaseRestoring, Before: t.intent.Context.StageModules, PreservedModules: preserved, Dependencies: dependencies, BootID: boot, RestoreKind: "stage_stock_fallback"}
		if err = t.saveRadioRecord(record); err != nil {
			return err
		}
		err = nil
	}
	if err != nil {
		return err
	}
	if record.Phase == radioPhaseRestored {
		return t.completeParentLineage()
	}
	boot, e := radioBootID(t)
	if e != nil || boot != record.BootID {
		return errors.New("runtime recovery journal belongs to a different boot; use independent-kernel file recovery")
	}
	record.Phase = radioPhaseRestoring
	if err = t.saveRadioRecord(record); err != nil {
		return err
	}
	if err = unloadRadioCohort(ctx, t); err != nil {
		return err
	}
	t.deferParent = true
	if err = t.rollback(ctx); err != nil {
		return err
	}
	for _, module := range record.Before {
		if !module.Loaded {
			continue
		}
		args := []string{"-C", "/dev/null", "-S", nobleABI, "--ignore-install", "--", module.Name}
		var keys []string
		for name := range module.Parameters {
			if !nativeName.MatchString(name) {
				return errors.New("invalid retained radio parameter")
			}
			keys = append(keys, name)
		}
		slices.Sort(keys)
		for _, key := range keys {
			value := module.Parameters[key]
			if len(value) > 512 || strings.ContainsAny(value, "\x00\r\n") {
				return errors.New("invalid retained radio parameter value")
			}
			args = append(args, key+"="+value)
		}
		if _, err = t.radioSystem().command(ctx, "/usr/sbin/modprobe", args...); err != nil {
			return err
		}
	}
	// Only newly loaded members of the predicted dependency closure can be
	// removed. Existing dependencies and concurrent unrelated modules survive.
	for i := len(record.Dependencies) - 1; i >= 0; i-- {
		name := record.Dependencies[i]
		if slices.Contains(record.PreservedModules, name) || slices.Contains(radioModuleNames(t.intent.Plan.Profile), name) {
			continue
		}
		if _, err = t.root.Stat("sys/module/" + name); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err = t.radioSystem().unload(ctx, name); err != nil {
			return fmt.Errorf("new dependency %s retained while busy: %w", name, err)
		}
	}
	for _, module := range record.Before {
		loaded, _, stock, e := t.radioSystem().identity(ctx, t, module.Name)
		if e != nil {
			return e
		}
		if module.Loaded && (!loaded || !stock) || !module.Loaded && loaded {
			return errors.New("original runtime module population not restored")
		}
	}
	if err = t.verifyProtected(ctx); err != nil {
		return err
	}
	record.Phase = radioPhaseRestored
	if record.RestoreKind == "" {
		record.RestoreKind = "runtime"
	}
	if err = t.saveRadioRecord(record); err != nil {
		return err
	}
	return t.completeParentLineage()
}

func radioBootID(t *durableWireless) (string, error) {
	data, err := radioRead(t.root, "proc/sys/kernel/random/boot_id", 128)
	if err != nil {
		return "", err
	}
	boot := strings.TrimSpace(string(data))
	if !recoveryUUID.MatchString(boot) {
		return "", errors.New("invalid running boot identity")
	}
	return boot, nil
}

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/netip"
	"os"
	"runtime"
	"slices"
	"strings"

	platform "github.com/vrooli/platform-go"
)

func verifyOpenRGBConsumedClaim(f managedUpdateExecFence, intent ManagedUpdateLaunchIntent, contract ManagedUpdateContract, ref platform.ServiceRef, incarnation string) (openRGBExecWitness, error) {
	if f.State != managedUpdateExecFenceClaimed {
		return openRGBExecWitness{}, errors.New("OpenRGB serving launch has no active consumed claim")
	}
	return verifyOpenRGBRetainedClaim(f, intent, contract, ref, incarnation)
}

func verifyOpenRGBRetainedClaim(f managedUpdateExecFence, intent ManagedUpdateLaunchIntent, contract ManagedUpdateContract, ref platform.ServiceRef, incarnation string) (openRGBExecWitness, error) {
	if f.Resource != openRGBResourceName || f.Intent != intent || f.Claim == nil || !f.Claim.Equal(ref) || f.OpenRGBWitness == nil || f.validate() != nil {
		return openRGBExecWitness{}, errors.New("OpenRGB serving launch has no consumed device witness for its exact native owner")
	}
	w := *f.OpenRGBWitness
	if w.Incarnation != incarnation || w.Policy.ArtifactSHA256 != contract.Artifact.SHA256 || w.Policy.ConfigSHA256 != contract.ConfigSHA256 || intent.ContractDigest != contract.Digest {
		return openRGBExecWitness{}, errors.New("OpenRGB serving launch differs from the witnessed incarnation, artifact or settings")
	}
	return w, nil
}

type openRGBServingObservation struct {
	ref      platform.ServiceRef
	witness  openRGBExecWitness
	endpoint netip.AddrPort
	base     managedUpdateInvocation
}

func newOpenRGBServingObservation(f managedUpdateExecFence, intent ManagedUpdateLaunchIntent, contract ManagedUpdateContract, invocation managedUpdateInvocation, ref platform.ServiceRef, incarnation string) (openRGBServingObservation, error) {
	witness, err := verifyOpenRGBConsumedClaim(f, intent, contract, ref, incarnation)
	if err != nil {
		return openRGBServingObservation{}, err
	}
	if err := verifyOpenRGBExecInvocation(invocation, intent.Runtime.Port); err != nil {
		return openRGBServingObservation{}, err
	}
	if invocation.Path != contract.Artifact.Path || invocation.ConfigContent != string(contract.ConfigBytes) {
		return openRGBServingObservation{}, errors.New("OpenRGB serving invocation differs from the witnessed envelope and frozen configuration")
	}
	address, err := netip.ParseAddr(invocation.Args[2])
	if err != nil || address.String() != "127.0.0.1" {
		return openRGBServingObservation{}, errors.New("OpenRGB native listener observation supports the release's literal IPv4 loopback only")
	}
	return openRGBServingObservation{ref: ref, witness: witness, endpoint: netip.AddrPortFrom(address, uint16(intent.Runtime.Port)), base: invocation}, nil
}

func (o openRGBServingObservation) verify(ctx context.Context, imageSHA string, observeContext func(platform.ServiceRef, string) (string, error), observeListener func(netip.AddrPort, int) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if imageSHA != o.witness.Policy.ArtifactSHA256 {
		return errors.New("OpenRGB serving executable is not the witnessed direct release image")
	}
	return o.verifyNativeContextAndListener(observeContext, observeListener)
}

func (o openRGBServingObservation) verifyRoleImage(ctx context.Context, stage *openRGBAppImageStage, image os.FileInfo, digest string) error {
	if stage != nil {
		return o.observeAppImageStage(ctx, *stage, image, digest)
	}
	return o.verify(ctx, digest, openRGBServingContext, observeOpenRGBHeadlessListener)
}

type openRGBAppImageServingProcess struct {
	image       os.FileInfo
	imageSHA256 string
	args, env   []string
	dir         string
	config      []byte
}

func (o openRGBServingObservation) verifyAppImageStage(ctx context.Context, stage openRGBAppImageStage, live openRGBAppImageServingProcess, observeContext func(platform.ServiceRef, string) (string, error), observeListener func(netip.AddrPort, int) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if stage.envelope.SHA256 != o.witness.Policy.ArtifactSHA256 || stage.payload == nil || live.image == nil || live.imageSHA256 != stage.payloadSHA256 || !os.SameFile(stage.payload, live.image) {
		return errors.New("OpenRGB process is not executing the exact sealed payload of the witnessed envelope")
	}
	want, err := stage.servingInvocation(o.base, int(o.endpoint.Port()))
	if err != nil {
		return err
	}
	if !sameOpenRGBDirectory(live.dir, want.Dir) || !slices.Equal(live.args, append([]string{want.Path}, want.Args...)) || !sameOpenRGBEnvironment(live.env, want.Env) {
		return errors.New("OpenRGB payload arguments, environment or working directory differ from the admitted staged invocation")
	}
	if err := verifyOpenRGBServingDetectors([]byte(want.ConfigContent), live.config); err != nil {
		return err
	}
	return o.verifyNativeContextAndListener(observeContext, observeListener)
}

func verifyOpenRGBServingDetectors(frozen, live []byte) error {
	want, err := readOpenRGBDetectorState(frozen)
	if err != nil {
		return err
	}
	actual, err := readOpenRGBDetectorState(live)
	if err != nil {
		return err
	}
	return verifyOpenRGBDetectorSubset(want, actual)
}

func verifyOpenRGBDetectorSubset(want, actual map[string]bool) error {
	// OpenRGB prunes unregistered detector keys on load. It may omit disabled
	// entries while preserving the full selected set and adding no detectors.
	for name, enabled := range actual {
		if expected, present := want[name]; !present || expected != enabled {
			return fmt.Errorf("OpenRGB serving detector %q differs from its witnessed selection", name)
		}
	}
	for name, enabled := range want {
		if enabled && !actual[name] {
			return fmt.Errorf("OpenRGB serving configuration lost selected detector %q", name)
		}
	}
	return nil
}

func (o openRGBServingObservation) verifyNativeContextAndListener(observeContext func(platform.ServiceRef, string) (string, error), observeListener func(netip.AddrPort, int) error) error {
	proof, err := observeContext(o.ref, o.witness.Incarnation)
	if err != nil || proof != o.witness.ContextSHA256 {
		return errors.Join(errors.New("OpenRGB serving executable left the witnessed security context"), err)
	}
	pid, err := o.servingPID()
	if err != nil {
		return err
	}
	if err := observeListener(o.endpoint, pid); err != nil {
		return fmt.Errorf("observe witnessed OpenRGB listener: %w", err)
	}
	return nil
}

// The consumed witness keeps the original frozen byte digest. Only the
// qualified runtime's disabled-key pruning and JSON formatting may normalize
// on disk; critical scope is checked before reconstructing those keys.
func verifyOpenRGBNormalizedServingSettings(frozen, live []byte) error {
	want, err := readOpenRGBDetectorState(frozen)
	if err != nil {
		return err
	}
	actual, err := readOpenRGBDetectorState(live)
	if err != nil {
		return err
	}
	if err := verifyOpenRGBDetectorSubset(want, actual); err != nil {
		return err
	}
	var root, detectors map[string]json.RawMessage
	if err := json.Unmarshal(live, &root); err != nil {
		return err
	}
	if err := json.Unmarshal(root["Detectors"], &detectors); err != nil {
		return err
	}
	for key, enabled := range want {
		if !enabled {
			if _, present := actual[key]; !present {
				actual[key] = false
			}
		}
	}
	detectors["detectors"], err = json.Marshal(actual)
	if err != nil {
		return err
	}
	root["Detectors"], err = json.Marshal(detectors)
	if err != nil {
		return err
	}
	normalized, err := marshalOpenRGBSettings(root)
	if err != nil {
		return err
	}
	var original map[string]json.RawMessage
	if err := json.Unmarshal(frozen, &original); err != nil {
		return err
	}
	canonical, err := marshalOpenRGBSettings(original)
	if err != nil {
		return err
	}
	if string(normalized) != string(canonical) {
		return errors.New("OpenRGB serving settings changed beyond qualified runtime normalization")
	}
	return nil
}

func sameOpenRGBEnvironment(a, b []string) bool {
	decode := func(entries []string) (map[string]string, bool) {
		result := map[string]string{}
		for _, entry := range entries {
			key, value, ok := strings.Cut(entry, "=")
			if !ok || key == "" {
				return nil, false
			}
			if runtime.GOOS == "windows" {
				key = strings.ToUpper(key)
			}
			if _, duplicate := result[key]; duplicate {
				return nil, false
			}
			result[key] = value
		}
		return result, true
	}
	left, ok := decode(a)
	if !ok {
		return false
	}
	right, ok := decode(b)
	return ok && maps.Equal(left, right)
}
func sameOpenRGBDirectory(a, b string) bool {
	if a == b {
		return true
	}
	if runtime.GOOS != "windows" {
		return false
	}
	x, e := os.Stat(a)
	if e != nil || !x.IsDir() {
		return false
	}
	y, e := os.Stat(b)
	return e == nil && y.IsDir() && os.SameFile(x, y)
}

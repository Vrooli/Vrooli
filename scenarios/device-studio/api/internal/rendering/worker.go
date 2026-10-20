// Package rendering implements the private, data-only still worker. It owns no
// job database, media lookup, process lifecycle, or retention policy.
package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"device-studio/internal/composition"
	"device-studio/internal/sourceimage"
)

const (
	ManifestName       = "render-manifest.json"
	OutputName         = "output.png"
	ReceiptName        = "receipt.json"
	FailureName        = "worker-error.json"
	CompositorProfile  = composition.StillProfile
	manifestVersion    = 1
	maxManifestBytes   = 64 << 10
	maxImageBytes      = composition.StillMaxImageBytes
	maxTotalImageBytes = composition.StillMaxInputBytes
	maxImageSide       = composition.StillMaxDimension
	maxImagePixels     = composition.StillMaxPixels
	maxOutputBytes     = 160 << 20
	maxReceiptBytes    = 64 << 10
	maxDiagnosticBytes = 2 << 10
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Manifest struct {
	Version          int           `json:"version"`
	JobID            string        `json:"job_id"`
	AttemptID        string        `json:"attempt_id"`
	InputFingerprint string        `json:"input_fingerprint"`
	Profile          string        `json:"profile"`
	Plate            ImageBinding  `json:"plate"`
	Screen           ImageBinding  `json:"screen"`
	ActiveMask       ImageBinding  `json:"active_mask"`
	HardwareMask     *ImageBinding `json:"hardware_mask,omitempty"`
	SurfaceWidth     int           `json:"surface_width"`
	SurfaceHeight    int           `json:"surface_height"`
	Quad             [4]Corner     `json:"quad"`
	Samples          int           `json:"samples_per_axis"`
}

type ImageBinding struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type Corner struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type InputReceipt struct {
	Role      string `json:"role"`
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type Receipt struct {
	Version          int                 `json:"version"`
	JobID            string              `json:"job_id"`
	AttemptID        string              `json:"attempt_id"`
	InputFingerprint string              `json:"input_fingerprint"`
	ManifestSHA256   string              `json:"manifest_sha256"`
	Profile          string              `json:"profile"`
	Algorithm        string              `json:"algorithm"`
	Inputs           []InputReceipt      `json:"inputs"`
	OutputName       string              `json:"output_name"`
	OutputSHA256     string              `json:"output_sha256"`
	OutputSizeBytes  int64               `json:"output_size_bytes"`
	OutputWidth      int                 `json:"output_width"`
	OutputHeight     int                 `json:"output_height"`
	Composition      composition.Receipt `json:"composition"`
	CompletedAt      time.Time           `json:"completed_at"`
}

type failureDiagnostic struct {
	Version   int       `json:"version"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	JobID     string    `json:"job_id,omitempty"`
	AttemptID string    `json:"attempt_id,omitempty"`
	FailedAt  time.Time `json:"failed_at"`
}

type loadedImage struct {
	role    string
	binding ImageBinding
	data    []byte
	image   image.Image
	format  string
}

// IsWorkerInvocation selects only the reserved worker subcommand. Its parser
// rejects every unrecognized flag before service preflight can run.
func IsWorkerInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "render-worker"
}

// RunWorkerCommand runs the exact private CLI form:
// render-worker --manifest /absolute/attempt/render-manifest.json
func RunWorkerCommand(args []string) error {
	if len(args) != 3 || args[0] != "render-worker" || args[1] != "--manifest" {
		return errors.New("render-worker requires exactly --manifest <absolute-manifest-path>")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunManifest(ctx, args[2])
}

// RunManifest validates and executes one immutable shot input closure in its
// attempt workspace, then atomically publishes a PNG and identity-bound receipt.
func RunManifest(ctx context.Context, manifestPath string) (retErr error) {
	if ctx == nil {
		return errors.New("render worker context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || filepath.Base(manifestPath) != ManifestName {
		return errors.New("render worker manifest must be the fixed filename in an absolute attempt workspace")
	}
	workspace := filepath.Dir(manifestPath)
	if err := validateWorkspacePath(workspace); err != nil {
		return err
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return fmt.Errorf("open attempt workspace: %w", err)
	}
	defer root.Close()

	var manifest Manifest
	var manifestBytes []byte
	var manifestSHA string
	var inputs []loadedImage
	defer func() {
		if retErr != nil {
			if diagnosticErr := writeFailure(root, manifest, retErr); diagnosticErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("write bounded worker diagnostic: %w", diagnosticErr))
			}
		}
	}()
	manifestBytes, err = readBoundedRegular(ctx, root, ManifestName, maxManifestBytes, 0)
	if err != nil {
		return fmt.Errorf("read worker manifest: %w", err)
	}
	manifestSHA = digest(manifestBytes)
	if err := decodeCanonicalManifest(manifestBytes, &manifest); err != nil {
		return err
	}
	if err := validateManifest(manifest); err != nil {
		return err
	}
	if err := requireNoPublishedOutputs(root); err != nil {
		return err
	}

	roles := []string{"plate", "screen", "active_mask"}
	bindings := []ImageBinding{manifest.Plate, manifest.Screen, manifest.ActiveMask}
	if manifest.HardwareMask != nil {
		roles = append(roles, "hardware_mask")
		bindings = append(bindings, *manifest.HardwareMask)
	}
	inputs = make([]loadedImage, 0, len(bindings))
	var totalBytes int64
	for _, binding := range bindings {
		if err := ctx.Err(); err != nil {
			return err
		}
		data, err := readBoundedRegular(ctx, root, binding.Name, maxImageBytes, binding.SizeBytes)
		if err != nil {
			return fmt.Errorf("read input %q: %w", binding.Name, err)
		}
		totalBytes += int64(len(data))
		if totalBytes > maxTotalImageBytes {
			return errors.New("render worker input bytes exceed aggregate limit")
		}
		if digest(data) != binding.SHA256 {
			return fmt.Errorf("input %q digest does not match manifest", binding.Name)
		}
		decoded, format, err := decodeImage(data)
		if err != nil {
			return fmt.Errorf("decode input %q: %w", binding.Name, err)
		}
		if err := validateInputMetadata(data, format); err != nil {
			return fmt.Errorf("input %q metadata: %w", binding.Name, err)
		}
		if binding.Name == manifest.ActiveMask.Name && format != "png" {
			return errors.New("active mask must use PNG grayscale or alpha coverage")
		}
		if manifest.HardwareMask != nil && binding.Name == manifest.HardwareMask.Name && format != "png" {
			return errors.New("hardware mask must use PNG grayscale or alpha coverage")
		}
		inputs = append(inputs, loadedImage{role: roles[len(inputs)], binding: binding, data: data, image: decoded, format: format})
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	plate, screen, active := inputs[0].image, inputs[1].image, inputs[2].image
	var hardware image.Image
	if len(inputs) == 4 {
		hardware = inputs[3].image
	}
	var quad [4]composition.Point
	for i, point := range manifest.Quad {
		quad[i] = composition.Point{X: point.X, Y: point.Y}
	}
	composed, err := composition.Compose(ctx, composition.Inputs{
		Plate: plate, Screen: screen, ActiveMask: active, HardwareMask: hardware,
		SurfaceSize: image.Pt(manifest.SurfaceWidth, manifest.SurfaceHeight), Quad: quad, Samples: manifest.Samples,
	})
	if err != nil {
		return fmt.Errorf("compose still: %w", err)
	}
	var output bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(contextWriter{ctx: ctx, dst: &output}, composed.Image); err != nil {
		return fmt.Errorf("encode output PNG: %w", err)
	}
	if output.Len() == 0 || output.Len() > maxOutputBytes {
		return errors.New("encoded output PNG is empty or exceeds the output limit")
	}
	decodedOutput, err := png.Decode(contextReader{ctx: ctx, src: bytes.NewReader(output.Bytes())})
	if err != nil {
		return fmt.Errorf("verify encoded output PNG: %w", err)
	}
	if decodedOutput.Bounds().Dx() != composed.Image.Bounds().Dx() || decodedOutput.Bounds().Dy() != composed.Image.Bounds().Dy() {
		return errors.New("encoded output dimensions do not match composition")
	}
	// The parent stages immutable files, but re-read and re-hash the manifest and
	// each input immediately before publication to detect mutation in the attempt.
	if err := verifyUnchangedInputs(ctx, root, manifestBytes, manifestSHA, inputs); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	outputBytes := output.Bytes()
	outputSHA := digest(outputBytes)
	if err := publishExclusive(ctx, root, OutputName, outputBytes, maxOutputBytes); err != nil {
		return fmt.Errorf("publish output PNG: %w", err)
	}
	receipt := Receipt{
		Version: manifestVersion, JobID: manifest.JobID, AttemptID: manifest.AttemptID,
		InputFingerprint: manifest.InputFingerprint, ManifestSHA256: manifestSHA, Profile: manifest.Profile,
		Algorithm: composed.Receipt.Algorithm, Inputs: inputReceipts(inputs),
		OutputName: OutputName, OutputSHA256: outputSHA, OutputSizeBytes: int64(len(outputBytes)),
		OutputWidth: composed.Image.Bounds().Dx(), OutputHeight: composed.Image.Bounds().Dy(),
		Composition: composed.Receipt, CompletedAt: time.Now().UTC(),
	}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("encode worker receipt: %w", err)
	}
	receiptBytes = append(receiptBytes, '\n')
	if len(receiptBytes) > maxReceiptBytes {
		return errors.New("worker receipt exceeds size limit")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := publishExclusive(ctx, root, ReceiptName, receiptBytes, maxReceiptBytes); err != nil {
		return fmt.Errorf("publish worker receipt: %w", err)
	}
	return nil
}

func validateManifest(m Manifest) error {
	if m.Version != manifestVersion {
		return fmt.Errorf("unsupported worker manifest version %d", m.Version)
	}
	if !identifierPattern.MatchString(m.JobID) || !identifierPattern.MatchString(m.AttemptID) {
		return errors.New("worker manifest job and attempt IDs are invalid")
	}
	if !digestPattern.MatchString(m.InputFingerprint) {
		return errors.New("worker manifest input fingerprint is invalid")
	}
	if m.Profile != CompositorProfile {
		return errors.New("worker manifest compositor profile is unsupported")
	}
	if m.SurfaceWidth <= 0 || m.SurfaceHeight <= 0 || m.SurfaceWidth > maxImageSide || m.SurfaceHeight > maxImageSide || int64(m.SurfaceWidth)*int64(m.SurfaceHeight) > maxImagePixels {
		return errors.New("worker surface dimensions exceed limits")
	}
	if m.Samples < 0 || m.Samples > 4 {
		return errors.New("worker samples per axis must be in [0,4]")
	}
	bindings := []ImageBinding{m.Plate, m.Screen, m.ActiveMask}
	if m.HardwareMask != nil {
		bindings = append(bindings, *m.HardwareMask)
	}
	seen := make(map[string]struct{}, len(bindings))
	var total int64
	for _, binding := range bindings {
		if err := validateBinding(binding); err != nil {
			return err
		}
		if _, ok := seen[binding.Name]; ok {
			return fmt.Errorf("worker manifest reuses input name %q", binding.Name)
		}
		seen[binding.Name] = struct{}{}
		total += binding.SizeBytes
		if total > maxTotalImageBytes {
			return errors.New("worker manifest input sizes exceed aggregate limit")
		}
	}
	for i, corner := range m.Quad {
		if math.IsNaN(corner.X) || math.IsInf(corner.X, 0) || math.IsNaN(corner.Y) || math.IsInf(corner.Y, 0) {
			return fmt.Errorf("worker quad corner %d is not finite", i)
		}
	}
	return nil
}

func validateBinding(b ImageBinding) error {
	if b.Name == "" || len(b.Name) > 128 || filepath.IsAbs(b.Name) || filepath.Clean(b.Name) != b.Name || b.Name == "." || b.Name == ".." || strings.ContainsAny(b.Name, "/\\:\x00") {
		return fmt.Errorf("worker input name %q must be a relative single path component", b.Name)
	}
	if b.Name == ManifestName || b.Name == OutputName || b.Name == ReceiptName || b.Name == FailureName || strings.HasSuffix(b.Name, ".tmp") {
		return fmt.Errorf("worker input name %q is reserved", b.Name)
	}
	if !digestPattern.MatchString(b.SHA256) || b.SizeBytes <= 0 || b.SizeBytes > maxImageBytes {
		return fmt.Errorf("worker input %q has invalid digest or byte size", b.Name)
	}
	return nil
}

func decodeCanonicalManifest(data []byte, out *Manifest) error {
	if len(data) == 0 || len(data) > maxManifestBytes {
		return errors.New("worker manifest is empty or exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode worker manifest: %w", err)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("worker manifest contains trailing JSON data")
	}
	canonical, err := json.Marshal(*out)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, canonical) && !bytes.Equal(data, append(canonical, '\n')) {
		return errors.New("worker manifest is not canonical JSON")
	}
	return nil
}

func validateWorkspacePath(workspace string) error {
	if !filepath.IsAbs(workspace) || filepath.Clean(workspace) != workspace {
		return errors.New("attempt workspace path must be absolute and clean")
	}
	volume := filepath.VolumeName(workspace)
	remaining := strings.TrimPrefix(workspace, volume)
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(remaining, string(filepath.Separator)), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect attempt workspace path: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("attempt workspace path contains a symlink")
		}
		if !info.IsDir() && current != workspace {
			return errors.New("attempt workspace path has a non-directory parent")
		}
	}
	info, err := os.Lstat(workspace)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("attempt workspace is not a regular directory")
	}
	return nil
}

func readBoundedRegular(ctx context.Context, root *os.Root, name string, maximum, expectedSize int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("workspace input must be a regular non-symlink file")
	}
	if info.Size() <= 0 || info.Size() > maximum || (expectedSize > 0 && info.Size() != expectedSize) {
		return nil, errors.New("workspace input has an invalid byte size")
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("workspace input changed while opening")
	}
	data, err := readContextBounded(ctx, f, maximum)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != opened.Size() || int64(len(data)) > maximum {
		return nil, errors.New("workspace input changed or exceeded its byte limit")
	}
	return data, nil
}

func decodeImage(data []byte) (image.Image, string, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	if format != "png" && format != "jpeg" {
		return nil, "", fmt.Errorf("unsupported image format %q", format)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxImageSide || config.Height > maxImageSide || int64(config.Width)*int64(config.Height) > maxImagePixels {
		return nil, "", errors.New("decoded image dimensions exceed limits")
	}
	decoded, fullFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	if fullFormat != format || decoded.Bounds().Dx() != config.Width || decoded.Bounds().Dy() != config.Height {
		return nil, "", errors.New("image changed between configuration and full decode")
	}
	return decoded, format, nil
}

func verifyUnchangedInputs(ctx context.Context, root *os.Root, manifest []byte, manifestHash string, inputs []loadedImage) error {
	currentManifest, err := readBoundedRegular(ctx, root, ManifestName, maxManifestBytes, int64(len(manifest)))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("reread worker manifest: %w", err)
	}
	if digest(currentManifest) != manifestHash || !bytes.Equal(currentManifest, manifest) {
		return errors.New("worker manifest changed during render")
	}
	for _, input := range inputs {
		current, err := readBoundedRegular(ctx, root, input.binding.Name, maxImageBytes, input.binding.SizeBytes)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("reread input %q: %w", input.binding.Name, err)
		}
		if digest(current) != input.binding.SHA256 || !bytes.Equal(current, input.data) {
			return fmt.Errorf("worker input %q changed during render", input.binding.Name)
		}
	}
	return nil
}

func requireNoPublishedOutputs(root *os.Root) error {
	for _, name := range []string{OutputName, ReceiptName, FailureName} {
		if _, err := root.Lstat(name); err == nil {
			return fmt.Errorf("worker output %q already exists", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect worker output %q: %w", name, err)
		}
	}
	return nil
}

func publishExclusive(ctx context.Context, root *os.Root, name string, data []byte, limit int64) error {
	if int64(len(data)) == 0 || int64(len(data)) > limit {
		return errors.New("worker publication has invalid byte size")
	}
	temporary := name + ".tmp"
	f, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = root.Remove(temporary)
		}
	}()
	if err := writeContext(ctx, f, data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := root.Lstat(name); err == nil {
		return fmt.Errorf("worker output %q already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := root.Link(temporary, name); err != nil {
		return err
	}
	if err := root.Remove(temporary); err != nil {
		return err
	}
	removeTemporary = false
	return nil
}

func writeFailure(root *os.Root, m Manifest, cause error) error {
	message := strings.ReplaceAll(cause.Error(), root.Name(), "<attempt-workspace>")
	message = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || r >= 0x20 {
			return r
		}
		return -1
	}, message)
	message = truncateUTF8(message, 768)
	diagnostic := failureDiagnostic{Version: manifestVersion, Code: "render_failed", Message: message, JobID: m.JobID, AttemptID: m.AttemptID, FailedAt: time.Now().UTC()}
	var data []byte
	var err error
	for limit := len(message); ; limit /= 2 {
		diagnostic.Message = truncateUTF8(message, limit)
		data, err = json.Marshal(diagnostic)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if len(data) <= maxDiagnosticBytes {
			break
		}
		if limit == 0 {
			return errors.New("failure diagnostic envelope exceeds limit")
		}
	}
	return publishExclusive(context.Background(), root, FailureName, data, maxDiagnosticBytes)
}

func validateInputMetadata(data []byte, format string) error {
	metadata, err := sourceimage.Inspect(data, format)
	if err != nil {
		return err
	}
	if metadata.Orientation != 1 {
		return errors.New("non-identity EXIF orientation is unsupported by the still worker")
	}
	if metadata.HasProfile {
		return errors.New("embedded color profiles are unsupported by the still worker")
	}
	return nil
}

func inputReceipts(inputs []loadedImage) []InputReceipt {
	receipts := make([]InputReceipt, 0, len(inputs))
	for _, input := range inputs {
		receipts = append(receipts, InputReceipt{Role: input.role, Name: input.binding.Name, SHA256: input.binding.SHA256, SizeBytes: input.binding.SizeBytes})
	}
	return receipts
}

type contextWriter struct {
	ctx context.Context
	dst io.Writer
}

type contextReader struct {
	ctx context.Context
	src io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.src.Read(p)
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.dst.Write(p)
}

func readContextBounded(ctx context.Context, reader io.Reader, maximum int64) ([]byte, error) {
	var data bytes.Buffer
	chunk := make([]byte, 64<<10)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, err := reader.Read(chunk)
		if n > 0 {
			if int64(data.Len())+int64(n) > maximum {
				return nil, errors.New("workspace input exceeds byte limit")
			}
			_, _ = data.Write(chunk[:n])
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, io.ErrNoProgress
		}
	}
	return data.Bytes(), nil
}

func writeContext(ctx context.Context, writer io.Writer, data []byte) error {
	for len(data) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk := data
		if len(chunk) > 64<<10 {
			chunk = chunk[:64<<10]
		}
		n, err := writer.Write(chunk)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func truncateUTF8(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	value = value[:maximum]
	for len(value) > 0 && !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

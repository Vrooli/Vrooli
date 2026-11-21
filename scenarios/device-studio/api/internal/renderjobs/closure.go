package renderjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"device-studio/internal/composition"
	"device-studio/internal/jobs"
	"device-studio/internal/presentations"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
)

const closureVersion = 1

type closureInput struct {
	Role      string `json:"role"`
	MediaID   string `json:"media_id"`
	MediaSHA  string `json:"media_sha256"`
	MediaSize int64  `json:"media_size_bytes"`
	BlobKey   string `json:"blob_key"`
	Name      string `json:"name"`
}

// immutableClosure contains every renderer choice and source binding needed
// after Submit. Dispatch never recompiles a shot or opens mutable project media.
type immutableClosure struct {
	Version             int                   `json:"version"`
	ProjectID           string                `json:"project_id"`
	ShotID              string                `json:"shot_id"`
	Revision            int64                 `json:"revision"`
	RecipeSHA256        string                `json:"recipe_sha256"`
	SceneSHA256         string                `json:"scene_sha256"`
	PresentationSHA256  string                `json:"presentation_sha256"`
	SceneVersion        string                `json:"scene_version"`
	PresentationVersion string                `json:"presentation_version"`
	Profile             string                `json:"profile"`
	RendererVersion     string                `json:"renderer_version"`
	OutputProfile       string                `json:"output_profile"`
	WorkerKey           string                `json:"worker_key"`
	WorkerSHA256        string                `json:"worker_sha256"`
	SurfaceWidth        int                   `json:"surface_width"`
	SurfaceHeight       int                   `json:"surface_height"`
	Quad                [4]rendering.Corner   `json:"quad"`
	Samples             int                   `json:"samples_per_axis"`
	Presentation        *presentations.Config `json:"presentation,omitempty"`
	PresentationLayout  *presentations.Layout `json:"presentation_layout,omitempty"`
	Inputs              []closureInput        `json:"inputs"`
}

func (c *Coordinator) freezeClosure(ctx context.Context, route routePin, resolved shots.ResolvedShot, workerKey, workerSHA string, workerBytes []byte, outputProfile string) (immutableClosure, []byte, string, error) {
	rev := resolved.Revision
	if !validDigest(workerSHA) || digestBytes(workerBytes) != workerSHA {
		return immutableClosure{}, nil, "", errors.New("renderjobs: worker bytes do not match their immutable digest")
	}
	scene := rev.Snapshot.GetScene()
	if scene == nil || scene.GetPlateQuad() == nil {
		return immutableClosure{}, nil, "", errors.New("renderjobs: eligible revision lacks scene geometry")
	}
	quad := scene.GetPlateQuad()
	closure := immutableClosure{
		Version: closureVersion, ProjectID: resolved.ProjectID, ShotID: resolved.Shot.ID, Revision: rev.Revision,
		RecipeSHA256: rev.RecipeSHA256, SceneSHA256: rev.SceneSHA256, PresentationSHA256: rev.PresentationSHA256,
		SceneVersion: rev.SceneVersion, PresentationVersion: rev.PresentationVersion,
		Profile: rev.CompositorProfile, RendererVersion: RendererVersion, OutputProfile: outputProfile,
		WorkerKey: workerKey, WorkerSHA256: workerSHA, SurfaceWidth: int(scene.GetDisplayWidthPixels()), SurfaceHeight: int(scene.GetDisplayHeightPixels()), Samples: 2,
		Quad: [4]rendering.Corner{{X: quad.GetTopLeft().GetX(), Y: quad.GetTopLeft().GetY()}, {X: quad.GetTopRight().GetX(), Y: quad.GetTopRight().GetY()}, {X: quad.GetBottomRight().GetX(), Y: quad.GetBottomRight().GetY()}, {X: quad.GetBottomLeft().GetX(), Y: quad.GetBottomLeft().GetY()}},
	}
	if rev.CompositorProfile == presentations.SimulatedCompositorProfile {
		if rev.PresentationVersion != "device-studio-presentation-v2" && rev.PresentationVersion != "device-studio-presentation-v3" {
			return immutableClosure{}, nil, "", errors.New("renderjobs: unsupported simulated presentation semantic version")
		}
		cfg, err := presentations.ConfigFromSavedVersion(image.Pt(closure.SurfaceWidth, closure.SurfaceHeight), rev.Snapshot.GetPresentation(), rev.PresentationVersion)
		if err != nil {
			return immutableClosure{}, nil, "", err
		}
		closure.Presentation = &cfg
	} else if rev.PresentationVersion != "device-studio-presentation-v1" || rev.CompositorProfile != rendering.CompositorProfile {
		return immutableClosure{}, nil, "", errors.New("renderjobs: unsupported legacy presentation semantic version")
	}
	mediaByRole := resolvedMedia(rev)
	fileNames := map[string]string{"plate": "plate.image", "screen": "screen.image", "active_mask": "active-mask.png", "hardware_mask": "hardware-mask.png"}
	var totalBytes int64
	for _, role := range []string{"plate", "screen", "active_mask", "hardware_mask"} {
		media := mediaByRole[role]
		if media == nil {
			if role == "hardware_mask" {
				continue
			}
			return immutableClosure{}, nil, "", fmt.Errorf("renderjobs: revision lacks required %s binding", role)
		}
		if err := c.requireRoute(ctx, route); err != nil {
			return immutableClosure{}, nil, "", err
		}
		opened, reader, err := c.media.Open(ctx, resolved.ProjectID, media.GetId())
		if err != nil {
			return immutableClosure{}, nil, "", fmt.Errorf("open project %s media: %w", role, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, maxWorkerInputBytes+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			return immutableClosure{}, nil, "", errors.Join(readErr, closeErr)
		}
		if int64(len(data)) != opened.SizeBytes || opened.SizeBytes != media.GetSizeBytes() || digestBytes(data) != media.GetSha256() || opened.SHA256 != media.GetSha256() || opened.ID != media.GetId() || opened.ProjectID != resolved.ProjectID || opened.MediaType != media.GetMediaType() || opened.Width != media.GetWidthPixels() || opened.Height != media.GetHeightPixels() {
			return immutableClosure{}, nil, "", fmt.Errorf("renderjobs: project %s media no longer matches immutable revision", role)
		}
		if role == "screen" && closure.Presentation != nil {
			decoded, format, decodeErr := image.Decode(bytes.NewReader(data))
			if decodeErr != nil || (format != "png" && format != "jpeg") || decoded.Bounds().Dx() != int(media.GetWidthPixels()) || decoded.Bounds().Dy() != int(media.GetHeightPixels()) {
				return immutableClosure{}, nil, "", errors.New("renderjobs: screen source decode or dimensions differ from immutable metadata")
			}
			layout, err := presentations.BuildLayout(decoded, *closure.Presentation)
			if err != nil {
				return immutableClosure{}, nil, "", fmt.Errorf("validate saved screen presentation against source: %w", err)
			}
			closure.PresentationLayout = &layout
		}
		totalBytes += int64(len(data))
		if totalBytes > maxWorkerInputBytes {
			return immutableClosure{}, nil, "", errors.New("renderjobs: aggregate source media exceeds staging limit")
		}
		key := inputBlobPrefix + media.GetSha256() + "/" + role
		if err := c.putVerified(ctx, route, key, data, opened.MediaType, media.GetSha256()); err != nil {
			return immutableClosure{}, nil, "", fmt.Errorf("pin %s source bytes: %w", role, err)
		}
		closure.Inputs = append(closure.Inputs, closureInput{Role: role, MediaID: media.GetId(), MediaSHA: media.GetSha256(), MediaSize: opened.SizeBytes, BlobKey: key, Name: fileNames[role]})
	}
	closure.RendererVersion = rendererVersionForProfile(closure.Profile)
	data, err := json.Marshal(closure)
	if err != nil {
		return immutableClosure{}, nil, "", err
	}
	data = append(data, '\n')
	if len(data) > maxManifestBytes {
		return immutableClosure{}, nil, "", errors.New("renderjobs: immutable render closure exceeds manifest limit")
	}
	digest := digestBytes(data)
	key := closureBlobPrefix + digest + ".json"
	if err := c.putVerified(ctx, route, key, data, "application/json", digest); err != nil {
		return immutableClosure{}, nil, "", fmt.Errorf("pin immutable render closure: %w", err)
	}
	return closure, data, key, nil
}

func rendererVersionForProfile(profile string) string {
	if profile == presentations.SimulatedCompositorProfile {
		return profile
	}
	return RendererVersion
}

func closureBindings(closure immutableClosure, key, digest string) []jobs.ArtifactBinding {
	out := []jobs.ArtifactBinding{{Role: "render_closure", ArtifactID: key, RevisionID: "sha256:" + digest, SHA256: digest}}
	for _, input := range closure.Inputs {
		out = append(out, jobs.ArtifactBinding{Role: "source_" + input.Role, ArtifactID: input.BlobKey, RevisionID: "sha256:" + input.MediaSHA, SHA256: input.MediaSHA})
	}
	return out
}

func decodeClosure(data []byte) (immutableClosure, error) {
	var closure immutableClosure
	if err := decodeJSON(data, &closure); err != nil {
		return immutableClosure{}, err
	}
	canonical, err := json.Marshal(closure)
	if err != nil || !bytes.Equal(data, append(canonical, '\n')) {
		return immutableClosure{}, errors.New("renderjobs: immutable closure is not canonical JSON")
	}
	if closure.Version != closureVersion || closure.ProjectID == "" || closure.ShotID == "" || closure.Revision <= 0 || !validDigest(closure.RecipeSHA256) || !validDigest(closure.SceneSHA256) || !validDigest(closure.PresentationSHA256) || !validDigest(closure.WorkerSHA256) || closure.WorkerKey == "" || len(closure.Inputs) < 3 || closure.SurfaceWidth <= 0 || closure.SurfaceHeight <= 0 || closure.SurfaceWidth > composition.StillMaxDimension || closure.SurfaceHeight > composition.StillMaxDimension || int64(closure.SurfaceWidth)*int64(closure.SurfaceHeight) > composition.StillMaxPixels {
		return immutableClosure{}, errors.New("renderjobs: immutable closure identity is incomplete")
	}
	if closure.Profile == rendering.CompositorProfile {
		if closure.Presentation != nil || closure.PresentationVersion != "device-studio-presentation-v1" || closure.RendererVersion != RendererVersion {
			return immutableClosure{}, errors.New("renderjobs: legacy closure semantic version is unsupported")
		}
	} else if closure.Profile == presentations.SimulatedCompositorProfile {
		if closure.Presentation == nil || closure.PresentationLayout == nil || (closure.PresentationVersion != "device-studio-presentation-v2" && closure.PresentationVersion != "device-studio-presentation-v3") || closure.RendererVersion != presentations.SimulatedCompositorProfile || closure.Presentation.Target.X != closure.SurfaceWidth || closure.Presentation.Target.Y != closure.SurfaceHeight {
			return immutableClosure{}, errors.New("renderjobs: simulated closure semantic version is unsupported")
		}
		wantProfileVersion := presentations.PhoneProfileVersion
		if closure.Presentation.Browser != nil {
			wantProfileVersion = presentations.BrowserProfileVersion
		}
		if closure.PresentationVersion == "device-studio-presentation-v3" {
			if closure.Presentation.Phone == nil || closure.Presentation.Phone.ProfileVersion != presentations.PhoneProfileVersionAuto {
				return immutableClosure{}, errors.New("renderjobs: v3 closure requires the phone v3 profile")
			}
			wantProfileVersion = presentations.PhoneProfileVersionAuto
		}
		if closure.PresentationLayout.ProfileID == "" || closure.PresentationLayout.ProfileVersion != wantProfileVersion || closure.PresentationLayout.DisplaySize != closure.Presentation.Target || !validDigest(closure.PresentationLayout.ConfigSHA256) || !validDigest(closure.PresentationLayout.FontSHA256) || closure.PresentationLayout.Rasterizer == "" || closure.PresentationLayout.FillPolicy == "" {
			return immutableClosure{}, errors.New("renderjobs: simulated closure layout identity is incomplete")
		}
		if closure.PresentationVersion == "device-studio-presentation-v3" && (closure.PresentationLayout.ResolvedStatusInk == nil || closure.PresentationLayout.ResolvedHomeInk == nil) {
			return immutableClosure{}, errors.New("renderjobs: v3 closure omits resolved phone inks")
		}
		if closure.PresentationVersion == "device-studio-presentation-v2" && (closure.PresentationLayout.ResolvedStatusInk != nil || closure.PresentationLayout.ResolvedHomeInk != nil || (closure.Presentation.Phone != nil && closure.Presentation.Phone.ProfileVersion != "")) {
			return immutableClosure{}, errors.New("renderjobs: v2 closure contains v3 phone profile fields")
		}
	} else {
		return immutableClosure{}, errors.New("renderjobs: immutable closure profile is unsupported")
	}
	seen := map[string]bool{}
	for _, input := range closure.Inputs {
		if input.Role == "" || seen[input.Role] || input.MediaID == "" || !validDigest(input.MediaSHA) || input.MediaSize <= 0 || input.MediaSize > maxWorkerInputBytes || input.BlobKey != inputBlobPrefix+input.MediaSHA+"/"+input.Role {
			return immutableClosure{}, errors.New("renderjobs: immutable closure input binding is malformed")
		}
		seen[input.Role] = true
	}
	if !seen["plate"] || !seen["screen"] || !seen["active_mask"] {
		return immutableClosure{}, errors.New("renderjobs: immutable closure omits a required source role")
	}
	return closure, nil
}

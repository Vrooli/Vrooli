package ai

import (
	"strings"

	"google.golang.org/protobuf/proto"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	"image-tools/internal/capability"
)

// presetRequest turns a request for a preset operation (a manifest `preset`) into
// the request for the operation it runs: the caller's prompt is wrapped in the
// preset's, the preset's model, sampling defaults and adapters are added unless the
// caller chose their own, and each adapter reads the upload of the port it names.
// Anything that is not a preset comes back unchanged. The uploads stay on the
// request, so the input review and the face and consent checks see them as they do
// for any operation that takes images.
func presetRequest(op string, params *aiv1.AIParams, inputKey, referenceKey string) (string, *aiv1.AIParams) {
	manifest, ok := capability.Get(op)
	if !ok || manifest.Preset == nil {
		return op, params
	}
	preset := manifest.Preset
	out := proto.Clone(params).(*aiv1.AIParams)
	if preset.Prompt != "" && strings.TrimSpace(out.GetPrompt()) != "" {
		out.Prompt = strings.ReplaceAll(preset.Prompt, capability.PromptPlaceholder, strings.TrimSpace(out.GetPrompt()))
	}
	if out.GetModelOverride() == "" {
		out.ModelOverride = preset.Model
	}
	if out.GetNegativePrompt() == "" {
		out.NegativePrompt = preset.Negative
	}
	if out.GetSteps() == 0 {
		out.Steps = int32(preset.Steps)
	}
	if out.GetWidth() == 0 {
		out.Width = int32(preset.Width)
	}
	if out.GetHeight() == 0 {
		out.Height = int32(preset.Height)
	}
	if out.GetSeed() == 0 {
		out.Seed = preset.Seed
	}
	if out.GetCfgScale() == 0 {
		out.CfgScale = preset.CfgScale
	}
	uploads := map[string]string{capability.ImagePort: inputKey, capability.ReferencePort: referenceKey}
	for _, a := range preset.Adapters {
		key := uploads[a.Input]
		if key == "" {
			continue
		}
		scale := a.Scale
		if given := out.GetOpParams().GetFields()[a.ScaleParam]; a.ScaleParam != "" && given.GetNumberValue() != 0 {
			scale = given.GetNumberValue()
		}
		out.Adapters = append(out.Adapters, &aiv1.AdapterRef{AdapterId: a.ID, Scale: scale, ConditioningImageKey: key})
	}
	return preset.Runs, out
}

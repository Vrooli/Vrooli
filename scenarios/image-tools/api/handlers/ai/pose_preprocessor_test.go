package ai

import (
	"testing"

	"image-tools/internal/adapters"
)

func TestBindPosePreprocessorInput(t *testing.T) {
	t.Run("binds source image to pose control", func(t *testing.T) {
		resolved := []adapters.ResolvedAdapter{{Kind: adapters.KindControlNet, Preprocessor: adapters.PreprocessorPose}}
		if err := bindPosePreprocessorInput(resolved, "in/source.png"); err != nil {
			t.Fatal(err)
		}
		if got := resolved[0].ConditioningImageKey; got != "in/source.png" {
			t.Fatalf("conditioning key=%q, want source image key", got)
		}
	})

	t.Run("preserves an explicit control image", func(t *testing.T) {
		resolved := []adapters.ResolvedAdapter{{Kind: adapters.KindControlNet, Preprocessor: adapters.PreprocessorPose, ConditioningImageKey: "in/prepared-pose.png"}}
		if err := bindPosePreprocessorInput(resolved, "in/source.png"); err != nil {
			t.Fatal(err)
		}
		if got := resolved[0].ConditioningImageKey; got != "in/prepared-pose.png" {
			t.Fatalf("conditioning key=%q, explicit value was replaced", got)
		}
	})

	t.Run("requires source image", func(t *testing.T) {
		resolved := []adapters.ResolvedAdapter{{Kind: adapters.KindControlNet, Preprocessor: adapters.PreprocessorPose}}
		if err := bindPosePreprocessorInput(resolved, ""); err == nil {
			t.Fatal("expected missing source image error")
		}
	})

	t.Run("ignores non-pose adapters", func(t *testing.T) {
		resolved := []adapters.ResolvedAdapter{
			{Kind: adapters.KindLoRA, Preprocessor: adapters.PreprocessorPose},
			{Kind: adapters.KindControlNet, Preprocessor: adapters.PreprocessorCanny},
		}
		if err := bindPosePreprocessorInput(resolved, ""); err != nil {
			t.Fatal(err)
		}
		if resolved[0].ConditioningImageKey != "" || resolved[1].ConditioningImageKey != "" {
			t.Fatalf("non-pose adapters changed: %+v", resolved)
		}
	})
}

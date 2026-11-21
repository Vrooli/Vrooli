package routing

import (
	"reflect"
	"testing"

	"ai-gateway/internal/requestvocab"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestRequiredInputModalitiesFollowRequestKindCatalog(t *testing.T) {
	for _, definition := range requestvocab.RequestKindCatalog() {
		t.Run(definition.Kind.String(), func(t *testing.T) {
			got := requiredInputModalities(&sharedv1.GatewayRequest{Kind: definition.Kind})
			want := append([]sharedv1.Modality(nil), definition.RequiredInputs...)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("requiredInputModalities() = %v, want %v", got, want)
			}
		})
	}
}

func TestRequiredInputModalitiesPreserveAttachmentModalities(t *testing.T) {
	req := &sharedv1.GatewayRequest{
		Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Attachments: []*sharedv1.Attachment{
			{Modality: sharedv1.Modality_MODALITY_VIDEO},
			{Modality: sharedv1.Modality_MODALITY_IMAGE},
			{Modality: sharedv1.Modality_MODALITY_TEXT},
			{Modality: sharedv1.Modality_MODALITY_IMAGE},
		},
	}
	want := []sharedv1.Modality{
		sharedv1.Modality_MODALITY_TEXT,
		sharedv1.Modality_MODALITY_IMAGE,
		sharedv1.Modality_MODALITY_VIDEO,
	}
	if got := requiredInputModalities(req); !reflect.DeepEqual(got, want) {
		t.Fatalf("requiredInputModalities() = %v, want %v", got, want)
	}
}

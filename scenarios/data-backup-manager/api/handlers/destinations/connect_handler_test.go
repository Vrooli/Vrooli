package destinations

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
)

// TestDestinationsService_Contract exercises every DestinationsService RPC
// against the handler backed by a FakeService and asserts request→domain and
// domain→response translation, including enum mapping and typed-error codes.
func TestDestinationsService_Contract(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateDestination maps backend enum and returns wire destination", func(t *testing.T) {
		svc := &mocks.FakeService{CreateOut: destinations.Destination{
			ID:                  "dst-1",
			Name:                "primary",
			BackendKind:         destinations.BackendFilesystem,
			Location:            "/mnt/backup",
			CapBytes:            0,
			CapPolicy:           destinations.CapPolicyAlertBlock,
			EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		SecretRef:           "vrooli/kopia/primary:repository-passphrase",
			CreatedAt:           time.Unix(1700000000, 0).UTC(),
			UpdatedAt:           time.Unix(1700000000, 0).UTC(),
		}}
		h := NewConnectHandler(Deps{Service: svc})

		resp, err := h.CreateDestination(ctx, connect.NewRequest(&destinationsv1.CreateDestinationRequest{
			Name:        "primary",
			BackendKind: destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM,
			Location:    "/mnt/backup",
		}))
		if err != nil {
			t.Fatalf("CreateDestination: %v", err)
		}
		if len(svc.CreateInputs) != 1 {
			t.Fatalf("service got wrong number of CreateInputs: %d", len(svc.CreateInputs))
		}
		if svc.CreateInputs[0].Backend != destinations.BackendFilesystem {
			t.Fatalf("backend not translated: %q", svc.CreateInputs[0].Backend)
		}
		got := resp.Msg.Destination
		if got.Id != "dst-1" || got.BackendKind != destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM {
			t.Fatalf("response destination wrong: %+v", got)
		}
		if got.EncryptionAlgorithm != "AES256-GCM-HMAC-SHA256" {
			t.Fatalf("encryption_algorithm not returned: %q", got.EncryptionAlgorithm)
		}
		if got.CreatedAt == nil || got.UpdatedAt == nil {
			t.Fatal("response destination missing timestamps")
		}
		if got.CapPolicy != destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK {
			t.Fatalf("cap_policy not translated: %v", got.CapPolicy)
		}
	})

	t.Run("CreateDestination surfaces invalid-argument", func(t *testing.T) {
		svc := &mocks.FakeService{CreateErr: destinations.ErrInvalidDestination{Field: "name", Reason: "required"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.CreateDestination(ctx, connect.NewRequest(&destinationsv1.CreateDestinationRequest{}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
		}
	})

	t.Run("GetDestination returns destination", func(t *testing.T) {
		svc := &mocks.FakeService{GetOut: destinations.Destination{
			ID:          "dst-2",
			Name:        "secondary",
			BackendKind: destinations.BackendS3,
			Location:    "my-bucket",
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.GetDestination(ctx, connect.NewRequest(&destinationsv1.GetDestinationRequest{Id: "dst-2"}))
		if err != nil {
			t.Fatalf("GetDestination: %v", err)
		}
		if svc.GetID != "dst-2" {
			t.Fatalf("id not passed to service: %q", svc.GetID)
		}
		if resp.Msg.Destination.BackendKind != destinationsv1.BackendKind_BACKEND_KIND_S3 {
			t.Fatalf("backend_kind not translated: %v", resp.Msg.Destination.BackendKind)
		}
	})

	t.Run("GetDestination surfaces not-found", func(t *testing.T) {
		svc := &mocks.FakeService{GetErr: destinations.ErrDestinationNotFound{ID: "missing"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetDestination(ctx, connect.NewRequest(&destinationsv1.GetDestinationRequest{Id: "missing"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
		}
	})

	t.Run("ListDestinations maps the collection", func(t *testing.T) {
		svc := &mocks.FakeService{ListOut: []destinations.Destination{
			{ID: "a", Name: "alpha", BackendKind: destinations.BackendFilesystem, Location: "/a"},
			{ID: "b", Name: "beta", BackendKind: destinations.BackendS3, Location: "bucket"},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.ListDestinations(ctx, connect.NewRequest(&destinationsv1.ListDestinationsRequest{}))
		if err != nil {
			t.Fatalf("ListDestinations: %v", err)
		}
		if len(resp.Msg.Destinations) != 2 {
			t.Fatalf("expected 2 destinations, got %d", len(resp.Msg.Destinations))
		}
		if resp.Msg.Destinations[1].BackendKind != destinationsv1.BackendKind_BACKEND_KIND_S3 {
			t.Fatalf("backend_kind mapping wrong: %v", resp.Msg.Destinations[1].BackendKind)
		}
	})

	t.Run("UpdateDestination passes cap fields and returns updated destination", func(t *testing.T) {
		svc := &mocks.FakeService{UpdateOut: destinations.Destination{
			ID:        "dst-1",
			Name:      "primary",
			CapBytes:  5000,
			CapPolicy: destinations.CapPolicyAlertOnly,
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.UpdateDestination(ctx, connect.NewRequest(&destinationsv1.UpdateDestinationRequest{
			Id:        "dst-1",
			CapBytes:  5000,
			CapPolicy: destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY,
		}))
		if err != nil {
			t.Fatalf("UpdateDestination: %v", err)
		}
		if len(svc.UpdateInputs) != 1 || svc.UpdateInputs[0].CapPolicy != destinations.CapPolicyAlertOnly {
			t.Fatalf("cap_policy not translated in update input: %+v", svc.UpdateInputs)
		}
		if resp.Msg.Destination.CapPolicy != destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY {
			t.Fatalf("cap_policy not translated in response: %v", resp.Msg.Destination.CapPolicy)
		}
	})

	t.Run("DeleteDestination returns removed flag", func(t *testing.T) {
		svc := &mocks.FakeService{DeleteOut: true}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.DeleteDestination(ctx, connect.NewRequest(&destinationsv1.DeleteDestinationRequest{Id: "dst-1"}))
		if err != nil {
			t.Fatalf("DeleteDestination: %v", err)
		}
		if !resp.Msg.Removed {
			t.Fatal("removed = false, want true")
		}
		if len(svc.DeleteIDs) != 1 || svc.DeleteIDs[0] != "dst-1" {
			t.Fatalf("delete id not passed: %v", svc.DeleteIDs)
		}
	})

	t.Run("GetDestinationUsage maps usage state and policy enums", func(t *testing.T) {
		svc := &mocks.FakeService{UsageOut: destinations.UsageReport{
			UsageBytes: 950,
			CapBytes:   1000,
			UsageState: destinations.UsageStateNear,
			CapPolicy:  destinations.CapPolicyAlertBlock,
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.GetDestinationUsage(ctx, connect.NewRequest(&destinationsv1.GetDestinationUsageRequest{Id: "dst-1"}))
		if err != nil {
			t.Fatalf("GetDestinationUsage: %v", err)
		}
		if svc.UsageID != "dst-1" {
			t.Fatalf("usage id not passed: %q", svc.UsageID)
		}
		if resp.Msg.UsageState != destinationsv1.UsageState_USAGE_STATE_NEAR {
			t.Fatalf("usage_state = %v, want NEAR", resp.Msg.UsageState)
		}
		if resp.Msg.CapPolicy != destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK {
			t.Fatalf("cap_policy = %v, want ALERT_BLOCK", resp.Msg.CapPolicy)
		}
		if resp.Msg.UsageBytes != 950 || resp.Msg.CapBytes != 1000 {
			t.Fatalf("usage/cap wrong: %d/%d", resp.Msg.UsageBytes, resp.Msg.CapBytes)
		}
	})

	t.Run("GetDestinationUsage surfaces not-found", func(t *testing.T) {
		svc := &mocks.FakeService{UsageErr: destinations.ErrDestinationNotFound{ID: "nope"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetDestinationUsage(ctx, connect.NewRequest(&destinationsv1.GetDestinationUsageRequest{Id: "nope"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
		}
	})
}

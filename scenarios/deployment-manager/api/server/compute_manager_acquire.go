package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"deployment-manager/releases"
	"github.com/vrooli/api-core/discovery"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

// cloudComputeAcquirer consumes Compute Manager's staged contract. It only
// returns after owner readback confirms the admitted instance; DM persists
// that exact target before calling EnrollCloudCompute.
type cloudComputeAcquirer struct {
	resolve func(context.Context) (string, error)
	client  *http.Client
	now     func() time.Time
}

func newCloudComputeAcquirer() *cloudComputeAcquirer {
	return &cloudComputeAcquirer{
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "compute-manager")
		},
		client: &http.Client{Timeout: 20 * time.Second}, now: func() time.Time { return time.Now().UTC() },
	}
}

func (a *cloudComputeAcquirer) cmClient(ctx context.Context) (instanceconnect.InstanceServiceClient, *http.Client, error) {
	if a == nil || a.resolve == nil {
		return nil, nil, fmt.Errorf("Compute Manager acquire route is unavailable")
	}
	base, err := a.resolve(ctx)
	if err != nil || strings.TrimSpace(base) == "" {
		return nil, nil, fmt.Errorf("Compute Manager is unavailable; no provider request was sent")
	}
	client := a.client
	if client == nil {
		client = http.DefaultClient
	}
	return instanceconnect.NewInstanceServiceClient(client, strings.TrimRight(base, "/")), client, nil
}

func (a *cloudComputeAcquirer) AcquireCloudCompute(ctx context.Context, request releases.CloudAcquireRequest) (*releases.CloudAcquireResult, error) {
	if request.Provider != "hetzner" || request.Size != "cx23" || (request.Region != "nbg1" && request.Region != "fsn1") || request.CapCurrency != "USD" || request.CapMonthlyMinor != a2MonthlyCapMinor || request.CapTaxBasis != "pre-tax" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return nil, fmt.Errorf("acquire request does not match the approved A2 provider, offer, region, key, and cap")
	}
	service, client, err := a.cmClient(ctx)
	if err != nil {
		return nil, err
	}
	var instance *instancev1.Instance
	var receipt *instancev1.SpendReceipt
	var persistedReceipt *instancev1.GetAcquireReceiptResponse
	acquireState, acquireReason := "", ""
	acquireAttempted := false
	prior, readErr := service.GetAcquireReceipt(ctx, connect.NewRequest(&instancev1.GetAcquireReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
	if readErr == nil && prior != nil && prior.Msg != nil {
		persistedReceipt = prior.Msg
	} else if connect.CodeOf(readErr) == connect.CodeNotFound {
		acquireAttempted = true
		_, createErr := service.AcquireInstance(ctx, connect.NewRequest(&instancev1.AcquireInstanceRequest{
			IdempotencyKey: request.IdempotencyKey, ProfileId: webConsoleStagingProfile, Region: request.Region, LifetimeSeconds: 3600,
		}))
		if createErr == nil {
			// The typed operation receipt must be the persisted GetAcquireReceipt
			// shape, not reconstructed from AcquireInstance's immediate response.
			prior, readErr = service.GetAcquireReceipt(ctx, connect.NewRequest(&instancev1.GetAcquireReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
			if readErr == nil && prior != nil && prior.Msg != nil {
				persistedReceipt = prior.Msg
			}
		} else {
			readErr = createErr
		}
	}
	if readErr != nil {
		// A begin may have committed before its response was lost. Read back by
		// the same key; never issue a replacement acquisition key.
		prior, readErr = service.GetAcquireReceipt(ctx, connect.NewRequest(&instancev1.GetAcquireReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
		if readErr != nil {
			if connect.CodeOf(readErr) == connect.CodeNotFound {
				if acquireAttempted {
					return nil, fmt.Errorf("%w: AcquireInstance reply was lost and no persisted acquisition receipt is visible yet", releases.ErrCloudAcquireAmbiguous)
				}
				return nil, fmt.Errorf("Compute Manager has no persisted acquisition for this key: %v", readErr)
			}
			return nil, fmt.Errorf("%w: acquisition reply and exact receipt readback unavailable: %v", releases.ErrCloudAcquireAmbiguous, readErr)
		}
		if prior != nil && prior.Msg != nil {
			persistedReceipt = prior.Msg
		}
	}
	if persistedReceipt != nil {
		instance, receipt = persistedReceipt.GetInstance(), persistedReceipt.GetSpendReceipt()
		acquireState, acquireReason = persistedReceipt.GetState(), persistedReceipt.GetReason()
	}
	if acquireState == "blocked" {
		if receipt != nil && receipt.GetAdmissionReason() != "" {
			acquireReason = receipt.GetAdmissionReason()
		}
		return nil, fmt.Errorf("Compute Manager blocked persisted acquisition: %s", acquireReason)
	}
	if acquireState != "acquired" {
		return nil, fmt.Errorf("%w: Compute Manager persisted acquisition state %q: %s", releases.ErrCloudAcquireAmbiguous, acquireState, acquireReason)
	}
	if instance == nil || receipt == nil {
		return nil, fmt.Errorf("%w: Compute Manager acquisition response lacks the persisted target or spend receipt", releases.ErrCloudAcquireAmbiguous)
	}
	instanceID := strings.TrimSpace(instance.GetId())
	if instanceID == "" || receipt.GetIdempotencyKey() != request.IdempotencyKey || receipt.GetInstanceId() != instanceID || receipt.GetAdmissionDecision() != "approved" || receipt.GetQuoteState() != "approved" {
		return nil, fmt.Errorf("Compute Manager blocked or mismatched the persisted acquisition receipt: %s", receipt.GetAdmissionReason())
	}
	verifier := &cloudComputeAdmissionVerifier{resolve: a.resolve, client: client, now: a.now}
	if err := verifier.VerifyCloudComputeAdmission(ctx, request.IdempotencyKey, instanceID); err != nil {
		return nil, fmt.Errorf("%w: persisted receipt or exact instance readback failed: %v", releases.ErrCloudAcquireAmbiguous, err)
	}
	receiptJSON, err := protojson.Marshal(persistedReceipt)
	if err != nil {
		return nil, fmt.Errorf("%w: encode persisted acquisition receipt: %v", releases.ErrCloudAcquireAmbiguous, err)
	}
	return &releases.CloudAcquireResult{QuoteIdempotencyKey: request.IdempotencyKey, InstanceID: instanceID, Receipt: receiptJSON}, nil
}

func (a *cloudComputeAcquirer) EnrollCloudCompute(ctx context.Context, request releases.CloudEnrollmentRequest) (*releases.CloudEnrollmentResult, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" || strings.TrimSpace(request.AcquisitionIdempotencyKey) == "" || strings.TrimSpace(request.InstanceID) == "" {
		return nil, fmt.Errorf("exact persisted acquisition and enrollment keys are required")
	}
	service, _, err := a.cmClient(ctx)
	if err != nil {
		return nil, err
	}
	var receipt *instancev1.EnrollmentReceipt
	var instance *instancev1.Instance
	prior, readErr := service.GetEnrollmentReceipt(ctx, connect.NewRequest(&instancev1.GetEnrollmentReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
	if readErr == nil && prior != nil && prior.Msg != nil {
		receipt, instance = prior.Msg.GetReceipt(), prior.Msg.GetInstance()
	} else if connect.CodeOf(readErr) == connect.CodeNotFound {
		created, createErr := service.EnrollInstance(ctx, connect.NewRequest(&instancev1.EnrollInstanceRequest{
			IdempotencyKey: request.IdempotencyKey, AcquisitionIdempotencyKey: request.AcquisitionIdempotencyKey, InstanceId: request.InstanceID,
		}))
		if createErr == nil && created != nil && created.Msg != nil {
			receipt, instance = created.Msg.GetReceipt(), created.Msg.GetInstance()
		}
		readErr = createErr
	}
	if readErr != nil {
		prior, readErr = service.GetEnrollmentReceipt(ctx, connect.NewRequest(&instancev1.GetEnrollmentReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
		if readErr != nil {
			return nil, fmt.Errorf("%w: enrollment reply and exact receipt readback unavailable: %v", releases.ErrCloudEnrollmentAmbiguous, readErr)
		}
		if prior != nil && prior.Msg != nil {
			receipt, instance = prior.Msg.GetReceipt(), prior.Msg.GetInstance()
		}
	}
	if receipt == nil {
		return nil, fmt.Errorf("%w: Compute Manager returned no persisted enrollment receipt", releases.ErrCloudEnrollmentAmbiguous)
	}
	if receipt.GetIdempotencyKey() != request.IdempotencyKey || receipt.GetAcquisitionIdempotencyKey() != request.AcquisitionIdempotencyKey || receipt.GetInstanceId() != request.InstanceID || instance == nil || instance.GetId() != request.InstanceID {
		return nil, fmt.Errorf("%w: enrollment receipt does not bind the exact saved target and acquisition key", releases.ErrCloudEnrollmentAmbiguous)
	}
	result := &releases.CloudEnrollmentResult{InstanceID: instance.GetId(), BridgeMachineID: receipt.GetBridgeMachineId(), BridgeOperationID: receipt.GetBridgeOperationId()}
	receiptJSON, marshalErr := protojson.Marshal(receipt)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: encode persisted enrollment receipt: %v", releases.ErrCloudEnrollmentAmbiguous, marshalErr)
	}
	result.Receipt = receiptJSON
	switch receipt.GetState() {
	case instancev1.EnrollmentState_ENROLLMENT_STATE_READY:
		result.State = "ready"
		if result.BridgeMachineID == "" || instance.GetBridgeMachineId() != result.BridgeMachineID || result.BridgeOperationID == "" {
			return nil, fmt.Errorf("%w: ready receipt lacks matching persisted Bridge machine and operation evidence", releases.ErrCloudEnrollmentAmbiguous)
		}
		return result, nil
	case instancev1.EnrollmentState_ENROLLMENT_STATE_AWAITING_ENROLLMENT:
		result.State = "awaiting_enrollment"
		return result, fmt.Errorf("%w: persisted Compute Manager state is awaiting_enrollment", releases.ErrCloudEnrollmentPending)
	case instancev1.EnrollmentState_ENROLLMENT_STATE_ENROLLING:
		result.State = "enrolling"
		return result, fmt.Errorf("%w: persisted Compute Manager state is enrolling", releases.ErrCloudEnrollmentInProgress)
	case instancev1.EnrollmentState_ENROLLMENT_STATE_BLOCKED:
		return result, fmt.Errorf("Compute Manager blocked Bridge enrollment: %s", receipt.GetReason())
	case instancev1.EnrollmentState_ENROLLMENT_STATE_AMBIGUOUS:
		return result, fmt.Errorf("%w: %s", releases.ErrCloudEnrollmentAmbiguous, receipt.GetReason())
	default:
		return nil, fmt.Errorf("%w: persisted enrollment state is unspecified", releases.ErrCloudEnrollmentAmbiguous)
	}
}

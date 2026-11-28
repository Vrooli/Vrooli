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
)

const (
	webConsoleStagingProfile = releases.CloudA2ProfileID
)

type cloudComputeAdmissionVerifier struct {
	resolve func(context.Context) (string, error)
	client  *http.Client
	now     func() time.Time
}

func newCloudComputeAdmissionVerifier() *cloudComputeAdmissionVerifier {
	return &cloudComputeAdmissionVerifier{
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "compute-manager")
		},
		client: &http.Client{Timeout: 10 * time.Second},
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (v *cloudComputeAdmissionVerifier) VerifyCloudComputeAdmission(ctx context.Context, quoteKey, instanceID string) error {
	if v == nil || v.resolve == nil {
		return fmt.Errorf("Compute Manager resolver is not configured")
	}
	quoteKey, instanceID = strings.TrimSpace(quoteKey), strings.TrimSpace(instanceID)
	if quoteKey == "" || instanceID == "" {
		return fmt.Errorf("quote key and Compute Manager instance ID are required")
	}
	base, err := v.resolve(ctx)
	if err != nil || strings.TrimSpace(base) == "" {
		return fmt.Errorf("Compute Manager is unavailable for read-only quote and instance verification")
	}
	client := v.client
	if client == nil {
		client = http.DefaultClient
	}
	service := instanceconnect.NewInstanceServiceClient(client, strings.TrimRight(base, "/"))
	quoteResponse, err := service.GetSpendReceipt(ctx, connect.NewRequest(&instancev1.GetSpendReceiptRequest{IdempotencyKey: quoteKey}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return fmt.Errorf("persisted Compute Manager spend receipt %q was not found", quoteKey)
		}
		return fmt.Errorf("read Compute Manager spend receipt: %w", err)
	}
	quote := quoteResponse.Msg.GetReceipt()
	if quote == nil || quote.GetIdempotencyKey() != quoteKey {
		return fmt.Errorf("Compute Manager returned no exact persisted spend receipt")
	}
	if quote.GetAdmissionDecision() != "approved" || quote.GetQuoteState() != "approved" {
		return fmt.Errorf("Compute Manager quote/admission is %q/%q, not approved", quote.GetQuoteState(), quote.GetAdmissionDecision())
	}
	plan, planErr := releases.A2CloudAcquirePlan(quote.GetRegion(), quoteKey)
	if planErr != nil || quote.GetProvider() != plan.Provider || strings.ToLower(quote.GetSize()) != plan.Size {
		return fmt.Errorf("A2 permits only a Hetzner CX23 in NBG1 or FSN1")
	}
	if !strings.Contains(strings.ToLower(quote.GetOfferId()), plan.Size) {
		return fmt.Errorf("persisted offer ID does not identify the approved CX23 size")
	}
	if !quote.GetIpv4() || quote.GetTerm() != "monthly" || quote.GetCurrency() != plan.CapCurrency || quote.GetTaxBasis() != plan.CapTaxBasis || quote.GetServerMinorPerMonth() < 0 || quote.GetIpv4MinorPerMonth() < 0 || quote.GetTotalMinorPerMonth() != quote.GetServerMinorPerMonth()+quote.GetIpv4MinorPerMonth() || quote.GetTotalMinorPerMonth() > plan.CapMonthlyMinor {
		return fmt.Errorf("persisted quote exceeds or mismatches the A2 USD 709/month pre-tax cap")
	}
	if quote.GetCapCurrency() != plan.CapCurrency || quote.GetCapMonthlyMinor() != plan.CapMonthlyMinor || quote.GetCapTaxBasis() != plan.CapTaxBasis {
		return fmt.Errorf("persisted Compute Manager admission is not bound to the approved A2 cap")
	}
	if strings.TrimSpace(quote.GetSourceUrl()) == "" || strings.TrimSpace(quote.GetSourceVersion()) == "" || quote.GetObservedAt() == nil || quote.GetExpiresAt() == nil {
		return fmt.Errorf("persisted Compute Manager quote lacks source or validity facts")
	}
	now := time.Now().UTC()
	if v.now != nil {
		now = v.now().UTC()
	}
	if quote.GetObservedAt().AsTime().After(now) || !now.Before(quote.GetExpiresAt().AsTime()) {
		return fmt.Errorf("persisted Compute Manager quote is not current")
	}
	if quote.GetInstanceId() != instanceID {
		return fmt.Errorf("spend receipt belongs to Compute Manager instance %q, not requested instance %q", quote.GetInstanceId(), instanceID)
	}
	instanceResponse, err := service.GetInstance(ctx, connect.NewRequest(&instancev1.GetInstanceRequest{Id: instanceID}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return fmt.Errorf("Compute Manager instance %q was not found", instanceID)
		}
		return fmt.Errorf("read Compute Manager instance: %w", err)
	}
	instance := instanceResponse.Msg.GetInstance()
	if instance == nil || instance.GetId() != instanceID {
		return fmt.Errorf("Compute Manager returned no exact instance reference")
	}
	if instance.GetProvider() != plan.Provider || strings.ToLower(instance.GetSize()) != plan.Size || strings.ToLower(instance.GetRegion()) != plan.Region {
		return fmt.Errorf("Compute Manager instance does not match the approved Hetzner CX23 region constraints")
	}
	if instance.GetState() != instancev1.InstanceState_INSTANCE_STATE_RUNNING || strings.TrimSpace(instance.GetAddress()) == "" {
		return fmt.Errorf("Compute Manager instance is not running with a verified address")
	}
	if instance.GetExpiresAt() == nil || !now.Before(instance.GetExpiresAt().AsTime()) {
		return fmt.Errorf("Compute Manager instance lease is expired or missing")
	}
	return nil
}

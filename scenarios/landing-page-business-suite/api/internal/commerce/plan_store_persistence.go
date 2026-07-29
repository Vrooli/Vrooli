package commerce

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// LoadAll loads plan configuration from JSON file into memory.
func (ps *PlanStore) LoadAll() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.plansPath == "" {
		ps.logEvent("plans_path_not_set", map[string]interface{}{"fallback": "empty plans"})
		return nil
	}
	data, err := os.ReadFile(ps.plansPath)
	if err != nil {
		if os.IsNotExist(err) {
			ps.logEvent("plans_file_not_found", map[string]interface{}{"path": ps.plansPath, "fallback": "empty plans"})
			return nil
		}
		return err
	}
	var fileData plansFileFormat
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("parse plans JSON: %w", err)
	}
	bundle := &BundleProduct{BundleKey: fileData.Bundle.BundleKey, Name: fileData.Bundle.Name, StripeProductId: fileData.Bundle.StripeProductID, CreditsPerUsd: fileData.Bundle.CreditsPerUSD, DisplayCreditsMultiplier: fileData.Bundle.DisplayCreditsMultiplier, DisplayCreditsLabel: fileData.Bundle.DisplayCreditsLabel, Environment: fileData.Bundle.Environment}
	if fileData.Bundle.Metadata != nil {
		bundle.Metadata = convertMetadataToProto(fileData.Bundle.Metadata)
	}
	if err := NormalizeBundle(bundle, ps.bundleKey, ps.displayEnv); err != nil {
		return fmt.Errorf("invalid bundle config: %w", err)
	}
	plans := make([]*PlanOption, 0, len(fileData.Plans))
	seenPriceIDs := make(map[string]struct{}, len(fileData.Plans))
	for _, planFile := range fileData.Plans {
		plan := &PlanOption{StripePriceId: planFile.StripePriceID, PlanName: planFile.PlanName, PlanTier: planFile.PlanTier, BillingInterval: MapBillingInterval(planFile.BillingInterval), AmountCents: planFile.AmountCents, Currency: planFile.Currency, DisplayWeight: planFile.DisplayWeight, DisplayEnabled: planFile.DisplayEnabled, MonthlyIncludedCredits: planFile.MonthlyIncludedCredits, OneTimeBonusCredits: planFile.OneTimeBonusCredits, PlanRank: planFile.PlanRank, BonusType: planFile.BonusType, Kind: MapPlanKind(planFile.Kind), IntroEnabled: planFile.IntroEnabled, IntroPeriods: planFile.IntroPeriods, IntroPriceLookupKey: planFile.IntroPriceLookupKey, IsVariableAmount: planFile.IsVariableAmount, BundleKey: ps.bundleKey}
		if planFile.IntroAmountCents != nil {
			plan.IntroAmountCents = proto.Int64(*planFile.IntroAmountCents)
		}
		if planFile.IntroType != "" {
			plan.IntroType = MapIntroPricingType(planFile.IntroType)
		}
		if planFile.Metadata != nil {
			plan.Metadata = convertMetadataToProto(planFile.Metadata)
		}
		if err := NormalizePlanOption(plan, ps.bundleKey); err != nil {
			return fmt.Errorf("invalid plan %s: %w", plan.StripePriceId, err)
		}
		if _, exists := seenPriceIDs[plan.StripePriceId]; exists {
			return fmt.Errorf("duplicate stripe_price_id detected: %s", plan.StripePriceId)
		}
		seenPriceIDs[plan.StripePriceId] = struct{}{}
		plans = append(plans, plan)
	}
	if fileData.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, fileData.UpdatedAt); err == nil {
			ps.updatedAt = t
		}
	}
	couponMappings := make(map[string]string)
	for priceID, couponID := range fileData.CouponMappings {
		if strings.TrimSpace(priceID) != "" && strings.TrimSpace(couponID) != "" {
			couponMappings[strings.TrimSpace(priceID)] = strings.TrimSpace(couponID)
		}
	}
	ps.bundle, ps.plans, ps.couponMappings = bundle, plans, couponMappings
	ps.logEvent("plans_loaded", map[string]interface{}{"path": ps.plansPath, "plan_count": len(ps.plans), "bundle_key": ps.bundle.BundleKey, "coupon_mapping_count": len(ps.couponMappings)})
	return nil
}

// SavePlans writes plan configuration back to the JSON file.
func (ps *PlanStore) SavePlans() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.savePlansLocked()
}

func (ps *PlanStore) savePlansLocked() error {
	if ps.plansPath == "" {
		return fmt.Errorf("plans path not configured")
	}
	if err := ps.validatePlanCatalogLocked(); err != nil {
		return err
	}
	fileData := plansFileFormat{UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	if ps.bundle != nil {
		fileData.Bundle = bundleFileFormat{BundleKey: ps.bundle.BundleKey, Name: ps.bundle.Name, StripeProductID: ps.bundle.StripeProductId, CreditsPerUSD: ps.bundle.CreditsPerUsd, DisplayCreditsMultiplier: ps.bundle.DisplayCreditsMultiplier, DisplayCreditsLabel: ps.bundle.DisplayCreditsLabel, Environment: ps.bundle.Environment}
		if ps.bundle.Metadata != nil {
			fileData.Bundle.Metadata = convertProtoMetadataToMap(ps.bundle.Metadata)
		}
	}
	fileData.Plans = make([]planFileFormat, 0, len(ps.plans))
	for _, plan := range ps.plans {
		planFile := planFileFormat{StripePriceID: plan.StripePriceId, PlanName: plan.PlanName, PlanTier: plan.PlanTier, BillingInterval: BillingIntervalLabel(plan.BillingInterval), AmountCents: plan.AmountCents, Currency: plan.Currency, DisplayWeight: plan.DisplayWeight, DisplayEnabled: plan.DisplayEnabled, MonthlyIncludedCredits: plan.MonthlyIncludedCredits, OneTimeBonusCredits: plan.OneTimeBonusCredits, PlanRank: plan.PlanRank, BonusType: plan.BonusType, Kind: PlanKindString(plan.Kind), IntroEnabled: plan.IntroEnabled, IntroPeriods: plan.IntroPeriods, IntroPriceLookupKey: plan.IntroPriceLookupKey, IsVariableAmount: plan.IsVariableAmount}
		if plan.IntroAmountCents != nil {
			value := *plan.IntroAmountCents
			planFile.IntroAmountCents = &value
		}
		if plan.IntroType != shared.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED {
			planFile.IntroType = IntroPricingTypeString(plan.IntroType)
		}
		if plan.Metadata != nil {
			planFile.Metadata = convertProtoMetadataToMap(plan.Metadata)
		}
		fileData.Plans = append(fileData.Plans, planFile)
	}
	if len(ps.couponMappings) > 0 {
		fileData.CouponMappings = make(map[string]string)
		for priceID, couponID := range ps.couponMappings {
			if strings.TrimSpace(priceID) != "" && strings.TrimSpace(couponID) != "" {
				fileData.CouponMappings[priceID] = couponID
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(ps.plansPath), 0o750); err != nil {
		return fmt.Errorf("create plans directory: %w", err)
	}
	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plans: %w", err)
	}
	if err := writeFileAtomic(ps.plansPath, data, 0o644); err != nil {
		return fmt.Errorf("write plans file: %w", err)
	}
	ps.updatedAt = time.Now()
	ps.logEvent("plans_saved", map[string]interface{}{"path": ps.plansPath, "plan_count": len(ps.plans)})
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tempFile, err := os.CreateTemp(dir, ".plans-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempName := tempFile.Name()
	defer func() { _ = tempFile.Close(); _ = os.Remove(tempName) }()
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tempFile.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace plans file: %w", err)
	}
	return nil
}

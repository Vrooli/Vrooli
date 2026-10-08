package credentials

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// [REQ:STC-P0-032] P13-A03: two descriptors that the target could no longer
// tell apart after normalisation are refused before anything is provisioned;
// the same descriptor declared twice is refused too.
func TestPlanBindingsRefusesDescriptorCollisions(t *testing.T) {
	plans := []domain.BundleSecretPlan{
		{ID: "a", Class: domain.SecretClassPerInstallGenerated, Target: domain.BundleSecretTarget{Type: "env", Name: "API_KEY"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/app", Field: "api_key"}},
		{ID: "b", Class: domain.SecretClassPerInstallGenerated, Target: domain.BundleSecretTarget{Type: "env", Name: "api.key"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/app", Field: "api-key"}},
	}
	_, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: plans})
	if !apierrors.Is(err, CodeDescriptorCollision) {
		t.Fatalf("collision error = %v, want %s", err, CodeDescriptorCollision)
	}
	typed := apierrors.As(err)
	if typed.HTTPStatus != 409 || typed.NextAction == nil {
		t.Fatalf("collision refusal = %+v", typed)
	}
	addresses, _ := typed.Details["descriptors"].([]string)
	if len(addresses) != 2 || addresses[0] != "fixture/app:api-key" || addresses[1] != "fixture/app:api_key" {
		t.Fatalf("collision details = %v", typed.Details)
	}

	duplicate := []domain.BundleSecretPlan{plans[0], plans[0]}
	if _, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: duplicate}); !apierrors.Is(err, CodeDescriptorCollision) {
		t.Fatalf("duplicate descriptor = %v", err)
	}
	if _, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: []domain.BundleSecretPlan{{ID: "no-descriptor", Class: domain.SecretClassUserPrompt}}}); !apierrors.Is(err, apierrors.CodeManifestInvalid) {
		t.Fatalf("missing descriptor = %v", err)
	}
}

// [REQ:STC-P0-032] Descriptors are bound byte-for-byte: a distinct target
// name keeps two similar descriptors apart, and the binding id is stable
// across replanning so a redeploy finds its version.
func TestPlanBindingsPreservesExactDescriptorsAndStableIDs(t *testing.T) {
	plans := []domain.BundleSecretPlan{
		{ID: "a", Class: domain.SecretClassPerInstallGenerated, Target: domain.BundleSecretTarget{Type: "env", Name: "PRIMARY_API_KEY"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/app", Field: "Api_Key"}},
		{ID: "b", Class: domain.SecretClassPerInstallGenerated, Target: domain.BundleSecretTarget{Type: "env", Name: "SECONDARY_API_KEY"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/app", Field: "api-key"}},
	}
	now := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	first, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: plans, Now: now})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	second, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: plans, Now: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("replan: %v", err)
	}
	if len(first) != 2 || first[0].ID != second[0].ID || first[1].ID != second[1].ID {
		t.Fatalf("binding ids not stable: %v vs %v", first, second)
	}
	for _, b := range first {
		if b.Descriptor.Field != "Api_Key" && b.Descriptor.Field != "api-key" {
			t.Fatalf("descriptor field was normalised: %q", b.Descriptor.Field)
		}
		if strings.ContainsAny(b.ID, "/:") {
			t.Fatalf("binding id %q is not URL-safe", b.ID)
		}
	}
	other, _ := PlanBindings(PlanInputs{DeploymentID: "dep-2", Plans: plans, Now: now})
	if other[0].ID == first[0].ID {
		t.Fatal("binding id does not include the deployment")
	}
}

// [REQ:STC-P0-032] A descriptor bound on several deployments is a typed
// refusal when the deployment is not named; naming it resolves exactly.
func TestResolveBindingRefusesCrossDeploymentAmbiguity(t *testing.T) {
	store := NewMemoryStore()
	descriptor := domain.CredentialDescriptor{LogicalID: "fixture/store", Field: "password"}
	for _, dep := range []string{"dep-1", "dep-2"} {
		if err := store.UpsertBinding(context.Background(), &domain.CredentialBinding{ID: BindingID(dep, descriptor), DeploymentID: dep, Descriptor: descriptor}); err != nil {
			t.Fatal(err)
		}
	}
	_, err := ResolveBinding(context.Background(), store, "", descriptor)
	if !apierrors.Is(err, CodeBindingAmbiguous) {
		t.Fatalf("ambiguous resolve = %v", err)
	}
	deployments, _ := apierrors.As(err).Details["deployments"].([]string)
	if len(deployments) != 2 {
		t.Fatalf("ambiguity does not list deployments: %v", apierrors.As(err).Details)
	}
	exact, err := ResolveBinding(context.Background(), store, "dep-2", descriptor)
	if err != nil || exact.DeploymentID != "dep-2" {
		t.Fatalf("exact resolve = %v, %v", exact, err)
	}
	if _, err := ResolveBinding(context.Background(), store, "", domain.CredentialDescriptor{LogicalID: "fixture/none", Field: "x"}); !apierrors.Is(err, CodeBindingNotFound) {
		t.Fatalf("unknown descriptor = %v", err)
	}
}

// [REQ:STC-P0-032] Consumers and classes derive from the closure: a
// credential_of reason names the consumer, a generated password read by a
// database resource is a database password, operator values are external.
func TestConsumersAndClassesDeriveFromClosure(t *testing.T) {
	closure := &domain.Closure{PersistentData: []domain.ClosurePersistentData{{ID: "store-data", Owner: "store", MigrationOwner: "resource"}}, Components: []domain.ClosureComponent{
		{ID: "resource:store", Kind: domain.ClosureKindResource},
		{ID: "scenario:app", Kind: domain.ClosureKindScenario},
		{ID: "credential:fixture/store:password", Kind: domain.ClosureKindCredentialDescriptor, Credential: &domain.ClosureCredential{LogicalID: "fixture/store", Field: "password"}, Reasons: []domain.ClosureReason{{Kind: domain.ClosureReasonCredentialOf, From: "resource:store"}, {Kind: domain.ClosureReasonCredentialOf, From: "scenario:app"}, {Kind: domain.ClosureReasonCredentialOf, From: "scenario:app"}}},
		{ID: "credential:fixture/mailer:api-token", Kind: domain.ClosureKindCredentialDescriptor, Credential: &domain.ClosureCredential{LogicalID: "fixture/mailer", Field: "api-token"}, Reasons: []domain.ClosureReason{{Kind: domain.ClosureReasonCredentialOf, From: "scenario:app"}}},
	}}
	consumers := ConsumersFromClosure(closure)
	if got := consumers["fixture/store:password"]; len(got) != 2 || got[0] != "resource:store" || got[1] != "scenario:app" {
		t.Fatalf("store consumers = %v", got)
	}
	databases := DatabaseResourcesFromClosure(closure)
	if !databases["resource:store"] {
		t.Fatalf("database resources = %v", databases)
	}
	plans := []domain.BundleSecretPlan{
		{ID: "store", Class: domain.SecretClassPerInstallGenerated, Target: domain.BundleSecretTarget{Type: "env", Name: "STORE_PASSWORD"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/store", Field: "password"}},
		{ID: "mailer", Class: domain.SecretClassUserPrompt, Target: domain.BundleSecretTarget{Type: "env", Name: "MAILER_TOKEN"}, Descriptor: &domain.DescriptorAddress{LogicalID: "fixture/mailer", Field: "api-token"}},
		{ID: "enroll", Class: domain.SecretClassInfrastructure, Target: domain.BundleSecretTarget{Type: "env", Name: "BRIDGE_TOKEN"}, Descriptor: &domain.DescriptorAddress{LogicalID: "vrooli/bridge", Field: "enrollment-token"}},
	}
	bindings, err := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: plans, Consumers: consumers, DatabaseResources: databases, ClassOverrides: map[string]domain.CredentialClass{}})
	if err != nil {
		t.Fatal(err)
	}
	classes := map[string]domain.CredentialClass{}
	for _, b := range bindings {
		classes[b.Descriptor.Address()] = b.Class
	}
	if classes["fixture/store:password"] != domain.CredentialClassGeneratedDatabasePassword || classes["fixture/mailer:api-token"] != domain.CredentialClassExternalAPICredential || classes["vrooli/bridge:enrollment-token"] != domain.CredentialClassMachineEnrollment {
		t.Fatalf("classes = %v", classes)
	}
	overridden, _ := PlanBindings(PlanInputs{DeploymentID: "dep-1", Plans: plans[:1], Consumers: consumers, ClassOverrides: map[string]domain.CredentialClass{"fixture/store:password": domain.CredentialClassSharedDependency}})
	if overridden[0].Class != domain.CredentialClassSharedDependency {
		t.Fatalf("override ignored: %s", overridden[0].Class)
	}
}

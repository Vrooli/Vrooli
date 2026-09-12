package credentials

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// PlanInputs is everything binding planning reads. Values are absent by
// construction: planning only maps descriptors to consumers and targets.
type PlanInputs struct {
	DeploymentID string
	Plans        []domain.BundleSecretPlan
	// Consumers maps a descriptor address (logical_id:field) to the closure
	// components that read it. Use ConsumersFromClosure to derive it.
	Consumers map[string][]string
	// ClassOverrides pins a lifecycle class per descriptor address when the
	// heuristic classification is wrong for a deployment.
	ClassOverrides map[string]domain.CredentialClass
	// DatabaseResources names the closure components that are database
	// resources, so a generated password bound to one is classed as such.
	DatabaseResources map[string]bool
	Now               time.Time
}

// ConsumersFromClosure derives the descriptor→consumer map: every credential
// descriptor component lists the components that declared it (reason
// credential_of) as its consumers.
func ConsumersFromClosure(closure *domain.Closure) map[string][]string {
	out := map[string][]string{}
	if closure == nil {
		return out
	}
	for _, component := range closure.Components {
		if component.Kind != domain.ClosureKindCredentialDescriptor || component.Credential == nil {
			continue
		}
		address := domain.CredentialDescriptor{LogicalID: component.Credential.LogicalID, Field: component.Credential.Field}.Address()
		for _, reason := range component.Reasons {
			if reason.Kind != domain.ClosureReasonCredentialOf || strings.TrimSpace(reason.From) == "" {
				continue
			}
			out[address] = appendUnique(out[address], reason.From)
		}
	}
	return out
}

// DatabaseResourcesFromClosure names the resource components that own
// declared persistent data (a declaration-derived marker) or whose id carries
// a database marker. The closure has no resource-kind field yet, so the id
// marker set is explicit here and documented in credential-lifecycle.md.
func DatabaseResourcesFromClosure(closure *domain.Closure) map[string]bool {
	out := map[string]bool{}
	if closure == nil {
		return out
	}
	for _, data := range closure.PersistentData {
		if owner := strings.TrimSpace(data.Owner); owner != "" {
			out[owner] = true
			if !strings.Contains(owner, ":") {
				out["resource:"+owner] = true
			}
		}
	}
	for _, component := range closure.Components {
		if component.Kind != domain.ClosureKindResource {
			continue
		}
		for _, marker := range []string{"postgres", "mysql", "mariadb", "redis", "mongo", "database"} {
			if strings.Contains(strings.ToLower(component.ID), marker) {
				out[component.ID] = true
				break
			}
		}
	}
	return out
}

// PlanBindings maps every declared secret to an exact descriptor binding and
// refuses the plan when two descriptors would collapse onto one target.
func PlanBindings(in PlanInputs) ([]domain.CredentialBinding, error) {
	if strings.TrimSpace(in.DeploymentID) == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "deployment id is required to plan credential bindings")
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	byTarget := map[string][]domain.CredentialDescriptor{}
	byAddress := map[string]bool{}
	bindings := make([]domain.CredentialBinding, 0, len(in.Plans))
	for _, plan := range in.Plans {
		descriptor := descriptorForPlan(plan)
		if descriptor.IsZero() {
			return nil, apierrors.Newf(apierrors.CodeManifestInvalid, "secret %q declares no credential descriptor and no target name", plan.ID)
		}
		address := descriptor.Address()
		if byAddress[address] {
			return nil, newError(CodeDescriptorCollision, "the same credential descriptor is declared twice").WithDetail("descriptor", address)
		}
		byAddress[address] = true
		target := plan.Target
		if strings.TrimSpace(target.Name) == "" {
			target = domain.BundleSecretTarget{Type: "env", Name: descriptor.Field}
		}
		key := normalisedTargetKey(target, descriptor)
		byTarget[key] = append(byTarget[key], descriptor)
		consumers := append([]string(nil), in.Consumers[address]...)
		sort.Strings(consumers)
		class := in.ClassOverrides[address]
		if class == "" {
			class = Classify(plan, descriptor, consumers, in.DatabaseResources)
		}
		binding := domain.CredentialBinding{
			ID:           BindingID(in.DeploymentID, descriptor),
			DeploymentID: in.DeploymentID,
			Descriptor:   descriptor,
			Class:        class,
			SourceClass:  plan.Class,
			Target:       target,
			ConsumerRefs: consumers,
			State:        domain.CredentialBindingPlanned,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if class == domain.CredentialClassEncryptionRecoveryKey {
			binding.RecoveryKeyRef = descriptor.Address()
		}
		bindings = append(bindings, binding)
	}
	keys := make([]string, 0, len(byTarget))
	for key := range byTarget {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		descriptors := byTarget[key]
		if len(descriptors) < 2 {
			continue
		}
		addresses := make([]string, 0, len(descriptors))
		for _, d := range descriptors {
			addresses = append(addresses, d.Address())
		}
		sort.Strings(addresses)
		return nil, newError(CodeDescriptorCollision, "two credential descriptors would normalise to the same injection target; refusing ambiguous provisioning").
			WithDetail("target", key).
			WithDetail("descriptors", addresses).
			WithNextAction(apierrors.NextAction{Owner: "secrets-manager", Kind: "declaration", Reference: key, Label: "Give each descriptor a distinct target name"})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].ID < bindings[j].ID })
	return bindings, nil
}

// Classify picks the lifecycle class from the manifest class, the descriptor
// and the consumer set. Machine enrollment and encryption keys are recognised
// by field name; a generated password consumed by a database resource is a
// database password; operator or fetched values are external credentials.
func Classify(plan domain.BundleSecretPlan, descriptor domain.CredentialDescriptor, consumers []string, databases map[string]bool) domain.CredentialClass {
	field := strings.ToLower(descriptor.Field)
	switch {
	case strings.Contains(field, "enrollment") || strings.Contains(field, "enroll-token") || strings.Contains(field, "machine-token"):
		return domain.CredentialClassMachineEnrollment
	case strings.Contains(field, "recovery-key") || strings.Contains(field, "encryption-key") || strings.Contains(field, "backup-key") || strings.Contains(field, "kek"):
		return domain.CredentialClassEncryptionRecoveryKey
	case strings.Contains(field, "signing") || strings.Contains(field, "verification") || strings.Contains(field, "jwt"):
		return domain.CredentialClassSigningKey
	}
	switch plan.Class {
	case domain.SecretClassUserPrompt, domain.SecretClassRemoteFetch:
		return domain.CredentialClassExternalAPICredential
	case domain.SecretClassPerInstallGenerated:
		if strings.Contains(field, "password") {
			for _, consumer := range consumers {
				if databases[consumer] || databases[strings.TrimPrefix(consumer, "resource:")] {
					return domain.CredentialClassGeneratedDatabasePassword
				}
			}
			if isDatabaseLogicalID(descriptor.LogicalID) {
				return domain.CredentialClassGeneratedDatabasePassword
			}
		}
	}
	return domain.CredentialClassSharedDependency
}

func isDatabaseLogicalID(logicalID string) bool {
	lower := strings.ToLower(logicalID)
	for _, marker := range []string{"postgres", "mysql", "mariadb", "mongo", "database"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// BindingID is deterministic per (deployment, descriptor) so a replanned
// deployment finds its existing binding and preserves its version. It is a
// digest rather than the raw address so it is URL-safe; the binding record
// carries the exact descriptor for reading.
func BindingID(deploymentID string, descriptor domain.CredentialDescriptor) string {
	sum := sha256.Sum256([]byte(deploymentID + "\x00" + descriptor.LogicalID + "\x00" + descriptor.Field))
	return "cb_" + hex.EncodeToString(sum[:12])
}

func descriptorForPlan(plan domain.BundleSecretPlan) domain.CredentialDescriptor {
	if plan.Descriptor != nil && strings.TrimSpace(plan.Descriptor.LogicalID) != "" && strings.TrimSpace(plan.Descriptor.Field) != "" {
		return domain.CredentialDescriptor{LogicalID: plan.Descriptor.LogicalID, Field: plan.Descriptor.Field}
	}
	return domain.CredentialDescriptor{}
}

// normalisedTargetKey applies the target-side normalisation the credential
// authority and the env injector perform (lower-case identity, `_` and `.`
// folded to `-`) so that descriptors which the target could no longer tell
// apart are caught before anything is provisioned.
func normalisedTargetKey(target domain.BundleSecretTarget, descriptor domain.CredentialDescriptor) string {
	fold := strings.NewReplacer("_", "-", ".", "-")
	kind := strings.ToLower(strings.TrimSpace(target.Type))
	if kind == "" {
		kind = "env"
	}
	name := strings.ToLower(fold.Replace(strings.TrimSpace(target.Name)))
	authority := strings.ToLower(strings.Trim(descriptor.LogicalID, "/")) + "/" + strings.ToLower(fold.Replace(descriptor.Field))
	return kind + ":" + name + "|" + authority
}

func appendUnique(list []string, value string) []string {
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

// ResolveBinding finds one binding. With a deployment id the lookup is exact;
// without one the descriptor must be bound to exactly one deployment, and a
// descriptor materialised on several deployments is a typed refusal rather
// than a guess.
func ResolveBinding(ctx context.Context, store Store, deploymentID string, descriptor domain.CredentialDescriptor) (*domain.CredentialBinding, error) {
	if strings.TrimSpace(deploymentID) != "" {
		binding, err := store.GetBinding(ctx, deploymentID, BindingID(deploymentID, descriptor))
		if err != nil {
			return nil, err
		}
		if binding == nil {
			return nil, newError(CodeBindingNotFound, "no credential binding for this descriptor on the deployment").WithDetail("descriptor", descriptor.Address())
		}
		return binding, nil
	}
	matches, err := store.FindBindingsByDescriptor(ctx, descriptor)
	if err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, newError(CodeBindingNotFound, "no deployment binds this descriptor").WithDetail("descriptor", descriptor.Address())
	case 1:
		return &matches[0], nil
	}
	deployments := make([]string, 0, len(matches))
	for _, m := range matches {
		deployments = append(deployments, m.DeploymentID)
	}
	sort.Strings(deployments)
	return nil, newError(CodeBindingAmbiguous, fmt.Sprintf("descriptor %s is bound on %d deployments; name the deployment", descriptor.Address(), len(matches))).
		WithDetail("descriptor", descriptor.Address()).
		WithDetail("deployments", deployments)
}

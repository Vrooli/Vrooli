package destinations

import (
	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func recoveryJournalToProto(j destinationreadiness.RecoveryJournal) *destinationsv1.VolumeRecoveryJournal {
	out := &destinationsv1.VolumeRecoveryJournal{
		Version:      j.Version,
		Id:           j.ID,
		Location:     j.Location,
		RelativePath: j.RelativePath,
		Identity:     deviceIdentityToProto(j.Identity),
		Current:      int32(j.Current),
		State:        string(j.State),
		LastError:    j.LastError,
		Steps:        make([]*destinationsv1.VolumeRecoveryStep, 0, len(j.Steps)),
	}
	if !j.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(j.UpdatedAt)
	}
	for _, step := range j.Steps {
		out.Steps = append(out.Steps, &destinationsv1.VolumeRecoveryStep{
			Plan:     preparationPlanToProto(step.Plan),
		Status:   string(step.Status),
			Attempts: int32(step.Attempts),
			Detail:   step.Detail,
		})
	}
	return out
}

func protoToPreparationPlan(p *destinationsv1.DestinationPreparationPlan) destinationreadiness.Plan {
	if p == nil {
		return destinationreadiness.Plan{}
	}
	return destinationreadiness.Plan{
		ID:                 p.Id,
		Action:             protoToPreparationAction(p.Action),
		Location:           p.Location,
		TargetPath:         p.TargetPath,
		RelativePath:       p.RelativePath,
		Identity:           protoToDeviceIdentity(p.Identity),
		DesiredLabel:       p.DesiredLabel,
		DesiredFS:          p.DesiredFilesystem,
		RequiresConfirm:    p.RequiresConfirmation,
		Destructive:        p.Destructive,
		ConfirmationPhrase: p.ConfirmationPhrase,
		Supported:          p.Supported,
		UnsupportedReason:  p.UnsupportedReason,
	}
}

func deviceIdentityToProto(i destinationreadiness.DeviceIdentity) *destinationsv1.DestinationDeviceIdentity {
	return &destinationsv1.DestinationDeviceIdentity{
		DevicePath: i.DevicePath,
		Mountpoint: i.Mountpoint,
		Label:      i.Label,
		Filesystem: i.Filesystem,
		TotalBytes: i.TotalBytes,
		Model:      i.Model,
		Serial:     i.Serial,
		Uuid:       i.UUID,
	}
}

func protoToDeviceIdentity(i *destinationsv1.DestinationDeviceIdentity) destinationreadiness.DeviceIdentity {
	if i == nil {
		return destinationreadiness.DeviceIdentity{}
	}
	return destinationreadiness.DeviceIdentity{
		DevicePath: i.DevicePath,
		Mountpoint: i.Mountpoint,
		Label:      i.Label,
		Filesystem: i.Filesystem,
		TotalBytes: i.TotalBytes,
		Model:      i.Model,
		Serial:     i.Serial,
		UUID:       i.Uuid,
	}
}

// protoToBackend / backendToProto translate the proto BackendKind enum to the
// domain vocabulary so domain code never imports the generated enum.
func protoToBackend(k destinationsv1.BackendKind) destinations.BackendKind {
	switch k {
	case destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM:
		return destinations.BackendFilesystem
	case destinationsv1.BackendKind_BACKEND_KIND_S3:
		return destinations.BackendS3
	default:
		return ""
	}
}

func backendToProto(k destinations.BackendKind) destinationsv1.BackendKind {
	switch k {
	case destinations.BackendFilesystem:
		return destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM
	case destinations.BackendS3:
		return destinationsv1.BackendKind_BACKEND_KIND_S3
	default:
		return destinationsv1.BackendKind_BACKEND_KIND_UNSPECIFIED
	}
}

// protoToCapPolicy / capPolicyToProto translate the proto CapPolicy enum.
func protoToCapPolicy(p destinationsv1.CapPolicy) destinations.CapPolicy {
	switch p {
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK:
		return destinations.CapPolicyAlertBlock
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY:
		return destinations.CapPolicyAlertOnly
	default:
		return ""
	}
}

func capPolicyToProto(p destinations.CapPolicy) destinationsv1.CapPolicy {
	switch p {
	case destinations.CapPolicyAlertBlock:
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK
	case destinations.CapPolicyAlertOnly:
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY
	default:
		return destinationsv1.CapPolicy_CAP_POLICY_UNSPECIFIED
	}
}

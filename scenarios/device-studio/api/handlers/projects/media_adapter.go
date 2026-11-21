package projects

import (
	"device-studio/internal/projects"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sourceMediaToProto(m projects.SourceMedia) *projectsv1.SourceMedia {
	out := &projectsv1.SourceMedia{Id: m.ID, ProjectId: m.ProjectID, Sha256: m.SHA256, SizeBytes: m.SizeBytes, MediaType: m.MediaType, OriginalFilename: m.OriginalFilename, ImportedAt: timestamppb.New(m.ImportedAt.UTC()), ImportMethod: m.ImportMethod, Decoder: m.Decoder, Kind: mediaKindToProto(m.Kind), Readiness: mediaReadinessToProto(m.Readiness), InspectionJobId: m.InspectionJobID, InspectionIssueCode: m.InspectionIssueCode, InspectionIssueMessage: m.InspectionIssueMessage}
	out.InspectionAttemptId = m.InspectionAttemptID
	if m.RasterKnown {
		out.Width = &m.Width
		out.Height = &m.Height
	}
	if m.OrientationKnown {
		out.Orientation = &m.Orientation
	}
	if m.ColorModelKnown {
		out.ColorModel = &m.ColorModel
	}
	if m.ColorProfileKnown {
		out.ColorProfileStatus = &m.ColorProfileStatus
	}
	if r := m.InspectionReceipt; r != nil {
		out.InspectionReceipt = inspectionReceiptToProto(r)
	}
	return out
}

func inspectionReceiptToProto(r *projects.InspectionReceipt) *projectsv1.MediaInspectionReceipt {
	var audio *projectsv1.AudioInspectionFacts
	if r.Audio != nil {
		audio = &projectsv1.AudioInspectionFacts{Version: r.Audio.Version}
		for _, stream := range r.Audio.Streams {
			audio.Streams = append(audio.Streams, &projectsv1.AudioStreamFact{OriginalStreamIndex: stream.OriginalStreamIndex, Codec: stream.Codec})
		}
	}
	out := &projectsv1.MediaInspectionReceipt{Audio: audio, Version: r.Version, Id: r.ID, Sha256: r.SHA256, ClockSha256: r.ClockSHA256, SourceSha256: r.SourceSHA256, JobId: r.JobID, SourceSizeBytes: r.SourceSizeBytes, AttemptId: r.AttemptID, ToolProfileId: r.ToolProfileID, ToolProfileDigest: r.ToolProfileDigest, InspectionProtocolVersion: r.InspectionProtocolVersion, ClockArtifactRef: r.ClockArtifactRef, ClockArtifactSizeBytes: r.ClockArtifactSizeBytes, Classification: r.Classification, IssueCode: r.IssueCode, IssueMessage: r.IssueMessage, Container: r.Container, Codec: r.Codec, ColorPolicy: r.ColorPolicy, ColorPrimaries: r.ColorPrimaries, ColorTransfer: r.ColorTransfer, ColorMatrix: r.ColorMatrix, ColorRange: r.ColorRange, EofPolicy: r.EOFPolicy}
	if r.VideoWidth > 0 {
		v := r.VideoWidth
		out.VideoWidth = &v
	}
	if r.VideoHeight > 0 {
		v := r.VideoHeight
		out.VideoHeight = &v
	}
	if r.RotationDegrees != 0 || r.Classification == "accepted" {
		v := r.RotationDegrees
		out.RotationDegrees = &v
	}
	if r.DurationNumerator != 0 {
		v := r.DurationNumerator
		out.DurationNumerator = &v
	}
	if r.DurationDenominator != 0 {
		v := r.DurationDenominator
		out.DurationDenominator = &v
	}
	if r.FrameCount != 0 {
		v := r.FrameCount
		out.FrameCount = &v
	}
	if r.StreamCount != 0 {
		v := r.StreamCount
		out.StreamCount = &v
	}
	if r.VideoStreamIndex != nil {
		v := uint32(*r.VideoStreamIndex)
		out.VideoStreamIndex = &v
	}
	if r.CodedWidth != 0 {
		v := r.CodedWidth
		out.CodedWidth = &v
	}
	if r.CodedHeight != 0 {
		v := r.CodedHeight
		out.CodedHeight = &v
	}
	if r.SampleAspectNum != 0 {
		v := r.SampleAspectNum
		out.SampleAspectNum = &v
	}
	if r.SampleAspectDen != 0 {
		v := r.SampleAspectDen
		out.SampleAspectDen = &v
	}
	if r.TimeBaseNumerator != 0 {
		v := r.TimeBaseNumerator
		out.TimeBaseNumerator = &v
	}
	if r.TimeBaseDenominator != 0 {
		v := r.TimeBaseDenominator
		out.TimeBaseDenominator = &v
	}
	if r.StreamStartPTS != nil {
		v := *r.StreamStartPTS
		out.StreamStartPts = &v
	}
	if r.SourceEndPTS != nil {
		v := *r.SourceEndPTS
		out.SourceEndPts = &v
	}
	if r.LastFrameDurationTicks != 0 {
		v := r.LastFrameDurationTicks
		out.LastFrameDurationTicks = &v
	}
	return out
}

func mediaKindToProto(k projects.MediaKind) projectsv1.SourceMediaKind {
	if k == projects.MediaKindImage {
		return projectsv1.SourceMediaKind_SOURCE_MEDIA_KIND_IMAGE
	}
	if k == projects.MediaKindVideo {
		return projectsv1.SourceMediaKind_SOURCE_MEDIA_KIND_VIDEO
	}
	return projectsv1.SourceMediaKind_SOURCE_MEDIA_KIND_UNSPECIFIED
}

func mediaReadinessToProto(r projects.Readiness) projectsv1.MediaReadiness {
	switch r {
	case projects.ReadinessReady:
		return projectsv1.MediaReadiness_MEDIA_READINESS_READY
	case projects.ReadinessPending:
		return projectsv1.MediaReadiness_MEDIA_READINESS_PENDING
	case projects.ReadinessRejected:
		return projectsv1.MediaReadiness_MEDIA_READINESS_REJECTED
	case projects.ReadinessReconciliationRequired:
		return projectsv1.MediaReadiness_MEDIA_READINESS_RECONCILIATION_REQUIRED
	default:
		return projectsv1.MediaReadiness_MEDIA_READINESS_UNSPECIFIED
	}
}

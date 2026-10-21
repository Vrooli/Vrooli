package shots

import (
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"
	projects "device-studio/internal/projects"
	domain "device-studio/internal/shots"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func toConnectError(err error, logger *log.Logger) error {
	if err == nil {
		return nil
	}
	var invalid domain.ErrInvalidShot
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid shot input"))
	}
	var invalidPhysical domain.ErrInvalidPhysicalBinding
	if errors.As(err, &invalidPhysical) {
		reason := strings.TrimSpace(invalidPhysical.Reason)
		if reason == "" || len(reason) > 240 || strings.ContainsAny(reason, "\r\n\\/") {
			reason = "check the physical device, scene, motion, and screen selections"
		}
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid physical shot binding: "+reason))
	}
	var mediaUnavailable domain.ErrSourceMediaUnavailable
	if errors.As(err, &mediaUnavailable) {
		return connect.NewError(connect.CodeUnavailable, errors.New("shot source media is unavailable"))
	}
	var unsupportedVersion domain.ErrUnsupportedPresentationVersion
	if errors.As(err, &unsupportedVersion) {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("this saved presentation version is no longer supported; create a new revision to use a supported presentation"))
	}
	if errors.Is(err, domain.ErrVideoSampleUnavailable) {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("the video sample claim is no longer valid; regenerate the sample and accept it before saving"))
	}
	var missing domain.ErrShotNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, errors.New("shot not found"))
	}
	if errors.Is(err, domain.ErrPhysicalPresetNotFound) {
		return connect.NewError(connect.CodeNotFound, errors.New("physical setup preset revision not found"))
	}
	if errors.Is(err, domain.ErrPhysicalPresetConflict) {
		return connect.NewError(connect.CodeAborted, errors.New("physical setup preset changed since it was opened"))
	}
	var projectMissing projects.ErrProjectNotFound
	if errors.As(err, &projectMissing) {
		return connect.NewError(connect.CodeNotFound, errors.New("project not found"))
	}
	var revisionMissing domain.ErrRevisionNotFound
	if errors.As(err, &revisionMissing) {
		return connect.NewError(connect.CodeNotFound, errors.New("shot revision not found"))
	}
	var conflict domain.ErrRevisionConflict
	if errors.As(err, &conflict) {
		ce := connect.NewError(connect.CodeAborted, errors.New("shot changed since it was opened"))
		detail, detailErr := connect.NewErrorDetail(&shotsv1.ShotConflict{ShotId: conflict.ShotID, ExpectedRevision: conflict.Expected, CurrentRevision: conflict.Current})
		if detailErr == nil {
			ce.AddDetail(detail)
		}
		return ce
	}
	if logger != nil {
		logger.Printf("shots operation failed: %v", err)
	}
	return connect.NewError(connect.CodeInternal, errors.New("shot operation failed"))
}

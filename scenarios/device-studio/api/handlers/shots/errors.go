package shots

import (
	"connectrpc.com/connect"
	projects "device-studio/internal/projects"
	domain "device-studio/internal/shots"
	"errors"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"log"
)

func toConnectError(err error, logger *log.Logger) error {
	if err == nil {
		return nil
	}
	var invalid domain.ErrInvalidShot
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid shot input"))
	}
	var mediaUnavailable domain.ErrSourceMediaUnavailable
	if errors.As(err, &mediaUnavailable) {
		return connect.NewError(connect.CodeUnavailable, errors.New("shot source media is unavailable"))
	}
	var missing domain.ErrShotNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, errors.New("shot not found"))
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

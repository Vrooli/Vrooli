package projects

import (
	"connectrpc.com/connect"
	domain "device-studio/internal/projects"
	"errors"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"log"
)

func toConnectError(err error, logger *log.Logger) error {
	if err == nil {
		return nil
	}
	var invalid domain.ErrInvalidProject
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var missing domain.ErrProjectNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, missing)
	}
	var conflict domain.ErrRevisionConflict
	if errors.As(err, &conflict) {
		ce := connect.NewError(connect.CodeAborted, errors.New("project changed since it was opened"))
		detail, e := connect.NewErrorDetail(&projectsv1.ProjectConflict{ProjectId: conflict.ID, ExpectedRevision: conflict.Expected, CurrentRevision: conflict.Current})
		if e == nil {
			ce.AddDetail(detail)
		}
		return ce
	}
	var invalidMedia domain.ErrInvalidSourceMedia
	if errors.As(err, &invalidMedia) {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid source media"))
	}
	var tooLarge domain.ErrSourceMediaTooLarge
	if errors.As(err, &tooLarge) {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("source media exceeds the upload limit"))
	}
	var mediaMissing domain.ErrSourceMediaNotFound
	if errors.As(err, &mediaMissing) {
		return connect.NewError(connect.CodeNotFound, errors.New("source media not found"))
	}
	if logger != nil {
		logger.Printf("projects operation failed: %v", err)
	}
	return connect.NewError(connect.CodeInternal, errors.New("project operation failed"))
}

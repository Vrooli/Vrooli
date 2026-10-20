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
	if logger != nil {
		logger.Printf("projects operation failed: %v", err)
	}
	return connect.NewError(connect.CodeInternal, errors.New("project operation failed"))
}

package authcli

import (
	"io"

	authapp "github.com/vrooli/vrooli/internal/app/auth"
	"github.com/vrooli/vrooli/internal/cliout"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// authStatusResponse maps the internal auth report onto the vrooli.cli.v1 wire
// contract. A proto field rename breaks this mapping at compile time.
func authStatusResponse(report authapp.Report) *cliv1.AuthStatusResponse {
	data := &cliv1.AuthStatusData{}
	for _, status := range report.Statuses {
		data.Statuses = append(data.Statuses, &cliv1.AuthToolStatus{
			Name: status.Name,
			Result: &cliv1.AuthProbeResult{
				State:         string(status.Result.State),
				Detail:        status.Result.Detail,
				SignInCommand: status.Result.SignInCommand,
			},
		})
	}
	return &cliv1.AuthStatusResponse{Success: true, Data: data}
}

func writeAuthStatusJSON(w io.Writer, report authapp.Report) error {
	return cliout.WriteProtoJSON(w, authStatusResponse(report))
}

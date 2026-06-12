package authcli

import (
	"fmt"
	"io"
	"strings"

	authapp "github.com/vrooli/vrooli/internal/app/auth"
	"github.com/vrooli/vrooli/internal/cliout"
)

func RenderStatus(w io.Writer, format cliout.Format, report authapp.Report) error {
	if format == cliout.FormatJSON {
		return writeAuthStatusJSON(w, report)
	}
	rows := make([][]string, 0, len(report.Statuses))
	for _, s := range report.Statuses {
		signIn := strings.Join(s.Result.SignInCommand, " ")
		rows = append(rows, []string{
			s.Name,
			string(s.Result.State),
			s.Result.Detail,
			signIn,
		})
	}
	if err := cliout.RenderTable(w, []string{"Tool", "State", "Detail", "Sign-in command"}, rows); err != nil {
		return err
	}
	signedOutOrExpired := false
	for _, s := range report.Statuses {
		switch s.Result.State {
		case authapp.StateSignedOut, authapp.StateExpired:
			signedOutOrExpired = true
		}
	}
	if signedOutOrExpired {
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Tools above with state=signed_out or state=expired need attention.")
	}
	return nil
}

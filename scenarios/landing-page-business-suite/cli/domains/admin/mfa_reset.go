package admin

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite/cli/internal/support"
)

// runMFAReset is the recovery path for an administrator who lost both the
// authenticator and the recovery codes. It authenticates with the LPBS
// service credential from the credential authority, so only an operator on a
// host that holds that credential can use it; a browser session cannot.
func runMFAReset(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("admin-mfa-reset", flag.ContinueOnError)
	email := fs.String("email", "", "Administrator email whose two-factor authentication to turn off (defaults to the seeded admin)")
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-mfa-reset [--email <email>]")
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return fmt.Errorf("initialize credential authority: %w", err)
	}
	secret, err := authority.Require(credentialauthority.Identity(support.LPBSCredentialIdentity), "service-secret")
	if err != nil {
		return fmt.Errorf("resolve the LPBS service credential (run on the host that owns it): %w", err)
	}
	target, err := deps.ResolveURL("/admin/mfa/reset", false, nil)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"email": support.ResolveAdminEmail(*email)})
	request, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(secret))
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return fmt.Errorf("reset two-factor authentication: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("reset two-factor authentication: %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	fmt.Printf("Two-factor authentication is off for %s. Sign in with the password, then set it up again from Admin → Profile.\n", support.ResolveAdminEmail(*email))
	return nil
}

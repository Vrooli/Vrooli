package admin

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite/cli/internal/support"
)

// runCredentialReset is the operator recovery path for a locked-out
// administrator. It runs against the local API by default and can target a
// stored remote profile, so a local control plane can recover a deployment
// without an interactive session on the remote host.
//
// The new password is read from stdin and never appears in argv, logs, or the
// response.
func runCredentialReset(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("admin-credential-reset", flag.ContinueOnError)
	email := fs.String("email", "", "Administrator email to reset (defaults to the seeded admin)")
	newEmail := fs.String("new-email", "", "Optional replacement sign-in email")
	passwordStdin := fs.Bool("new-password-stdin", false, "Read the new administrator password from stdin (required)")
	profileIDFlag := fs.String("profile-id", "", "Remote profile id to recover (alternative to --profile-tag)")
	profileTag := fs.String("profile-tag", "", "Remote profile tag to recover (for example: prod)")
	tagAlias := fs.String("tag", "", "Alias for --profile-tag")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: admin-credential-reset --new-password-stdin [--email <email>] [--new-email <email>] [--profile-tag <tag>]")
	}
	if !*passwordStdin {
		return fmt.Errorf("--new-password-stdin is required so the new password never appears in argv")
	}
	resolvedEmail := support.ResolveAdminEmail(*email)
	resolvedNewEmail := strings.TrimSpace(*newEmail)
	if strings.TrimSpace(*email) == "" && resolvedNewEmail == "" {
		return fmt.Errorf("provide --email and/or --new-email")
	}
	passwordBytes, err := io.ReadAll(io.LimitReader(os.Stdin, 8<<10))
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	newPassword := strings.TrimSpace(string(passwordBytes))
	if newPassword == "" {
		return fmt.Errorf("new password is empty")
	}
	payload, err := json.Marshal(map[string]string{
		"email":        resolvedEmail,
		"new_email":    resolvedNewEmail,
		"new_password": newPassword,
	})
	if err != nil {
		return fmt.Errorf("encode reset request: %w", err)
	}

	profileID := strings.TrimSpace(*profileIDFlag)
	tag := strings.TrimSpace(*profileTag)
	if tag == "" {
		tag = strings.TrimSpace(*tagAlias)
	}
	if profileID == "" && tag != "" {
		profileID, err = deps.ResolveRemoteProfileIDByTag(tag)
		if err != nil {
			return err
		}
	}

	var response []byte
	if profileID != "" {
		headers := map[string]string{"Content-Type": "application/json"}
		response, err = deps.RequestRemoteProxy(profileID, http.MethodPost, "/admin/admin-credentials/reset", nil, headers, payload)
		if err != nil {
			return fmt.Errorf("reset administrator credential: %w", err)
		}
	} else {
		response, err = postCredentialResetWithServiceSecret(deps, payload)
		if err != nil {
			return err
		}
	}
	if *jsonOut && len(response) > 0 {
		cliutil.PrintJSON(response)
		return nil
	}
	fmt.Printf("Administrator credential reset for %s.\n", resolvedEmail)
	return nil
}

// postCredentialResetWithServiceSecret authenticates the local recovery call
// with the LPBS service credential from the credential authority, so only an
// operator on a host that holds that credential can use it.
func postCredentialResetWithServiceSecret(deps support.Dependencies, payload []byte) ([]byte, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, fmt.Errorf("initialize credential authority: %w", err)
	}
	secret, err := authority.Require(credentialauthority.Identity(support.LPBSCredentialIdentity), "service-secret")
	if err != nil {
		return nil, fmt.Errorf("resolve the LPBS service credential (run on the host that owns it): %w", err)
	}
	target, err := deps.ResolveURL("/admin/admin-credentials/reset", false, nil)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(secret))
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("reset administrator credential: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reset administrator credential: %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"connectrpc.com/connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity/identity_v1connect"
	"golang.org/x/term"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core     *cliapp.ScenarioApp
	client   identityconnect.IdentityServiceClient
	password passwordSource
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:     core,
		client:   identityconnect.NewIdentityServiceClient(httpClient, baseURL),
		password: newPasswordSource(),
	}
}

// login obtains an owner JWT through the same Bridge facade the UI uses. The
// password is only read from a masked TTY prompt or stdin; it is never an argv
// flag, environment variable, report field, or log line.
func (h *handlers) login(ctx cliapp.RunContext) error {
	email := strings.TrimSpace(ctx.Flag("email"))
	if email == "" {
		return fmt.Errorf("--email is required")
	}
	password, err := h.password.resolve(ctx.BoolFlag("password-stdin"))
	if err != nil {
		return err
	}
	defer clear(password)
	if len(password) == 0 {
		return fmt.Errorf("password must not be empty")
	}

	resp, err := h.client.Login(context.Background(), connect.NewRequest(&identityv1.LoginRequest{
		Email:    email,
		Password: string(password),
	}))
	if err != nil {
		return cliapp.WrapAPIError("sign in", err, nil)
	}
	if resp == nil || resp.Msg == nil || strings.TrimSpace(resp.Msg.Token) == "" {
		return fmt.Errorf("sign in returned no access token")
	}

	// Do not alter an existing token until a complete new login has succeeded.
	h.core.Config.Token = resp.Msg.Token
	if err := h.core.SaveConfig(); err != nil {
		return fmt.Errorf("save owner session: %w", err)
	}
	// Deliberately custom output: generic proto rendering could expose Token.
	fmt.Fprintf(ctx.Stdout(), "Signed in as %s. Owner session saved for this CLI.\n", displayIdentity(resp.Msg.Email, resp.Msg.UserId))
	return nil
}

func displayIdentity(email, userID string) string {
	if strings.TrimSpace(email) != "" {
		return email
	}
	if strings.TrimSpace(userID) != "" {
		return userID
	}
	return "owner"
}

type passwordSource struct {
	isTerminal func() bool
	readSecret func() ([]byte, error)
	stdin      io.Reader
	prompt     io.Writer
}

func newPasswordSource() passwordSource {
	fd := int(os.Stdin.Fd())
	return passwordSource{
		isTerminal: func() bool { return term.IsTerminal(fd) },
		readSecret: func() ([]byte, error) { return term.ReadPassword(fd) },
		stdin:      os.Stdin,
		prompt:     os.Stderr,
	}
}

// resolve prompts by default because `auth login` is explicitly interactive.
// --password-stdin makes non-interactive use possible without placing a secret
// in argv or an environment variable.
func (p passwordSource) resolve(fromStdin bool) ([]byte, error) {
	if fromStdin {
		raw, err := io.ReadAll(p.stdin)
		if err != nil {
			return nil, fmt.Errorf("read password from stdin: %w", err)
		}
		return []byte(strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")), nil
	}
	if !p.isTerminal() {
		return nil, fmt.Errorf("auth login needs a TTY; use --password-stdin to provide the password securely")
	}
	fmt.Fprint(p.prompt, "Bridge password: ")
	secret, err := p.readSecret()
	fmt.Fprintln(p.prompt)
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}
	return secret, nil
}

func clear(secret []byte) {
	for i := range secret {
		secret[i] = 0
	}
}

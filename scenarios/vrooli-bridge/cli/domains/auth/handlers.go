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
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	core          *cliapp.ScenarioApp
	client        identityconnect.IdentityServiceClient
	password      passwordSource
	localExchange func(context.Context) (string, string, error)
	tokenFile     func() (string, error)
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{
		core:          core,
		client:        identityconnect.NewIdentityServiceClient(httpClient, baseURL),
		password:      newPasswordSource(),
		localExchange: session.ExchangeLocal,
		tokenFile:     session.TokenFile,
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
	if h.localExchange != nil {
		if token, refresh, err := h.localExchange(context.Background()); err == nil {
			if err := h.saveSession(token, refresh); err != nil {
				return err
			}
			fmt.Fprintln(ctx.Stdout(), "Signed in through the local machine binding. Owner session saved for this CLI.")
			return nil
		}
	}
	if h.tokenFile != nil {
		if token, err := h.tokenFile(); err == nil {
			if err := h.saveSession(token, ""); err != nil {
				return err
			}
			fmt.Fprintln(ctx.Stdout(), "Loaded the owner session from the configured token file.")
			return nil
		}
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
	if err := h.saveSession(resp.Msg.Token, resp.Msg.RefreshToken); err != nil {
		return err
	}
	// Deliberately custom output: generic proto rendering could expose Token.
	fmt.Fprintf(ctx.Stdout(), "Signed in as %s. Owner session saved for this CLI.\n", displayIdentity(resp.Msg.Email, resp.Msg.UserId))
	return nil
}

func (h *handlers) saveSession(token, refresh string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("sign in returned no access token")
	}
	h.core.Config.Token = token
	h.core.Config.RefreshToken = refresh
	if err := h.core.SaveConfig(); err != nil {
		return fmt.Errorf("save owner session: %w", err)
	}
	return nil
}

func (h *handlers) refresh(ctx cliapp.RunContext) error {
	if strings.TrimSpace(h.core.Config.RefreshToken) == "" {
		return fmt.Errorf("no refresh token is saved; run `auth login` first")
	}
	resp, err := h.client.Refresh(context.Background(), connect.NewRequest(&identityv1.RefreshRequest{
		RefreshToken: h.core.Config.RefreshToken,
	}))
	if err != nil {
		return cliapp.WrapAPIError("refresh owner session", err, nil)
	}
	if resp == nil || resp.Msg == nil || strings.TrimSpace(resp.Msg.Token) == "" || strings.TrimSpace(resp.Msg.RefreshToken) == "" {
		return fmt.Errorf("refresh returned no complete owner session")
	}
	previous := h.core.Config
	h.core.Config.Token = resp.Msg.Token
	h.core.Config.RefreshToken = resp.Msg.RefreshToken
	if err := h.core.SaveConfig(); err != nil {
		h.core.Config = previous
		return fmt.Errorf("save refreshed owner session: %w", err)
	}
	fmt.Fprintln(ctx.Stdout(), "Owner session refreshed and saved.")
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

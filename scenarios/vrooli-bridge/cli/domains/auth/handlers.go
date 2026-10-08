package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity/identity_v1connect"
	"golang.org/x/term"

	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	core          *cliapp.ScenarioApp
	client        identityClient
	password      passwordSource
	localExchange func(context.Context) (string, string, error)
	tokenFile     func() (string, error)
	enroll        func(context.Context, *cliapp.ScenarioApp, string) error
	mintLocal     func(time.Time) (string, error)
}

// identityClient is deliberately the subset used by auth commands. Keeping
// this seam narrow means adding an owner-session RPC cannot invalidate every
// auth test double or force unrelated callers to implement it.
type identityClient interface {
	Login(context.Context, *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error)
	Refresh(context.Context, *connect.Request[identityv1.RefreshRequest]) (*connect.Response[identityv1.RefreshResponse], error)
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{
		core:          core,
		client:        identityconnect.NewIdentityServiceClient(httpClient, baseURL),
		password:      newPasswordSource(),
		localExchange: session.ExchangeLocal,
		tokenFile:     session.TokenFile,
		enroll: func(ctx context.Context, app *cliapp.ScenarioApp, token string) error {
			return session.EnrollLocalWithToken(ctx, app, token)
		},
		mintLocal: func(now time.Time) (string, error) {
			token, _, err := session.ResolveLocal(now)
			return token, err
		},
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
		if token, _, err := h.localExchange(context.Background()); err == nil {
			if err := h.enroll(context.Background(), h.core, token); err == nil {
				// The durable local enrollment replaces the bearer-token config;
				// the access token is intentionally not persisted by this path.
				h.core.Config.Token = ""
				h.core.Config.RefreshToken = ""
				if err := h.core.SaveConfig(); err != nil {
					return fmt.Errorf("save local operator enrollment: %w", err)
				}
				fmt.Fprintln(ctx.Stdout(), "Enrolled through the local machine binding. Future owner sessions will be minted locally.")
				return nil
			}
		}
	}
	if h.tokenFile != nil {
		if token, err := h.tokenFile(); err == nil {
			if err := h.enroll(context.Background(), h.core, token); err != nil {
				return err
			}
			fmt.Fprintln(ctx.Stdout(), "Enrolled through the configured owner token. Future owner sessions will be minted locally.")
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

	if err := h.enroll(context.Background(), h.core, resp.Msg.Token); err != nil {
		return fmt.Errorf("enroll local owner session: %w", err)
	}
	// Deliberately custom output: generic proto rendering could expose Token.
	fmt.Fprintf(ctx.Stdout(), "Signed in as %s. Local owner enrollment saved; future sessions will be minted locally.\n", displayIdentity(resp.Msg.Email, resp.Msg.UserId))
	return nil
}

func (h *handlers) refresh(ctx cliapp.RunContext) error {
	if h.mintLocal == nil {
		return fmt.Errorf("local operator-session resolver is not configured")
	}
	if _, err := h.mintLocal(time.Now()); err != nil {
		return fmt.Errorf("refresh local owner session: %w", err)
	}
	fmt.Fprintln(ctx.Stdout(), "Local owner session refreshed without contacting the identity provider.")
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

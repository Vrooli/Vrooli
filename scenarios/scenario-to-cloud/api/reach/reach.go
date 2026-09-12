// Package reach is the one seam through which scenario-to-cloud touches a
// target machine. A command is a typed vrooli verb with an argument vector;
// there is no shell-string payload and no implicit transport. The transport
// is selected by the deployment's target binding and never falls back: a
// revoked enrollment or an unavailable Bridge is a typed refusal, not an SSH
// retry (D3, D11).
package reach

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/identity"
)

// Command is one typed target-local control-plane invocation. Verb is the
// space-separated vrooli command path (for example "cloud-target receipt
// get"); Args are passed verbatim as separate argv entries. RequiredScope is
// the governed scope the transport must hold (for example "vrooli:write").
type Command struct {
	Verb          string
	Args          []string
	RequiredScope string
	Effectful     bool
	Timeout       time.Duration
	Stdin         []byte
	// Program names a read-only host observation program (see
	// ObservationPrograms) instead of a vrooli verb. It exists for the few
	// facts the target owner does not report yet (disk, sockets, processes,
	// files under the bound workdir). A Program command is never effectful,
	// runs outside the bound workdir's vrooli binary and is refused by
	// transports that only relay scenario verbs.
	Program string
	// Observation is the typed target-owner observation contract. Program is
	// retained only for compatibility with older callers; new callers must use
	// NewObservation so transports carry a cloud-target owner verb instead of
	// an arbitrary host executable.
	Observation *ObservationSpec
}

// ObservationSpec names a bounded read owned by the target control plane.
// Args are semantic arguments for Kind and are validated again by the target
// owner before it performs the read.
type ObservationSpec struct {
	Kind string
	Args []string
}

// NewObservation converts the historical probe name into the canonical typed
// target observation operation. The mapping is deliberately closed: adding a
// host fact requires an owner operation and its authorization tests.
func NewObservation(program string, args ...string) (Command, error) {
	kind, ok := ObservationKinds[strings.TrimSpace(program)]
	if !ok {
		return Command{}, &Error{Kind: KindInvalidArgument, Detail: "program " + strings.TrimSpace(program) + " has no typed target observation"}
	}
	return Command{Program: strings.TrimSpace(program), Observation: &ObservationSpec{Kind: kind, Args: append([]string(nil), args...)}, RequiredScope: "vrooli:read", Timeout: DefaultObservationTimeout}, nil
}

const DefaultObservationTimeout = 45 * time.Second

// ObservationKinds is the compatibility map from old probe names to typed
// owner operations. The program name never crosses the transport boundary.
var ObservationKinds = map[string]string{
	"cat":        "file",
	"df":         "disk",
	"du":         "directory_usage",
	"find":       "directory_entries",
	"grep":       "grep",
	"journalctl": "journal",
	"ls":         "directory_listing",
	"pgrep":      "process_match",
	"ps":         "processes",
	"ss":         "sockets",
	"stat":       "file_stat",
	"uname":      "os",
	"head":       "file_head",
	"sudo":       "privilege",
}

// ObservationPrograms is the closed set of host programs a Program command
// may name. Every entry is a read-only inspection tool; nothing here can
// change host state whatever its arguments. Adding an entry is an
// architecture decision (archtest pins the set).
var ObservationPrograms = map[string]bool{
	"cat":        true,
	"df":         true,
	"du":         true,
	"find":       true,
	"grep":       true,
	"sudo":       true,
	"journalctl": true,
	"ls":         true,
	"pgrep":      true,
	"ps":         true,
	"ss":         true,
	"stat":       true,
	"uname":      true,
	"head":       true,
}

// IsObservation reports whether the command names a host observation
// program rather than a vrooli verb.
func (c Command) IsObservation() bool {
	return c.Observation != nil || strings.TrimSpace(c.Program) != ""
}

// Argv returns the full argument vector for the command: the vrooli verb
// path, or the observation program, followed by the arguments.
func (c Command) Argv() []string {
	if c.Observation != nil {
		argv := []string{"vrooli", "cloud-target", "host", "observe", "--kind", c.Observation.Kind}
		for _, arg := range c.Observation.Args {
			argv = append(argv, "--arg", arg)
		}
		return argv
	}
	if c.IsObservation() {
		return append([]string{c.Program}, c.Args...)
	}
	argv := append([]string{"vrooli"}, strings.Fields(c.Verb)...)
	return append(argv, c.Args...)
}

// Result is what the target answered. RunID and CorrelationID are the
// transport's durable identities so an operation can reattach after a
// disconnect instead of creating a second job.
type Result struct {
	ExitCode      int
	Stdout        string
	Stderr        string
	RunID         string
	CorrelationID string
	Transport     string
}

// Capabilities is the negotiated state of one target before dispatch.
// Platform is "<goos>/<goarch>" when the transport can observe it; release
// delivery selects the native control-plane binary from it.
type Capabilities struct {
	Transport       string
	Online          bool
	ProtocolVersion string
	Scopes          []string
	NativeCLI       bool
	Platform        string
}

// ArtifactFile is one local file to place on the target. Mode 0 keeps the
// transport default (0644); SHA256 is the local digest the receipt repeats so
// the target owner can verify the bytes independently.
type ArtifactFile struct {
	Role       string
	LocalPath  string
	RemotePath string
	Mode       uint32
	SHA256     string
}

// Delivery is one artifact set placed on the target by the bound transport.
type Delivery struct {
	Files []ArtifactFile
}

// DeliveredFile is what the transport reports for one placed file.
type DeliveredFile struct {
	Role       string `json:"role"`
	RemotePath string `json:"remote_path"`
	SHA256     string `json:"sha256"`
}

// DeliveryReceipt is the transport's report of a delivery.
type DeliveryReceipt struct {
	Transport string          `json:"transport"`
	Files     []DeliveredFile `json:"files"`
}

// Reach executes typed commands against a target, places artifacts on it and
// negotiates its capabilities. Implementations: bridge (nodereach) and
// sshadapter. Deliver is the only byte-transfer seam; everything else is a
// typed verb.
type Reach interface {
	Exec(ctx context.Context, target identity.TargetRef, cmd Command) (Result, error)
	Deliver(ctx context.Context, target identity.TargetRef, delivery Delivery) (DeliveryReceipt, error)
	Negotiate(ctx context.Context, target identity.TargetRef) (Capabilities, error)
}

// ValidateDelivery refuses a delivery whose remote paths are relative or
// could be read as shell syntax by a joining transport.
func ValidateDelivery(delivery Delivery) error {
	if len(delivery.Files) == 0 {
		return &Error{Kind: KindInvalidArgument, Detail: "delivery carries no files"}
	}
	for i, f := range delivery.Files {
		if strings.TrimSpace(f.LocalPath) == "" {
			return &Error{Kind: KindInvalidArgument, Detail: fmt.Sprintf("file %d has no local path", i)}
		}
		if !strings.HasPrefix(f.RemotePath, "/") {
			return &Error{Kind: KindInvalidArgument, Detail: fmt.Sprintf("file %d remote path must be absolute", i)}
		}
		if err := ValidateArgs([]string{f.RemotePath}); err != nil {
			return err
		}
	}
	return nil
}

// Kind classifies a reach failure without inspecting message text.
type Kind string

const (
	KindUnavailable         Kind = "reach_unavailable"
	KindTargetOffline       Kind = "target_offline"
	KindScopeMissing        Kind = "reach_scope_missing"
	KindProtocolUnsupported Kind = "reach_protocol_unsupported"
	KindEnrollmentRevoked   Kind = "enrollment_revoked"
	KindInvalidArgument     Kind = "invalid_request"
	KindTransport           Kind = "transport_failure"
)

// Error is a typed reach failure. Kind maps to a stable apierrors code.
type Error struct {
	Kind      Kind
	Transport string
	Target    string
	Detail    string
	Err       error
}

func (e *Error) Error() string {
	parts := []string{string(e.Kind)}
	if e.Transport != "" {
		parts = append(parts, "transport="+e.Transport)
	}
	if e.Target != "" {
		parts = append(parts, "target="+e.Target)
	}
	if e.Detail != "" {
		parts = append(parts, e.Detail)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *Error) Unwrap() error { return e.Err }

// IsKind reports whether err is a reach error of the given kind.
func IsKind(err error, kind Kind) bool {
	var re *Error
	return errors.As(err, &re) && re.Kind == kind
}

// APIError converts a reach error to the shared typed error model.
func APIError(err error) *apierrors.Error {
	var re *Error
	if !errors.As(err, &re) {
		return apierrors.Internal("reach failed", err)
	}
	code := apierrors.CodeReachUnavailable
	switch re.Kind {
	case KindTargetOffline:
		code = apierrors.CodeTargetOffline
	case KindScopeMissing:
		code = apierrors.CodeReachScopeMissing
	case KindProtocolUnsupported:
		code = apierrors.CodeReachProtocolUnsupported
	case KindEnrollmentRevoked:
		code = apierrors.CodeEnrollmentRevoked
	case KindInvalidArgument:
		code = apierrors.CodeInvalidRequest
	}
	out := apierrors.New(code, re.Error()).WithRetryable(re.Kind == KindTargetOffline || re.Kind == KindTransport)
	if re.Transport != "" {
		out = out.WithDetail("transport", re.Transport)
	}
	if re.Target != "" {
		out = out.WithDetail("target", re.Target)
	}
	return out
}

// ValidateArgs refuses any argument that could be read as shell syntax by a
// transport that must join argv into one remote string. Cloud composes
// identifiers, flags, digests and paths; none of them legitimately contain
// these bytes.
func ValidateArgs(args []string) error {
	for i, arg := range args {
		if arg == "" {
			return &Error{Kind: KindInvalidArgument, Detail: fmt.Sprintf("argument %d is empty", i)}
		}
		if len(arg) > 4096 {
			return &Error{Kind: KindInvalidArgument, Detail: fmt.Sprintf("argument %d exceeds 4096 bytes", i)}
		}
		if strings.ContainsAny(arg, ";|&$`<>\n\r\x00'\"\\") {
			return &Error{Kind: KindInvalidArgument, Detail: fmt.Sprintf("argument %d contains shell syntax", i)}
		}
	}
	return nil
}

// ValidateCommand checks the verb and arguments before any transport work.
// An observation program must come from ObservationPrograms, carry no verb,
// no stdin and no effect.
func ValidateCommand(cmd Command) error {
	if cmd.Observation != nil {
		if cmd.Observation == nil || strings.TrimSpace(cmd.Observation.Kind) == "" {
			return &Error{Kind: KindInvalidArgument, Detail: "typed observation kind is required"}
		}
		if strings.TrimSpace(cmd.Verb) != "" {
			return &Error{Kind: KindInvalidArgument, Detail: "a command names a typed observation or a verb, not both"}
		}
		if cmd.Effectful || len(cmd.Stdin) > 0 {
			return &Error{Kind: KindInvalidArgument, Detail: "typed observations are read-only and take no stdin"}
		}
		known := false
		for _, kind := range ObservationKinds {
			if kind == cmd.Observation.Kind {
				known = true
				break
			}
		}
		if !known {
			return &Error{Kind: KindInvalidArgument, Detail: "unknown typed observation " + cmd.Observation.Kind}
		}
		return ValidateArgs(cmd.Observation.Args)
	}
	if cmd.IsObservation() {
		switch {
		case !ObservationPrograms[cmd.Program]:
			return &Error{Kind: KindInvalidArgument, Detail: "program " + cmd.Program + " is not an observation program"}
		case cmd.Effectful:
			return &Error{Kind: KindInvalidArgument, Detail: "observation programs are never effectful"}
		case strings.TrimSpace(cmd.Verb) != "":
			return &Error{Kind: KindInvalidArgument, Detail: "a command names a verb or a program, not both"}
		case len(cmd.Stdin) > 0:
			return &Error{Kind: KindInvalidArgument, Detail: "observation programs take no stdin"}
		}
		if cmd.Program == "sudo" && !sameArgs(cmd.Args, "-n", "-l") {
			return &Error{Kind: KindInvalidArgument, Detail: "sudo observation is limited to non-interactive privilege listing"}
		}
		return ValidateArgs(cmd.Args)
	}
	fields := strings.Fields(cmd.Verb)
	if len(fields) == 0 {
		return &Error{Kind: KindInvalidArgument, Detail: "verb is empty"}
	}
	for _, f := range fields {
		for _, r := range f {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			default:
				return &Error{Kind: KindInvalidArgument, Detail: "verb must be lowercase words separated by spaces"}
			}
		}
	}
	return ValidateArgs(cmd.Args)
}

func sameArgs(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// RevocationPolicy answers whether a target's authority has been withdrawn.
// It is consulted before every Exec so a revoked enrollment cannot be
// reached by any transport.
type RevocationPolicy interface {
	Revoked(ctx context.Context, target identity.TargetRef) (revoked bool, reason string, err error)
}

// Router selects the adapter from the target binding's explicit transport.
// A binding without a transport, or a transport with no configured adapter,
// is reach_unavailable; the router never tries another transport.
type Router struct {
	Bridge     Reach
	SSH        Reach
	Revocation RevocationPolicy
}

var _ Reach = (*Router)(nil)

func (r *Router) adapterFor(target identity.TargetRef) (Reach, error) {
	switch target.Transport {
	case identity.TransportBridge:
		if r.Bridge == nil {
			return nil, &Error{Kind: KindUnavailable, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge reach is not configured"}
		}
		if strings.TrimSpace(target.NodeID) == "" {
			return nil, &Error{Kind: KindEnrollmentRevoked, Transport: identity.TransportBridge, Target: target.Key(), Detail: "target binding has no enrolled node"}
		}
		return r.Bridge, nil
	case identity.TransportSSH:
		if r.SSH == nil {
			return nil, &Error{Kind: KindUnavailable, Transport: identity.TransportSSH, Target: target.Key(), Detail: "ssh reach is not configured"}
		}
		if strings.TrimSpace(target.Locator.Host) == "" {
			return nil, &Error{Kind: KindUnavailable, Transport: identity.TransportSSH, Target: target.Key(), Detail: "target binding has no host locator"}
		}
		return r.SSH, nil
	default:
		return nil, &Error{Kind: KindUnavailable, Target: target.Key(), Detail: "target binding selects no transport; bind the deployment to a machine first"}
	}
}

func (r *Router) checkRevocation(ctx context.Context, target identity.TargetRef) error {
	if r.Revocation == nil {
		return nil
	}
	revoked, reason, err := r.Revocation.Revoked(ctx, target)
	if err != nil {
		return &Error{Kind: KindUnavailable, Target: target.Key(), Detail: "revocation state unknown", Err: err}
	}
	if revoked {
		return &Error{Kind: KindEnrollmentRevoked, Transport: target.Transport, Target: target.Key(), Detail: reason}
	}
	return nil
}

// Exec validates, checks revocation, then dispatches on the bound transport.
func (r *Router) Exec(ctx context.Context, target identity.TargetRef, cmd Command) (Result, error) {
	if err := ValidateCommand(cmd); err != nil {
		return Result{}, err
	}
	if err := r.checkRevocation(ctx, target); err != nil {
		return Result{}, err
	}
	adapter, err := r.adapterFor(target)
	if err != nil {
		return Result{}, err
	}
	return adapter.Exec(ctx, target, cmd)
}

// Deliver validates, checks revocation, then places the files through the
// bound transport.
func (r *Router) Deliver(ctx context.Context, target identity.TargetRef, delivery Delivery) (DeliveryReceipt, error) {
	if err := ValidateDelivery(delivery); err != nil {
		return DeliveryReceipt{}, err
	}
	if err := r.checkRevocation(ctx, target); err != nil {
		return DeliveryReceipt{}, err
	}
	adapter, err := r.adapterFor(target)
	if err != nil {
		return DeliveryReceipt{}, err
	}
	return adapter.Deliver(ctx, target, delivery)
}

// Negotiate reports the bound transport's capabilities for the target.
func (r *Router) Negotiate(ctx context.Context, target identity.TargetRef) (Capabilities, error) {
	if err := r.checkRevocation(ctx, target); err != nil {
		return Capabilities{}, err
	}
	adapter, err := r.adapterFor(target)
	if err != nil {
		return Capabilities{}, err
	}
	return adapter.Negotiate(ctx, target)
}

// RequireScope refuses when the negotiated scopes do not cover the required
// one. Wildcards follow the shared catalog rule (exact, "*", "ns:*", "*:effect").
func RequireScope(held []string, required string) error {
	if required == "" {
		return nil
	}
	ns, effect, _ := strings.Cut(required, ":")
	for _, h := range held {
		if h == required || h == "*" || h == ns+":*" || h == "*:"+effect {
			return nil
		}
	}
	return &Error{Kind: KindScopeMissing, Detail: "target does not hold scope " + required}
}

// SessionSpec describes the interactive session a caller wants on a target.
// Term and the initial size are presentation facts; the shell is the
// target's login shell in the bound workdir.
type SessionSpec struct {
	Term string
	Cols int
	Rows int
}

// Session is one interactive PTY on a target. Reads deliver the merged
// terminal stream; writes go to the session's stdin. Close ends the session
// and releases the transport; Wait blocks until the remote side ends it.
type Session interface {
	io.Reader
	io.Writer
	Resize(rows, cols int) error
	Wait() error
	Close() error
}

// SessionOpener is implemented by transports that can open an interactive
// session. The SSH adapter keeps its PTY implementation; the bridge adapter
// opens the node's session channel. A transport without a session
// capability is a typed reach_protocol_unsupported, never a fallback.
type SessionOpener interface {
	OpenSession(ctx context.Context, target identity.TargetRef, spec SessionSpec) (Session, error)
}

// OpenSession opens a session on the transport the target binding selects,
// after the same revocation check every Exec performs.
func (r *Router) OpenSession(ctx context.Context, target identity.TargetRef, spec SessionSpec) (Session, error) {
	if err := r.checkRevocation(ctx, target); err != nil {
		return nil, err
	}
	adapter, err := r.adapterFor(target)
	if err != nil {
		return nil, err
	}
	opener, ok := adapter.(SessionOpener)
	if !ok {
		return nil, &Error{Kind: KindProtocolUnsupported, Transport: target.Transport, Target: target.Key(), Detail: "transport has no interactive session capability"}
	}
	return opener.OpenSession(ctx, target, spec)
}

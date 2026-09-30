// Package failures owns the stable, redacted failure vocabulary shared by
// preflight, execution, health, and presentation. Human messages are evidence
// for an operator; Code and Scope are the durable contract.
package failures

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Category string

const (
	CategoryDestination  Category = "destination"
	CategoryCredential   Category = "credential"
	CategoryRepository   Category = "repository"
	CategorySource       Category = "source"
	CategoryCapacity     Category = "capacity"
	CategoryPlatform     Category = "platform"
	CategoryExecution    Category = "execution"
	CategoryVerification Category = "verification"
	CategoryPersistence  Category = "persistence"
)

// Code is intentionally a string rather than an enum so new codes can be
// introduced without breaking old clients. These values are safe to persist
// and expose in JSON.
type Code string

const (
	// #nosec G101 -- these are public failure-code vocabulary values, never credentials.
	DestinationMissing      Code = "destination_missing"
	DestinationUnmounted    Code = "destination_unmounted"
	DestinationDirty        Code = "destination_dirty"
	DestinationReadOnly     Code = "destination_read_only"
	DestinationWrongDevice  Code = "destination_wrong_device"
	DestinationInaccessible Code = "destination_inaccessible"
	DestinationCapacity     Code = "destination_capacity"
	CredentialMissing       Code = "credential_missing"    // #nosec G101 -- public failure vocabulary, not a credential.
	CredentialUnreadable    Code = "credential_unreadable" // #nosec G101 -- public failure vocabulary, not a credential.
	RepositoryInvalid       Code = "repository_invalid"
	SourceMissing           Code = "source_missing"
	SourceUnsupported       Code = "source_unsupported"
	UnsupportedPlatform     Code = "unsupported_platform"
	Transient               Code = "transient"
	RetentionFailed         Code = "retention_failed"
	CleanupFailed           Code = "cleanup_failed"
	VerificationFailed      Code = "verification_failed"
	PersistenceFailed       Code = "persistence_failed"
	ProcessInterrupted      Code = "process_interrupted"
	Unknown                 Code = "unknown"
)

type Scope string

const (
	ScopePlan        Scope = "plan"
	ScopeDestination Scope = "destination"
	ScopeTarget      Scope = "target"
	ScopeRun         Scope = "run"
)

// Cause is a redacted incident. Raw command output and secret values do not
// belong in this type.
type Cause struct {
	Code          Code
	Category      Category
	Scope         Scope
	Message       string
	NextAction    string
	Retryable     bool
	RetryAfter    time.Duration
	FirstObserved time.Time
	LastObserved  time.Time
	LastKnownGood time.Time
	DestinationID string
	TargetIDs     []string
}

func (c Cause) Error() string {
	if c.Message == "" {
		return string(c.Code)
	}
	return fmt.Sprintf("%s: %s", c.Code, c.Message)
}

// Classify converts errors from resource adapters into the stable vocabulary.
// Matching is deliberately conservative and only uses lower-cased, bounded
// error text; callers should replace Message with a redacted operator message.
func Classify(err error) Cause {
	if err == nil {
		return Cause{}
	}
	text := strings.ToLower(err.Error())
	c := Cause{Code: Unknown, Category: CategoryExecution, Scope: ScopeRun, Message: "backup operation failed"}
	switch {
	case strings.Contains(text, "credential is not configured"), strings.Contains(text, "passphrase") && strings.Contains(text, "not configured"):
		c.Code, c.Category, c.Scope = CredentialMissing, CategoryCredential, ScopeDestination
		c.Message, c.NextAction = "repository credential is unavailable", "restore the repository credential in the credential authority, then run readiness again"
	case strings.Contains(text, "credential authority"),
		strings.Contains(text, "read repository passphrase"),
		strings.Contains(text, "repository credential"):
		c.Code, c.Category, c.Scope = CredentialUnreadable, CategoryCredential, ScopeDestination
		c.Message, c.NextAction = "repository credential could not be read", "restore credential-authority access and retry readiness"
	case strings.Contains(text, "read-only"), strings.Contains(text, "read only"):
		c.Code, c.Category, c.Scope = DestinationReadOnly, CategoryDestination, ScopeDestination
		c.Message, c.NextAction = "destination is mounted read-only", "run destination readiness to attribute the cause, then remediate it under explicit confirmation"
	case strings.Contains(text, "not mounted"), strings.Contains(text, "unmounted"), strings.Contains(text, "no such file or directory"):
		c.Code, c.Category, c.Scope = DestinationUnmounted, CategoryDestination, ScopeDestination
		c.Message, c.NextAction = "destination is not mounted or its path is absent", "mount the intended volume, then recheck device identity and readiness"
	case strings.Contains(text, "dirty"), strings.Contains(text, "needs-check"), strings.Contains(text, "needs check"):
		c.Code, c.Category, c.Scope = DestinationDirty, CategoryDestination, ScopeDestination
		c.Message, c.NextAction = "destination filesystem reports a dirty or needs-check state", "check and repair the destination filesystem under explicit confirmation, then recheck readiness"
	case strings.Contains(text, "capacity"), strings.Contains(text, "storage cap"):
		c.Code, c.Category, c.Scope = DestinationCapacity, CategoryCapacity, ScopeDestination
		c.Message, c.NextAction = "destination capacity policy blocks this backup", "free capacity or change the destination policy deliberately"
	case strings.Contains(text, "repo status"), strings.Contains(text, "repository"), strings.Contains(text, "kopia"):
		c.Code, c.Category, c.Scope = RepositoryInvalid, CategoryRepository, ScopeDestination
		c.Message, c.NextAction = "repository readiness probe failed", "inspect the repository and credential reference before retrying"
	case strings.Contains(text, "no capturer"), strings.Contains(text, "unsupported source"):
		c.Code, c.Category, c.Scope = SourceUnsupported, CategorySource, ScopeTarget
		c.Message, c.NextAction = "source kind is not available on this platform", "use a supported source adapter or mark this target partial"
	case strings.Contains(text, "stat "), strings.Contains(text, "lstat "), strings.Contains(text, "source") && strings.Contains(text, "not found"):
		c.Code, c.Category, c.Scope = SourceMissing, CategorySource, ScopeTarget
		c.Message, c.NextAction = "source path is missing or inaccessible", "restore the source path or deregister the stale target"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		c.Code, c.Category, c.Scope = ProcessInterrupted, CategoryExecution, ScopeRun
		c.Message, c.NextAction = "backup process was interrupted", "retry after the service is healthy"
	}
	return c
}

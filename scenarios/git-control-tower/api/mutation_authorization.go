package main

import (
	"context"
	"fmt"
	"strings"

	"git-control-tower/internal/policygate"
	"github.com/vrooli/cli-core/cliutil"
)

// requireHumanMutation is the final domain-level guard for every repository
// writer. HTTP/Connect middleware is defense in depth, not the authority
// contract: callers that invoke a writer service directly must carry both a
// verified human principal and a consumed server-issued intent. Caller-
// supplied headers are intentionally invisible here.
func requireHumanMutation(ctx context.Context, operation string) error {
	if err := requireHumanPrincipal(ctx, operation); err != nil {
		return err
	}
	if _, ok := policygate.ConsumedIntentFromContext(ctx); !ok {
		return fmt.Errorf("consumed mutation intent is required for %s", operation)
	}
	intent, _ := policygate.ConsumedIntentFromContext(ctx)
	if expected := writerIntentOperation(operation); expected != "" && intent.Operation != "" && intent.Operation != expected {
		return fmt.Errorf("mutation intent for %s cannot authorize %s", intent.Operation, operation)
	}
	return nil
}

func requireHumanPrincipal(ctx context.Context, operation string) error {
	principal, ok := policygate.PrincipalFromContext(ctx)
	if !ok || principal.Kind != cliutil.CallerKindHuman {
		return fmt.Errorf("verified human authority is required for %s", operation)
	}
	return nil
}

func writerIntentOperation(operation string) string {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "create commit":
		return mutationOperationCommit
	case "stage files":
		return "repo.stage"
	case "unstage files":
		return "repo.unstage"
	case "discard files":
		return "repo.discard"
	case "push to remote", "upstream action":
		return "repo.push"
	case "pull from remote":
		return "repo.pull"
	case "create branch":
		return "repo.branch.create"
	case "switch branch":
		return "repo.branch.switch"
	case "publish branch":
		return "repo.branch.publish"
	case "move gitignore entry":
		return "repo.gitignore.move"
	case "install precommit hook":
		return "repo.precommit.hook.install"
	case "uninstall precommit hook":
		return "repo.precommit.hook.uninstall"
	case "delete path":
		return "repo.files.delete"
	case "save file content":
		return "repo.files.content"
	case "ignore path":
		return "repo.ignore"
	case "untrack binary":
		return "repo.tracked-binaries.untrack"
	case "update remote url":
		return "repo.remote.url"
	case "save credential", "delete credential":
		return "repo.credentials"
	case "save precommit configuration":
		return "repo.precommit"
	case "save grouping rules":
		return "repo.grouping-rules"
	case "set active repository":
		return "repo.active"
	case "remove repository":
		return "repo.remove"
	case "open repository":
		return "repo.open"
	case "clone repository":
		return "repo.clone"
	case "apply auditor fix":
		return auditorFixOperation
	default:
		return ""
	}
}

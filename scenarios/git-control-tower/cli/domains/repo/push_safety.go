package repo

import (
	"context"
	"flag"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	"google.golang.org/protobuf/encoding/protojson"
)

func runPushSafety(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("push-safety", flag.ContinueOnError)
	req := &repov1.InspectPushSafetyRequest{}
	jsonOutput := false
	fs.StringVar(&req.RepositoryId, "repo", "", "Repository ID")
	fs.StringVar(&req.Remote, "remote", "", "Remote name")
	fs.StringVar(&req.Branch, "branch", "", "Destination branch")
	fs.BoolVar(&jsonOutput, "json", false, "Typed JSON report")
	if e := fs.Parse(args); e != nil {
		return e
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	r, e := repoClientFactory(core).InspectPushSafety(context.Background(), connect.NewRequest(req))
	if e != nil {
		return e
	}
	if jsonOutput {
		b, e := (protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}).Marshal(r.Msg)
		if e != nil {
			return e
		}
		fmt.Println(string(b))
		return nil
	}
	formatPushSafety(r.Msg)
	return nil
}

func formatPushSafety(r *repov1.PushSafetyReport) {
	fmt.Printf("Push safety: %s\n%s\nSource: %s\nRemote base: %s\nOutgoing commits: %d\n", r.State, r.Reason, r.Head, r.Base, len(r.Commits))
	for _, f := range r.Files {
		fmt.Printf("  %.2f MiB — %v\n", float64(f.Bytes)/1048576, f.Paths)
	}
	if r.CanPrepare {
		fmt.Println("Open Push in the UI to review and authorize isolated preparation. Applying recovery requires a separate coordinated checkpoint.")
	}
	if r.Fingerprint != "" {
		fmt.Printf("Preview identity: %s\n", r.Fingerprint)
	}
}

func runRecoveryStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("push-recovery-status", flag.ContinueOnError)
	req := &repov1.GetPushRecoveryRequest{}
	jsonOutput := false
	fs.StringVar(&req.RepositoryId, "repo", "", "Repository ID")
	fs.StringVar(&req.Fingerprint, "fingerprint", "", "Operation ID (omit to find the most recently updated preparation for this repository)")
	fs.BoolVar(&jsonOutput, "json", false, "Typed JSON artifact")
	if e := fs.Parse(args); e != nil {
		return e
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	r, e := repoClientFactory(core).GetPushRecovery(context.Background(), connect.NewRequest(req))
	if e != nil {
		return e
	}
	if jsonOutput {
		b, e := (protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}).Marshal(r.Msg)
		if e != nil {
			return e
		}
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("Recovery: %s\nOperation ID: %s\n%s\nOriginal bundle: %s\nReplacement bundle: %s\n", r.Msg.State, r.Msg.Fingerprint, r.Msg.Message, r.Msg.OriginalBundle, r.Msg.RepairedBundle)
	return nil
}

func runPrepareRecovery(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("prepare-push-recovery", flag.ContinueOnError)
	fingerprint := ""
	fs.StringVar(&fingerprint, "fingerprint", "", "Exact reviewed preview identity")
	if e := fs.Parse(args); e != nil {
		return e
	}
	if fingerprint == "" || fs.NArg() != 0 {
		return fmt.Errorf("--fingerprint is required")
	}
	client := repoClientFactory(core)
	r, e := client.InspectPushSafety(context.Background(), connect.NewRequest(&repov1.InspectPushSafetyRequest{}))
	if e != nil {
		return e
	}
	if r.Msg.Fingerprint != fingerprint || !r.Msg.CanPrepare {
		return fmt.Errorf("preview is stale or not eligible; run repo push-safety again")
	}
	formatPushSafety(r.Msg)
	fmt.Println("Preparation excludes every version of the blocked paths from outgoing commits. It preserves other paths and committed history in isolated artifacts. It does not back up live files or the index, add ignore rules, apply history, or push. It may require several copies of committed history on disk.")
	i, e := confirmRepoMutationContext(core, "repo.recovery.prepare", fingerprint)
	if e != nil {
		return e
	}
	a, e := client.PreparePushRecovery(context.Background(), connect.NewRequest(&repov1.PreparePushRecoveryRequest{RepositoryId: i.repositoryID, IntentId: i.intentID, Remote: r.Msg.Remote, Branch: r.Msg.Branch, Fingerprint: fingerprint}))
	if e != nil {
		return e
	}
	b, e := (protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}).Marshal(a.Msg)
	if e != nil {
		return e
	}
	fmt.Println(string(b))
	return nil
}

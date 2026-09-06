# Discover supported commands before execution; adapt spelling only when the current CLI help proves it changed.
vrooli help
vrooli scenario requirements validate web-console
vrooli scenario requirements report web-console
business-health validate scenario web-console
storage-manager validate scenario web-console

# Start and test only through supported lifecycle surfaces. Capture each returned run ID.
vrooli scenario test web-console
test-genie runs wait --json web-console <web-console-run-id>
vrooli scenario test agent-manager
test-genie runs wait --json agent-manager <agent-manager-run-id>
vrooli scenario test search-hub
test-genie runs wait --json search-hub <search-hub-run-id>

# Contract, local-recovery, and reconciliation acceptance probes.
web-console conversation integrity --json --dry-run
web-console conversation inspect --json --thread-id 01a06a6b-88da-7422-b391-bb59c5f5e5e0
web-console conversation search --json --query "shared intent data type lifecycle" --state all
web-console conversation reconcile --json --dry-run --manifest <backup-manifest>
web-console conversation reconcile --json --apply --manifest <backup-manifest>
web-console conversation reconcile --json --apply --manifest <backup-manifest>

# Cross-run discovery acceptance probes; use current help to confirm exact filters.
agent-manager search query "shared intent data type lifecycle" --json
search-hub query "shared intent data type lifecycle" --type record --json

# Validate every new/changed skill and governed program using the current prompt-manager/program-runtime help.
prompt-manager skills validate web-console-conversation-recovery
prompt-manager skills validate web-console-conversation-improvement
program-runtime validate web-console-conversation-integrity-audit
program-runtime run web-console-conversation-integrity-audit -- --dry-run --json

# Repository review, baseline comparison, and residual bypass scan.
git-control-tower review unified --scenario web-console --json
rg -n "Store\\.Delete|time\\.AfterFunc|go func.*archive|_ = .*Archive|_ = .*Delete" scenarios/web-console/api scenarios/web-console/cli scenarios/web-console/ui


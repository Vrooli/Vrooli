# git-control-tower — Quickstart

Agent-friendly git repository control plane: proto/Connect-RPC API + CLI + UI.
Operates on the local Vrooli working tree to expose safe, structured git
operations for agents and humans.

## What you get

- A Connect-RPC API on `API_PORT` for typed status, settings, and mutation
  operations, plus REST endpoints only for remaining migration residue and
  documented transport exceptions.
- A CLI (`git-control-tower`) wrapping the same endpoints for shell use.
- A React UI on `UI_PORT` with a 3-pane layout (file list / diff /
  history), markdown preview for `.md` files, and AI-assisted commit
  message helpers.

## First run

```bash
# From the repo root
make setup                                       # one-time host setup
vrooli scenario start git-control-tower          # boots API + UI
vrooli scenario port  git-control-tower          # prints assigned ports
```

Open `http://localhost:<UI_PORT>` and you should see the file-list panel
populated with the current working-tree changes.

The CLI installs alongside (`git-control-tower --help`). Common flows:

```bash
git-control-tower repo status                    # working-tree status
git-control-tower repo diff <path>               # unified diff for a file
git-control-tower repo blame <path>              # native blame; add --enrich for linked evidence
git-control-tower branch list                    # branches + ahead/behind
git-control-tower review <slug>                  # scenario review report
```

## Authentication modes

The normal bundled desktop experience is private to the local operator and
does not require a sign-in. The desktop supervisor token used to start and
inspect local processes is not a human credential.

Choose an explicit deployment mode when the installation must support shared
users or remote access:

| Mode | Sign-in | Authority |
|---|---|---|
| `personal_local` | No human sign-in by default | Local app and OS boundary |
| `local_multi_user` | Required | `scenario-authenticator` |
| `remote_vrooli` | Required | `scenario-authenticator` plus deployment edge |
| `shared_provider` | Required | Declared shared identity provider |

LPBS website sign-in is separate from local GCT access. Link a local identity
to an LPBS business account only when paid features require it; do not copy
website tokens or infer a link from matching email addresses. See the project
[Identity and Authentication contract](../../../docs/concepts/IDENTITY-AND-AUTHENTICATION.md)
and the [authenticator integration](internal/AUTHENTICATOR-INTEGRATION.md).

## Where to go next

- **Architecture overview:** [docs/concepts/ARCHITECTURE.md][arch] — how
  API/CLI/UI fit together; planned [REQ: OT-P1-002], [REQ: OT-P1-003],
  [REQ: OT-P1-004], [REQ: OT-P2-002], [REQ: OT-P2-003].
- **API reference:** [docs/reference/api-endpoints.md][api]
- **CLI reference:** [docs/reference/cli-commands.md][cli]
- **Configuration:** [docs/reference/configuration.md][config]
- **Performance audits:** [docs/perf/][perf]
- **Plans / proposals:** [docs/plans/][plans]
- **Internal notes** (problems, seams, invariants, etc.):
  [docs/internal/][internal]

[arch]: concepts/ARCHITECTURE.md
[api]: reference/api-endpoints.md
[cli]: reference/cli-commands.md
[config]: reference/configuration.md
[perf]: perf/
[plans]: plans/
[internal]: internal/

## Stop / restart

```bash
vrooli scenario restart git-control-tower
vrooli scenario stop    git-control-tower
```

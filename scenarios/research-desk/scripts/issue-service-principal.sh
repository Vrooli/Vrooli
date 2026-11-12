#!/usr/bin/env bash
# Give research-desk its own Web Console credential, so automatic launch stops
# depending on Web Console's local availability policy (docs/DECISIONS.md).
#
# The operator runs this; no agent does. It is safe to run again:
#   1. adds the `research-desk` service-principal profile (resource web-console,
#      scopes web-console:read and web-console:write) to scenario-authenticator's
#      VROOLI_AUTH_SERVICE_PRINCIPALS, copying the issuers from the
#      `agent-manager` entry; an existing identical profile is left alone;
#   2. restarts scenario-authenticator when the profile was added or changed,
#      and waits for it once;
#   3. issues (or rotates) the credential with
#      `scenario-authenticator auth issue-service-principal --name research-desk`,
#      which stores it in the credential authority and never prints it.
#
# Until the desk's credential client is built, the desk keeps reaching Web
# Console under Web Console's local availability policy and says so in
# `research-desk now` and /health.
set -euo pipefail

root="${VROOLI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)}"
manifest="$root/scenarios/scenario-authenticator/.vrooli/service.json"
[[ -f "$manifest" ]] || { echo "scenario-authenticator manifest not found at $manifest" >&2; exit 1; }

changed="$(python3 -I - "$manifest" <<'PY'
import json, sys
path = sys.argv[1]
text = open(path, encoding="utf-8").read()
env = json.loads(text).get("environment", {})
old = env.get("VROOLI_AUTH_SERVICE_PRINCIPALS", "")
if not old:
    sys.exit("VROOLI_AUTH_SERVICE_PRINCIPALS is not set in " + path + "; there is no agent-manager entry to copy the issuers from")
profiles = json.loads(old)
agent = next((p for p in profiles if p.get("name") == "agent-manager"), None)
if agent is None or not agent.get("issuers"):
    sys.exit("no agent-manager profile with issuers in VROOLI_AUTH_SERVICE_PRINCIPALS; add the desk's profile by hand")
desk = {"name": "research-desk", "resource": "web-console",
        "scopes": ["web-console:read", "web-console:write"],
        "issuers": list(agent["issuers"])}
current = next((p for p in profiles if p.get("name") == "research-desk"), None)
if current == desk:
    print("unchanged")
    sys.exit(0)
profiles = [p for p in profiles if p.get("name") != "research-desk"] + [desk]
new = json.dumps(profiles, separators=(",", ":"))
# Replace only the value's text so the rest of the file keeps its layout.
before, after = json.dumps(old), json.dumps(new)
if text.count(before) != 1:
    sys.exit("could not find the VROOLI_AUTH_SERVICE_PRINCIPALS value exactly once in " + path)
open(path, "w", encoding="utf-8").write(text.replace(before, after))
print("changed")
PY
)"

if [[ "$changed" == "changed" ]]; then
  echo "Added the research-desk profile to scenario-authenticator; restarting it."
  vrooli scenario restart scenario-authenticator
  vrooli scenario wait scenario-authenticator --json >/dev/null
else
  echo "The research-desk profile is already present; scenario-authenticator is not restarted."
fi

echo "Issuing the research-desk credential into the credential authority (it is never printed)."
scenario-authenticator auth issue-service-principal --name research-desk
echo "Done. research-desk now and /health say which Web Console access the desk uses."

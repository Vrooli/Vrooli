#!/usr/bin/env bash
#
# Step-function tests for bootstrap.sh. No network, no real toolchain: every
# external command the script drives (git, make, loginctl, journalctl) and both
# binaries it produces (the node-agent and the vrooli-bridge CLI) are replaced
# with fakes that record state on disk. The test then runs the script TWICE and
# asserts the first run does the work (step-ok) while the second converges to a
# no-op (step-skip) — the idempotency contract in the README.
#
# Run:  ./bootstrap_test.sh      (exit 0 = all assertions passed)

set -euo pipefail

SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
readonly SCRIPT="${SCRIPT_DIR}/bootstrap.sh"

WORKROOT="$(mktemp -d)"
trap 'rm -rf "$WORKROOT"' EXIT

FAKEBIN="${WORKROOT}/bin"
STATE="${WORKROOT}/state"
CHECKOUT="${WORKROOT}/checkout"
FAKE_AGENT="${FAKEBIN}/fake-agent"
FAKE_CLI="${FAKEBIN}/fake-cli"
mkdir -p "$FAKEBIN"

PASS=0
FAIL=0
check() { # check <description> <condition-result 0/1>
  if [ "$2" -eq 0 ]; then PASS=$((PASS + 1)); printf 'ok   %s\n' "$1"
  else FAIL=$((FAIL + 1)); printf 'FAIL %s\n' "$1"; fi
}
# marker_is <file> <event> <step> — 0 if a line with that event+step exists.
marker_is() { grep -qE "event=$2 step=$3( |\$)" "$1"; }

# --- fakes -------------------------------------------------------------------

make_fakes() {
  # git: `clone` makes a checkout with a .git dir; everything else is a no-op
  # that still lets `rev-parse HEAD` print a stable sha.
  cat >"${FAKEBIN}/git" <<'GIT'
#!/usr/bin/env bash
set -euo pipefail
args=("$@")
for ((i=0; i<${#args[@]}; i++)); do
  case "${args[$i]}" in
    clone) dest="${args[-1]}"; mkdir -p "${dest}/.git"; exit 0 ;;
    rev-parse) echo "deadbeefcafe0000000000000000000000000000"; exit 0 ;;
  esac
done
exit 0
GIT

  # make: setup / agent build are no-ops (the agent binary is injected via
  # --agent-bin so no real compile happens).
  cat >"${FAKEBIN}/make" <<'MAKE'
#!/usr/bin/env bash
exit 0
MAKE

  # loginctl: linger state persists in a file so run 2 sees run 1's enable.
  cat >"${FAKEBIN}/loginctl" <<LOGINCTL
#!/usr/bin/env bash
set -euo pipefail
f="${STATE}/.linger"
case "\$1" in
  show-user) [ -f "\$f" ] && echo yes || echo no ;;
  enable-linger) touch "\$f" ;;
esac
exit 0
LOGINCTL

  # journalctl: emits the exact line the agent logs once the control plane
  # accepts its dial-out stream, so verify-online sees "connected".
  cat >"${FAKEBIN}/journalctl" <<'JOURNAL'
#!/usr/bin/env bash
# -n0 availability probe used by the script prints nothing and succeeds.
for a in "$@"; do [ "$a" = "-n0" ] && exit 0; done
echo 'Jul 14 00:00:00 host vrooli-bridge-agent[1]: channel: dial-out stream open to http://cp (node "node-xyz")'
exit 0
JOURNAL

  # Fake node-agent: --print-public-key, --print-service-unit, and the
  # service verbs (install/status) backed by a state file.
  cat >"$FAKE_AGENT" <<AGENT
#!/usr/bin/env bash
set -euo pipefail
svc="${STATE}/.service.json"
unit="${STATE}/.unit"
if [ "\$1" = "--print-public-key" ]; then echo "AAAApublickeyBBBB="; exit 0; fi
if [ "\$1" = "--print-service-unit" ]; then printf '[Service]\\nExecStart=fake-agent\\n'; exit 0; fi
if [ "\$1" = "service" ]; then
  verb="\$2"
  case "\$verb" in
    install)
      printf '[Service]\\nExecStart=fake-agent\\n' >"\$unit"
      printf '{"installed":true,"running":true,"unit_path":"%s"}\\n' "\$unit" >"\$svc"
      cat "\$svc"; exit 0 ;;
    status)
      if [ -f "\$svc" ]; then cat "\$svc"; else echo '{"installed":false,"running":false}'; fi
      exit 0 ;;
  esac
fi
exit 0
AGENT

  # Fake vrooli-bridge CLI: `pair redeem` pins control_plane.pub and prints a
  # node_id, refusing a second redeem (single-use) so re-runs must skip it.
  cat >"$FAKE_CLI" <<CLI
#!/usr/bin/env bash
set -euo pipefail
# find --state-dir value
sd=""
args=("\$@")
for ((i=0; i<\${#args[@]}; i++)); do
  [ "\${args[\$i]}" = "--state-dir" ] && sd="\${args[\$((i+1))]}"
done
# redeem requires the code from the environment, never argv.
if [ -z "\${BRIDGE_PAIRING_CODE:-}" ]; then echo "no code in env" >&2; exit 1; fi
if grep -qE ' --code ' <<<" \$* "; then echo "code must not be on argv" >&2; exit 1; fi
mkdir -p "\$sd"; printf 'FAKECPKEY=' >"\${sd}/control_plane.pub"; chmod 600 "\${sd}/control_plane.pub"
echo '{"node_id":"node-xyz","control_plane_public_key":"FAKECPKEY="}'
exit 0
CLI

  chmod +x "${FAKEBIN}/git" "${FAKEBIN}/make" "${FAKEBIN}/loginctl" "${FAKEBIN}/journalctl" "$FAKE_AGENT" "$FAKE_CLI"
}

run_bootstrap() { # run_bootstrap <stdout-file>
  PATH="${FAKEBIN}:${PATH}" \
  BRIDGE_PAIRING_CODE="TESTCODE1234" \
    bash "$SCRIPT" \
      --control-plane-url "http://cp.test" \
      --node-name "test-node" \
      --checkout-dir "$CHECKOUT" \
      --state-dir "$STATE" \
      --skip-prereqs \
      --agent-bin "$FAKE_AGENT" \
      --bridge-cli "$FAKE_CLI" \
      --verify-timeout 10 \
      >"$1" 2>"${1}.err"
}

# --- test --------------------------------------------------------------------

make_fakes

OUT1="${WORKROOT}/run1.out"
OUT2="${WORKROOT}/run2.out"

echo "== run 1 (fresh) =="
run_bootstrap "$OUT1"; rc1=$?
check "run 1 exits 0" "$rc1"
check "run 1 emits run-ok" "$(grep -q 'event=run-ok' "$OUT1" && echo 0 || echo 1)"
check "run 1 secret never on stdout" "$(grep -q 'TESTCODE1234' "$OUT1" && echo 1 || echo 0)"
check "run 1 secret never on stderr" "$(grep -q 'TESTCODE1234' "${OUT1}.err" && echo 1 || echo 0)"
check "run 1 pair-redeem did work (step-ok)" "$(marker_is "$OUT1" step-ok pair-redeem && echo 0 || echo 1)"
check "run 1 service-install did work (step-ok)" "$(marker_is "$OUT1" step-ok service-install && echo 0 || echo 1)"
check "run 1 autostart did work (step-ok)" "$(marker_is "$OUT1" step-ok autostart && echo 0 || echo 1)"
check "run 1 setup ran (step-ok)" "$(marker_is "$OUT1" step-ok setup && echo 0 || echo 1)"
check "run 1 verify-online connected (step-ok)" "$(marker_is "$OUT1" step-ok verify-online && echo 0 || echo 1)"
check "run 1 pinned control_plane.pub exists" "$([ -s "${STATE}/control_plane.pub" ] && echo 0 || echo 1)"
check "run 1 recorded node id" "$([ -s "${STATE}/node_id" ] && echo 0 || echo 1)"

echo "== run 2 (converge, must be no-op) =="
run_bootstrap "$OUT2"; rc2=$?
check "run 2 exits 0" "$rc2"
check "run 2 emits run-ok" "$(grep -q 'event=run-ok' "$OUT2" && echo 0 || echo 1)"
check "run 2 pair-redeem skipped (already paired)" "$(marker_is "$OUT2" step-skip pair-redeem && echo 0 || echo 1)"
check "run 2 setup skipped (sentinel)" "$(marker_is "$OUT2" step-skip setup && echo 0 || echo 1)"
check "run 2 service-install skipped (converged)" "$(marker_is "$OUT2" step-skip service-install && echo 0 || echo 1)"
check "run 2 autostart skipped (linger already on)" "$(marker_is "$OUT2" step-skip autostart && echo 0 || echo 1)"
check "run 2 verify-online still connected" "$(marker_is "$OUT2" step-ok verify-online && echo 0 || echo 1)"

echo "== missing pairing code on a fresh node fails cleanly (exit 2) =="
rm -rf "$STATE" "$CHECKOUT"
OUT3="${WORKROOT}/run3.out"
set +e
PATH="${FAKEBIN}:${PATH}" bash "$SCRIPT" \
  --control-plane-url "http://cp.test" --checkout-dir "$CHECKOUT" --state-dir "$STATE" \
  --skip-prereqs --agent-bin "$FAKE_AGENT" --bridge-cli "$FAKE_CLI" --verify-timeout 10 \
  >"$OUT3" 2>"${OUT3}.err"
rc3=$?
set -e
check "no-code run exits 2 (usage)" "$([ "$rc3" -eq 2 ] && echo 0 || echo 1)"
check "no-code run emits run-fail" "$(grep -q 'event=run-fail' "$OUT3" && echo 0 || echo 1)"
check "no-code run fails at pair-redeem" "$(marker_is "$OUT3" step-fail pair-redeem && echo 0 || echo 1)"

echo
echo "PASS=${PASS} FAIL=${FAIL}"
[ "$FAIL" -eq 0 ]

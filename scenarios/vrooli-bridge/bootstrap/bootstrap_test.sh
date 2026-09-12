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
# Toolchain fakes (go, pnpm) live in a SEPARATE dir from the core fakes so a test
# can present them on PATH (default: guard finds them) or withhold them (the
# guard's off-PATH-recovery and missing paths) without disturbing git/make/etc.
TOOLBIN="${WORKROOT}/toolbin"
STATE="${WORKROOT}/state"
CHECKOUT="${WORKROOT}/checkout"
TEST_HOME="${WORKROOT}/home"
FAKE_AGENT="${FAKEBIN}/fake-agent"
FAKE_CLI="${FAKEBIN}/fake-cli"
FAKE_VROOLI="${FAKEBIN}/fake-vrooli"
mkdir -p "$FAKEBIN" "$TOOLBIN" "$TEST_HOME"

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
  # that still lets `rev-parse HEAD` print a stable sha. `--git-dir` models git's
  # ability to recognize both directory and file-form .git metadata.
  cat >"${FAKEBIN}/git" <<'GIT'
#!/usr/bin/env bash
set -euo pipefail
args=("$@")
for ((i=0; i<${#args[@]}; i++)); do
  case "${args[$i]}" in
    clone) dest="${args[-1]}"; mkdir -p "${dest}/.git"; exit 0 ;;
    rev-parse)
      checkout=""
      has_git_dir=0
      for ((j=0; j<${#args[@]}; j++)); do
        [ "${args[$j]}" = "-C" ] && checkout="${args[$((j + 1))]}"
        [ "${args[$j]}" = "--git-dir" ] && has_git_dir=1
      done
      if [ "$has_git_dir" -eq 1 ]; then
        [ -n "$checkout" ] && [ -e "${checkout}/.git" ] || exit 1
        echo "${checkout}/.git"
      else
        echo "deadbeefcafe0000000000000000000000000000"
      fi
      exit 0
      ;;
  esac
done
exit 0
GIT

  # make: setup / agent build are no-ops (the agent binary is injected via
  # --agent-bin so no real compile happens). It records every invocation's argv to
  # $FAKE_MAKE_LOG so tests can assert the profile reached SETUP_ARGS, and — when
  # $FAKE_SETUP_NEEDS_SUDO is set on a `setup` invocation — prints the grouped
  # "Needs sudo" requirements block the real renderer emits, so the loud-skip path
  # can be exercised without a real toolchain.
  cat >"${FAKEBIN}/make" <<'MAKE'
#!/usr/bin/env bash
[ -n "${FAKE_MAKE_LOG:-}" ] && printf '%s\n' "$*" >>"$FAKE_MAKE_LOG"
for a in "$@"; do
  if [ "$a" = "setup" ]; then
    if [ -n "${FAKE_SETUP_NEEDS_SUDO:-}" ]; then
      cat <<'REPORT'
To finish setup:
  → Re-run with sudo to install 2 blocked item(s):        sudo vrooli setup

Needs sudo — re-run with `sudo vrooli setup` (2):
  ✗ kdump-tools                 requires root to install
       apt-get needs root privileges
  ✗ sysctl-hardening            requires root to apply
Applied (1):
  ✓ go-toolchain                installed
Δ installed=1  applied=0  failed=0  pending=2  unchanged=0
REPORT
    fi
    exit 0
  fi
done
exit 0
MAKE

  # sudo: emulate the `sudo -n` probe + elevation. $FAKE_SUDO_MODE=passwordless
  # makes `sudo -n true` succeed and `sudo -n <cmd>` run the command (via the fake
  # PATH); anything else makes `sudo -n` fail exactly as real sudo does when a
  # password would be required — so the script's non-interactive branch is
  # deterministic and never hangs.
  cat >"${FAKEBIN}/sudo" <<'SUDO'
#!/usr/bin/env bash
args=("$@")
[ "${args[0]:-}" = "-n" ] && args=("${args[@]:1}")
if [ "${FAKE_SUDO_MODE:-denied}" != "passwordless" ]; then
  echo "sudo: a password is required" >&2
  exit 1
fi
[ "${#args[@]}" -eq 0 ] && exit 0
exec "${args[@]}"
SUDO

  # Ownership-repair fixture: the bootstrap's exact-path chown remains safe to
  # exercise without root by restoring owner-readable permissions on its final
  # path argument. Real runs use the non-interactive sudo/root path above.
  cat >"${FAKEBIN}/chown" <<'CHOWN'
#!/usr/bin/env bash
set -euo pipefail
target="${!#}"
[ -n "${FAKE_CHOWN_LOG:-}" ] && printf '%s\n' "$target" >>"$FAKE_CHOWN_LOG"
chmod -R u+rwX "$target"
CHOWN

  # uname: keep the normal Linux fixture deterministic and allow the platform
  # default socket contract to be exercised as a Darwin run below.
  cat >"${FAKEBIN}/uname" <<'UNAME'
#!/usr/bin/env bash
case "${1:-}" in
  -s) echo "${FAKE_UNAME_S:-Linux}" ;;
  -m) echo "${FAKE_UNAME_M:-x86_64}" ;;
  *) echo "Linux" ;;
esac
UNAME

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
[ -n "\${FAKE_AGENT_LOG:-}" ] && printf '%s\n' "\$*" >>"\$FAKE_AGENT_LOG"
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

  # Fake project Vrooli CLI: the prebuilt-bundle path invokes this directly for
  # setup with VROOLI_SOURCE_ROOT pointing at the shipped tree. It records the
  # call so the no-Go test proves make/go were never used on the node.
  cat >"$FAKE_VROOLI" <<'VROOLI'
#!/usr/bin/env bash
set -euo pipefail
[ -n "${FAKE_VROOLI_LOG:-}" ] && printf 'root=%s argv=%s\n' "${VROOLI_SOURCE_ROOT:-}" "$*" >>"$FAKE_VROOLI_LOG"
[ "${1:-}" = "setup" ] || { echo "expected setup invocation" >&2; exit 2; }
result_file=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--result-file" ]; then result_file="$arg"; break; fi
  prev="$arg"
done
if [ -n "${FAKE_VROOLI_RESULT_CATEGORY:-}" ] && [ -n "$result_file" ]; then
  printf '{"version":"v1","status":"failed","category":"%s","stage":"requirements","retryable":true,"remediation":"fixture"}\n' "${FAKE_VROOLI_RESULT_CATEGORY}" >"$result_file"
fi
exit "${FAKE_VROOLI_EXIT:-0}"
VROOLI

  # apt-get: records every invocation's argv to $FAKE_APT_LOG so a test can assert
  # the prereqs step installs ONLY git/curl and never a toolchain package. Any
  # `install` also drops a matching command shim into $FAKEBIN so the post-install
  # presence re-check passes.
  cat >"${FAKEBIN}/apt-get" <<'APT'
#!/usr/bin/env bash
[ -n "${FAKE_APT_LOG:-}" ] && printf '%s\n' "$*" >>"$FAKE_APT_LOG"
install=0
for a in "$@"; do [ "$a" = "install" ] && install=1; done
if [ "$install" -eq 1 ]; then
  for a in "$@"; do
    case "$a" in
      -*|install|update) ;;
      *) printf '#!/usr/bin/env bash\nexit 0\n' >"${FAKE_INSTALL_BIN:-/dev/null}/$a" 2>/dev/null && chmod +x "${FAKE_INSTALL_BIN}/$a" 2>/dev/null ;;
    esac
  done
fi
exit 0
APT

  # Fake go + pnpm toolchains — presence is the only thing the guard checks, so a
  # trivial exit-0 stub suffices. They live in $TOOLBIN (not $FAKEBIN) so tests can
  # add or withhold them from PATH independently.
  printf '#!/usr/bin/env bash\nexit 0\n' >"${TOOLBIN}/go"
  printf '#!/usr/bin/env bash\nexit 0\n' >"${TOOLBIN}/pnpm"

  chmod +x "${FAKEBIN}/git" "${FAKEBIN}/make" "${FAKEBIN}/sudo" "${FAKEBIN}/chown" "${FAKEBIN}/uname" "${FAKEBIN}/loginctl" "${FAKEBIN}/journalctl" "${FAKEBIN}/apt-get" "${TOOLBIN}/go" "${TOOLBIN}/pnpm" "$FAKE_AGENT" "$FAKE_CLI" "$FAKE_VROOLI"
}

FAKE_MAKE_LOG="${WORKROOT}/make.calls"

run_bootstrap() { # run_bootstrap <stdout-file> [extra bootstrap args...]
  local out="$1"; shift
  HOME="$TEST_HOME" PATH="${FAKEBIN}:${TOOLBIN}:${PATH}" \
  FAKE_MAKE_LOG="$FAKE_MAKE_LOG" \
  FAKE_AGENT_LOG="${FAKE_AGENT_LOG:-$WORKROOT/agent.calls}" \
  FAKE_UNAME_S="${FAKE_UNAME_S:-Linux}" FAKE_UNAME_M="${FAKE_UNAME_M:-x86_64}" \
  FAKE_SUDO_MODE="${FAKE_SUDO_MODE:-denied}" \
  FAKE_CHOWN_LOG="${FAKE_CHOWN_LOG:-$WORKROOT/chown.calls}" \
  FAKE_SETUP_NEEDS_SUDO="${FAKE_SETUP_NEEDS_SUDO:-}" \
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
      "$@" \
      >"$out" 2>"${out}.err"
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
check "run 1 toolchain guard passed on PATH (step-ok)" "$(marker_is "$OUT1" step-ok toolchain && echo 0 || echo 1)"
check "run 1 toolchain detail says on PATH" "$(grep 'step=toolchain' "$OUT1" | grep -q 'resolve on PATH' && echo 0 || echo 1)"
check "run 1 verify-online connected (step-ok)" "$(marker_is "$OUT1" step-ok verify-online && echo 0 || echo 1)"
check "run 1 pinned control_plane.pub exists" "$([ -s "${STATE}/control_plane.pub" ] && echo 0 || echo 1)"
check "run 1 recorded node id" "$([ -s "${STATE}/node_id" ] && echo 0 || echo 1)"
check "run 1 writes install record" "$([ -s "${TEST_HOME}/.vrooli/state/install-record.json" ] && jq -e '.version == 1 and (.entries | length) > 0' "${TEST_HOME}/.vrooli/state/install-record.json" >/dev/null && echo 0 || echo 1)"

echo "== run 2 (converge, must be no-op) =="
run_bootstrap "$OUT2"; rc2=$?
check "run 2 exits 0" "$rc2"
check "run 2 emits run-ok" "$(grep -q 'event=run-ok' "$OUT2" && echo 0 || echo 1)"
check "run 2 pair-redeem skipped (already paired)" "$(marker_is "$OUT2" step-skip pair-redeem && echo 0 || echo 1)"
check "run 2 setup skipped (sentinel)" "$(marker_is "$OUT2" step-skip setup && echo 0 || echo 1)"
check "run 2 service-install skipped (converged)" "$(marker_is "$OUT2" step-skip service-install && echo 0 || echo 1)"
check "run 2 autostart skipped (linger already on)" "$(marker_is "$OUT2" step-skip autostart && echo 0 || echo 1)"
check "run 2 verify-online still connected" "$(marker_is "$OUT2" step-ok verify-online && echo 0 || echo 1)"

echo "== unreadable managed state directory is repaired narrowly =="
rm -rf "$STATE" "$CHECKOUT"; mkdir -p "$STATE"; chmod 000 "$STATE"
OUTOWNERSHIP="${WORKROOT}/ownership-repair.out"
set +e
FAKE_SUDO_MODE=passwordless run_bootstrap "$OUTOWNERSHIP"
ownership_rc=$?
set -e
check "ownership repair run exits 0" "$ownership_rc"
if [ "$ownership_rc" -ne 0 ]; then
  sed -n '1,120p' "${OUTOWNERSHIP}.err" >&2
fi
check "ownership repair reaches run-ok" "$(grep -q 'event=run-ok' "$OUTOWNERSHIP" && echo 0 || echo 1)"
check "ownership repair targets only the managed state directory" "$(grep -Fxq "$STATE" "$WORKROOT/chown.calls" && echo 0 || echo 1)"

echo "== Darwin provisioning helper uses writable per-user state socket =="
rm -rf "$STATE" "$CHECKOUT"
OUTDARWIN="${WORKROOT}/darwin-provision.out"
FAKE_UNAME_S=Darwin FAKE_UNAME_M=x86_64 FAKE_SUDO_MODE=passwordless run_bootstrap "$OUTDARWIN" \
  --provision-service-user root
check "Darwin helper run exits 0" "$?"
check "Darwin helper run reaches run-ok" "$(grep -q 'event=run-ok' "$OUTDARWIN" && echo 0 || echo 1)"
check "Darwin helper defaults socket under state dir" "$(grep -q -- "--provision-socket ${STATE}/provision.sock" "$WORKROOT/agent.calls" && echo 0 || echo 1)"

echo "== existing Git worktree with file-form .git metadata converges =="
rm -rf "$STATE" "$CHECKOUT"; mkdir -p "$CHECKOUT"
printf 'gitdir: /tmp/worktree.git\n' >"$CHECKOUT/.git"
printf 'existing worktree\n' >"$CHECKOUT/WORKTREE.txt"
OUTWTG="${WORKROOT}/worktree-git.out"
run_bootstrap "$OUTWTG" --revision e767613fca
check "file-form .git run exits 0" "$?"
check "file-form .git uses existing checkout" "$(grep -q 'existing checkout — fetching' "$OUTWTG" "$OUTWTG.err" && echo 0 || echo 1)"
check "file-form .git reaches run-ok" "$(grep -q 'event=run-ok' "$OUTWTG" && echo 0 || echo 1)"

echo "== populated non-Git checkout fails with safe recovery =="
rm -rf "$STATE" "$CHECKOUT"; mkdir -p "$CHECKOUT"
printf 'unmanaged source tree\n' >"$CHECKOUT/README.md"
OUTNONGIT="${WORKROOT}/nongit.out"
set +e
run_bootstrap "$OUTNONGIT"
rcnongit=$?
set -e
check "populated non-Git checkout exits 1" "$([ "$rcnongit" -eq 1 ] && echo 0 || echo 1)"
check "non-Git failure names source mode recovery" "$(grep 'step=clone' "$OUTNONGIT" | grep -q -- '--source-dir' && echo 0 || echo 1)"
check "non-Git failure never invokes clone" "$(grep -q 'cloning fresh' "$OUTNONGIT" && echo 1 || echo 0)"

echo "== setup profile flags reach make setup SETUP_ARGS (unprivileged path) =="
rm -rf "$STATE" "$CHECKOUT"; : >"$FAKE_MAKE_LOG"
OUTP="${WORKROOT}/profile.out"
run_bootstrap "$OUTP" \
  --setup-environment production --setup-resources enabled \
  --setup-scenarios none --include-optional
check "profile run exits 0" "$?"
check "SETUP_ARGS carries --environment production" "$(grep -q -- '--environment production' "$FAKE_MAKE_LOG" && echo 0 || echo 1)"
check "SETUP_ARGS carries --resources enabled" "$(grep -q -- '--resources enabled' "$FAKE_MAKE_LOG" && echo 0 || echo 1)"
check "SETUP_ARGS carries --scenarios none" "$(grep -q -- '--scenarios none' "$FAKE_MAKE_LOG" && echo 0 || echo 1)"
check "SETUP_ARGS carries --include-optional" "$(grep -q -- '--include-optional' "$FAKE_MAKE_LOG" && echo 0 || echo 1)"
check "setup ran unprivileged (no passwordless sudo)" "$(grep 'step=setup' "$OUTP" | grep -q 'unprivileged' && echo 0 || echo 1)"

echo "== setup runs elevated when passwordless sudo is available =="
rm -rf "$STATE" "$CHECKOUT"; : >"$FAKE_MAKE_LOG"
OUTS="${WORKROOT}/sudo.out"
FAKE_SUDO_MODE=passwordless run_bootstrap "$OUTS"
check "sudo run exits 0" "$?"
check "setup ran elevated under sudo" "$(grep 'step=setup' "$OUTS" | grep -q 'elevated' && echo 0 || echo 1)"
check "make setup still invoked" "$(grep -q 'setup' "$FAKE_MAKE_LOG" && echo 0 || echo 1)"

echo "== unprivileged setup surfaces the skipped Needs-sudo items loudly (not a failure) =="
rm -rf "$STATE" "$CHECKOUT"; : >"$FAKE_MAKE_LOG"
OUTN="${WORKROOT}/needsudo.out"
FAKE_SETUP_NEEDS_SUDO=1 run_bootstrap "$OUTN"
check "needs-sudo run exits 0" "$?"
check "setup is step-ok (loud warning, not a failure)" "$(marker_is "$OUTN" step-ok setup && echo 0 || echo 1)"
check "setup detail names the skipped count" "$(grep 'step=setup' "$OUTN" | grep -q '2 root-required requirement' && echo 0 || echo 1)"
check "setup detail names a skipped item" "$(grep 'step=setup' "$OUTN" | grep -q 'kdump-tools' && echo 0 || echo 1)"

echo "== the setup sentinel is keyed on the profile (change re-runs; identical no-ops) =="
rm -rf "$STATE" "$CHECKOUT"
OUTA="${WORKROOT}/sentA1.out"; OUTB="${WORKROOT}/sentA2.out"; OUTC="${WORKROOT}/sentB.out"
run_bootstrap "$OUTA" --setup-environment development
check "profile A first run: setup ran" "$(marker_is "$OUTA" step-ok setup && echo 0 || echo 1)"
run_bootstrap "$OUTB" --setup-environment development
check "profile A re-run: setup skipped (sentinel)" "$(marker_is "$OUTB" step-skip setup && echo 0 || echo 1)"
run_bootstrap "$OUTC" --setup-environment production
check "profile B (changed): setup re-ran" "$(marker_is "$OUTC" step-ok setup && echo 0 || echo 1)"

echo "== a shell-metachar setup value is rejected before it can reach the script (exit 2) =="
rm -rf "$STATE" "$CHECKOUT"
OUTM="${WORKROOT}/meta.out"
set +e
run_bootstrap "$OUTM" --setup-resources 'a;rm -rf x'
rcm=$?
set -e
check "metachar setup value exits 2 (usage)" "$([ "$rcm" -eq 2 ] && echo 0 || echo 1)"

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

echo "== prereqs installs ONLY clone tools (git/curl), never a toolchain =="
# git+curl are present (faked in FAKEBIN), so prereqs must converge to a skip
# WITHOUT invoking apt-get at all — proving it no longer tries to install
# go/pnpm/node. Run without --skip-prereqs so the real prereqs logic executes.
rm -rf "$STATE" "$CHECKOUT"
FAKE_APT_LOG="${WORKROOT}/apt.calls"; : >"$FAKE_APT_LOG"
OUTPQ="${WORKROOT}/prereq.out"
PATH="${FAKEBIN}:${TOOLBIN}:${PATH}" \
FAKE_APT_LOG="$FAKE_APT_LOG" FAKE_INSTALL_BIN="$FAKEBIN" \
FAKE_MAKE_LOG="$FAKE_MAKE_LOG" BRIDGE_PAIRING_CODE="TESTCODE1234" \
  bash "$SCRIPT" \
    --control-plane-url "http://cp.test" --node-name "test-node" \
    --checkout-dir "$CHECKOUT" --state-dir "$STATE" \
    --agent-bin "$FAKE_AGENT" --bridge-cli "$FAKE_CLI" --verify-timeout 10 \
    >"$OUTPQ" 2>"${OUTPQ}.err"
check "prereqs run exits 0" "$?"
check "prereqs skipped (git+curl already present)" "$(marker_is "$OUTPQ" step-skip prereqs && echo 0 || echo 1)"
check "prereqs never invoked apt-get (no toolchain install)" "$([ -s "$FAKE_APT_LOG" ] && echo 1 || echo 0)"
check "prereqs step detail is git/curl-only" "$(grep 'step=prereqs' "$OUTPQ" | grep -qE 'go|pnpm|node' && echo 1 || echo 0)"

# make_toolless_bin <dir> — a bin dir with symlinks to only the coreutils
# bootstrap.sh needs, deliberately EXCLUDING go/pnpm, so the toolchain guard's
# negative paths are deterministic no matter what the test host has in /usr/bin
# (this host, for instance, ships /usr/bin/pnpm).
make_toolless_bin() {
  local d="$1" b real
  mkdir -p "$d"
  for b in bash sh uname hostname id mktemp cat rm cp mv ls grep egrep awk sed tr cut date sleep \
           mkdir chmod chown touch dirname basename sha256sum shasum cksum jq env ln \
           readlink dd head tail wc sort; do
    real="$(command -v "$b" 2>/dev/null || true)"
    [ -n "$real" ] && ln -sf "$real" "$d/$b"
  done
}

TOOLLESS="${WORKROOT}/toolless"; make_toolless_bin "$TOOLLESS"

echo "== complete prebuilt bundle reaches online with no node Go or source build =="
rm -rf "$STATE" "$CHECKOUT"; mkdir -p "$CHECKOUT"
printf 'shipped tree\n' >"${CHECKOUT}/WORKING.txt"
mkdir -p "${CHECKOUT}/.vrooli"
printf '{}\n' >"${CHECKOUT}/.vrooli/repo-contract.json"
: >"$FAKE_MAKE_LOG"
FAKE_VROOLI_LOG="${WORKROOT}/vrooli.calls"; : >"$FAKE_VROOLI_LOG"
FAKE_AGENT_LOG="${WORKROOT}/agent.calls"; : >"$FAKE_AGENT_LOG"
PREBUILT_DEFAULT_CHECKOUT="${WORKROOT}/prebuilt-default-checkout"
for bin in "$FAKE_VROOLI" "$FAKE_CLI" "$FAKE_AGENT"; do
  printf 'live-tree-fingerprint\n' >"${bin}.fp"
done
OUTPRE="${WORKROOT}/prebuilt.out"
set +e
PATH="${FAKEBIN}:${TOOLLESS}" HOME="${WORKROOT}/prebuilt-home" \
FAKE_MAKE_LOG="$FAKE_MAKE_LOG" FAKE_VROOLI_LOG="$FAKE_VROOLI_LOG" FAKE_AGENT_LOG="$FAKE_AGENT_LOG" \
BRIDGE_PAIRING_CODE="TESTCODE1234" \
  bash "$SCRIPT" \
    --control-plane-url "http://cp.test" --node-name "test-node" \
    --checkout-dir "$PREBUILT_DEFAULT_CHECKOUT" --source-dir "$CHECKOUT" --source-digest "live-tree-fingerprint" \
    --state-dir "$STATE" --vrooli-bin "$FAKE_VROOLI" \
    --agent-bin "$FAKE_AGENT" --bridge-cli "$FAKE_CLI" --verify-timeout 10 \
    >"$OUTPRE" 2>"${OUTPRE}.err"
rcpre=$?
set -e
check "prebuilt no-Go run exits 0" "$([ "$rcpre" -eq 0 ] && echo 0 || echo 1)"
check "prebuilt bundle reports received binaries" "$(grep 'step=prebuilt-artifacts' "$OUTPRE" | grep -q 'received prebuilt binaries' && echo 0 || echo 1)"
check "prebuilt setup invokes transferred vrooli directly" "$(grep -q 'argv=setup' "$FAKE_VROOLI_LOG" && echo 0 || echo 1)"
check "prebuilt setup passes no global flags" "$(grep -q 'argv=setup' "$FAKE_VROOLI_LOG" && ! grep -q 'argv=--' "$FAKE_VROOLI_LOG" && echo 0 || echo 1)"
# The exact argv is pinned in argv-fixture.txt so the control plane's invoker
# registry can parse what this script really runs through the root parser.
fixture_argv="$(grep -v '^#' "${SCRIPT_DIR}/argv-fixture.txt" | sed -n '1p')"
recorded_argv="$(sed -n 's/^root=.* argv=//p' "$FAKE_VROOLI_LOG" | head -1 | sed 's#--result-file .*#--result-file <result-file>#')"
check "prebuilt setup argv matches argv-fixture.txt" "$([ "$recorded_argv" = "$fixture_argv" ] && echo 0 || echo 1)"
check "prebuilt setup points freshness at shipped tree" "$(grep -q "root=${CHECKOUT}" "$FAKE_VROOLI_LOG" && echo 0 || echo 1)"
check "prebuilt service work-dir follows shipped tree" "$(grep -q -- "--work-dir ${CHECKOUT}" "$FAKE_AGENT_LOG" && echo 0 || echo 1)"
check "prebuilt run never invokes make" "$([ -s "$FAKE_MAKE_LOG" ] && echo 1 || echo 0)"
check "prebuilt run never invokes go build" "$(grep -q 'go build\|build from source' "${OUTPRE}" "${OUTPRE}.err" && echo 1 || echo 0)"
check "prebuilt toolchain guard skips source-build requirement" "$(marker_is "$OUTPRE" step-skip toolchain && echo 0 || echo 1)"
check "prebuilt run reaches ONLINE" "$(grep -q 'event=run-ok' "$OUTPRE" && echo 0 || echo 1)"

echo "== setup result categories control Bridge failure classification =="
run_prebuilt_setup_failure() { # category exit output
  local category="$1" expected_exit="$2" out="$3"
  rm -rf "$STATE" "$CHECKOUT"; mkdir -p "$CHECKOUT"
  printf 'shipped tree\n' >"${CHECKOUT}/WORKING.txt"
  mkdir -p "${CHECKOUT}/.vrooli"
  printf '{}\n' >"${CHECKOUT}/.vrooli/repo-contract.json"
  PATH="${FAKEBIN}:${TOOLLESS}" HOME="${WORKROOT}/result-home-${category}" \
  FAKE_VROOLI_RESULT_CATEGORY="$category" FAKE_VROOLI_EXIT="$expected_exit" BRIDGE_PAIRING_CODE="TESTCODE1234" \
    bash "$SCRIPT" \
      --control-plane-url "http://cp.test" --node-name "test-node" \
      --checkout-dir "${WORKROOT}/result-checkout-${category}" --source-dir "$CHECKOUT" --source-digest "live-tree-fingerprint" \
      --state-dir "$STATE" --vrooli-bin "$FAKE_VROOLI" \
      --agent-bin "$FAKE_AGENT" --bridge-cli "$FAKE_CLI" --verify-timeout 10 \
      >"$out" 2>"${out}.err"
  return $?
}
OUTREQ="${WORKROOT}/result-required.out"
set +e; run_prebuilt_setup_failure required_requirement_blocked 1 "$OUTREQ"; rcreq=$?; set -e
check "required setup failure stays exit 1" "$([ "$rcreq" -eq 1 ] && echo 0 || echo 1)"
check "required setup failure is not platform-gated" "$(grep -q 'platform is unsupported' "${OUTREQ}.err" && echo 1 || echo 0)"
OUTUNSUPPORTED="${WORKROOT}/result-unsupported.out"
set +e; run_prebuilt_setup_failure unsupported_platform 3 "$OUTUNSUPPORTED"; rcunsupported=$?; set -e
check "explicit unsupported-platform result maps to exit 3" "$([ "$rcunsupported" -eq 3 ] && echo 0 || echo 1)"
check "unsupported result has platform guidance" "$(grep -q 'platform is unsupported' "${OUTUNSUPPORTED}.err" && echo 0 || echo 1)"

echo "== toolchain guard recovers a tool installed OFF the non-interactive PATH =="
# go/pnpm are absent from PATH (TOOLBIN withheld; TOOLLESS has no toolchains) but
# present in a known install dir under a controlled HOME ($HOME/.vrooli/bin) — the
# guard must find them there, amend PATH, and continue to a full onboard.
rm -rf "$STATE" "$CHECKOUT"; : >"$FAKE_MAKE_LOG"
RECHOME="${WORKROOT}/rechome"; mkdir -p "${RECHOME}/.vrooli/bin"
printf '#!/usr/bin/env bash\nexit 0\n' >"${RECHOME}/.vrooli/bin/go"
printf '#!/usr/bin/env bash\nexit 0\n' >"${RECHOME}/.vrooli/bin/pnpm"
chmod +x "${RECHOME}/.vrooli/bin/go" "${RECHOME}/.vrooli/bin/pnpm"
OUTREC="${WORKROOT}/recover.out"
set +e
PATH="${FAKEBIN}:${TOOLLESS}" HOME="$RECHOME" \
FAKE_MAKE_LOG="$FAKE_MAKE_LOG" BRIDGE_PAIRING_CODE="TESTCODE1234" \
  bash "$SCRIPT" \
    --control-plane-url "http://cp.test" --node-name "test-node" \
    --checkout-dir "$CHECKOUT" --state-dir "$STATE" --skip-prereqs \
    --agent-bin "$FAKE_AGENT" --bridge-cli "$FAKE_CLI" --verify-timeout 10 \
    >"$OUTREC" 2>"${OUTREC}.err"
rcrec=$?
set -e
check "recovery run exits 0" "$([ "$rcrec" -eq 0 ] && echo 0 || echo 1)"
check "toolchain guard recovered off-PATH (step-ok)" "$(marker_is "$OUTREC" step-ok toolchain && echo 0 || echo 1)"
check "toolchain detail names the recovery" "$(grep 'step=toolchain' "$OUTREC" | grep -q 'recovered off-PATH' && echo 0 || echo 1)"
check "recovery run reaches run-ok" "$(grep -q 'event=run-ok' "$OUTREC" && echo 0 || echo 1)"

echo "== toolchain guard fails actionably when setup did not deliver a toolchain =="
# Neither on PATH nor in any known install dir (empty HOME) — the guard must
# step-fail, name the gap, and exit 1 instead of letting a build die confusingly.
rm -rf "$STATE" "$CHECKOUT"; : >"$FAKE_MAKE_LOG"
EMPTYHOME="${WORKROOT}/emptyhome"; mkdir -p "$EMPTYHOME"
OUTMISS="${WORKROOT}/missing.out"
set +e
PATH="${FAKEBIN}:${TOOLLESS}" HOME="$EMPTYHOME" \
FAKE_MAKE_LOG="$FAKE_MAKE_LOG" BRIDGE_PAIRING_CODE="TESTCODE1234" \
  bash "$SCRIPT" \
    --control-plane-url "http://cp.test" --node-name "test-node" \
    --checkout-dir "$CHECKOUT" --state-dir "$STATE" --skip-prereqs \
    --agent-bin "$FAKE_AGENT" --bridge-cli "$FAKE_CLI" --verify-timeout 10 \
    >"$OUTMISS" 2>"${OUTMISS}.err"
rcmiss=$?
set -e
check "missing-toolchain run exits 1" "$([ "$rcmiss" -eq 1 ] && echo 0 || echo 1)"
check "toolchain guard step-fail" "$(marker_is "$OUTMISS" step-fail toolchain && echo 0 || echo 1)"
check "toolchain fail detail names the gap" "$(grep 'step=toolchain' "$OUTMISS" | grep -q 'missing after setup' && echo 0 || echo 1)"
check "toolchain fail detail points at vrooli setup" "$(grep 'step=toolchain' "$OUTMISS" | grep -q 'vrooli setup' && echo 0 || echo 1)"
check "missing-toolchain run emits run-fail" "$(grep -q 'event=run-fail' "$OUTMISS" && echo 0 || echo 1)"
check "missing-toolchain never reached build-agent" "$(marker_is "$OUTMISS" step-start build-agent && echo 1 || echo 0)"

echo "== working-tree source mode: verify a pre-synced tree instead of cloning =="
# The control plane pre-ships its local tree into SOURCE_DIR; --source-dir switches
# step_clone to verify-and-record instead of clone/fetch, and provenance reads dirty.
rm -rf "$STATE"; : >"$FAKE_MAKE_LOG"
SRCTREE="${WORKROOT}/synced-tree"; rm -rf "$SRCTREE"; mkdir -p "$SRCTREE/scenarios/vrooli-bridge"
printf 'uncommitted work\n' >"$SRCTREE/WORKING.txt"
mkdir -p "$SRCTREE/.vrooli"
printf '{}\n' >"$SRCTREE/.vrooli/repo-contract.json"
OUTWT="${WORKROOT}/worktree.out"
run_bootstrap "$OUTWT" --revision e767613fca --source-dir "$SRCTREE" --source-digest "abc123def456"
check "working-tree run exits 0" "$?"
check "clone step-ok in working-tree mode" "$(marker_is "$OUTWT" step-ok clone && echo 0 || echo 1)"
check "clone detail names the pre-synced tree" "$(grep 'step=clone' "$OUTWT" | grep -q 'pre-synced working tree' && echo 0 || echo 1)"
check "clone detail marks provenance dirty (base+dirty)" "$(grep 'step=clone' "$OUTWT" | grep -q 'e767613fca+dirty' && echo 0 || echo 1)"
check "clone detail names the digest" "$(grep 'step=clone' "$OUTWT" | grep -q 'abc123def456' && echo 0 || echo 1)"
check "working-tree run reaches run-ok" "$(grep -q 'event=run-ok' "$OUTWT" && echo 0 || echo 1)"
check "working-tree setup ran against the synced tree" "$(grep -q -- "-C ${SRCTREE}" "$FAKE_MAKE_LOG" && echo 0 || echo 1)"

echo "== working-tree: an empty pre-synced tree fails cleanly (incomplete ship) =="
rm -rf "$STATE"
EMPTYSRC="${WORKROOT}/empty-tree"; rm -rf "$EMPTYSRC"; mkdir -p "$EMPTYSRC"
OUTES="${WORKROOT}/emptysrc.out"
set +e
run_bootstrap "$OUTES" --source-dir "$EMPTYSRC" --source-digest "deadbeef"
rces=$?
set -e
check "empty source-dir run exits 1" "$([ "$rces" -eq 1 ] && echo 0 || echo 1)"
check "empty source-dir fails at clone" "$(marker_is "$OUTES" step-fail clone && echo 0 || echo 1)"
check "empty source-dir detail says incomplete" "$(grep 'step=clone' "$OUTES" | grep -q 'incomplete' && echo 0 || echo 1)"

echo "== working-tree: the setup sentinel keys on the content digest (re-ship re-runs) =="
rm -rf "$STATE"; : >"$FAKE_MAKE_LOG"
OUTD1="${WORKROOT}/digest1.out"; OUTD2="${WORKROOT}/digest2.out"; OUTD3="${WORKROOT}/digest3.out"
run_bootstrap "$OUTD1" --source-dir "$SRCTREE" --source-digest "digestAAAA"
check "digest A first run: setup ran" "$(marker_is "$OUTD1" step-ok setup && echo 0 || echo 1)"
run_bootstrap "$OUTD2" --source-dir "$SRCTREE" --source-digest "digestAAAA"
check "digest A re-run: setup skipped (sentinel)" "$(marker_is "$OUTD2" step-skip setup && echo 0 || echo 1)"
run_bootstrap "$OUTD3" --source-dir "$SRCTREE" --source-digest "digestBBBB"
check "digest B (changed ship): setup re-ran" "$(marker_is "$OUTD3" step-ok setup && echo 0 || echo 1)"

echo "== bootstrap.sh contains no bash-4-only constructs (macOS ships bash 3.2) =="
# Regression backstop: macOS stock bash is 3.2 (GPLv2 freeze). These constructs
# only exist in bash 4+, so their presence would silently break a darwin onboard.
# The SETUP_METACHARS literal legitimately contains '|&', so that shorthand is not
# scanned; the constructs below have no false-positive source in the script.
# Scan a comment-stripped view so the lint judges CODE, not the explanatory
# comments (which necessarily name the very constructs they forbid). Full-line
# comments are dropped; an inline comment (whitespace + '#' … EOL) is trimmed.
# Parameter-expansion '#' (e.g. "${a# }") is preceded by a non-space and so is
# never mistaken for a comment.
SCRIPT_CODE="${WORKROOT}/bootstrap.code.sh"
sed -e '/^[[:space:]]*#/d' -e 's/[[:space:]]#.*$//' "$SCRIPT" >"$SCRIPT_CODE"
b4_hit() { # b4_hit <label> <extended-regex>
  local label="$1" re="$2" hits
  hits="$(grep -nE "$re" "$SCRIPT_CODE" || true)"
  if [ -n "$hits" ]; then
    check "no bash-4 construct: ${label}" 1
    printf '     matched (in comment-stripped view):\n%s\n' "$hits"
  else
    check "no bash-4 construct: ${label}" 0
  fi
}
b4_hit "mapfile"                 'mapfile'
b4_hit "readarray"               'readarray'
b4_hit "declare -A / local -A"   '(declare|local)[[:space:]]+-[a-zA-Z]*A'
b4_hit "case-conversion (,,/^^)" '\$\{[A-Za-z_][A-Za-z0-9_]*(,,|\^\^)'
b4_hit "case-conversion (single ,/^)" '\$\{[A-Za-z_][A-Za-z0-9_]*[,^]\}'
b4_hit "negative substr offset"  '\$\{[A-Za-z_][A-Za-z0-9_]*:[[:space:]]+-'
b4_hit "truncating substr :n:-"  '\$\{[A-Za-z_][A-Za-z0-9_]*:[0-9]+:-'
b4_hit "coproc"                  '(^|[^[:alnum:]_])coproc([^[:alnum:]_]|$)'
b4_hit "&>> append redirect"     '&>>'
b4_hit "param transform @Q/@U/@L" '\$\{[^}]*@[QEPAKaLU]\}'
b4_hit "wait -n"                 'wait[[:space:]]+-n'

echo
echo "PASS=${PASS} FAIL=${FAIL}"
[ "$FAIL" -eq 0 ]

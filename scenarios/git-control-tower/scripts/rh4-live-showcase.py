#!/usr/bin/env python3
"""Exercise RH4 through the managed GCT UI/API using disposable linked worktrees.

This is a bounded development fixture. All Git metadata and the routed SQLite
lease live below one temporary directory; no path in the operator checkout is
opened, staged, or committed. Its explicitly allowlisted provider proves gate
composition and admission wiring, not RH2 secret detection.
"""

from __future__ import annotations

import importlib.util
import json
import shutil
import sqlite3
import subprocess
import sys
import tempfile
import time
import urllib.request
import uuid
from pathlib import Path

from playwright.sync_api import Error as PlaywrightError
from playwright.sync_api import sync_playwright


HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("rh3_live_showcase", HERE / "rh3-live-showcase.py")
rh3 = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(rh3)

API, UI = rh3.API, rh3.UI
ROUTING, REPO = rh3.ROUTING, rh3.REPO
LEASE_TTL_MS, TEST_HEADER = rh3.LEASE_TTL_MS, rh3.TEST_HEADER


def stat_number(stats: dict, key: str) -> int:
    return int(stats.get(key, 0) or 0)


def launch(repo_id: str, retry: bool = False) -> dict:
    return rh3.request(f"{REPO}/StartPrecommitCandidateCheck",
        {"repositoryId": repo_id, "retryTerminal": retry}, test_mode=True)


def wait_terminal(repo_id: str, job_id: str, timeout: float = 90) -> dict:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        row = rh3.request(f"{REPO}/GetPrecommitCandidateCheck",
            {"repositoryId": repo_id, "jobId": job_id}, test_mode=True)
        if row.get("state") not in ("running", "queued", "cancelling"):
            return row
        time.sleep(.2)
    raise RuntimeError(f"candidate job {job_id} did not settle")


def main() -> int:
    lease = f"rh4-live-{uuid.uuid4().hex}"
    root = Path(tempfile.mkdtemp(prefix="gct-rh4-live-"))
    common = root / "main checkout – snow ☃"
    one = root / "linked worktree α with spaces"
    two = root / "linked worktree β (client 2)"
    db_path = root / "fixture.sqlite"
    installed = False
    browser = None
    evidence: dict[str, object] = {"lease": lease, "root": str(root)}
    try:
        with urllib.request.urlopen(f"{API}/health", timeout=10) as response:
            rh3.require(response.status == 200, "managed GCT API is unhealthy")
        with urllib.request.urlopen(UI, timeout=10) as response:
            rh3.require(response.status == 200, "managed GCT UI is unhealthy")

        rh3.run("git", "init", "--initial-branch=main", str(common))
        rh3.run("git", "config", "user.name", "RH4 Fixture", cwd=common)
        rh3.run("git", "config", "user.email", "rh4-fixture@example.invalid", cwd=common)
        (common / "tracked.txt").write_text("base\n", encoding="utf-8")
        rh3.run("git", "add", "tracked.txt", cwd=common)
        rh3.run("git", "commit", "-m", "test: initialize isolated RH4 fixture", cwd=common)
        rh3.run("git", "worktree", "add", "-b", "rh4/client-alpha", str(one), cwd=common)
        rh3.run("git", "worktree", "add", "-b", "rh4/client-beta", str(two), cwd=common)

        # Each root gets a distinct candidate. Unusual path bytes are retained
        # through Git's index and the browser/API repository identity.
        (common / "unusual name ☃.txt").write_text("first candidate\n", encoding="utf-8")
        rh3.run("git", "add", "--", "unusual name ☃.txt", cwd=common)
        (one / "path with spaces").mkdir()
        (one / "path with spaces" / "δ.txt").write_text("candidate B\n", encoding="utf-8")
        rh3.run("git", "add", "--", "path with spaces/δ.txt", cwd=one)
        (two / "tracked.txt").write_text("first candidate\n", encoding="utf-8")
        rh3.run("git", "add", "tracked.txt", cwd=two)

        request = rh3.request
        request(f"{ROUTING}/InstallTestPool", {
            "dsn": str(db_path), "leaseId": lease,
            "leaseTtlMs": str(LEASE_TTL_MS), "emptyConfig": True,
        })
        installed = True
        (root / ".gct-rh4-benign-security").write_text(lease + "\n", encoding="utf-8")

        repos: list[tuple[Path, str]] = []
        for path in (common, one, two):
            opened = request(f"{REPO}/OpenRepository", {"path": str(path)}, test_mode=True)
            repo_id = str(opened.get("repo", {}).get("id", ""))
            rh3.require(repo_id not in ("", "0"), f"could not register fixture root {path}: {opened}")
            repos.append((path, repo_id))
            with sqlite3.connect(db_path, timeout=10) as db:
                db.execute("""INSERT INTO git_repo_precommit
                    (repo_path, enabled, command, working_directory, timeout_seconds,
                     run_before_commit, allow_override, declared_inputs_json)
                    VALUES (?, 1, ?, '.', 60, 1, 0, '[]')
                    ON CONFLICT(repo_path) DO UPDATE SET enabled=1, command=excluded.command,
                     working_directory='.', timeout_seconds=60, run_before_commit=1,
                     allow_override=0, declared_inputs_json='[]'""",
                    (str(path), "sleep 12; git rev-parse --verify HEAD >/dev/null"))
                db.commit()

        # UI creates the first durable job. A second API client joins it by
        # candidate identity. Remaining roots exercise bounded scheduling.
        browser = sync_playwright().start().chromium.launch(channel="chrome", headless=True)
        context = browser.new_context(extra_http_headers={TEST_HEADER: "1"})
        page = context.new_page()
        commit_responses: list[dict] = []
        page.goto(UI, wait_until="domcontentloaded", timeout=30_000)
        request(f"{REPO}/SetActiveRepository", {"repositoryId": repos[0][1]}, test_mode=True)
        page.reload(wait_until="domcontentloaded", timeout=30_000)
        status = rh3.wait_status(page, "Commit checks running", 30_000)
        first = launch(repos[0][1])
        rh3.require(first.get("state") in ("running", "queued"), f"UI job was not live: {first}")
        joined = launch(repos[0][1])
        rh3.require(joined.get("jobId") == first.get("jobId"), "second client did not join the UI-created job")
        rh3.require(first.get("candidateDigest", "").removeprefix("tree:")[:7] in status,
                    "UI and API clients did not identify the same candidate")
        evidence["ui_api_join"] = first.get("jobId")

        second = launch(repos[1][1])
        third = launch(repos[2][1])
        rh3.require(second.get("jobId") != first.get("jobId") and third.get("jobId") != second.get("jobId"),
                    "worktree candidate identities collided")
        evidence["initial_states"] = [first.get("state"), second.get("state"), third.get("state")]
        rh3.require("queued" in evidence["initial_states"],
                    "three concurrent candidates did not demonstrate bounded worker capacity")

        # Cancel the queued candidate through the managed API, then mutate its
        # index. The replacement candidate must acquire a new identity.
        cancelled = request(f"{REPO}/CancelPrecommitCandidateCheck",
            {"repositoryId": repos[2][1], "jobId": third.get("jobId")}, test_mode=True)
        rh3.require(cancelled.get("state") == "cancelled", f"queued cancellation failed: {cancelled}")
        (two / "tracked.txt").write_text("second candidate\n", encoding="utf-8")
        rh3.run("git", "add", "tracked.txt", cwd=two)
        replacement = launch(repos[2][1], retry=True)
        rh3.require(replacement.get("jobId") != third.get("jobId"), "cancelled candidate identity was reused")
        evidence["cancelled"] = third.get("jobId")
        evidence["replacement"] = replacement.get("jobId")

        # Managed restart exercises owner shutdown and durable receipt recovery.
        # The temporary database remains on disk; after restart it is re-leased
        # before any test-mode application request resumes.
        checkpoint = request(f"{ROUTING}/PrepareTestPoolRestart", {"leaseId": lease})
        ticket = checkpoint.get("restartTicket")
        rh3.require(bool(ticket), f"routing owner did not prepare the lease for restart: {checkpoint}")
        rh3.run("vrooli", "scenario", "restart", "git-control-tower")
        rh3.run("vrooli", "scenario", "wait", "git-control-tower", "--json")
        resumed = request(f"{ROUTING}/ResumeTestPool", {"leaseId": lease, "restartTicket": ticket})
        rh3.require(resumed.get("activeLeaseId") == lease and resumed.get("fileRootsResumed") is True,
                    f"routed test lease did not resume after owner restart: {resumed}")
        installed = True
        (root / ".gct-rh4-benign-security").write_text(lease + "\n", encoding="utf-8")
        # Restore the known harmless staged blob in the third root after using
        # its earlier candidate solely to exercise cancellation and identity
        # replacement. Its fresh post-restart receipt is then admissible.
        (two / "tracked.txt").write_text("first candidate\n", encoding="utf-8")
        rh3.run("git", "add", "tracked.txt", cwd=two)
        recovered = [launch(repo_id, retry=True) for _, repo_id in repos]
        evidence["post_restart_jobs"] = [row.get("jobId") for row in recovered]
        evidence["owner_restart"] = "managed lifecycle restart completed; fixture DB re-leased"
        rh3.require(all(row.get("state") in ("running", "queued", "passed") for row in recovered),
                    f"post-restart receipts did not reconcile: {recovered}")

        terminals = [wait_terminal(repo_id, row["jobId"])
                     for (_, repo_id), row in zip(repos, recovered)]
        rh3.require(all(row.get("state") == "passed" for row in terminals),
                    f"required checks did not pass after restart: {terminals}")
        evidence["required_receipts"] = [
            {"repo": str(path), "job": row.get("jobId"), "state": row.get("state"),
             "candidate": row.get("candidateDigest")}
            for (path, _), row in zip(repos, terminals)
        ]

        # Exercise final admission in the browser for each root. The API's
        # final gate re-derives the candidate and may launch/join a fresh check;
        # it must never consume one of the other worktree's receipts.
        page.on("response", lambda response: commit_responses.append(response.json())
                if f"{REPO}/CreateCommit" in response.url else None)
        admissions = []
        # With no configured RH2 scanner, removal of the exact RH4 fixture
        # marker must produce incomplete coverage and block the real commit
        # path. Restore it before required admission checks.
        path, repo_id = repos[0]
        request(f"{REPO}/SetActiveRepository", {"repositoryId": repo_id}, test_mode=True)
        page.evaluate("id => localStorage.setItem('gct.activeRepoId', id)", repo_id)
        page.reload(wait_until="domcontentloaded", timeout=30_000)
        rh3.wait_status(page, "Checks passed for this staged candidate", 45_000)
        (root / ".gct-rh4-benign-security").unlink()
        page.get_by_test_id("commit-message-input").fill("test: RH4 unavailable security provider")
        before = len(commit_responses)
        rh3.authorize_exact_fixture_commit(page, path, ["unusual name ☃.txt"])
        unavailable = rh3.wait_for_commit_response(page, commit_responses, before + 1, timeout_ms=30_000)
        rh3.require(not unavailable.get("success") and
                    unavailable.get("precommit", {}).get("status") == "incomplete",
                    f"unavailable candidate-security provider did not fail closed: {unavailable}")
        serialized = json.dumps(unavailable)
        rh3.require("first candidate" not in serialized and lease not in serialized,
                    "unavailable-provider response exposed fixture content or lease identity")
        evidence["security_unavailable"] = {
            "blocked": not unavailable.get("success"),
            "coverage": unavailable.get("precommit", {}).get("status"),
            "redacted": True,
        }
        (root / ".gct-rh4-benign-security").write_text(lease + "\n", encoding="utf-8")

        # A complete benign security result cannot cover failed workspace
        # health. Change only the isolated fixture's configured command, make
        # sure its candidate receipt reaches failed, then restore it.
        path, repo_id = repos[1]
        with sqlite3.connect(db_path, timeout=10) as db:
            db.execute("UPDATE git_repo_precommit SET command='exit 7' WHERE repo_path=?", (str(path),))
            db.commit()
        health_failure = launch(repo_id, retry=True)
        health_terminal = wait_terminal(repo_id, health_failure["jobId"])
        rh3.require(health_terminal.get("state") == "failed",
                    f"workspace-health failure was not independent: {health_terminal}")
        evidence["workspace_health_failure"] = health_terminal.get("state")
        with sqlite3.connect(db_path, timeout=10) as db:
            db.execute("UPDATE git_repo_precommit SET command=? WHERE repo_path=?",
                       ("sleep 12; git rev-parse --verify HEAD >/dev/null", str(path)))
            db.commit()
        health_recovery = launch(repo_id, retry=True)
        health_recovered = wait_terminal(repo_id, health_recovery["jobId"])
        rh3.require(health_recovered.get("state") == "passed",
                    f"workspace-health did not recover after fixture command restoration: {health_recovered}")
        evidence["workspace_health_recovered"] = health_recovered.get("state")

        for index, ((path, repo_id), paths) in enumerate(zip(repos, [
            ["unusual name ☃.txt"], ["path with spaces/δ.txt"], ["tracked.txt"],
        ])):
            request(f"{REPO}/SetActiveRepository", {"repositoryId": repo_id}, test_mode=True)
            page.evaluate("id => localStorage.setItem('gct.activeRepoId', id)", repo_id)
            page.reload(wait_until="domcontentloaded", timeout=30_000)
            rh3.wait_status(page, "Checks passed for this staged candidate", 45_000)
            page.get_by_test_id("commit-message-input").fill(f"test: RH4 linked worktree admission {index}")
            before = len(commit_responses)
            rh3.authorize_exact_fixture_commit(page, path, paths)
            response = rh3.wait_for_commit_response(page, commit_responses, before + 1, timeout_ms=90_000)
            rh3.require(response.get("success") is True,
                        f"final admission failed for current matching receipt at {path}: {response}")
            rh3.require(bool(response.get("hash")), f"successful admission did not write a fixture commit at {path}: {response}")
            summary = response.get("precommit", {}).get("summary", "")
            rh3.require("Fixture-only benign provider" in summary,
                        f"candidate security boundary was not labeled for {path}: {response}")
            admissions.append({"repo": str(path), "success": True, "commit": response.get("hash"), "summary": summary})
        evidence["final_admissions"] = admissions

        # Ensure one root cannot consume another worktree's receipt: mutate its
        # staged tree, then require a new check before that root is admissible.
        (one / "path with spaces" / "δ.txt").write_text("changed after receipt\n", encoding="utf-8")
        rh3.run("git", "add", "--", "path with spaces/δ.txt", cwd=one)
        mismatched = launch(repos[1][1])
        rh3.require(mismatched.get("candidateDigest") != terminals[1].get("candidateDigest"),
                    "changed worktree reused a stale candidate receipt")
        rh3.require(mismatched.get("jobId") != terminals[1].get("jobId"),
                    "changed worktree reused another candidate's job identity")
        evidence["changed_candidate_job"] = mismatched.get("jobId")

        # The two independent gates have separate positive and negative live
        # evidence; only the benign fixture provider is synthetic.
        evidence["candidate_security"] = "complete fixture-only allowlist; no staged-secret detection asserted"
        evidence["workspace_health"] = "configured precommit receipts were independently required and passed"
        cleared_reply = request(f"{ROUTING}/ClearTestPool", {"leaseId": lease})
        stats = cleared_reply.get("stats", {})
        rh3.require(stat_number(stats, "testPoolRequests") > 0, f"test lease had no routed requests: {stats}")
        rh3.require(stat_number(stats, "primaryDuringTestModeRequests") == 0 and
                    stat_number(stats, "primaryRootWritesDuringTestMode") == 0,
                    f"fixture touched primary storage: {stats}")
        evidence["cleanup_stats"] = stats
        process_list = subprocess.run(["ps", "-eo", "args="], text=True, capture_output=True, check=True).stdout
        lingering = [line for line in process_list.splitlines() if str(root) in line]
        rh3.require(not lingering, f"fixture child processes remain after routed lease clear: {lingering}")
        installed = False
        shutil.rmtree(root)
        evidence["cleanup"] = "lease cleared with zero primary writes; fixture backing removed"
        print(json.dumps({"result": "passed_fixture_scope", **evidence}, indent=2))
        return 0
    except (RuntimeError, PlaywrightError, sqlite3.Error, OSError, KeyError) as error:
        print(json.dumps({"result": "incomplete", "error": str(error), **evidence}, indent=2), file=sys.stderr)
        return 1
    finally:
        if browser is not None:
            browser.close()
        if installed:
            try:
                result = rh3.request(f"{ROUTING}/ClearTestPool", {"leaseId": lease})
                stats = result.get("stats", {})
                if stat_number(stats, "primaryDuringTestModeRequests") or stat_number(stats, "primaryRootWritesDuringTestMode"):
                    print(f"fixture routing touched primary storage: {stats}", file=sys.stderr)
                    print(f"fixture backing retained at {root}", file=sys.stderr)
                else:
                    installed = False
                    process_list = subprocess.run(["ps", "-eo", "args="], text=True, capture_output=True, check=True).stdout
                    lingering = [line for line in process_list.splitlines() if str(root) in line]
                    if lingering:
                        print(f"fixture child processes remain; backing retained at {root}: {lingering}", file=sys.stderr)
                    else:
                        shutil.rmtree(root, ignore_errors=False)
            except Exception as error:
                print(f"fixture lease cleanup failed; backing retained at {root}: {error}", file=sys.stderr)
        elif root.exists():
            shutil.rmtree(root, ignore_errors=False)


if __name__ == "__main__":
    raise SystemExit(main())

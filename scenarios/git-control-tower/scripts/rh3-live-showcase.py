#!/usr/bin/env python3
"""Run the RH3 browser acceptance flow against the managed GCT API.

The database and repository are temporary. All application requests from the
browser carry the dev-routing marker and therefore use the installed lease.
The fixture never selects or stages anything in the operator's checkout.
"""

from __future__ import annotations

import json
import os
import shutil
import sqlite3
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path

from playwright.sync_api import Error as PlaywrightError
from playwright.sync_api import sync_playwright


API = os.environ.get("GCT_API_URL", "http://127.0.0.1:18710").rstrip("/")
UI = os.environ.get("GCT_UI_URL", "http://127.0.0.1:21400").rstrip("/")
LEASE_TTL_MS = 15 * 60 * 1000
TEST_HEADER = "X-Vrooli-Test-Mode"
ROUTING = "vrooli.dev_routing.v1.routing.RoutingService"
REPO = "vrooli.git_control_tower.v1.repo.RepoService"


def request(method: str, payload: dict, *, test_mode: bool = False) -> dict:
    headers = {"Content-Type": "application/json"}
    if test_mode:
        headers[TEST_HEADER] = "1"
    body = json.dumps(payload).encode()
    req = urllib.request.Request(f"{API}/{method}", data=body, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=20) as response:
            return json.loads(response.read())
    except urllib.error.HTTPError as error:
        details = error.read().decode(errors="replace")
        raise RuntimeError(f"{method} returned HTTP {error.code}: {details}") from error


def run(*args: str, cwd: Path | None = None) -> str:
    result = subprocess.run(args, cwd=cwd, text=True, capture_output=True)
    if result.returncode:
        raise RuntimeError(
            f"command failed ({result.returncode}): {' '.join(args)}\n"
            f"stdout: {result.stdout}\nstderr: {result.stderr}"
        )
    return result.stdout.strip()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise RuntimeError(message)


def wait_status(page, expected: str, timeout_ms: int = 30_000) -> str:
    status = page.get_by_test_id("precommit-candidate-status")
    status.wait_for(state="visible", timeout=timeout_ms)
    status.get_by_text(expected, exact=False).wait_for(state="visible", timeout=timeout_ms)
    return status.inner_text()


def authorize_exact_fixture_commit(page, repo: Path, expected_paths: list[str]) -> None:
    page.get_by_test_id("commit-button").click(timeout=10_000)
    dialog = page.get_by_test_id("commit-authorization-dialog")
    dialog.wait_for(state="visible", timeout=10_000)
    require(str(repo) in dialog.inner_text(), "commit confirmation did not name the temporary fixture repository")
    staged = dialog.locator("ul").inner_text().splitlines()
    require(staged == expected_paths, f"commit confirmation staged paths were {staged}, expected {expected_paths}")
    skip = dialog.get_by_test_id("skip-precommit-confirmation-checkbox")
    if skip.count():
        require(not skip.is_checked(), "fixture attempted to bypass configured precommit checks")
    dialog.get_by_test_id("confirm-commit-authorization").click()


def wait_for_commit_response(page, responses: list[dict], count: int, timeout_ms: int = 30_000) -> dict:
    deadline = time.monotonic() + timeout_ms / 1000
    while time.monotonic() < deadline:
        if len(responses) >= count:
            return responses[count - 1]
        page.wait_for_timeout(100)
    raise RuntimeError(f"commit API response {count} was not observed; collected {len(responses)}")


def wait_for_head_change(repo: Path, original_head: str, page, timeout_s: float = 45) -> str:
    deadline = time.monotonic() + timeout_s
    while time.monotonic() < deadline:
        head = run("git", "rev-parse", "HEAD", cwd=repo)
        if head != original_head:
            return head
        page.wait_for_timeout(200)
    raise RuntimeError("managed UI/API did not create the expected fixture-only commit")


def main() -> int:
    lease = f"rh3-browser-{uuid.uuid4().hex}"
    root = Path(tempfile.mkdtemp(prefix="gct-rh3-live-"))
    repo = root / "repo"
    repo.mkdir()
    db_path = root / "fixture.sqlite"
    installed = False
    cleared = False
    backing_removed = False
    browser = None
    evidence: dict[str, object] = {"lease": lease, "repo": str(repo)}

    try:
        # Confirm that the live managed API/UI are available before allocating
        # a routed storage lease.
        with urllib.request.urlopen(f"{API}/health", timeout=10) as response:
            require(response.status == 200, "managed GCT API is unhealthy")
        with urllib.request.urlopen(UI, timeout=10) as response:
            require(response.status == 200, "managed GCT UI is unhealthy")

        run("git", "init", "--initial-branch=main", str(repo))
        run("git", "config", "user.name", "RH3 Fixture", cwd=repo)
        run("git", "config", "user.email", "rh3-fixture@example.invalid", cwd=repo)
        (repo / "tracked.txt").write_text("base\n", encoding="utf-8")
        run("git", "add", "tracked.txt", cwd=repo)
        run("git", "commit", "-m", "test: initialize isolated fixture", cwd=repo)
        initial_head = run("git", "rev-parse", "HEAD", cwd=repo)
        (repo / "tracked.txt").write_text("first candidate\n", encoding="utf-8")
        run("git", "add", "tracked.txt", cwd=repo)

        # The lease endpoint is development-only and is mounted by the managed
        # API. No test-mode application request is sent before installation.
        installed_reply = request(
            f"{ROUTING}/InstallTestPool",
            {
                "dsn": str(db_path),
                "leaseId": lease,
                "leaseTtlMs": str(LEASE_TTL_MS),
                "emptyConfig": True,
            },
        )
        require(installed_reply.get("activeLeaseId") == lease, "fixture database lease was not installed")
        require(installed_reply.get("fileRootsInstalled") is True, "fixture file roots were not installed")
        installed = True
        (root / ".gct-rh3-benign-security").write_text(lease + "\n", encoding="utf-8")

        opened = request(
            f"{REPO}/OpenRepository",
            {"path": str(repo)},
            test_mode=True,
        )
        repository_id = str(opened.get("repo", {}).get("id", ""))
        require(repository_id not in ("", "0"), f"temporary repo was not registered: {opened}")
        request(f"{REPO}/SetActiveRepository", {"repositoryId": repository_id}, test_mode=True)

        # Configure a deliberately slow passing check in the isolated DB. This
        # is fixture state, not a change to the shared GCT configuration.
        with sqlite3.connect(db_path, timeout=10) as db:
            db.execute(
                """INSERT INTO git_repo_precommit
                   (repo_path, enabled, command, working_directory, timeout_seconds,
                    run_before_commit, allow_override, declared_inputs_json)
                   VALUES (?, 1, ?, '.', 60, 1, 0, '[]')
                   ON CONFLICT(repo_path) DO UPDATE SET enabled=1, command=excluded.command,
                    working_directory='.', timeout_seconds=60, run_before_commit=1,
                    allow_override=0, declared_inputs_json='[]'""",
                (str(repo), "sleep 8; git rev-parse --verify HEAD >/dev/null"),
            )
            db.commit()

        browser = sync_playwright().start().chromium.launch(channel="chrome", headless=True)
        context = browser.new_context(extra_http_headers={TEST_HEADER: "1"})
        page = context.new_page()
        commit_responses: list[dict] = []
        commit_request_headers: list[dict[str, str]] = []

        def capture_commit(response) -> None:
            if f"{REPO}/CreateCommit" not in response.url:
                return
            try:
                commit_responses.append(response.json())
            except Exception:
                return

        page.on("response", capture_commit)
        page.on(
            "request",
            lambda request: commit_request_headers.append(request.headers)
            if f"{REPO}/CreateCommit" in request.url else None,
        )
        page.goto(UI, wait_until="domcontentloaded", timeout=30_000)
        running = wait_status(page, "Commit checks running", 30_000)
        first = request(
            f"{REPO}/StartPrecommitCandidateCheck",
            {"repositoryId": repository_id},
            test_mode=True,
        )
        require(first.get("state") == "running", f"first UI check was not pending: {first}")
        job_id = first.get("jobId")
        require(bool(job_id), "candidate check returned no durable job identity")
        evidence["initial_job_id"] = job_id
        evidence["initial_candidate"] = first.get("candidateDigest")
        require(first.get("candidateDigest", "").removeprefix("tree:")[:7] in running,
                "UI candidate status does not match the isolated repo receipt")

        page.get_by_test_id("commit-message-input").fill("test: exercise isolated durable check")
        require("message drafting can continue" in running, "UI did not expose drafting during the pending check")
        page.reload(wait_until="domcontentloaded", timeout=30_000)
        reloaded = wait_status(page, "Commit checks running", 30_000)
        after_reload = request(
            f"{REPO}/StartPrecommitCandidateCheck",
            {"repositoryId": repository_id},
            test_mode=True,
        )
        require(after_reload.get("jobId") == job_id, "reload did not attach to the original candidate job")
        require(after_reload.get("candidateDigest", "").removeprefix("tree:")[:7] in reloaded,
                "reloaded UI did not attach to the isolated candidate receipt")
        evidence["reload_job_id"] = after_reload.get("jobId")
        evidence["reload_status"] = reloaded.splitlines()[0]

        wait_status(page, "Checks passed for this staged candidate", 45_000)
        reused = request(
            f"{REPO}/StartPrecommitCandidateCheck",
            {"repositoryId": repository_id},
            test_mode=True,
        )
        require(reused.get("jobId") == job_id and reused.get("state") == "passed", "matching candidate did not reuse its terminal receipt")
        evidence["receipt_reuse"] = reused.get("jobId")

        # The browser performs final admission inside the temporary Git
        # repository. The test provider is enabled only by the pinned lease,
        # exact marker, known-safe fixture blobs, and in-root Git metadata.
        # It proves the admission/receipt path can consume complete scanner
        # coverage; it does not test secret detection.
        message = page.get_by_test_id("commit-message-input")
        if not message.input_value().strip():
            message.fill("test: exercise isolated durable check")
        authorize_exact_fixture_commit(page, repo, ["tracked.txt"])
        first_response = wait_for_commit_response(page, commit_responses, 1)
        evidence["commit_test_marker"] = bool(
            commit_request_headers and commit_request_headers[-1].get(TEST_HEADER.lower()) == "1"
        )
        require(evidence["commit_test_marker"],
                "browser commit request did not carry the isolated routing marker")
        require(first_response.get("success") is True, f"first isolated commit failed: {first_response}")
        first_summary = first_response.get("precommit", {}).get("summary", "")
        require("Fixture-only benign provider" in first_summary and "not staged-secret coverage" in first_summary,
                f"fixture security boundary was not labeled in API response: {first_response}")
        first_commit = wait_for_head_change(repo, initial_head, page)
        evidence["first_commit"] = first_commit
        evidence["first_security_label"] = first_summary

        # Begin a second final admission while candidate B's check is pending.
        # Once the API is waiting on that receipt, mutate the index to candidate
        # C; admission must attach to a distinct receipt for the changed tree.
        (repo / "later.txt").write_text("candidate B\n", encoding="utf-8")
        run("git", "add", "later.txt", cwd=repo)
        page.reload(wait_until="domcontentloaded", timeout=30_000)
        wait_status(page, "Commit checks running", 30_000)
        second = request(
            f"{REPO}/StartPrecommitCandidateCheck",
            {"repositoryId": repository_id},
            test_mode=True,
        )
        require(second.get("state") == "running", f"second candidate did not start pending: {second}")
        require(second.get("jobId") != job_id, "stage mutation reused the previous candidate job")
        evidence["second_candidate"] = second.get("candidateDigest")
        second_message = page.get_by_test_id("commit-message-input")
        second_message.fill("test: recheck changed final candidate")
        authorize_exact_fixture_commit(page, repo, ["later.txt"])
        page.locator('[data-testid="commit-button"]').get_by_text("Committing", exact=False).wait_for(
            state="visible", timeout=5_000
        )
        (repo / "during-check.txt").write_text("candidate C\n", encoding="utf-8")
        run("git", "add", "during-check.txt", cwd=repo)
        third = request(
            f"{REPO}/StartPrecommitCandidateCheck",
            {"repositoryId": repository_id},
            test_mode=True,
        )
        require(third.get("state") in ("running", "passed"), f"changed candidate check did not start: {third}")
        require(third.get("jobId") not in (job_id, second.get("jobId")),
                "concurrent stage mutation reused an earlier candidate job")
        evidence["third_candidate"] = third.get("candidateDigest")
        second_response = wait_for_commit_response(page, commit_responses, 2, timeout_ms=45_000)
        require(second_response.get("success") is True,
                f"final admission did not commit the changed fixture candidate: {second_response}")
        second_summary = second_response.get("precommit", {}).get("summary", "")
        require("Fixture-only benign provider" in second_summary and "later-index revalidation" in second_summary,
                f"changed-candidate security boundary was not labeled: {second_response}")
        second_commit = wait_for_head_change(repo, first_commit, page)
        committed_paths = run("git", "show", "--pretty=format:", "--name-only", second_commit, cwd=repo).splitlines()
        require("later.txt" in committed_paths and "during-check.txt" in committed_paths,
                f"final commit did not include the latest staged candidate: {committed_paths}")
        evidence["fresh_final_commit"] = second_commit
        evidence["final_commit_paths"] = committed_paths
        evidence["final_security_label"] = second_summary

        with sqlite3.connect(db_path, timeout=10) as db:
            rows = db.execute(
                "SELECT job_id, candidate_digest, state FROM git_precommit_candidate_checks "
                "WHERE repo_path = ? ORDER BY created_at, job_id",
                (str(repo),),
            ).fetchall()
        require(len(rows) >= 3, f"concurrent stage mutation did not produce a fresh receipt: {rows}")
        require(len({row[0] for row in rows}) == len(rows), f"duplicate durable job rows: {rows}")
        evidence["candidate_receipts"] = [
            {"job_id": row[0], "candidate": row[1], "state": row[2]} for row in rows
        ]

        cleared_reply = request(f"{ROUTING}/ClearTestPool", {"leaseId": lease})
        stats = cleared_reply.get("stats", {})
        require(stats.get("primaryDuringTestModeRequests", 0) == 0,
                f"test requests hit primary DB: {stats}")
        require(stats.get("primaryRootWritesDuringTestMode", 0) == 0,
                f"test requests wrote primary files: {stats}")
        evidence["cleanup_stats"] = stats
        cleared = True
        installed = False
        shutil.rmtree(root)
        backing_removed = True
        evidence["cleanup"] = "lease cleared; temporary database and repository removed"

        evidence["fixture_provider_scope"] = "Known benign fixture blobs only; proves the real admission path consumes a complete provider result, not RH2 secret detection or staged-security revalidation."
        evidence["rh2_status"] = "unchanged; live Security Health staged-secret coverage is not asserted"
        print(json.dumps({"result": "passed_fixture_scope", **evidence}, indent=2))
        return 0
    except (RuntimeError, PlaywrightError, sqlite3.Error, OSError) as error:
        print(json.dumps({"result": "incomplete", "error": str(error), **evidence}, indent=2), file=sys.stderr)
        return 1
    finally:
        if browser is not None:
            browser.close()
        if installed:
            try:
                result = request(f"{ROUTING}/ClearTestPool", {"leaseId": lease})
                stats = result.get("stats", {})
                if stats.get("primaryDuringTestModeRequests", 0) != 0 or stats.get("primaryRootWritesDuringTestMode", 0) != 0:
                    print(f"fixture routing used primary storage: {stats}", file=sys.stderr)
                cleared = True
            except Exception as error:  # retain fixture backing if route could not be cleared
                print(f"fixture lease cleanup failed; backing retained at {root}: {error}", file=sys.stderr)
        if backing_removed:
            pass
        elif cleared or not installed:
            shutil.rmtree(root, ignore_errors=False)
        else:
            print(f"fixture backing retained for safe lease expiry/recovery: {root}", file=sys.stderr)


if __name__ == "__main__":
    raise SystemExit(main())

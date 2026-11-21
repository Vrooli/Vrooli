#!/usr/bin/env python3
"""Qualify Security Health's existing exact-candidate scanner on a temp Git repo.

This read-only commit-safety fixture sends only Git index blobs to the existing
Security Health Connect endpoint. It never enables the GCT commit policy and
never reads or mutates the operator's checkout or index.
"""

from __future__ import annotations

import base64
import json
import os
import shutil
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request
from pathlib import Path


API = os.environ.get("SECURITY_HEALTH_API_URL", "http://127.0.0.1:17104").rstrip("/")
SCAN_PATH = "/vrooli.security_health.v1.CandidateScannerService/ScanCandidate"
CANDIDATE_PATH = "config candidate.env"
# A documentation-only sample credential; this value is a test marker, not a credential.
STAGED_SAMPLE = b"AWS_ACCESS_KEY_ID=AKIA" + b"1234567890ABCDEF\n"
SAFE_VALUE = b"MODE=demo\n"


def git(repo: Path, *args: str, input_bytes: bytes | None = None) -> bytes:
    result = subprocess.run(
        ["git", *args], cwd=repo, input=input_bytes,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
    )
    if result.returncode:
        raise RuntimeError(f"git {args[0]} failed with exit {result.returncode}")
    return result.stdout


def git_diff_exit(repo: Path, *args: str) -> int:
    result = subprocess.run(
        ["git", "diff", *args], cwd=repo,
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
    )
    if result.returncode not in (0, 1):
        raise RuntimeError("git diff could not verify fixture state")
    return result.returncode


def staged_blob(repo: Path) -> bytes:
    return git(repo, "show", f":{CANDIDATE_PATH}")


def scan_candidate(blob: bytes) -> dict:
    payload = {"files": {CANDIDATE_PATH: base64.b64encode(blob).decode("ascii")}}
    request = urllib.request.Request(
        API + SCAN_PATH,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.loads(response.read())
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as error:
        raise RuntimeError(f"Security Health candidate scan request failed ({type(error).__name__})") from None


def require(ok: bool, message: str) -> None:
    if not ok:
        raise RuntimeError(message)


def main() -> int:
    root = Path(tempfile.mkdtemp(prefix="gct-rh2-security-"))
    repo = root / "isolated repo with spaces"
    repo.mkdir()
    evidence: dict[str, object] = {"scope": "existing Security Health candidate endpoint; no GCT commit policy"}
    try:
        git(repo, "init", "--initial-branch=main")
        git(repo, "config", "user.name", "RH2 Candidate Fixture")
        git(repo, "config", "user.email", "rh2-fixture@example.invalid")

        source = repo / CANDIDATE_PATH
        source.parent.mkdir(parents=True, exist_ok=True)
        source.write_bytes(SAFE_VALUE)
        git(repo, "add", "--", CANDIDATE_PATH)
        git(repo, "commit", "-m", "test: initialize isolated candidate fixture")

        # Mismatch direction 1: index carries a sample secret while the checked-out
        # file is clean. Security Health must see the exact staged blob and redact it.
        source.write_bytes(STAGED_SAMPLE)
        git(repo, "add", "--", CANDIDATE_PATH)
        source.write_bytes(SAFE_VALUE)
        require(staged_blob(repo) == STAGED_SAMPLE, "Git index did not retain staged-only sample")
        require(git_diff_exit(repo, "HEAD", "--quiet", "--", CANDIDATE_PATH) == 0,
                "worktree should match committed base in staged-only case")
        staged_only = scan_candidate(staged_blob(repo))
        require(staged_only.get("complete") is True, f"Security Health coverage incomplete: {staged_only.get('failureCode', 'unknown')}")
        secret_findings = staged_only.get("findings", [])
        require(bool(secret_findings), "staged-only sample was not detected")
        safe_response = json.dumps(staged_only, sort_keys=True)
        require(STAGED_SAMPLE.decode().strip() not in safe_response, "Security Health response leaked staged sample value")
        require(any(CANDIDATE_PATH in finding.get("filePath", "") for finding in secret_findings),
                "redacted finding did not retain the candidate path")
        evidence["staged_secret_clean_worktree"] = {
            "complete": True, "finding_count": len(secret_findings),
            "candidate_path": CANDIDATE_PATH, "redacted": True,
        }

        # Mismatch direction 2: the index is clean and only the worktree contains
        # the sample. The caller sends the staged blob, so no secret should appear.
        source.write_bytes(SAFE_VALUE)
        git(repo, "add", "--", CANDIDATE_PATH)
        source.write_bytes(STAGED_SAMPLE)
        require(staged_blob(repo) == SAFE_VALUE, "Git index should contain safe inverse candidate")
        require(git_diff_exit(repo, "--cached", "--quiet", "--", CANDIDATE_PATH) == 0,
                "index should match committed base in inverse case")
        require(git_diff_exit(repo, "--quiet", "--", CANDIDATE_PATH) == 1,
                "inverse sample should exist only in the worktree")
        inverse = scan_candidate(staged_blob(repo))
        require(inverse.get("complete") is True, f"inverse scan coverage incomplete: {inverse.get('failureCode', 'unknown')}")
        require(not inverse.get("findings"), f"worktree-only sample was incorrectly attributed to staged candidate: {inverse.get('findings')}")
        require(STAGED_SAMPLE.decode().strip() not in json.dumps(inverse), "inverse response leaked worktree-only sample")
        evidence["clean_candidate_secret_worktree"] = {"complete": True, "finding_count": 0, "redacted": True}
        evidence["candidate_identity_source"] = "git show :<path> from the disposable repository's index"
        evidence["shared_index_touched"] = False
        print(json.dumps({"result": "passed_candidate_scanner", **evidence}, indent=2))
        return 0
    except (RuntimeError, OSError, subprocess.SubprocessError) as error:
        print(json.dumps({"result": "incomplete", "error": str(error), **evidence}, indent=2), file=sys.stderr)
        return 1
    finally:
        shutil.rmtree(root, ignore_errors=True)


if __name__ == "__main__":
    raise SystemExit(main())

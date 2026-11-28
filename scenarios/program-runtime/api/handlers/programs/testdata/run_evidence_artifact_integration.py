"""Execute the real Test Genie consumer against the Go bridge's synthetic stream."""
import contextlib
import io
import json
import os
import sys
from pathlib import Path
from types import SimpleNamespace

repo = Path(os.environ["VROOLI_REPO_ROOT"])
sys.path.insert(0, str(repo / "scenarios/program-runtime/kernel/host"))
import engine
from program_helper import ProgramHelper

artifact_id = "artifact_0123456789abcdef0123456789abcdef"
raw = "synthetic-private-artifact-bytes"


class ArtifactHandle:
    def head(self, limit):
        return [{"id": artifact_id, "kind": "findings", "sizeBytes": len(raw),
                 "accessCapability": "ARTIFACT_ACCESS_CAPABILITY_STREAM"}][:limit]

    def meta(self):
        return {"legacyDiscovered": False}


class Runs:
    def artifact(self, **kwargs):
        expected = {"scenario": "demo", "run_id": "run-1", "artifact_id": artifact_id}
        if kwargs != expected:
            raise AssertionError(f"consumer identity mismatch: {kwargs!r}")
        return ArtifactHandle()


inputs = {"view": "download", "scenario": "demo", "run_id": "run-1",
          "artifact_id": artifact_id, "path": os.environ["PR4_OUTPUT_PATH"],
          "overwrite": os.environ.get("PR4_OVERWRITE", "false") == "true"}
bridge = os.environ["PR4_BRIDGE_URL"]
helper = ProgramHelper(invocation_context=engine._INVOCATION_CONTEXT)
scope = {"inputs": inputs, "__vrooli__": SimpleNamespace(
    _bridge_url=bridge, _session_id="sess-1")}
scope["program"] = helper
scope["test_genie"] = SimpleNamespace(runs=Runs())
helper._bind(scope)
token = engine._INVOCATION_CONTEXT.set({"program_id": "prog-1", "provenance": "PROVENANCE_TEST"})
captured = io.StringIO()
try:
    source = repo / "scenarios/test-genie/.vrooli/program-runtime/run-evidence.py"
    with contextlib.redirect_stdout(captured):
        exec(compile(source.read_text(), str(source), "exec"), scope)
finally:
    engine._INVOCATION_CONTEXT.reset(token)
print(captured.getvalue(), end="")

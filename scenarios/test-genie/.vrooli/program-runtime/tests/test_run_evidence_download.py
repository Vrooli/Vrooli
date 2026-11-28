import contextlib
import hashlib
import io
import json
import runpy
import tempfile
from pathlib import Path

SOURCE = "scenarios/test-genie/.vrooli/program-runtime/run-evidence.py"


class Handle:
    def __init__(self, rows=None, meta=None):
        self.rows = list(rows or [])
        self._meta = meta or {}

    def head(self, count):
        return self.rows[:count]

    def count(self):
        return len(self.rows)

    def meta(self):
        return self._meta


class Runs:
    def __init__(self, artifact):
        self.artifact_row = artifact
        self.calls = []

    def artifact(self, **kwargs):
        self.calls.append(kwargs)
        return Handle([self.artifact_row], {"legacyDiscovered": False})


class FakeTestGenie:
    def __init__(self, artifact):
        self.runs = Runs(artifact)


class Program:
    def __init__(self, inputs, payload=b"hello", receipt_overrides=None):
        self._inputs = inputs
        self.payload = payload
        self.receipt_overrides = receipt_overrides or {}
        self.calls = []
        self.output = None
        self.workspace = Path(tempfile.mkdtemp())

    def inputs(self):
        return self._inputs

    def classify(self, _error):
        return "failed", "binding_error"

    def fail(self, status, klass, detail, where, detail_limit=240):
        self.envelope = {"status": status, "phase": where, "signals": {}, "errors": [{"class": klass, "detail": str(detail)[:detail_limit], "where": where}], "evidence": []}
        return "report"

    def write_artifact(self, **kwargs):
        self.calls.append(kwargs)
        dest = self.workspace / kwargs["path"]
        dest.parent.mkdir(parents=True, exist_ok=True)
        if dest.exists() and not kwargs["overwrite"]:
            raise RuntimeError("destination exists")
        dest.write_bytes(self.payload)
        receipt = {
            "scenario": kwargs["scenario"], "run_id": kwargs["run_id"], "artifact_id": kwargs["artifact_id"],
            "kind": "trace", "output_path": kwargs["path"], "bytes": len(self.payload),
            "sha256": hashlib.sha256(self.payload).hexdigest(), "legacy_discovered": False,
        }
        receipt.update(self.receipt_overrides)
        return receipt

    def run(self, steps, start):
        state = start
        while state:
            state = steps[state]()

    def report(self, envelope):
        self.output = envelope


def execute(inputs, *, artifact=None, payload=b"hello", receipt_overrides=None):
    artifact = artifact or {"id": "art-1", "kind": "trace", "sizeBytes": str(len(payload)),
                            "accessCapability": 1, "accessPath": "/private/run/artifact"}
    program = Program(inputs, payload, receipt_overrides)
    owner = FakeTestGenie(artifact)
    stdout = io.StringIO()
    with contextlib.redirect_stdout(stdout):
        runpy.run_path(SOURCE, init_globals={"program": program, "test_genie": owner})
    return json.loads(stdout.getvalue()), program, owner


def test_download_streams_synthetic_bytes_and_returns_only_metadata():
    output, program, owner = execute({"view": "download", "scenario": "demo", "run_id": "run-1",
                                      "artifact_id": "art-1", "path": "out/trace.bin"})
    assert output["status"] == "ok"
    assert program.calls == [{"scenario": "demo", "run_id": "run-1", "artifact_id": "art-1",
                              "path": "out/trace.bin", "overwrite": False}]
    assert owner.runs.calls == [{"scenario": "demo", "run_id": "run-1", "artifact_id": "art-1"}]
    assert (program.workspace / "out/trace.bin").read_bytes() == b"hello"
    receipt = output["signals"]["receipt"]
    assert receipt["sha256"] == hashlib.sha256(b"hello").hexdigest()
    assert receipt["bytes"] == 5 and receipt["source_digest_verified"] is False
    raw = json.dumps(output)
    assert "hello" not in raw and "/private/run/artifact" not in raw


def test_download_requires_explicit_overwrite_and_honors_it():
    output, program, _ = execute({"view": "download", "scenario": "demo", "run_id": "run-1",
                                  "artifact_id": "art-1", "path": "trace.bin", "overwrite": True})
    assert output["status"] == "ok"
    assert program.calls[0]["overwrite"] is True


def test_download_rejects_identity_path_and_non_stream_metadata_before_write():
    output, program, owner = execute({"view": "download", "scenario": "demo", "run_id": "run-1",
                                      "artifact_id": "art-1", "path": "../escape"})
    assert output["status"] == "failed" and not program.calls and not owner.runs.calls

    output, program, _ = execute({"view": "download", "scenario": "demo", "run_id": "run-1",
                                  "artifact_id": "wrong-id", "path": "trace.bin"})
    assert output["status"] == "failed" and not program.calls

    output, program, _ = execute({"view": "download", "scenario": "demo", "run_id": "run-1",
                                  "artifact_id": "art-1", "path": "trace.bin"},
                                 artifact={"id": "art-1", "kind": "trace", "sizeBytes": 5,
                                           "accessCapability": 0, "accessPath": "/private/path"})
    assert output["status"] == "failed" and not program.calls


def test_unverified_receipt_is_effect_uncertain_and_never_returns_a_false_success():
    output, program, _ = execute({"view": "download", "scenario": "demo", "run_id": "run-1",
                                  "artifact_id": "art-1", "path": "trace.bin"},
                                 receipt_overrides={"run_id": "another-run"})
    assert output["status"] == "partial"
    assert output["errors"][0]["class"] == "effect_uncertain"
    assert "receipt" not in output["signals"]
    assert (program.workspace / "trace.bin").read_bytes() == b"hello"

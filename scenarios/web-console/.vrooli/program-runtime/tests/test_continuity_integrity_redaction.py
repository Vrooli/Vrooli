import contextlib
import io
import json
import runpy


class FakeResult:
    def count(self):
        return 1

    def head(self, _count):
        return [{
            "classification": "closed",
            "state": "available",
            "transcript": "PRIVATE-TRANSCRIPT",
            "filesystem_path": "/private/agent/home",
        }]


class FakeBinding:
    def integrity(self):
        return FakeResult()


class FakeWebConsole:
    continuity = FakeBinding()


class FakeProgram:
    def inputs(self):
        return {"include_generation": False}

    def fail(self, status, klass, message, phase):
        raise AssertionError((status, klass, message, phase))

    def classify(self, error):
        raise error

    def run(self, steps, start):
        step = start
        while step:
            step = steps[step]()


stdout = io.StringIO()
with contextlib.redirect_stdout(stdout):
    runpy.run_path(
        "scenarios/web-console/.vrooli/program-runtime/continuity-integrity-audit.py",
        init_globals={"program": FakeProgram(), "web_console": FakeWebConsole},
    )
raw = stdout.getvalue()
for private in ("PRIVATE-TRANSCRIPT", "/private/agent/home"):
    assert private not in raw
# The output remains a valid bounded envelope; this also rejects accidental raw result dumps.
assert isinstance(json.loads(raw), dict)

class BrokenBinding:
    def integrity(self):
        raise RuntimeError("PRIVATE-STATE /private/session/path")


class BrokenWebConsole:
    continuity = BrokenBinding()


error_stdout = io.StringIO()
with contextlib.redirect_stdout(error_stdout):
    runpy.run_path(
        "scenarios/web-console/.vrooli/program-runtime/continuity-integrity-audit.py",
        init_globals={"program": FakeProgram(), "web_console": BrokenWebConsole},
    )
error_raw = error_stdout.getvalue()
assert "PRIVATE-STATE" not in error_raw
assert "/private/session/path" not in error_raw

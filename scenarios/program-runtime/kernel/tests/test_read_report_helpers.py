"""Read-only owner programs emit one parseable envelope through ProgramHelper."""
import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

KERNEL = Path(__file__).resolve().parents[1]
REPO = KERNEL.parents[2]
sys.path.insert(0, str(KERNEL))
sys.path.insert(0, str(KERNEL / "host"))

from host.program_helper import ProgramHelper  # noqa: E402


class Rows:
    def __init__(self, rows=(), meta=None):
        self.rows = list(rows)
        self.metadata = meta or {}

    def head(self, limit):
        return self.rows[:limit]

    def count(self):
        return len(self.rows)

    def meta(self):
        return self.metadata

    def filter(self, predicate):
        return Rows([row for row in self.rows if predicate(row)], self.metadata)

    def map(self, mapper):
        return Rows([mapper(row) for row in self.rows], self.metadata)

    def group_by(self, key):
        grouped = {}
        for row in self.rows:
            value = row[key]
            grouped[value] = grouped.get(value, 0) + 1
        return grouped


def empty(**_kwargs):
    return Rows()


def bindings_for(relative, *, broken=None):
    def read(name, value=None):
        if broken == name:
            raise RuntimeError("connection refused")
        return value if value is not None else Rows()

    if relative.endswith("agent-manager/.vrooli/program-runtime/friction-digest.py"):
        tools = Rows(meta={"coverage": {"sentence": "no calls observed"}})
        return {"agent_manager": SimpleNamespace(
            watch=SimpleNamespace(policy_outcomes=lambda **kw: read("policy")),
            usage=SimpleNamespace(friction=lambda **kw: read("tools", tools)),
        )}
    if relative.endswith("agent-manager/.vrooli/program-runtime/setpoint-read.py"):
        measure = Rows([{"validity": {"state": "available"}, "summary": {"coverage": {}}},
                        {"validity": {"state": "available"}, "rate": 0.9}])
        return {"agent_manager": SimpleNamespace(
            run=SimpleNamespace(list=lambda **kw: read("runs")),
            measures=SimpleNamespace(tool_friction=lambda **kw: read("tools", measure),
                                     run_success_rate=lambda **kw: read("success", measure)),
        ), "gather": lambda *calls: [call() for call in calls]}
    if relative.endswith("command-center/.vrooli/program-runtime/learning-read.py"):
        return {"vrooli_memory": SimpleNamespace(learning=SimpleNamespace(
            measure=lambda **kw: read("learning", Rows(meta={"reliable": True, "eligibleAttempts": 0}))))}
    if relative.endswith("command-center/.vrooli/program-runtime/setpoint-read.py"):
        return {"program_runtime": SimpleNamespace(bindings=SimpleNamespace(
            condition=lambda **kw: read("condition"))),
            "vrooli_memory": SimpleNamespace(learning=SimpleNamespace(
                measure=lambda **kw: read("learning", Rows(meta={"reliable": True})))),
            "gather": lambda *calls: [call() for call in calls]}
    if relative.endswith("plan-manager/.vrooli/program-runtime/setpoint-read.py"):
        return {"program_runtime": SimpleNamespace(bindings=SimpleNamespace(
            condition=lambda **kw: read("condition"))),
            "vrooli_memory": SimpleNamespace(learning=SimpleNamespace(
                measure=lambda **kw: read("learning", Rows([{}], {"reliable": True, "eligibleAttempts": 1})))),
            "gather": lambda *calls: [call() for call in calls]}
    if relative.endswith("prompt-manager/.vrooli/program-runtime/skill-set-read.py"):
        skills = SimpleNamespace(list=lambda: read("skills"),
                                 read=lambda **kw: read("skill_read", Rows(meta={"totalTokens": 0, "skillCount": 0, "missing": []})))
        usage = SimpleNamespace(skill_usage=lambda **kw: read("usage"))
        return {"prompt_manager": SimpleNamespace(skill=skills, skill_usage=usage),
                "program_runtime": SimpleNamespace(library=SimpleNamespace(list=lambda: read("programs")))}
    if relative.endswith("offer-desk/.vrooli/program-runtime/setpoint-read.py"):
        offers = SimpleNamespace(
            board_show=lambda **kw: read("board", Rows(meta={"defaultAliveGap": "No default-alive gap supplied"})),
            meters=lambda **kw: read("meters", Rows(meta={"deliverableMeterGaps": [], "undeclaredStreams": []})),
            space=lambda **kw: read("space", Rows(meta={"denominatorConfidence": "verified"})),
        )
        return {"offer_desk": SimpleNamespace(offers=offers),
                "vrooli_memory": SimpleNamespace(learning=SimpleNamespace(
                    measure=lambda **kw: read("learning", Rows(meta={"reliable": True, "eligibleAttempts": 0})))),
                "gather": lambda *calls: [call() for call in calls]}
    raise AssertionError(relative)


CASES = [
    ("scenarios/agent-manager/.vrooli/program-runtime/friction-digest.py", {"scenario": "fixture"}, "ok", None, None),
    ("scenarios/agent-manager/.vrooli/program-runtime/setpoint-read.py", {"scenario": "fixture"}, "ok", "runs", "partial"),
    ("scenarios/command-center/.vrooli/program-runtime/learning-read.py", {}, "ok", "learning", "partial"),
    ("scenarios/command-center/.vrooli/program-runtime/setpoint-read.py", {}, "ok", "condition", "partial"),
    ("scenarios/plan-manager/.vrooli/program-runtime/setpoint-read.py", {}, "ok", "condition", "partial"),
    ("scenarios/prompt-manager/.vrooli/program-runtime/skill-set-read.py", {"scenario": "fixture"}, "ok", "usage", "partial"),
    ("scenarios/offer-desk/.vrooli/program-runtime/setpoint-read.py", {}, "ok", "board", "partial"),
]


def run(relative, inputs, *, broken=None):
    path = REPO / relative
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    reports = []
    original = helper.report

    def report(envelope=None):
        reports.append(envelope)
        return original(envelope)

    helper.report = report
    namespace = {"inputs": inputs, **bindings_for(relative, broken=broken)}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    result = json.loads(output.getvalue())
    assert len(reports) == 1
    assert output.getvalue().endswith("\n")
    assert result["program"] == contract["name"]
    assert result["phase"] == "report"
    return result


def test_readers_report_once_and_keep_owner_status_policies():
    for relative, inputs, success, broken, failure in CASES:
        result = run(relative, inputs)
        assert result["status"] == success, relative
        assert isinstance(result["signals"], dict), relative
        if broken:
            failed = run(relative, inputs, broken=broken)
            assert failed["status"] == failure, relative
            assert failed["errors"], relative


def test_validation_failures_still_report_once_with_original_identity():
    invalid = {
        "scenarios/agent-manager/.vrooli/program-runtime/friction-digest.py": {},
        "scenarios/agent-manager/.vrooli/program-runtime/setpoint-read.py": {"window_token": "bogus"},
        "scenarios/command-center/.vrooli/program-runtime/learning-read.py": {"unexpected": True},
        "scenarios/plan-manager/.vrooli/program-runtime/setpoint-read.py": {"unexpected": True},
        "scenarios/prompt-manager/.vrooli/program-runtime/skill-set-read.py": {"scenario": ""},
    }
    for relative, inputs in invalid.items():
        result = run(relative, inputs)
        assert result["status"] == "failed", relative
        assert result["errors"][0]["class"] == "invalid_input", relative

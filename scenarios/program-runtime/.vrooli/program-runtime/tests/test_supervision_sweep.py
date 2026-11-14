"""A fleet prefix must not conceal a requested scenario from supervision."""
import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT.parents[1] / "kernel"))
from host.engine import Handle


def run_sweep(rows, scenarios):
    output = io.StringIO()
    def run(states, state):
        while state:
            state = states[state]()
    program = SimpleNamespace(inputs=lambda: {"scenarios": scenarios}, run=run,
                              fail=lambda *args: "report", BindingError=Exception)
    vrooli = SimpleNamespace(scenario=SimpleNamespace(list=lambda **kwargs: Handle(rows)))
    with contextlib.redirect_stdout(output):
        exec(compile((ROOT / "supervision-sweep.py").read_text(), "supervision-sweep", "exec"),
             {"program": program, "vrooli": vrooli})
    return json.loads(output.getvalue())


def test_requested_scenario_after_large_unrelated_fleet_is_found():
    rows = [{"name": f"other-{i}", "status": "running", "healthStatus": "healthy"} for i in range(300)]
    rows.append({"name": "target", "status": "running", "healthStatus": "healthy"})
    result = run_sweep(rows, "target")
    assert result["status"] == "ok"
    assert result["signals"]["found"] == 1
    assert result["signals"]["attention"] == []


def test_duplicate_matches_make_unseen_scenarios_unknown():
    rows = [{"name": "first", "status": "running", "healthStatus": "healthy"} for _ in range(33)]
    rows.append({"name": "second", "status": "running", "healthStatus": "healthy"})
    result = run_sweep(rows, "first,second")
    assert result["status"] == "partial"
    assert result["signals"]["source_truncated"] is True
    assert result["signals"]["attention"] == [{"name": "second", "status": "unknown", "health": "unknown"}]


def test_requested_list_truncation_never_reports_complete():
    rows = [{"name": f"s{i}", "status": "running", "healthStatus": "healthy"} for i in range(33)]
    result = run_sweep(rows, ",".join(f"s{i}" for i in range(33)))
    assert result["status"] == "partial"
    assert result["signals"]["truncated"] is True

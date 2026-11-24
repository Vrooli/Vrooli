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
    values = scenarios if isinstance(scenarios, dict) else {"scenarios": scenarios}
    program = SimpleNamespace(inputs=lambda: values, run=run,
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
    assert result["signals"]["scenarios"] == [
        {"name": "target", "status": "running", "health": "healthy"}]


def test_named_scenario_returns_exact_unhealthy_status_and_health():
    rows = [{"name": "target", "status": "stopped", "healthStatus": "degraded"}]
    result = run_sweep(rows, "target")
    assert result["status"] == "ok"
    assert result["signals"]["scenarios"] == [
        {"name": "target", "status": "stopped", "health": "degraded"}]
    assert result["signals"]["attention"] == result["signals"]["scenarios"]


def test_named_missing_slug_returns_absent_row():
    result = run_sweep([], "missing")
    assert result["status"] == "partial"
    assert result["signals"]["scenarios"] == [
        {"name": "missing", "status": "absent", "health": "unknown"}]
    assert result["signals"]["attention"] == result["signals"]["scenarios"]


def test_duplicate_matches_make_unseen_scenarios_unknown():
    rows = [{"name": "first", "status": "running", "healthStatus": "healthy"} for _ in range(161)]
    rows.append({"name": "second", "status": "running", "healthStatus": "healthy"})
    result = run_sweep(rows, "first,second")
    assert result["status"] == "partial"
    assert result["signals"]["source_truncated"] is True
    assert result["signals"]["attention"] == [{"name": "second", "status": "unknown", "health": "unknown"}]
    assert result["signals"]["scenarios"] == [
        {"name": "first", "status": "running", "health": "healthy"},
        {"name": "second", "status": "unknown", "health": "unknown"},
    ]


def test_requested_list_truncation_never_reports_complete():
    rows = [{"name": f"s{i}", "status": "running", "healthStatus": "healthy"} for i in range(33)]
    result = run_sweep(rows, ",".join(f"s{i}" for i in range(33)))
    assert result["status"] == "partial"
    assert result["signals"]["truncated"] is True


def test_named_status_output_stays_under_default_output_bound():
    rows = [{"name": f"scenario-{i:02}", "status": "running", "healthStatus": "healthy"}
            for i in range(32)]
    requested = ",".join(row["name"] for row in rows)
    result = run_sweep(rows, requested)
    encoded = json.dumps(result, separators=(",", ":")).encode()
    assert result["status"] == "ok"
    assert len(result["signals"]["scenarios"]) == 32
    assert len(encoded) <= 4096


def test_named_unhealthy_output_trims_suffix_and_attention_together():
    rows = [{"name": f"scenario-{i:02}-{'x' * 120}", "status": "stopped", "healthStatus": "degraded"}
            for i in range(32)]
    requested = ",".join(row["name"] for row in rows)
    result = run_sweep(rows, requested)
    signals = result["signals"]
    encoded = json.dumps(result, separators=(",", ":")).encode()
    assert len(encoded) <= 4096
    assert result["status"] == "partial"
    assert signals["truncated"] is True
    assert 0 < len(signals["scenarios"]) < len(rows)
    assert signals["attention"] == signals["scenarios"]


def test_single_oversized_slug_is_not_echoed_past_output_bound():
    slug = "x" * 5000
    result = run_sweep([{"name": slug, "status": "running", "healthStatus": "healthy"}], slug)
    encoded = json.dumps(result, separators=(",", ":")).encode()
    assert len(encoded) <= 4096
    assert result["status"] == "partial"
    assert result["signals"]["truncated"] is True
    assert result["signals"]["scenarios"] == []


def test_all_scenarios_returns_complete_bounded_fleet_summaries():
    rows = [{"name": f"scenario-{i}", "status": "running", "healthStatus": "healthy"}
            for i in range(133)]
    result = run_sweep(rows, {"all_scenarios": True})
    assert result["status"] == "ok"
    assert result["signals"]["requested"] == 133
    assert len(result["signals"]["scenarios"]) == 20
    assert result["signals"]["next_offset"] == 20
    assert result["signals"]["scenarios"][0] == {
        "name": "scenario-0", "status": "running", "health": "healthy"}


def test_all_scenarios_marks_fleet_beyond_bound_partial():
    rows = [{"name": f"scenario-{i}", "status": "running", "healthStatus": "healthy"}
            for i in range(161)]
    result = run_sweep(rows, {"all_scenarios": True})
    assert result["status"] == "partial"
    assert result["signals"]["source_truncated"] is True
    assert len(result["signals"]["scenarios"]) == 20


def test_all_scenarios_pages_reconstruct_133_rows_under_output_cap():
    rows = [{"name": f"scenario-{i:03}", "status": "running" if i % 2 else "available",
             "healthStatus": "healthy" if i % 3 else "unknown"} for i in range(133)]
    collected = []
    offset = 0
    while True:
        result = run_sweep(rows, {"all_scenarios": True, "page_offset": offset, "page_size": 20})
        signals = result["signals"]
        page = signals["scenarios"]
        assert result["status"] == "ok"
        assert signals["total_scenarios"] == 133
        assert len(page) <= 20
        assert len(json.dumps(result, separators=(",", ":")).encode()) <= 4096
        collected.extend(page)
        offset = signals["next_offset"]
        if offset is None:
            break
    assert len(collected) == 133
    assert len({row["name"] for row in collected}) == 133
    assert collected == sorted(collected, key=lambda row: row["name"])


def test_all_scenarios_rejects_page_over_default_output_bound():
    result = run_sweep([], {"all_scenarios": True, "page_size": 21})
    assert result["status"] == "failed"

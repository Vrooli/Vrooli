"""The ledger accepts only complete owner metrics and balanced source receipts."""

import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS

ROOT = Path(__file__).parents[1]


class Handle:
    def __init__(self, rows=None, metadata=None):
        self.rows = [] if rows is None else rows
        self.metadata = {} if metadata is None else metadata

    def count(self):
        return len(self.rows)

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return dict(self.metadata)


def receipt(source, identity, scope=None):
    row = {
        "id": identity,
        "source": source,
        "approach": "dead-path-deletion",
        "scenario": "alpha",
        "before_revision": "abc1234",
        "after_revision": "def5678",
        "boundary_complete": True,
        "change_boundary": [
            {"path": "api/main.go", "language": "go", "category": "production", "added": 3, "deleted": 12},
            {"path": "api/main_test.go", "language": "go", "category": "tests", "added": 5, "deleted": 4},
            {"path": "README.md", "language": "markdown", "category": "docs", "added": 2, "deleted": 1},
        ],
        "validation_refs": ["test-genie:run-1"],
    }
    if source == "ecosystem-hunt":
        row["hunt_round"] = "round-1"
    if source == "delivery":
        row["team_scope"] = scope
    return "DR6_REDUCTION_RECEIPT: " + json.dumps(row, separators=(",", ":"))


def compact_snapshot(snapshot):
    return {
        "schema": "dr6-weekly-debt-snapshot-store/1",
        "observed_at": snapshot["observed_at"],
        "window": snapshot["window"],
        "scenario_metrics": [
            [row.get("scenario"), row.get("production_lines"), row.get("source_hash"), row.get("config_hash"), row.get("metrics_revision"), row.get("complete")]
            for row in snapshot.get("scenario_metrics", [])
        ],
        "template_debt": snapshot.get("template_debt"),
        "lifecycle_inventory": snapshot.get("lifecycle_inventory"),
        "reduction_receipts": snapshot.get("reduction_receipts"),
        "approach_yield": snapshot.get("approach_yield", []),
        "source_trend": snapshot.get("source_trend"),
        "detector_snapshot": snapshot.get("detector_snapshot"),
        "source_coverage": snapshot.get("source_coverage"),
    }


def run_program(window_start="2026-11-08", window_end="2026-11-15", overrides=None, journal_overrides=None, lifecycle_snapshot=None, publish_snapshot=False):
    calls = []
    overrides = overrides or {}

    def read(name, **kwargs):
        calls.append((name, kwargs))
        value = overrides.get(name)
        if isinstance(value, Exception):
            raise value
        if callable(value):
            return value(**kwargs)
        return value

    def fact_metrics(**kwargs):
        scenario = kwargs["target"]["scenario"]
        n = 15 if scenario == "alpha" else 4
        return Handle(
            [{"attributes": {"role": "implementation", "line_count": str(n)}}],
            {"totalFacts": 1, "cache": {"sourceHash": "source-" + scenario, "configHash": "config-" + scenario, "analyzerVersion": "metric-v1"}},
        )

    scopes = Handle([{"id": "agent-memory"}, {"id": "debt-reduction"}, {"id": "ecosystem-hunt"}, {"id": "team:alpha-delivery"}])
    journals = {
        "agent-memory": Handle([{"id": "r1", "createdAt": "2026-11-10T00:00:00Z", "body": receipt("research", "receipt-research")}]),
        "debt-reduction": Handle([{"id": "r2", "createdAt": "2026-11-10T00:00:00Z", "body": receipt("research", "receipt-research")}]),
        "ecosystem-hunt": Handle([{"id": "h1", "createdAt": "2026-11-11T00:00:00Z", "body": receipt("ecosystem-hunt", "receipt-hunt")}]),
        "team:alpha-delivery": Handle([{"id": "d1", "createdAt": "2026-11-12T00:00:00Z", "body": receipt("delivery", "receipt-delivery", "team:alpha-delivery")}]),
    }
    journals.update(journal_overrides or {})
    defaults = {
        "roster": Handle([{"id": "alpha"}, {"id": "beta"}]),
        "template": Handle([{"status": "open"}, {"status": "closed"}]),
        "scopes": scopes,
        "snapshot_history": Handle([]),
        "snapshot_write": Handle([], {"entry": {"id": "snapshot-written"}}),
    }

    def journal(**kwargs):
        if kwargs["scope"] == "debt-ledger-snapshots":
            return read("snapshot_history", **kwargs)
        return journals[kwargs["scope"]]

    def emit(*functions):
        return [function() for function in functions]

    outputs = io.StringIO()
    program = NS(
        inputs=lambda: {
            "window_start": window_start,
            "window_end": window_end,
            "max_scenarios": 160,
            "max_journal_rows": 100,
            "lifecycle_snapshot": lifecycle_snapshot or {},
            "publish_snapshot": publish_snapshot,
        },
        classify=lambda exc: ("unavailable", "source_unavailable"),
    )
    namespace = {
        "program": program,
        "gather": emit,
        "flow_verifier": NS(scenarios=NS(list=lambda **kw: read("roster", **kw))),
        "code_facts": NS(facts=NS(describe=lambda **kw: fact_metrics(**kw))),
        "template_manager": NS(debt=NS(list=lambda **kw: read("template", **kw))),
        "source_ledger": NS(
            scopes=NS(list=lambda **kw: read("scopes", **kw)),
            journal=NS(list=lambda **kw: journal(**kw), note=lambda **kw: read("snapshot_write", **kw)),
        ),
    }
    for name, handle in defaults.items():
        overrides.setdefault(name, handle)
    with contextlib.redirect_stdout(outputs):
        exec(compile((ROOT / "weekly-debt-ledger.py").read_text(), "weekly-debt-ledger", "exec"), namespace)
    return json.loads(outputs.getvalue()), calls


def test_week_windows_must_be_exactly_seven_days_and_read_nothing_on_rejection():
    result, calls = run_program(window_end="2026-11-16")
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"
    assert calls == []


def test_saved_lifecycle_evidence_keeps_original_date_and_provenance():
    artifact = {
        "observed_at": "2026-11-14T15:03:00Z",
        "artifact": "research/topics/debt-reduction/runs/steward-free-host-20261114/evidence/flow-verifier-served-inventory-hunt-113-20261114T1503Z.json",
        "sha256": "1a1ad176536266a9951208849a1e856222965bfade4f4dcfdcee2351507e1ec7",
        "scenario_count": 133,
        "candidate_count": 1917,
        "persisted_count": 84,
    }
    result, _ = run_program(lifecycle_snapshot=artifact)
    inventory = result["signals"]["lifecycle_inventory"]
    assert inventory["status"] == "supplied_saved"
    assert inventory["candidate_count"] == 1917
    assert inventory["persisted_count"] == 84
    assert inventory["sha256"] == artifact["sha256"]
    assert result["signals"]["source_coverage"]["lifecycle_complete"] is False
    assert result["signals"]["source_coverage"]["lifecycle_saved_evidence_present"] is True


def test_saved_lifecycle_evidence_outside_window_is_rejected_before_owner_reads():
    result, calls = run_program(lifecycle_snapshot={"observed_at": "2026-11-07T10:00:00Z"})
    assert result["status"] == "failed"
    assert "inside the requested week" in result["errors"][0]["detail"]
    assert calls == []


def test_balanced_receipts_and_owner_metrics_are_counted_with_unknown_sources_kept_explicit():
    result, calls = run_program()
    signals = result["signals"]
    snapshot = signals["weekly_snapshot"]
    assert result["status"] == "partial"  # lifecycle inventory and detector snapshot are not supplied
    assert [(row["scenario"], row["production_lines"]) for row in snapshot["scenario_metrics"]] == [("alpha", 15), ("beta", 4)]
    assert snapshot["scenario_metrics"][0]["source_hash"] == "source-alpha"
    assert snapshot["template_debt"]["total"] == 2
    assert signals["reduction_receipts"]["research"]["net_reduction_lines"] == 9
    assert signals["reduction_receipts"]["ecosystem-hunt"]["valid_receipts"] == 1
    assert signals["reduction_receipts"]["delivery"]["valid_receipts"] == 1
    assert signals["approach_yield"][0]["mean_net_reduction_per_landed_unit"] == 9
    assert signals["line_trend"]["status"] == "unknown"
    assert signals["source_trend"]["status"] == "unknown"
    assert len([call for call in calls if call[0] == "scopes"]) == 1


def test_general_agent_memory_is_not_mislabeled_as_research_reduction():
    invalid = receipt("research", "bad").replace('"path":"README.md"', '"path":"api/main.go"')
    journals = {
        "agent-memory": Handle([{"id": "bad", "createdAt": "2026-11-10T00:00:00Z", "body": invalid}]),
        "debt-reduction": Handle([]),
    }
    result, _ = run_program(journal_overrides=journals)
    research = result["signals"]["reduction_receipts"]["research"]
    assert research["valid_receipts"] == 0
    assert research["invalid_receipts"] == 0
    assert research["status"] == "unknown"
    assert all(row["source"] != "research" for row in result["signals"]["approach_yield"])


def test_partial_source_population_does_not_publish_approach_yield():
    journals = {
        "debt-reduction": Handle([
            {"id": "valid", "createdAt": "2026-11-10T00:00:00Z", "body": receipt("research", "receipt-research")},
            {"id": "unstructured", "createdAt": "2026-11-11T00:00:00Z", "body": "unclassified research activity"},
        ]),
        "ecosystem-hunt": Handle([]),
        "team:alpha-delivery": Handle([]),
    }
    result, _ = run_program(journal_overrides=journals)
    research = result["signals"]["reduction_receipts"]["research"]
    assert research["valid_receipts"] == 1
    assert research["status"] == "partial"
    assert result["signals"]["approach_yield"] == []


def test_adjacent_week_snapshots_produce_unattributed_role_deltas():
    previous = {
        "schema": "dr6-weekly-debt-snapshot/1",
        "observed_at": "2026-11-09T00:00:00Z",
        "window": {"start_inclusive": "2026-11-01", "end_exclusive": "2026-11-08", "days": 7, "timezone": "UTC"},
        "scenario_metrics": [
            {"scenario": "alpha", "complete": True, "source_hash": "old-alpha", "config_hash": "cfg", "metrics_revision": "metric-v1", "production_lines": 10},
            {"scenario": "beta", "complete": True, "source_hash": "old-beta", "config_hash": "cfg", "metrics_revision": "metric-v1", "production_lines": 4},
        ],
        "template_debt": {"status": "measured", "total": 3, "by_status": {"open": 2, "resolved": 1}},
        "source_coverage": {
            "scenario_metrics_complete": True,
            "template_debt_complete": True,
            "research_receipts_complete": True,
            "ecosystem_hunt_receipts_complete": True,
            "delivery_receipts_complete": True,
        },
        "reduction_receipts": {
            "research": {"status": "measured", "valid_receipts": 1, "journal_rows": 1, "scopes": 1, "unstructured_rows": 0, "invalid_receipts": 0, "net_reduction_lines": 4},
            "ecosystem-hunt": {"status": "measured", "valid_receipts": 1, "journal_rows": 1, "scopes": 1, "unstructured_rows": 0, "invalid_receipts": 0, "net_reduction_lines": 7},
            "delivery": {"status": "measured", "valid_receipts": 1, "journal_rows": 1, "scopes": 1, "unstructured_rows": 0, "invalid_receipts": 0, "net_reduction_lines": 8},
        },
        "approach_yield": [
            {"source": source, "approach": "dead-path-deletion", "landed_units": 1, "mean_net_reduction_per_landed_unit": net}
            for source, net in (("research", 4), ("ecosystem-hunt", 7), ("delivery", 8))
        ],
    }
    prior_row = {
        "createdAt": "2026-11-09T00:00:00Z",
        "id": "prior-entry",
        "body": "DR6_WEEKLY_SNAPSHOT: " + json.dumps({"schema": "dr6-weekly-ledger-record/1", "snapshot": compact_snapshot(previous)}, separators=(",", ":")),
    }
    result, _ = run_program(overrides={"snapshot_history": Handle([prior_row])})
    assert result["signals"]["line_trend"]["status"] == "measured"
    alpha = result["signals"]["weekly_snapshot"]["scenario_metrics"][0]
    assert alpha["weekly_production_line_delta"] == 5
    assert alpha["prior_source_hash"] == "old-alpha"
    source_trend = result["signals"]["source_trend"]
    assert source_trend["status"] == "measured"
    assert {row["source"]: row["net_reduction_line_delta"] for row in source_trend["rows"]} == {
        "research": 5, "ecosystem-hunt": 2, "delivery": 1,
    }
    assert len(source_trend["approach_rows"]) == 3


def test_publish_appends_one_immutable_compact_snapshot_to_source_ledger():
    result, calls = run_program(publish_snapshot=True)
    write = result["signals"]["snapshot_write"]
    assert write == {"status": "stored", "entry_id": "snapshot-written"}
    assert result["signals"]["source_coverage"]["weekly_snapshot_persisted"] is True
    call = next(kwargs for name, kwargs in calls if name == "snapshot_write")
    assert call["scope"] == "debt-ledger-snapshots"
    body = call["body"]
    assert body.startswith("DR6_WEEKLY_SNAPSHOT: ")
    record = json.loads(body[len("DR6_WEEKLY_SNAPSHOT: "):])
    assert record["schema"] == "dr6-weekly-ledger-record/1"
    assert record["snapshot"]["schema"] == "dr6-weekly-debt-snapshot-store/1"
    assert len(record["snapshot"]["scenario_metrics"]) == 2


def test_incomplete_prior_history_is_not_exposed_or_used_as_a_trend():
    prior = {
        "schema": "dr6-weekly-debt-snapshot/1",
        "observed_at": "2026-11-09T00:00:00Z",
        "window": {"start_inclusive": "2026-11-01", "end_exclusive": "2026-11-08"},
        "scenario_metrics": [{"scenario": "alpha", "complete": True, "source_hash": "s", "config_hash": "c", "metrics_revision": "v1", "production_lines": 10}],
        "template_debt": {"status": "measured", "total": 2},
        "source_coverage": {"scenario_metrics_complete": True, "template_debt_complete": True},
        "reduction_receipts": {"research": {"secret_summary": "must not leak"}},
        "approach_yield": [{"private": "unverified"}],
    }
    row = {"createdAt": "2026-11-09T00:00:00Z", "id": "incomplete-prior", "body": "DR6_WEEKLY_SNAPSHOT: " + json.dumps({"schema": "dr6-weekly-ledger-record/1", "snapshot": compact_snapshot(prior)}, separators=(",", ":"))}
    result, _ = run_program(overrides={"snapshot_history": Handle([row])})
    assert result["signals"]["line_trend"]["status"] == "unknown"
    assert result["signals"]["source_trend"]["status"] == "unknown"
    assert result["signals"]["previous_window"] is None
    assert "secret_summary" not in json.dumps(result)
    assert "unverified" not in json.dumps(result)

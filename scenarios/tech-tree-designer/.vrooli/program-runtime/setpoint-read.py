"""Read diagnostic owners; never infer product acceptance from their inventory."""

import json

inputs = program.inputs()
envelope = program.envelope("tech-tree-designer.setpoint-read", "1")
OUTCOMES = [
    "OT-P0-001", "OT-P0-002", "OT-P0-003", "OT-P0-004", "OT-P0-005",
    "OT-P0-006", "OT-P0-007", "OT-P0-008", "OT-P0-009",
    "OT-P1-001", "OT-P1-002", "OT-P1-003", "OT-P1-004", "OT-P1-005",
    "OT-P1-006", "OT-P2-001", "OT-P2-002", "OT-P2-003",
]
envelope["signals"] = {
    "target_revision": "ecosystem-design-v1",
    "acceptance": "unknown",
    "rows": [{"row": key, "reading": None, "target": None,
              "in_band": None, "unavailable": True, "reason": "pending_telemetry"}
             for key in OUTCOMES],
    "diagnostics": [],
    "required_selection": "caller_mandate",
    "unresolved_outcomes": len(OUTCOMES),
    "diagnostics_requested": inputs.get("collect_diagnostics", True),
}
results = {}


# validate: no owner calls before admission.
def step_validate():
    envelope["phase"] = "validate"
    if not isinstance(inputs.get("collect_diagnostics", True), bool):
        return program.fail("failed", "invalid_input", "collect_diagnostics must be boolean", "validate")
    envelope["status"] = "ok"
    return "collect" if inputs.get("collect_diagnostics", True) else "report"


# collect: each read is independently guarded; no writes or inference.
def step_collect():
    envelope["phase"] = "collect"
    calls = [
        lambda: tech_tree_designer.plan.list(),
        lambda: tech_tree_designer.ontology.coverage(rows="classifications"),
        lambda: lib.prompt_manager.skill_set_read(scenario="tech-tree-designer"),
    ]
    values = gather(*[program.guarded(call) for call in calls])
    for key, value in zip(["proto-plans", "ontology", "skill-registry"], values):
        results[key] = value
    return "classify"


def diagnostic(key, reading=None, reason=None):
    envelope["signals"]["diagnostics"].append({
        "row": key, "reading": reading, "target": None, "in_band": None,
        "unavailable": reason is not None, "reason": reason,
    })


# classify: validity is distinct from a zero value and from product acceptance.
def step_classify():
    envelope["phase"] = "classify"
    for key, value in results.items():
        if isinstance(value, Exception):
            status, klass = program.classify(value)
            diagnostic(key, reason="scenario_unreachable" if status == "unavailable" else "unreliable:" + klass)
            envelope["status"] = "partial"
            envelope["errors"].append({"class": klass, "detail": "Diagnostic owner read failed", "where": key})
            continue
        if key == "proto-plans":
            diagnostic(key, value.count())
            envelope["evidence"].append("tech-tree-designer/plan/list")
        elif key == "ontology":
            meta = value.meta()
            if meta.get("graphError"):
                diagnostic(key, reason="unreliable:graph_source")
                envelope["status"] = "partial"
                envelope["errors"].append({"class": "invalid_evidence", "detail": "Ontology owner reports graphError", "where": key})
            else:
                diagnostic(key, {"capabilities": int(meta.get("totalCapabilities", 0)),
                                 "scenarios": int(meta.get("totalScenarios", 0))})
            envelope["evidence"].append("tech-tree-designer/ontology/coverage")
        else:
            child_rows = value.head(1)
            child = child_rows[0] if child_rows else {}
            child_signals = child.get("signals", {})
            child_sensor_rows = child_signals.get("rows", [])
            expected_rows = {"registered-under-scenario-pack", "usage-id-present", "improve-id-present",
                             "set-token-size", "read-counts", "programs-declared", "programs-named-in-usage-skill"}
            valid = (child.get("program") == "prompt-manager.skill-set-read"
                     and child.get("version") == "1"
                     and child.get("status") == "ok"
                     and not child.get("errors")
                     and isinstance(child_signals.get("usage_present"), bool)
                     and isinstance(child_signals.get("improve_present"), bool)
                     and len(child_sensor_rows) == 7
                     and all(isinstance(row, dict) and row.get("unavailable") is False
                             and row.get("reason") is None and row.get("reading") is not None
                             for row in child_sensor_rows)
                     and {row.get("row") for row in child_sensor_rows} == expected_rows)
            if valid:
                diagnostic(key, {"usage_present": child_signals["usage_present"],
                                 "improve_present": child_signals["improve_present"]})
            else:
                diagnostic(key, reason="unreliable:child_evidence")
                envelope["status"] = "partial"
                envelope["errors"].append({"class": "invalid_evidence", "detail": "Skill-set child failed or returned incomplete evidence", "where": key})
            child_meta = value.meta()
            envelope["signals"]["child"] = {
                "status": str(child.get("status", "missing"))[:32],
                "digest": str(child_meta.get("digest", ""))[:128],
                "acceptance_eligible": False,
            }
            envelope["evidence"].append("prompt-manager.skill-set-read")
    return "report"


# report: the helper emits one bounded envelope, including driver failures.
def step_report():
    envelope["phase"] = "report"
    program.report()


program.run({"validate": step_validate, "collect": step_collect,
             "classify": step_classify, "report": step_report}, "validate")

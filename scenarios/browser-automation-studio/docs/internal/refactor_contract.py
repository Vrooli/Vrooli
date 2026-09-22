#!/usr/bin/env python3
"""Validate rehabilitation preparation. This never certifies product behavior."""

import ast
import json
from pathlib import Path


SCENARIO = Path(__file__).resolve().parents[2]
ROOT = SCENARIO.parents[1]


def validate(contract, scenario=SCENARIO):
    errors = []
    policy = contract.get("continuation_policy", {})
    if (policy.get("kind") != "continuous_improvement_until_operator_stop"
            or policy.get("unavailable_validation_stops_work") is not False
            or policy.get("unknown_evidence_counts_as_pass") is not False):
        errors.append("Operator policy requires non-blocking validation without invented passes")
    if (policy.get("auto_complete_on_green") is not False
            or policy.get("auto_complete_after_clean_reviews") is not False):
        errors.append("Green checks and clean reviews must not end the continuous goal")
    work = contract.get("work_model", {})
    if (work.get("kind") != "file_based_continuous_improvement"
            or work.get("plan_manager_allowed") is not False
            or work.get("phased_plan_allowed") is not False):
        errors.append("Operator requires file-based work without a plan")
    for key in ("goal_file", "progress_file", "feedback_file", "decisions_file"):
        if not (scenario / work.get(key, "missing")).is_file():
            errors.append(f"Missing durable work file: {key}")
    goal_path = scenario / work.get("goal_file", "missing")
    if goal_path.is_file() and len(goal_path.read_text()) > 2048:
        errors.append("Launch goal exceeds the 2048-character harness limit")
    expected = {f"BAS-RH-J{i:02d}" for i in range(1, 25)}
    journeys = contract.get("journeys", [])
    ids = [j.get("id") for j in journeys]
    if set(ids) != expected or len(ids) != 24:
        errors.append("The original 24 preservation journeys must remain uniquely represented")
    rows = contract.get("rows", [])
    row_ids = [r.get("id") for r in rows]
    if len(row_ids) != len(set(row_ids)) or not rows:
        errors.append("Outcome IDs must be nonempty and unique")
    for row in rows:
        if not row.get("required") or not row.get("band") or not row.get("producer_owner"):
            errors.append(f"Incomplete required outcome: {row.get('id')}")
        if not row.get("producer_roots") or not row.get("implementation_obligation"):
            errors.append(f"No qualification producer route: {row.get('id')}")
    for journey in journeys:
        text = journey.get("description", "").lower()
        if not all(part in text for part in ("given ", "when ", "then ")):
            errors.append(f"Missing behavioral assertion: {journey.get('id')}")
        if not journey.get("oracle") or not journey.get("phases"):
            errors.append(f"Missing independent oracle/phase: {journey.get('id')}")
    for key in ("design", "assessment", "issue_register", "qualification_protocol"):
        if not (scenario / contract.get(key, "missing")).is_file():
            errors.append(f"Missing source: {key}")
    for item in rows + journeys:
        for name in item.get("producer_roots", []):
            base = ROOT if name.startswith("scenarios/") else scenario
            if not (base / name).exists():
                errors.append(f"Missing producer root: {name}")
    current = json.loads((scenario / ".vrooli/testing.json").read_text())["phases"]["tidiness"]["budgets"]
    for key, value in contract["protected_tidiness_budgets"].items():
        if isinstance(value, int) and not isinstance(value, bool) and key != "reserve":
            if current.get(key, value + 1) > value:
                errors.append(f"Tidiness budget was weakened: {key}")
    source = (scenario / ".vrooli/program-runtime/setpoint-read.py").read_text()
    tree = ast.parse(source)
    declared = None
    for node in tree.body:
        if isinstance(node, ast.Assign) and any(isinstance(t, ast.Name) and t.id == "REHABILITATION_ROWS" for t in node.targets):
            declared = ast.literal_eval(node.value)
    if declared != row_ids:
        errors.append("Governed rehabilitation board does not cover the canonical outcomes")
    registry = json.loads((scenario / "requirements/08-rehabilitation/module.json").read_text())
    if {r["id"] for r in registry["requirements"]} != expected:
        errors.append("Preservation requirement registry is incomplete")
    return errors


if __name__ == "__main__":
    contract = json.loads((SCENARIO / "docs/internal/REFRACTOR_CONTRACT.json").read_text())
    errors = validate(contract)
    print(json.dumps({
        "contract": contract["id"], "preparation_valid": not errors,
        "product_qualified": False,
        "meaning": "Preparation consistency only; owner-produced behavioral evidence is required for product qualification",
        "required_outcomes": len(contract["rows"]),
        "preservation_journeys": len(contract["journeys"]), "errors": errors,
    }, indent=2))
    raise SystemExit(bool(errors))

"""Bounded, read-only projection of one Agent Manager owner observation."""

NAME = "agent-manager.supervision-observation-read"
inputs = program.inputs()
envelope = program.envelope(NAME, "1")


def step_validate():
    if not isinstance(inputs, dict) or set(inputs) != {"owner_observation", "selected_effort_refs"}:
        return program.fail("failed", "invalid_input", "owner_observation and selected_effort_refs are required", "validate")
    selected = inputs["selected_effort_refs"]
    owner = inputs["owner_observation"]
    if (not isinstance(selected, list) or not selected or len(selected) > 16 or
            any(not isinstance(ref, str) or not ref.strip() or len(ref) > 512 for ref in selected) or
            len(set(selected)) != len(selected) or not isinstance(owner, dict)):
        return program.fail("failed", "invalid_input", "selection must contain 1-16 unique bounded effort references", "validate")
    rows = owner.get("efforts")
    if not isinstance(rows, list) or len(rows) > 16:
        return program.fail("failed", "invalid_input", "owner observation must contain at most 16 effort rows", "validate")
    envelope["inputs"] = {"selected_effort_refs": selected, "owner_observation_keys": sorted(owner.keys())[:16]}
    return "collect"


def step_collect():
    selected = inputs["selected_effort_refs"]
    by_id = {}
    for row in inputs["owner_observation"].get("efforts", []):
        if not isinstance(row, dict) or not isinstance(row.get("id"), str) or not row["id"].strip():
            return program.fail("failed", "invalid_input", "owner row is missing a bounded id", "collect")
        if row["id"] in by_id:
            return program.fail("failed", "identity_mismatch", "duplicate owner effort identity", "collect")
        by_id[row["id"]] = row

    projected, missing, refs = [], [], []
    for effort_id in selected:
        row = by_id.get(effort_id)
        if row is None:
            missing.append(effort_id)
            continue
        compact = {
            "id": effort_id,
            "targetRevision": row.get("targetRevision", ""),
            "evidenceRevision": row.get("evidenceRevision", ""),
            "eligible": bool(row.get("eligible", False)),
            "observationOnly": bool(row.get("observationOnly", False)),
            "retired": bool(row.get("retired", False)),
            "reason": str(row.get("reason", ""))[:512],
            "priorAssessment": row.get("priorAssessment"),
            "changedEvidence": list(row.get("changedEvidence", []))[:16],
            "usage": row.get("usage"),
            "quotaObservations": list(row.get("quotaObservations", []))[:16],
            "namedWaits": list(row.get("namedWaits", []))[:16],
            "detailRefs": list(row.get("detailRefs", []))[:16],
            "repairLinks": list(row.get("repairLinks", []))[:8],
        }
        compact["detailRefs"] = [ref for ref in compact["detailRefs"] if isinstance(ref, str) and ref.strip()][:16]
        refs.extend(compact["detailRefs"])
        projected.append(compact)

    refs = list(dict.fromkeys(refs))[:32]
    envelope["phase"] = "collect"
    envelope["status"] = "ok"
    envelope["signals"] = {
        "efforts": projected,
        # Agent Manager joins provider observations once at board scope. Keep
        # that cut separate from effort rows so a shared pool is not repeated.
        "quotaObservations": list(inputs["owner_observation"].get("quotaObservations", []))[:16],
        "selected_count": len(selected),
        "projected_count": len(projected),
        "missing_effort_refs": missing,
        "coverage": inputs["owner_observation"].get("coverage", "unknown"),
        "owner_error": str(inputs["owner_observation"].get("error", ""))[:256],
        "detail_refs": refs,
        "detail_refs_truncated": len(refs) >= 32,
        "next_action": "inspect_named_detail_refs" if projected else "retain_owner_observation_unknown",
    }
    envelope["evidence"] = list(dict.fromkeys(selected + refs))[:32]
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")

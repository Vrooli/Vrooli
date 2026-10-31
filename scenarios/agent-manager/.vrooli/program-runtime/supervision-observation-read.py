"""One bounded owner board read and deterministic supervision selection."""

NAME = "agent-manager.supervision-observation-read"
inputs = program.inputs()
envelope = program.envelope(NAME, "3")


def bounded_text(value, limit=512):
    return value[:limit] if isinstance(value, str) else ""


def owner_refs(row):
    refs = []
    for assignment in row.get("assignments", []):
        subject = assignment.get("subject") or {}
        role = subject.get("role", "")
        ref = subject.get("runId") or subject.get("reference") or ""
        if role in ("orchestrator", "worker") and isinstance(ref, str) and ref.strip():
            refs.append({
                "role": role,
                "reference": ref,
                "runtime_state": bounded_text(assignment.get("runtimeState"), 64) or "unknown",
                "unavailable_reason": bounded_text(assignment.get("unavailableReason"), 160),
            })
    return refs


def prioritize_owner_refs(refs):
    # Keep retained obligations ahead of historical terminal references when
    # the compact projection reaches its fixed cap. Unknown states remain
    # visible as potentially unfinished; only known terminal run states sort
    # behind them. Python's sort is stable, preserving source order in each
    # group.
    terminal_states = {"complete", "failed", "cancelled"}
    return sorted(refs, key=lambda ref: ref["runtime_state"].strip().lower() in terminal_states)


def step_validate():
    if not isinstance(inputs, dict):
        return program.fail("failed", "invalid_input", "inputs must be an object", "validate")
    page_size = inputs.get("page_size", 8)
    page_token = inputs.get("page_token", "")
    if (not isinstance(page_size, int) or isinstance(page_size, bool) or page_size < 1 or page_size > 8 or
            not isinstance(page_token, str) or len(page_token) > 512):
        return program.fail("failed", "invalid_input", "page_size must be 1-8 and page_token at most 512 characters", "validate")
    envelope["inputs"] = {"page_size": page_size, "page_token_supplied": bool(page_token)}
    envelope["phase"] = "collect"
    try:
        owner = agent_manager.effort.board(page_size=page_size, page_token=page_token, rows="rows")
        board = owner.raw()
    except Exception:
        envelope["signals"] = {
            "coverage": "unavailable",
            "selected_count": 0,
            "next_action": "retain_owner_observation_unknown",
        }
        return program.fail("unavailable", "owner_read_unavailable", "Agent Manager effort board read is unavailable; eligibility remains unknown", "collect")
    if not isinstance(board, dict) or not isinstance(board.get("rows"), list) or len(board["rows"]) > 8:
        return program.fail("failed", "identity_mismatch", "Agent Manager returned an invalid or oversized board page", "collect")
    return collect_board(board, page_size)


def collect_board(board, page_size):
    projected = []
    evidence = []
    for row in board["rows"]:
        if not isinstance(row, dict):
            return program.fail("failed", "identity_mismatch", "Agent Manager board contains a malformed row", "collect")
        enrollment = row.get("enrollment") or {}
        effort_ref = enrollment.get("effortRef")
        if not isinstance(effort_ref, str) or not effort_ref.strip() or len(effort_ref) > 512:
            return program.fail("failed", "identity_mismatch", "Agent Manager board row is missing its exact effort identity", "collect")
        if any(item["id"] == effort_ref for item in projected):
            return program.fail("failed", "identity_mismatch", "Agent Manager board contains a duplicate effort identity", "collect")

        withdrawn = bool(enrollment.get("withdrawn"))
        all_owner_refs = prioritize_owner_refs(owner_refs(row))
        owner_refs_total = len(all_owner_refs)
        owner_refs_truncated = owner_refs_total > 8
        retained_refs = all_owner_refs[:8]
        home = row.get("goalHome") or {}
        if withdrawn:
            eligible, disposition = False, "retired"
            reason = bounded_text(enrollment.get("withdrawalReason"), 256) or "owner enrollment is withdrawn"
        elif retained_refs:
            eligible, disposition = True, "unfinished_retained_owner"
            reason = "Agent Manager retained orchestrator/worker assignment; goal-home state is supporting evidence only"
        else:
            eligible, disposition = False, "admission_or_completion_unknown"
            reason = "no native retained orchestrator/worker assignment; populated goal-home state does not establish admission or retained ownership"

        detail_ref = "agent-manager:GetEffortBoard:" + effort_ref
        evidence.append(detail_ref)
        projected.append({
            "id": effort_ref,
            "display_name": bounded_text(enrollment.get("displayName"), 160),
            "disposition": disposition,
            "eligible": eligible,
            "reason": reason,
            "runtime_state": bounded_text(row.get("runtimeState"), 64) or "unknown",
            "freshness": bounded_text(row.get("freshness"), 64) or "unknown",
            "outcome_state": bounded_text((row.get("outcomeStanding") or {}).get("state"), 64) or "unknown",
            "owner_refs": retained_refs,
            "owner_refs_total": owner_refs_total,
            "owner_refs_truncated": owner_refs_truncated,
            "goal_home": {
                "path": bounded_text(home.get("path"), 512),
                "state": bounded_text(home.get("state"), 64) or ("not_registered" if not home else "unknown"),
                "active_epoch": bounded_text(home.get("activeEpoch"), 32),
                "active_status": bounded_text(home.get("activeStatus"), 64),
                "active_workers": [bounded_text(v, 96) for v in home.get("activeWorkers", [])[:4] if isinstance(v, str)],
                "needs_operator": [bounded_text(v, 160) for v in home.get("needsOperator", [])[:4] if isinstance(v, str)],
                "open_feedback_ids": [bounded_text(v, 96) for v in home.get("openFeedbackIds", [])[:4] if isinstance(v, str)],
                "open_workarounds": home.get("openWorkarounds"),
                "open_decisions": len(home.get("openDecisions", [])) if isinstance(home.get("openDecisions"), list) else None,
                "lint_findings": [bounded_text(v, 96) for v in home.get("lintFindings", [])[:4] if isinstance(v, str)],
                "limitations": [bounded_text(v, 160) for v in home.get("limitations", [])[:4] if isinstance(v, str)],
            },
            "next_action": bounded_text(row.get("nextAction"), 512),
            "blockers": [bounded_text(v, 160) for v in row.get("blockers", [])[:4] if isinstance(v, str)],
            "limitations": [bounded_text(v, 160) for v in row.get("limitations", [])[:4] if isinstance(v, str)],
            "evidence_refs": [bounded_text(v, 160) for v in row.get("evidenceRefs", [])[:4] if isinstance(v, str)],
            "detail_ref": detail_ref,
        })

    discovery = board.get("discovery") or {}
    next_page = bounded_text(board.get("nextPageToken"), 512)
    partial = bool(board.get("partial")) or bool(discovery.get("partial")) or bool(next_page)
    envelope["phase"] = "report"
    envelope["status"] = "partial" if partial else "ok"
    envelope["signals"] = {
        "efforts": projected,
        "selected_count": sum(1 for row in projected if row["eligible"]),
        "visible_count": len(projected),
        "coverage": "partial" if partial else "complete",
        "observed_at": board.get("observedAt"),
        "active_count": board.get("activeCount"),
        "discovery": {
            "partial": bool(discovery.get("partial")),
            "scanned_count": discovery.get("scannedCount"),
            "scan_limit": discovery.get("scanLimit"),
            "last_successful_scan_at": discovery.get("lastSuccessfulScanAt"),
            "findings": [
                {"source": bounded_text(item.get("source"), 128), "code": bounded_text(item.get("code"), 96), "reason": bounded_text(item.get("reason"), 256)}
                for item in discovery.get("findings", [])[:8] if isinstance(item, dict)
            ],
        },
        "board_limitations": [bounded_text(v, 256) for v in board.get("limitations", [])[:8] if isinstance(v, str)],
        "next_page_token": next_page,
        "page_has_more": bool(next_page),
        "authorized_by_used_for_selection": False,
        "prepared_team_state": "not represented distinctly by the current AM board",
        "next_action": "inspect_selected_owner_obligations" if any(row["eligible"] for row in projected) else "retain_owner_observation_unknown",
    }
    envelope["evidence"] = evidence[:8]
    program.report()
    return None


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


program.run({"validate": step_validate, "report": step_report}, "validate")

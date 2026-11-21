"""Reconcile one enrolled effort's retained decisions with Hub delivery evidence."""
import json

NAME = "program-runtime.research-attention-delivery"
inputs = program.inputs()
envelope = program.envelope(NAME, "1")


def step_validate():
    ref = inputs.get("effort_ref", "")
    token = inputs.get("decision_page_token", "")
    include_board = inputs.get("include_board", False)
    if not isinstance(ref, str) or not ref.strip() or len(ref) > 512:
        return program.fail("failed", "invalid_input", "an exact effort_ref is required", "validate")
    if not isinstance(token, str) or len(token) > 4096 or not isinstance(include_board, bool):
        return program.fail("failed", "invalid_input", "invalid page token or board option", "validate")
    envelope["inputs"] = {"effort_ref": ref, "decision_page_token": token, "include_board": include_board}
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    used = {"agent-manager/effort/decisions-page"}
    try:
        decision_page = agent_manager.effort.decisions_page(effort_ref=inputs["effort_ref"], limit=16, page_token=inputs.get("decision_page_token", "")).raw()
        decisions = decision_page.get("decisions", []) if isinstance(decision_page, dict) else None
        if not isinstance(decisions, list) or len(decisions) > 16:
            raise ValueError("invalid decision projection")
        if any(not isinstance(d, dict) or d.get("effortRef") != inputs["effort_ref"] for d in decisions):
            raise ValueError("decision effort identity mismatch")
    except Exception as exc:
        envelope["signals"] = {"coverage": "unknown", "items": []}
        return program.fail("unavailable", "owner_unavailable", str(exc)[:200], "collect")
    items = []
    incomplete = bool(decision_page.get("nextPageToken"))
    for decision in decisions:
        item = {"decision_id": decision.get("decisionId", ""), "state": decision.get("state", ""), "delivery": "unknown"}
        if decision.get("state") in ("answered", "defaulted", "expired"):
            item["delivery"] = "not_pending"
        elif decision.get("state") != "pending":
            item["delivery"] = "unknown_decision_state"
            incomplete = True
        elif not decision.get("eventId"):
            item["delivery"] = "unknown_missing_event_id"
            incomplete = True
        elif not decision.get("publishedAt"):
            item["delivery"] = "unpublished"
        else:
            try:
                event = decision["eventId"]
                used.add("notification-hub/conversations/list")
                page = notification_hub.conversations.list(query=event, limit=20, rows="asks").raw()
                asks = page.get("asks", [])
                if not isinstance(asks, list):
                    raise ValueError("invalid asks")
                matches = [a for a in asks if a.get("requesterRef") == event and a.get("source") == "agent-manager" and a.get("sourceEventType") == "agent_manager.effort.decision_requested.v1"]
                if len(matches) > 1:
                    raise ValueError("duplicate ask identities")
                if not matches:
                    item["delivery"] = "unknown_incomplete_search" if page.get("nextCursor") else "ask_not_observed"
                    incomplete = True
                else:
                    ask = matches[0]
                    item["ask_id"] = ask.get("id", "")
                    item["ask_state"] = ask.get("state", "")
                    if item["ask_state"] not in ("pending", "escalated", "answered", "defaulted", "expired", "withdrawn"):
                        item["resolution"] = "unknown_ask_state"
                        incomplete = True
                    elif item["ask_state"] not in ("pending", "escalated"):
                        # The selected Agent Manager decision is still pending.
                        # A terminal Hub ask requires owner reconciliation even
                        # when a delivery receipt exists.
                        item["resolution"] = "ask_terminal_decision_pending"
                        incomplete = True
                    notice_id = ask.get("notificationId", "")
                    if not notice_id:
                        raise ValueError("ask has no notification identity")
                    used.add("notification-hub/notifications/get")
                    notice = notification_hub.notifications.get(id=notice_id).raw()
                    info = notice.get("notification") or {}
                    if info.get("id") != notice_id:
                        raise ValueError("notification identity mismatch")
                    item["notification_id"] = notice_id
                    item["delivery"] = info.get("state") or "unknown"
                    receipts = notice.get("receipts", [])
                    item["delivered_receipt"] = any(r.get("notificationId") == notice_id and r.get("deliveredAt") for r in receipts)
                    if not item["delivered_receipt"]:
                        incomplete = True
            except Exception as exc:
                item["delivery"] = "unknown_owner_read"
                item["error"] = str(exc)[:160]
                incomplete = True
        items.append(item)
    board_coverage = "not_requested"
    if inputs.get("include_board", False):
        used.add("agent-manager/effort/board")
        try:
            board = agent_manager.effort.board(effort_ref=inputs["effort_ref"], page_size=1, rows="rows").raw()
            rows = board.get("rows") if isinstance(board, dict) else None
            if not isinstance(rows, list) or len(rows) != 1 or rows[0].get("enrollment", {}).get("effortRef") != inputs["effort_ref"]:
                raise ValueError("board identity unavailable")
            home = rows[0].get("goalHome") or {}
            lines = home.get("needsOperator", [])
            if not isinstance(lines, list):
                raise ValueError("board attention unavailable")
            known = {d.get("decisionId") for d in decisions if isinstance(d.get("decisionId"), str)}
            unlinked = sum(1 for line in lines if not isinstance(line, str) or not any(line.lstrip(" -*").startswith("[decision " + key + "]") for key in known))
            if unlinked:
                items.append({"decision_id": "unlinked-needs-operator", "delivery": "unknown_unlinked_attention", "count": unlinked})
            board_coverage = "partial" if board.get("partial") or home.get("state") != "available" or decision_page.get("nextPageToken") else "selected_effort"
        except Exception:
            board_coverage = "unknown"
        if board_coverage != "selected_effort" or any(item.get("delivery") == "unknown_unlinked_attention" for item in items):
            incomplete = True
    envelope["signals"] = {"coverage": "partial" if incomplete else "selected_page_only", "items": items, "decision_count": len(decisions), "next_page_token": decision_page.get("nextPageToken", ""), "board_coverage": board_coverage, "attention_view_coverage": "unknown"}
    # The narrative attention index is not represented by this owner board.
    # Never present this bounded join as complete research-attention coverage.
    envelope["status"] = "partial"
    envelope["evidence"] = [{"binding": name} for name in sorted(used)]
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None

program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")

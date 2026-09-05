"""agent-manager.supervision-evaluate v1 — one finite, read-only supervision decision. [REQ:REQ-P2-010]

Contract: supervision-evaluate.json. Nested data contracts live in schemas/.
Phases: validate -> collect -> classify -> decide -> report.
The program recommends an action; Agent Manager alone authorizes and applies it.
"""

import json

try:
    inputs
except NameError:
    inputs = {}

raw = inputs if isinstance(inputs, dict) else {}
watch_id = raw.get("watch_id", "")
events = raw.get("events", [])
friction = raw.get("friction_episodes", [])
runs = raw.get("run_summaries", [])
history = raw.get("prior_decisions", [])
policy = raw.get("policy", {})
current_cursor = raw.get("current_cursor", "")
proposed_cursor = raw.get("proposed_next_cursor", "")
cursor_reset = raw.get("cursor_reset_required", False)
reset_reason = raw.get("reset_reason", "")
allow_inference = raw.get("allow_inference", True)
input_byte_budget = raw.get("input_byte_budget", 32768)

envelope = {
    "program": "agent-manager.supervision-evaluate", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"watch_id": str(watch_id)[:128], "policy_version": "", "input_bytes": 0},
    "signals": {
        "disposition": "unavailable", "classification": "invalid_input",
        "confidence": None, "abstained": True, "recommended_action": "observe",
        "next_cursor": str(current_cursor)[:512] if isinstance(current_cursor, str) else "",
        "cursor_reset": False, "wake_condition": {"kind": "after", "after_seconds": 30},
        "policy_version": "", "inference_calls": 0,
        "event_count": 0, "friction_count": 0, "run_count": 0,
    },
    "errors": [], "evidence": [],
}
work = {"ambiguous": False, "classifier": None, "actions": []}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")


def remember(value):
    value = str(value or "")[:128]
    if value and value not in envelope["evidence"] and len(envelope["evidence"]) < 20:
        envelope["evidence"].append(value)


def allowed(action):
    configured = policy.get("allowed_actions", []) if isinstance(policy, dict) else []
    return action if action in configured else "observe"


def set_decision(disposition, classification, confidence, abstained, action, cursor, wake_kind, wake_seconds=0):
    sig = envelope["signals"]
    sig.update({
        "disposition": disposition,
        "classification": classification,
        "confidence": confidence,
        "abstained": abstained,
        "recommended_action": allowed(action),
        "next_cursor": cursor,
        "cursor_reset": disposition == "cursor_reset",
        "wake_condition": {"kind": wake_kind, "after_seconds": max(0, min(int(wake_seconds), 86400))},
    })


def valid_text(value, maximum, allow_empty=True):
    return isinstance(value, str) and len(value) <= maximum and (allow_empty or bool(value))


def step_validate():  # VALIDATE · no bindings
    global input_byte_budget
    if not isinstance(inputs, dict):
        return fail("failed", "invalid_input", "inputs must be an object", "validate")
    if not isinstance(input_byte_budget, int) or isinstance(input_byte_budget, bool) or input_byte_budget < 1024 or input_byte_budget > 65536:
        return fail("failed", "invalid_input", "input_byte_budget must be an integer in [1024, 65536]", "validate")
    try:
        input_bytes = len(json.dumps(inputs, sort_keys=True, separators=(",", ":")).encode("utf-8"))
    except Exception as exc:
        return fail("failed", "invalid_input", f"input is not JSON serializable: {exc}", "validate")
    envelope["inputs"]["input_bytes"] = input_bytes
    if input_bytes > input_byte_budget:
        envelope["signals"]["classification"] = "budget_exhausted"
        return fail("failed", "input_budget_exhausted", f"input uses {input_bytes} bytes; budget is {input_byte_budget}", "validate")
    if not isinstance(policy, dict):
        return fail("failed", "invalid_input", "policy must be an object", "validate")
    required = ("version", "event_count_threshold", "friction_threshold", "quiet_seconds", "event_count_enabled", "friction_enabled", "terminal_enabled", "deadline_reached", "quiet_reached", "allowed_actions")
    missing = [key for key in required if key not in policy]
    if missing:
        return fail("failed", "invalid_input", "policy missing: " + ", ".join(missing), "validate")
    if not valid_text(policy.get("version"), 64, False):
        return fail("failed", "invalid_input", "policy.version must be a non-empty string up to 64 bytes", "validate")
    event_threshold = policy.get("event_count_threshold")
    quiet_seconds = policy.get("quiet_seconds")
    friction_threshold = policy.get("friction_threshold")
    if not isinstance(event_threshold, int) or isinstance(event_threshold, bool) or event_threshold < 1 or event_threshold > 64:
        return fail("failed", "invalid_input", "policy.event_count_threshold must be in [1, 64]", "validate")
    if not isinstance(quiet_seconds, int) or isinstance(quiet_seconds, bool) or quiet_seconds < 1 or quiet_seconds > 86400:
        return fail("failed", "invalid_input", "policy.quiet_seconds must be in [1, 86400]", "validate")
    if not isinstance(friction_threshold, (int, float)) or isinstance(friction_threshold, bool) or friction_threshold < 0 or friction_threshold > 1:
        return fail("failed", "invalid_input", "policy.friction_threshold must be in [0, 1]", "validate")
    for flag in ("event_count_enabled", "friction_enabled", "terminal_enabled", "deadline_reached", "quiet_reached"):
        if not isinstance(policy.get(flag), bool):
            return fail("failed", "invalid_input", f"policy.{flag} must be boolean", "validate")
    valid_actions = {"observe", "nudge", "park", "continue", "stop", "escalate", "wake_parent"}
    actions = policy.get("allowed_actions")
    if not isinstance(actions, list) or len(actions) > 7 or len(set(actions)) != len(actions) or any(a not in valid_actions for a in actions):
        return fail("failed", "invalid_input", "policy.allowed_actions contains an invalid or duplicate action", "validate")
    for value, name, limit in ((events, "events", 64), (friction, "friction_episodes", 16), (runs, "run_summaries", 32), (history, "prior_decisions", 8)):
        if not isinstance(value, list) or len(value) > limit:
            envelope["signals"]["classification"] = "budget_exhausted"
            return fail("failed", "input_budget_exhausted", f"{name} must be an array with at most {limit} items", "validate")
    if not valid_text(watch_id, 128) or not valid_text(current_cursor, 512) or not valid_text(proposed_cursor, 512):
        return fail("failed", "invalid_input", "watch_id and cursors must be bounded strings", "validate")
    if not isinstance(cursor_reset, bool) or not isinstance(allow_inference, bool) or not valid_text(reset_reason, 240):
        return fail("failed", "invalid_input", "reset and inference controls have invalid types", "validate")
    for event in events:
        if not isinstance(event, dict) or any(key not in event for key in ("event_id", "run_id", "sequence", "event_type")):
            return fail("failed", "invalid_input", "each event requires event_id, run_id, sequence, and event_type", "validate")
        if not valid_text(event["event_id"], 128, False) or not valid_text(event["run_id"], 128, False) or not valid_text(event["event_type"], 96, False):
            return fail("failed", "invalid_input", "event identifiers and type must be bounded non-empty strings", "validate")
        if not isinstance(event["sequence"], int) or isinstance(event["sequence"], bool) or event["sequence"] < 0:
            return fail("failed", "invalid_input", "event sequence must be a non-negative integer", "validate")
    for episode in friction:
        if not isinstance(episode, dict) or any(key not in episode for key in ("evidence_id", "score", "pattern")):
            return fail("failed", "invalid_input", "each friction episode requires evidence_id, score, and pattern", "validate")
        if not valid_text(episode["evidence_id"], 128, False) or not valid_text(episode["pattern"], 240, False):
            return fail("failed", "invalid_input", "friction evidence_id and pattern must be bounded non-empty strings", "validate")
        if not isinstance(episode["score"], (int, float)) or isinstance(episode["score"], bool) or episode["score"] < 0 or episode["score"] > 1:
            return fail("failed", "invalid_input", "friction score must be in [0, 1]", "validate")
    for summary in runs:
        if not isinstance(summary, dict) or not valid_text(summary.get("run_id"), 128, False) or not valid_text(summary.get("status"), 48, False):
            return fail("failed", "invalid_input", "each run summary requires bounded run_id and status", "validate")
        if "blocked" in summary and not isinstance(summary["blocked"], bool):
            return fail("failed", "invalid_input", "run summary blocked must be boolean", "validate")
        if "needs_review" in summary and not isinstance(summary["needs_review"], bool):
            return fail("failed", "invalid_input", "run summary needs_review must be boolean", "validate")
    valid_dispositions = {"quiet", "signal", "terminal", "cursor_reset", "unavailable"}
    for decision in history:
        if not isinstance(decision, dict) or not valid_text(decision.get("decision_id"), 128, False) or decision.get("disposition") not in valid_dispositions:
            return fail("failed", "invalid_input", "each prior decision requires a bounded decision_id and valid disposition", "validate")
        if "classification" in decision and not valid_text(decision["classification"], 64):
            return fail("failed", "invalid_input", "prior decision classification must be a bounded string", "validate")
    envelope["inputs"]["policy_version"] = policy["version"]
    envelope["signals"]["policy_version"] = policy["version"]
    envelope["signals"]["wake_condition"]["after_seconds"] = quiet_seconds
    return "collect"


def step_collect():  # COLLECT · optional bounded read bindings
    global events, cursor_reset, reset_reason
    envelope["phase"] = "collect"
    if watch_id and not events:
        try:
            inspection, actions = gather(
                lambda: agent_manager.watch.inspect(watch_id=watch_id, event_limit=64),
                lambda: agent_manager.watch.actions(watch_id=watch_id, limit=8),
            )
            events = [{
                "event_id": row.get("eventId", ""), "run_id": row.get("runId", ""),
                "sequence": int(row.get("sequence", 0)), "event_type": row.get("eventType", ""),
            } for row in inspection.head(64)]
            meta = inspection.meta()
            cursor_reset = bool(meta.get("cursorResetRequired", False))
            reset_reason = str(meta.get("resetReason", ""))[:240]
            watch = meta.get("watch") or {}
            watch_policy = ((watch.get("spec") or {}).get("policyVersion"))
            if watch_policy and watch_policy != policy["version"]:
                return fail("failed", "invalid_input", f"watch policy {watch_policy} does not match input policy {policy['version']}", "collect")
            work["actions"] = actions.head(8)
            remember(watch.get("watchId") or watch_id)
            for action in work["actions"]:
                remember(action.get("actionId"))
        except Exception as exc:
            status, klass = classify_transport(exc)
            set_decision("unavailable", "dependency_unavailable", None, True, "observe", current_cursor, "after", policy["quiet_seconds"])
            return fail(status, klass, exc, "collect")
    envelope["signals"]["event_count"] = len(events)
    envelope["signals"]["friction_count"] = len(friction)
    envelope["signals"]["run_count"] = len(runs)
    for event in events:
        remember(event.get("event_id"))
    for episode in friction:
        remember(episode.get("evidence_id"))
    for summary in runs:
        remember(summary.get("run_id"))
    for decision in history:
        if isinstance(decision, dict):
            remember(decision.get("decision_id"))
    return "classify"


def step_classify():  # CLASSIFY · deterministic predicates before at most one inference
    envelope["phase"] = "classify"
    if cursor_reset:
        set_decision("cursor_reset", "cursor_gap", 1.0, False, "observe", None, "immediate")
        envelope["status"] = "ok"
        return "decide"
    terminal_states = {"completed", "failed", "canceled", "cancelled", "stopped", "succeeded"}
    failed_states = {"failed", "blocked", "stopped"}
    statuses = [str(row.get("status", "")).lower() for row in runs]
    if policy["terminal_enabled"] and statuses and all(status in terminal_states for status in statuses):
        classification = "failed" if any(status in failed_states for status in statuses) else "completed"
        set_decision("terminal", classification, 1.0, False, "wake_parent", proposed_cursor, "terminal")
        envelope["status"] = "ok"
        return "decide"
    if any(bool(row.get("blocked")) or bool(row.get("needs_review")) or str(row.get("status", "")).lower() in failed_states for row in runs):
        set_decision("signal", "blocked", 1.0, False, "escalate", proposed_cursor, "immediate")
        envelope["status"] = "ok"
        return "decide"
    event_types = [str(row.get("event_type", "")).lower() for row in events]
    max_friction = max([float(row.get("score", 0)) for row in friction] or [0.0])
    if policy["deadline_reached"]:
        set_decision("signal", "deadline", 1.0, False, "escalate", proposed_cursor, "immediate")
        envelope["status"] = "ok"
        return "decide"
    if policy["friction_enabled"] and friction and max_friction >= float(policy["friction_threshold"]):
        set_decision("signal", "stalled", 1.0, False, "escalate", proposed_cursor, "immediate")
        envelope["status"] = "ok"
        return "decide"
    # Projection lag remains unknown even when the batch is full or children park.
    if policy["friction_enabled"] and any(row.get("friction_unavailable", False) for row in runs):
        set_decision("unavailable", "dependency_unavailable", None, True, "observe", current_cursor, "after", policy["quiet_seconds"])
        envelope["status"] = "ok"
        return "decide"
    if statuses and all(status == "parked" for status in statuses):
        set_decision("quiet", "quiet", 1.0, False, "park", proposed_cursor, "after", policy["quiet_seconds"])
        envelope["status"] = "ok"
        return "decide"
    # Batch size is a wake trigger, not evidence of productive progress.
    if policy["quiet_reached"]:
        set_decision("signal", "quiet_time", 1.0, False, "escalate", proposed_cursor, "immediate")
        envelope["status"] = "ok"
        return "decide"
    if not events and not friction:
        set_decision("quiet", "quiet", 1.0, False, "park", proposed_cursor, "after", policy["quiet_seconds"])
        envelope["status"] = "ok"
        return "decide"

    work["ambiguous"] = True
    safe_action = "escalate" if max_friction >= float(policy["friction_threshold"]) * 0.75 else "observe"
    if not allow_inference:
        set_decision("unavailable", "abstained", None, True, safe_action, current_cursor, "after", policy["quiet_seconds"])
        envelope["status"] = "ok"
        return "decide"
    classifier_schema = {
        "type": "object",
        "required": ["classification", "confidence", "abstain", "recommended_action"],
        "properties": {
            "classification": {"type": "string", "enum": ["quiet", "progress", "stalled", "blocked", "failed", "completed", "ambiguous"]},
            "confidence": {"type": "number", "minimum": 0, "maximum": 1},
            "abstain": {"type": "boolean"},
            "recommended_action": {"type": "string", "enum": ["observe", "nudge", "park", "continue", "stop", "escalate", "wake_parent"]},
        },
    }
    compact = json.dumps({
        "event_types": event_types[:64],
        "friction": [{"score": row.get("score"), "pattern": str(row.get("pattern", ""))[:120], "fingerprint": str(row.get("fingerprint", ""))[:120], "owner": str(row.get("owner", ""))[:80]} for row in friction[:16]],
        "run_statuses": statuses[:32],
        "prior": [{"disposition": row.get("disposition"), "classification": row.get("classification")} for row in history[:8] if isinstance(row, dict)],
        "policy_version": policy["version"],
    }, sort_keys=True)[:4096]
    try:
        result = ai.classify(compact, classifier_schema, "Classify supervision evidence. Abstain when evidence does not support a safe conclusion.")
        envelope["signals"]["inference_calls"] = 1
        row = result.head(1)[0]
        if row.get("error"):
            set_decision("unavailable", "dependency_unavailable", None, True, safe_action, current_cursor, "after", policy["quiet_seconds"])
            envelope["errors"].append({"class": "classifier_unavailable", "detail": str(row.get("error"))[:240], "where": "classify"})
            envelope["status"] = "unavailable"
            return "decide"
        value = json.loads(row.get("valueJson", "null"))
        if isinstance(value, str):
            try:
                value = json.loads(value)
            except json.JSONDecodeError:
                pass
        if not isinstance(value, dict):
            return fail("failed", "classifier_invalid", f"classifier returned {type(value).__name__}, not an object; row keys={sorted(row.keys())}", "classify")
        identity = {"provider": row.get("provider"), "model": row.get("model"), "applied": row.get("applied")}
        if not identity["provider"] or not identity["model"] or not isinstance(identity["applied"], dict):
            set_decision("unavailable", "dependency_unavailable", None, True, "observe", current_cursor, "after", policy["quiet_seconds"])
            envelope["errors"].append({"class":"classifier_invalid","detail":"gateway inference identity is incomplete","where":"classify"})
            envelope["status"] = "unavailable"
            return "decide"
        envelope["signals"]["inference_identity"] = identity
        classification = value.get("classification")
        confidence = value.get("confidence")
        abstained = bool(value.get("abstain"))
        action = value.get("recommended_action")
        if classification not in classifier_schema["properties"]["classification"]["enum"] or action not in classifier_schema["properties"]["recommended_action"]["enum"] or isinstance(confidence, bool) or not isinstance(confidence, (int, float)) or not 0 <= confidence <= 1 or not isinstance(value.get("abstain"), bool):
            return fail("failed", "classifier_invalid", "classifier result violated the closed output vocabulary", "classify")
        work["classifier"] = {"classification": classification, "confidence": float(confidence), "abstained": abstained, "action": action}
    except Exception as exc:
        status, klass = classify_transport(exc)
        classification = "budget_exhausted" if klass == "inference_spend_exceeded" else "dependency_unavailable"
        set_decision("unavailable", classification, None, True, safe_action, current_cursor, "after", policy["quiet_seconds"])
        envelope["errors"].append({"class": "classifier_unavailable", "detail": str(exc)[:240], "where": "classify"})
        envelope["status"] = status
        return "decide"
    return "decide"


def step_decide():  # DECIDE · pure interpretation of a constrained classifier result
    envelope["phase"] = "decide"
    verdict = work.get("classifier")
    if verdict:
        if verdict["abstained"]:
            set_decision("unavailable", "abstained", verdict["confidence"], True, "observe", current_cursor, "after", policy["quiet_seconds"])
        elif verdict["classification"] == "completed":
            set_decision("signal", "ambiguous", verdict["confidence"], True, "wake_parent", proposed_cursor, "immediate")
        elif verdict["classification"] in ("stalled", "blocked", "failed"):
            set_decision("signal", verdict["classification"], verdict["confidence"], False, verdict["action"], proposed_cursor, "immediate")
        elif verdict["classification"] in ("quiet", "progress"):
            set_decision("quiet", verdict["classification"], verdict["confidence"], False, verdict["action"], proposed_cursor, "after", policy["quiet_seconds"])
        else:
            set_decision("unavailable", "ambiguous", verdict["confidence"], True, "observe", current_cursor, "after", policy["quiet_seconds"])
        envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT · one bounded envelope
    envelope["phase"] = "report"
    envelope["evidence"] = envelope["evidence"][:20]
    print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "decide": step_decide, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"

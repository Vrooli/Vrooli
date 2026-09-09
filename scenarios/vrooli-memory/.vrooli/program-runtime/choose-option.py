"""Use a scoped structured preference only within the caller's current allowed set.

This selects a recommendation, never authorizes or executes the selected operation.
Free prose, truncated entries, conflicting preferences and stale option IDs cannot
override the caller's fallback. Domain owners determine the allowed options.
"""
import json
inputs = program.inputs()
envelope = {"program": "vrooli-memory.choose-option", "version": "1", "status": "failed",
            "phase": "report", "signals": {}, "errors": [], "evidence": []}
try:
    options = inputs.get("options", [])
    fallback = inputs.get("default_id", "")
    if not isinstance(options, list) or not 1 <= len(options) <= 20 or any(
            not isinstance(o, str) or not o or len(o.encode()) > 256 for o in options):
        raise ValueError("Supply 1..20 bounded option IDs")
    if len(set(options)) != len(options) or fallback not in options:
        raise ValueError("Options must be unique and contain default_id")
    current = tasks.current()
    candidates = current.get("prepare", {}).get("signals", {}).get("advice_candidates", [])
    matches = []
    for candidate in candidates[:10]:
        text = candidate.get("text", "")
        if candidate.get("text_truncated") or not text.startswith("option-preference/v1 "):
            continue
        try:
            preference = json.loads(text[len("option-preference/v1 "):])
        except (ValueError, TypeError):
            continue
        if (isinstance(preference, dict) and preference.get("operation") == current.get("operation")
                and preference.get("context_key") == current.get("context_key")
                and preference.get("option_id") in options):
            matches.append((candidate["entry_id"], preference["option_id"]))
    choices = {option for _, option in matches}
    selected = next(iter(choices)) if len(choices) == 1 else fallback
    advice = []
    if len(choices) == 1:
        advice = [{"entry_id": entry_id, "decision": "applied",
                   "decision_change": "Selected allowed option " + selected,
                   "verdict": "unknown", "evidence_refs": []} for entry_id, _ in matches]
    envelope.update(status="ok", signals={"selected_id": selected,
        "source": "advice" if advice else "default", "conflicting": len(choices) > 1,
        "learning": {"advice": advice}}, evidence=[entry_id for entry_id, _ in matches])
except Exception as exc:
    envelope["errors"] = [{"class": "invalid_input" if isinstance(exc, ValueError) else "advice_unavailable",
                            "where": "select", "detail": str(exc)[:160]}]
print(json.dumps(envelope))

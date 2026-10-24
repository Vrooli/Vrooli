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
        if candidate.get("text_truncated"):
            continue
        preference = candidate.get("body") if isinstance(candidate.get("body"), dict) else None
        legacy = False
        for prefix in ("option-preference/v1 ", "preference/v1 "):
            if preference is None and isinstance(text, str) and text.startswith(prefix):
                legacy = prefix.startswith("option-")
                try:
                    preference = json.loads(text[len(prefix):])
                except (ValueError, TypeError):
                    preference = None
        if (isinstance(preference, dict) and preference.get("option_id") in options
                and (not legacy or (preference.get("operation") == current.get("operation")
                                    and preference.get("context_key") == current.get("context_key")))):
            matches.append((candidate["entry_id"], preference["option_id"], candidate.get("verdict", "unknown")))
    weights = {option: 0.0 for option in options}
    for _, option, verdict in matches:
        weights[option] += 1.0 if verdict == "supported" else (-1.0 if verdict == "contradicted" else 0.5)
    choices = {option for _, option, _ in matches}
    selected = max(weights, key=weights.get) if weights and max(weights.values()) > 0 else fallback
    advice = []
    if len(choices) == 1:
        advice = [{"entry_id": entry_id, "decision": "applied",
                   "decision_change": "Selected allowed option " + selected,
                   "verdict": "unknown", "evidence_refs": []} for entry_id, option, _ in matches if option == selected]
    envelope.update(status="ok", signals={"selected_id": selected,
        "source": "advice" if advice else "default", "conflicting": len(choices) > 1,
        "learning": {"advice": advice}}, evidence=[entry_id for entry_id, _, _ in matches])
except Exception as exc:
    envelope["errors"] = [{"class": "invalid_input" if isinstance(exc, ValueError) else "advice_unavailable",
                            "where": "select", "detail": str(exc)[:160]}]
print(json.dumps(envelope))

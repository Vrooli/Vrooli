"""Read-only voice evidence board; availability and replay are not acceptance."""
import json
import math

inputs = program.inputs()

envelope = {"status": "ok", "phase": "validate", "signals": {}, "errors": [], "evidence": []}


def row(name, target=None, reading=None, reason="pending_telemetry"):
    return {"row": name, "reading": reading, "target": target, "in_band": None,
            "unavailable": reading is None, "reason": reason}


def validate_phase(values):
    if not isinstance(values, dict) or set(values) - {"experiment_id"}:
        raise ValueError("inputs must contain only optional experiment_id")
    selected = values.get("experiment_id", "")
    if not isinstance(selected, str) or len(selected) > 128 or (selected and
            any(c not in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_" for c in selected)):
        raise ValueError("experiment_id must be an identifier of at most 128 characters")
    return selected


def read_binding(name, call):
    try:
        value = call()
        envelope["evidence"].append(name)
        return value
    except Exception as exc:
        status, kind = program.classify(exc)
        # Raw provider errors can contain private material; keep only typed metadata.
        envelope["errors"].append({"class": kind, "binding": name, "detail": status})
        return None


def collect_phase(selected):
    jobs = [
        ("audio-tools/stt/engines", lambda: audio_tools.stt.engines().head(17)),
        ("audio-tools/health/show", lambda: audio_tools.health.show().head(9)),
    ]
    results = gather(*[lambda name=name, call=call: read_binding(name, call) for name, call in jobs])
    report = None
    if selected:
        report = read_binding("audio-tools/experiment/report", lambda: audio_tools.experiment.report(id=selected).meta())
    return results[0], results[1], report


def finite(value, minimum=0):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value) and value >= minimum


def classify_phase(selected, engines, health, meta):
    outcomes = [
        row("dictation-trust", "OT-P0-001: processed, retained, or explicitly recoverable intervals"),
        row("provider-parity", "OT-P0-002: common provider trust floor"),
        row("speaker-policy", "OT-P0-003: applied/degraded and required-policy fail-closed"),
        row("provider-evidence", "OT-P1-001: provider-neutral persisted evidence"),
        row("mobile-recovery", "OT-P1-002: accessible recovery and diagnostics"),
        row("device-qualification", "OT-P2-001: native device qualification evidence"),
        row("interactive-latency", "OT-P0-005: genuine partials and final drain; numeric bands pending adoption"),
        row("corpus-quality", "OT-P0-006: versioned recognition cohorts; quality floors pending adoption"),
        row("owned-settlement", "OT-P0-007: shared owned delivery and settlement; policy details pending adoption"),
        row("explicit-routing", "OT-P0-004: explicit local/BYOK/owned route and prior fallback consent"),
        row("host-portability", "OT-P0-008: compatible host capabilities and scoped native support evidence"),
        row("private-lifecycle", "OT-P0-009: authorized bounded retention, recovery and metadata-only diagnostics"),
        row("acceptance-integrity", "OT-P0-010: complete current owner receipts; simulation and duration integrity"),
        row("adapter-maintainability", "OT-P1-003: replaceable shared adapters without consumer duplication"),
        row("shared-voice-capabilities", "OT-P1-004: operation-specific TTS and summarization qualification"),
    ]
    diagnostics = []
    if engines is None:
        diagnostics.append(row("available-engines", reason="scenario_unreachable" if any(e["class"] == "scenario_unreachable" and e["binding"].endswith("engines") for e in envelope["errors"]) else "unreliable:binding_read_failed"))
    elif len(engines) > 16 or any(not isinstance(e, dict) or not e.get("id") for e in engines):
        diagnostics.append(row("available-engines", reason="unreliable:engine_inventory_shape_or_bound"))
    else:
        diagnostics.append(row("available-engines", reading=sum(e.get("available") is True for e in engines), reason=None))
        diagnostics.append(row("native-streaming-engines", reading=sum(e.get("available") is True and e.get("nativeStreaming") is True for e in engines), reason=None))
    if health is None or len(health) > 8 or any(not isinstance(h, dict) or not h.get("capability") for h in health):
        diagnostics.append(row("available-capabilities", reason="unreliable:health_read_or_bound"))
    else:
        diagnostics.append(row("available-capabilities", reading=sum(h.get("effectiveState") == "PROVIDER_STATE_AVAILABLE" for h in health), reason=None))

    metrics = []
    experiment = {"requested": bool(selected), "id": selected or None, "status": None, "finished_at": None,
                  "acceptance_eligible": False, "reason": "unreliable:no_selected_experiment"}
    if selected and meta is None:
        experiment["reason"] = "unreliable:report_read_failed"
    elif selected:
        exp = meta.get("experiment", {})
        report = meta.get("report", {})
        experiment.update({"status": exp.get("status"), "finished_at": exp.get("finishedAt"),
                           "reason": "unreliable:replay_is_not_current_product_acceptance"})
        strategies = report.get("perStrategy", [])
        if exp.get("id") != selected or exp.get("status") != "EXPERIMENT_STATUS_SUCCEEDED":
            experiment["reason"] = "unreliable:experiment_not_succeeded_or_identity_mismatch"
        elif not isinstance(strategies, list) or not strategies or len(strategies) > 16:
            experiment["reason"] = "unreliable:empty_or_oversized_report"
        else:
            for cell in strategies:
                words = cell.get("refWords")
                valid = finite(words, 1) and finite(cell.get("wer"))
                # A zero absent proto scalar is not a measured zero: omit absent fields.
                metric = {"engine_id": str(cell.get("engineId", ""))[:80],
                          "strategy": str(cell.get("strategy", ""))[:80],
                          "lane": str(cell.get("replayLane", ""))[:80],
                          "verdict": str(cell.get("verdict", ""))[:80] or None,
                          "ref_words": words if finite(words) else None,
                          "wer": cell.get("wer") if valid else None,
                          "reason": None if valid else "unreliable:missing_quality_denominator"}
                for source, key in [("rtf", "rtf"), ("finalizationLatencyP95Ms", "replay_finalization_p95_ms")]:
                    metric[key] = cell.get(source) if valid and finite(cell.get(source)) and (source == "rtf" or report.get("latencyMeasured") is True) else None
                metrics.append(metric)
    readable = sum(not d["unavailable"] for d in diagnostics)
    envelope["signals"] = {"method": "audio-tools-evidence-board-v2", "target_revision": "portable-voice-v1", "acceptance": "unknown",
                           "rows": outcomes, "diagnostics": diagnostics, "readable": readable,
                           "unavailable": sum(r["unavailable"] for r in outcomes + diagnostics),
                           "experiment": experiment, "replay_metrics": metrics}
    if envelope["errors"]:
        envelope["status"] = "partial" if readable else "unavailable"


def main():
    try:
        selected = validate_phase(inputs)
    except (ValueError, TypeError) as exc:
        envelope.update(status="failed", errors=[{"class": "invalid_input", "detail": str(exc)}])
        return
    try:
        envelope["phase"] = "collect"
        engines, health, report = collect_phase(selected)
        envelope["phase"] = "classify"
        classify_phase(selected, engines, health, report)
        envelope["phase"] = "report"
    except Exception:
        envelope.update(status="failed", errors=envelope["errors"] + [{"class": "kernel_runtime", "detail": "board computation failed; inspect runtime diagnostics"}])


main()
print(json.dumps(envelope, allow_nan=False))

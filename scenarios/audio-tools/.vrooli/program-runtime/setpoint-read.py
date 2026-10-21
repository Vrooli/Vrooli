"""Read-only voice evidence board; availability and replay are not acceptance."""
import json
import math

inputs = program.inputs()

envelope = {"status": "ok", "phase": "validate", "signals": {}, "errors": [], "evidence": []}

TARGET_REVISION = "portable-voice-v1"
RECEIPT_METHOD = "audio-tools-acceptance-v1"
# Retain a bounded but comfortably larger owner history. Failed attempts are
# deliberately preserved for negative-control auditing; a 512-row ceiling was
# reached during the first sustained qualification pass and made the board
# unreadable before acceptance resolution could run.
MAX_EVIDENCE_ROWS = 1024
EXCLUDED_ROWS = {"shared-voice-capabilities", "device-qualification"}
NON_ACCEPTANCE_EVIDENCE_KINDS = {"QUALIFICATION_EVIDENCE_KIND_DEVICE"}
REQUIRED_ROWS = {
    "dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
    "mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
    "explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
    "adapter-maintainability",
}


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
        # Evidence is a repeated response field.  Program Runtime keeps it in
        # the raw response; meta() contains only scalar response metadata.
        ("audio-tools/experiment/list-evidence", lambda: audio_tools.experiment.list_evidence().raw()),
    ]
    results = gather(*[lambda name=name, call=call: read_binding(name, call) for name, call in jobs])
    report = None
    if selected:
        report = read_binding("audio-tools/experiment/report", lambda: audio_tools.experiment.report(id=selected).meta())
    evidence = results[2]
    if isinstance(evidence, dict):
        # The Connect response is an envelope, not an optional map.  Treat a
        # missing/retyped repeated field as an unreadable owner source; an
        # empty list is the only representation of measured empty telemetry.
        evidence = evidence.get("evidence") if "evidence" in evidence else {"malformed_envelope": True}
    if evidence is not None and not isinstance(evidence, list):
        evidence = {"malformed": evidence}
    return results[0], results[1], evidence, report


def finite(value, minimum=0):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value) and value >= minimum


def _snake(name):
    return {"engineId": "engine_id", "modelId": "model_id", "artifactRef": "artifact_ref"}.get(name, name)


def _json_object(value):
    if isinstance(value, dict):
        return value
    if isinstance(value, str):
        try:
            parsed = json.loads(value)
        except (TypeError, ValueError):
            return None
        return parsed if isinstance(parsed, dict) else None
    return None


def resolve_acceptance(rows, evidence):
    """Resolve required rows from the Audio-owned, transcript-free receipt join.

    A generic qualification record is not acceptance by itself. Only its
    explicit owner receipt payload can contribute, and all rows must share one
    target revision/method/cohort/build/device/policy/corpus tuple.
    """
    required = [name for name in rows if name not in EXCLUDED_ROWS]
    resolved = {name: row(name, reason="pending_telemetry:no_matching_receipt") for name in rows}
    for name in EXCLUDED_ROWS:
        resolved[name] = row(name, reading="excluded", reason="excluded:mandate_scope")
    candidates = {}
    inspected = 0
    rejected = {name: [] for name in required}
    for item in evidence or []:
        inspected += 1
        if not isinstance(item, dict):
            continue
        # Device receipts may share run metadata with a product or simulated
        # receipt, but they are not acceptance producers. In particular, a
        # failed device qualification must not turn a passing claim into a
        # contradictory candidate tuple. Other kinds remain eligible when
        # their explicit target-aware envelope and execution lane allow it.
        if item.get("kind") in NON_ACCEPTANCE_EVIDENCE_KINDS:
            continue
        artifact = item.get("artifactRef", item.get("artifact_ref"))
        receipt = _json_object(item.get("machineJson", item.get("machine_json")))
        if not isinstance(artifact, str) or not artifact.strip() or not receipt:
            continue
        if receipt.get("target_revision") != TARGET_REVISION or receipt.get("method") != RECEIPT_METHOD:
            continue
        cohort, build = receipt.get("cohort"), receipt.get("build")
        device, policy, corpus = (receipt.get("device_profile"), receipt.get("policy_revision"),
                                  receipt.get("corpus_revision"))
        context_fields = (device, policy, corpus)
        if (not isinstance(cohort, str) or not cohort.strip() or not isinstance(build, str) or not build.strip()
                or any(not isinstance(field, str) or not field.strip() for field in context_fields)):
            continue
        acceptance = receipt.get("acceptance")
        if not isinstance(acceptance, dict) or receipt.get("stale") is True:
            continue
        # A simulated producer may only resolve rows it explicitly exercised.
        # Older broad receipts are invalid: their hard-coded claims can never
        # certify measured duration, corpus quality, latency, or device rows.
        details = receipt.get("details") if isinstance(receipt.get("details"), dict) else {}
        simulated = receipt.get("execution_lane") == "simulated" or details.get("execution_lane") == "simulated"
        supported_rows = receipt.get("supported_rows", details.get("supported_rows"))
        if simulated:
            if not isinstance(supported_rows, list) or not supported_rows or any(
                    not isinstance(name, str) or name not in required for name in supported_rows):
                continue
            supported_rows = set(supported_rows)
        else:
            supported_rows = None
        key = (RECEIPT_METHOD, cohort.strip(), build.strip(), device.strip(), policy.strip(), corpus.strip())
        for name, claim in acceptance.items():
            if name not in required:
                continue
            if simulated and name not in supported_rows:
                rejected[name].append("unsupported_simulation_claim")
                continue
            if not isinstance(claim, dict):
                rejected[name].append("invalid_claim")
                continue
            if claim.get("stale") is True or claim.get("simulated_as_native") is True:
                rejected[name].append("rejected_negative_control")
                continue
            # A failed persisted artifact is itself negative evidence even if
            # its payload contains an optimistic claim. Never let that row
            # participate in a passing candidate tuple.
            claim_passed = (item.get("passed") is True and
                            claim.get("in_band") is True and
                            claim.get("negative_controls_passed") is True)
            if not claim_passed:
                rejected[name].append("claim_out_of_band")
            candidates.setdefault(key, {}).setdefault(name, []).append(
                (claim_passed, artifact)
            )
    # Resolve each row from a consistent current tuple. A simulated receipt
    # intentionally covers only the rows it exercised, so requiring one
    # receipt to contain every required row would discard legitimate partial
    # owner evidence and hide useful progress. Overall acceptance still stays
    # false until every required row resolves in the same tuple below.
    for name in required:
        passing_keys = [key for key, claims in candidates.items()
                        if name in claims and all(passed for passed, _ in claims[name])]
        if passing_keys:
            key = sorted(passing_keys)[-1]
            refs = [artifact for _, artifact in candidates[key][name]]
            resolved[name] = row(name, reading={"method": key[0], "cohort": key[1], "build": key[2],
                                                "device_profile": key[3], "policy_revision": key[4],
                                                "corpus_revision": key[5], "evidence_refs": refs},
                                 reason=None)
            resolved[name]["in_band"] = True
            resolved[name]["unavailable"] = False
        elif rejected[name]:
            resolved[name]["reason"] = ("failed:receipt_claim_out_of_band" if "claim_out_of_band" in rejected[name]
                                         else "failed:receipt_negative_control")
    complete = [key for key, claims in candidates.items()
                if all(name in claims and all(passed for passed, _ in claims[name]) for name in required)]
    return resolved, candidates, inspected, bool(complete)


def classify_phase(selected, engines, health, evidence, meta):
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

    integrity = {"receipt_count": None, "valid_receipts": 0, "failed_receipts": 0,
                 "invalid_receipts": 0, "in_band": False,
                 "reason": "unreliable:owner_receipt_read_failed"}
    if evidence is not None:
        if len(evidence) > MAX_EVIDENCE_ROWS or any(not isinstance(item, dict) for item in evidence):
            integrity["reason"] = "unreliable:owner_receipt_shape_or_bound"
        else:
            integrity["receipt_count"] = len(evidence)
            for item in evidence:
                artifact = item.get("artifactRef", item.get("artifact_ref"))
                identity = all(isinstance(item.get(key, item.get(_snake(key))), str) and item.get(key, item.get(_snake(key))).strip()
                               for key in ("engineId", "modelId", "strategy", "kind"))
                valid = identity and isinstance(artifact, str) and bool(artifact.strip())
                if not valid:
                    integrity["invalid_receipts"] += 1
                elif item.get("passed") is True:
                    integrity["valid_receipts"] += 1
                else:
                    integrity["failed_receipts"] += 1
            if not evidence:
                integrity["reason"] = "pending_telemetry:no_owner_receipts"
            elif integrity["invalid_receipts"]:
                integrity["reason"] = "failed:owner_receipt_shape_or_identity"
            elif integrity["valid_receipts"] == 0:
                # A history containing only failed attempts is useful negative
                # evidence, but it cannot establish a trustworthy positive
                # owner population yet.
                integrity["reason"] = "failed:owner_receipt_negative_control"
            else:
                # Failed attempts are intentionally retained by the evidence
                # contract. They remain visible in failed_receipts and are
                # excluded from acceptance joins; they do not corrupt the
                # structural audit when a valid owner population is present.
                integrity["in_band"] = True
                integrity["reason"] = "retained_failed_attempts" if integrity["failed_receipts"] else "owner_receipts_valid"
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
    outcome_map, receipt_candidates, inspected_receipts, acceptance_complete = resolve_acceptance(
        [r["row"] for r in outcomes], evidence)
    for outcome in outcomes:
        resolved = outcome_map[outcome["row"]]
        outcome.update({key: resolved[key] for key in ("reading", "in_band", "unavailable", "reason")})
    # Full acceptance is stricter than per-row visibility: all required rows
    # must resolve from one shared identity tuple. Partial receipts can make
    # their exercised rows green without allowing mixed cohorts to combine.
    accepted = acceptance_complete and all(outcome_map[name]["in_band"] is True for name in REQUIRED_ROWS)
    envelope["signals"] = {"method": "audio-tools-evidence-board-v3", "target_revision": TARGET_REVISION,
                           "acceptance": "in_band" if accepted else "unknown",
                           "rows": outcomes, "diagnostics": diagnostics, "readable": readable,
                           "unavailable": sum(r["unavailable"] for r in outcomes + diagnostics),
                           "experiment": experiment, "replay_metrics": metrics,
                           "evidence_integrity": integrity,
                           "acceptance_join": {"required_rows": len(REQUIRED_ROWS),
                                               "resolved_rows": sum(v["in_band"] is True for v in outcome_map.values()),
                                               "receipt_cohorts": len(receipt_candidates),
                                               "inspected_receipts": inspected_receipts if inspected_receipts <= MAX_EVIDENCE_ROWS else None,
                                               "in_band": accepted,
                                               "reason": None if accepted else (
                                                   "pending:no_target_aware_acceptance_claims" if inspected_receipts else
                                                   "pending:required_owner_receipts")}}
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
        engines, health, evidence, report = collect_phase(selected)
        envelope["phase"] = "classify"
        classify_phase(selected, engines, health, evidence, report)
        envelope["phase"] = "report"
    except Exception:
        envelope.update(status="failed", errors=envelope["errors"] + [{"class": "kernel_runtime", "detail": "board computation failed; inspect runtime diagnostics"}])


main()
print(json.dumps(envelope, allow_nan=False))

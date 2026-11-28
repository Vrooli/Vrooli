"""Read-only voice evidence board; availability and replay are not acceptance."""
import json
import math

inputs = program.inputs()

envelope = {"status": "ok", "phase": "validate", "signals": {}, "errors": [], "evidence": []}

TARGET_REVISION = "portable-voice-v2"
RECEIPT_METHOD = "audio-tools-acceptance-v2"
SIMULATED_ACCEPTANCE_KIND = "QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE"
SIMULATED_ROW_CHECKS = {
    "explicit-routing": ["route_local_zero_debit", "route_byok_zero_debit", "route_owned_requires_consent"],
    "owned-settlement": ["settlement_reserve_and_partial_finalize", "settlement_retry_is_idempotent", "settlement_local_byok_zero_debit"],
    "private-lifecycle": ["lifecycle_authorized_recoverable_metadata_only", "lifecycle_delete_removes_retained_item"],
    "acceptance-integrity": ["acceptance_rejects_missing_target", "acceptance_rejects_failed_artifact", "acceptance_rejects_simulated_as_native"],
    "adapter-maintainability": ["adapter_versioned_route_metadata", "adapter_metadata_only"],
}
# Fixed owner receipt cells. These keys and expected routes mirror the CLI's
# hostPortabilityMatrixComplete contract; parent evidence identity is an
# aggregate receipt identity and must not be imposed on these distinct cells.
REQUIRED_HOST_CELLS = {"d1_cpu", "d1_accelerator", "d2"}
# Retain a bounded but comfortably larger owner history. Failed attempts are
# deliberately preserved for negative-control auditing; a 512-row ceiling was
# reached during the first sustained qualification pass and made the board
# unreadable before acceptance resolution could run.
MAX_EVIDENCE_ROWS = 1024
EXCLUDED_ROWS = {"shared-voice-capabilities", "device-qualification"}
NON_ACCEPTANCE_EVIDENCE_KINDS = {"QUALIFICATION_EVIDENCE_KIND_DEVICE"}
REQUIRED_QUALITY_ACCENTS = {"uk", "us", "south_african"}
REQUIRED_QUALITY_NOISE = {"clean", "20db", "10db"}
REQUIRED_ROWS = {
    "dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
    "mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
    "explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
    "desktop-dictation-parity", "adapter-maintainability", "efficient-model-selection",
    "advanced-voice-workflows", "standalone-offer-upgrade", "offline-install-lifecycle",
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
        # The governed binding decodes this response against the declared proto
        # before returning it. Protojson omits zero-length repeated fields, so
        # an absent evidence field is the typed empty-list default, not a
        # malformed envelope. A present non-list remains unreadable below.
        evidence = evidence.get("evidence", [])
    if evidence is not None and not isinstance(evidence, list):
        evidence = {"malformed": evidence}
    return results[0], results[1], evidence, report


def finite(value, minimum=0):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value) and value >= minimum


def _snake(name):
    return {"engineId": "engine_id", "modelId": "model_id", "artifactRef": "artifact_ref",
            "providerTier": "provider_tier"}.get(name, name)


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


def _provider_tier(value):
    if not isinstance(value, (str, int)) or isinstance(value, bool):
        return None
    return {
        "PROVIDER_TIER_LOCAL": "local", "PROVIDER_TIER_BYOK": "byok",
        "PROVIDER_TIER_VROOLI": "vrooli", 1: "local", 2: "byok", 3: "vrooli",
    }.get(value, value if value in {"local", "byok", "vrooli"} else None)


def host_portability_matrix_status(details, item, artifact, source_artifacts):
    """Validate the CLI owner receipt's fixed D1/D2 cells and visible paths."""
    if not isinstance(details, dict) or any(key not in details for key in REQUIRED_HOST_CELLS):
        return False
    allowed_artifacts = {artifact}
    if isinstance(source_artifacts, list):
        allowed_artifacts.update(ref for ref in source_artifacts if isinstance(ref, str) and ref.strip())

    def text(value):
        return isinstance(value, str) and bool(value.strip())

    def sha256(value):
        return isinstance(value, str) and len(value) == 64 and all(c in "0123456789abcdefABCDEF" for c in value)

    def host(cell, profile, engine, model, route, accelerator, constrained=False):
        if not isinstance(cell, dict):
            return False
        expected = {"profile_id": profile, "state": "available", "inference": "passed",
                    "engine_id": engine, "model_id": model, "selected_route": route,
                    "os": "linux", "architecture": "amd64", "browser": "Chromium",
                    "accelerator_identity": accelerator, "product_path_status": "passed"}
        if any(cell.get(key) != value for key, value in expected.items() if key != "accelerator_identity"):
            return False
        actual_accelerator = cell.get("accelerator_identity")
        if accelerator == "assigned":
            if not text(actual_accelerator) or actual_accelerator.strip().lower() in {"none", "cpu-only", "cpu", "unassigned"}:
                return False
        elif actual_accelerator != accelerator:
            return False
        for key in ("os_version", "browser_version", "device_identity", "engine_version", "model_revision", "product_path"):
            if not text(cell.get(key)):
                return False
        if not finite(cell.get("cpu_count"), 1) or not finite(cell.get("memory_total_bytes"), 1):
            return False
        if constrained and (type(cell.get("resource_cpu_limit")) is not int or cell["resource_cpu_limit"] != 2
                            or type(cell.get("resource_memory_limit_bytes")) is not int
                            or cell["resource_memory_limit_bytes"] != 4 * 1024 ** 3):
            return False
        ref, digest = cell.get("artifact_ref"), cell.get("artifact_sha256")
        product = cell.get("product_path_receipt")
        if (not text(ref) or ref not in allowed_artifacts or not sha256(digest)
                or not isinstance(product, dict) or product.get("outcome") != "passed"
                or product.get("engine_id") != engine or product.get("model_id") != model
                or product.get("selected_route") != route or product.get("artifact_ref") != ref
                or product.get("artifact_sha256") != digest or not text(product.get("visible_transcript_ref"))):
            return False
        return True

    cpu = details.get("d1_cpu")
    if not host(cpu, "D1-linux-amd64-cpu", "whisper-local", "whisper-small", "whisper.cpp-cpu", "none"):
        return False
    d2 = details.get("d2")
    if not host(d2, "D2-linux-amd64-constrained", "sherpa-streaming",
                "sherpa-onnx-streaming-zipformer-en-2023-06-26",
                "sherpa-streaming-zipformer-en-2023-06-26", "none", constrained=True):
        return False
    accelerator = details.get("d1_accelerator")
    if not isinstance(accelerator, dict):
        return False
    if (accelerator.get("engine_id") == "kyutai" and accelerator.get("model_id") == "kyutai/stt-1b-en_fr"
            and accelerator.get("state") == "available"):
        return host(accelerator, "D1-linux-amd64-accelerator", "kyutai", "kyutai/stt-1b-en_fr", "kyutai", "assigned")
    if (accelerator.get("engine_id") != "kyutai" or accelerator.get("model_id") != "kyutai/stt-1b-en_fr"
            or accelerator.get("profile_id") != "D1-linux-amd64-accelerator"
            or accelerator.get("state") != "unavailable" or accelerator.get("inference") != "not_run"
            or not text(accelerator.get("reason")) or accelerator.get("accelerator_identity") != "none"):
        return False
    fallback = accelerator.get("fallback_product_path")
    if not isinstance(fallback, dict) or fallback.get("visible_fallback") is not True:
        return False
    return host({**accelerator, "engine_id": "sherpa-streaming",
                 "model_id": "sherpa-onnx-streaming-zipformer-en-2023-06-26",
                 "selected_route": "sherpa-streaming-zipformer-en-2023-06-26",
                 "state": "available", "inference": "passed",
                 "product_path": accelerator.get("product_path") or "visible Sherpa fallback",
                 "product_path_status": "passed", "product_path_receipt": fallback,
                 "model_revision": fallback.get("model_revision"),
                 "engine_version": fallback.get("engine_version")},
                "D1-linux-amd64-accelerator", "sherpa-streaming",
                "sherpa-onnx-streaming-zipformer-en-2023-06-26",
                "sherpa-streaming-zipformer-en-2023-06-26", "none")


def quality_subgroup_metrics_complete(receipt):
    """Require current transcript-free quality evidence for every accent/noise cell."""
    measurement = receipt.get("quality_measurement") if isinstance(receipt, dict) else None
    if not isinstance(measurement, dict) or measurement.get("schema_version") != "corpus-quality-subgroups-v1":
        return False
    metrics = measurement.get("metrics")
    if not isinstance(metrics, list) or len(metrics) != 3 + len(REQUIRED_QUALITY_ACCENTS) * len(REQUIRED_QUALITY_NOISE):
        return False
    expected = {
        "clean": 0.10, "noisy_20db": 0.20, "noisy_10db": 0.20,
    }
    expected.update({
        f"accent_noise:{accent}:{noise}": 0.10 if noise == "clean" else 0.20
        for accent in REQUIRED_QUALITY_ACCENTS for noise in REQUIRED_QUALITY_NOISE
    })
    indexed = {}
    for metric in metrics:
        if not isinstance(metric, dict):
            return False
        name = metric.get("name")
        if not isinstance(name, str) or name not in expected or name in indexed:
            return False
        indexed[name] = metric
    if set(indexed) != set(expected):
        return False
    for name, threshold in expected.items():
        metric = indexed[name]
        wer, max_wer, words = metric.get("wer"), metric.get("max_wer"), metric.get("ref_words")
        if (metric.get("available") is not True or not finite(wer, 0)
                or not finite(max_wer, 0) or not finite(words, 1)
                or max_wer > threshold):
            return False
    return True


def quality_subgroup_metrics_in_band(receipt):
    """Require complete subgroup metrics, passing WER, and a passing receipt."""
    if not quality_subgroup_metrics_complete(receipt):
        return False
    measurement = receipt["quality_measurement"]
    if measurement.get("passed") is not True:
        return False
    return all(metric["wer"] <= metric["max_wer"] for metric in measurement["metrics"])


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
        # The receipt's assertion must describe the same persisted execution
        # cell. Missing provenance stays unknown; a self-consistent but
        # different provider, route, or device profile cannot borrow claims.
        provider_cell = receipt.get("provider_cell")
        item_engine = item.get("engineId", item.get("engine_id"))
        item_model = item.get("modelId", item.get("model_id"))
        item_strategy = item.get("strategy")
        if (not isinstance(provider_cell, dict) or
                provider_cell.get("engine_id") != item_engine or
                provider_cell.get("model_id") != item_model or
                provider_cell.get("strategy") != item_strategy or
                provider_cell.get("device_profile") != device):
            continue
        # A simulated producer may only resolve rows it explicitly exercised.
        # Older broad receipts are invalid: their hard-coded claims can never
        # certify measured duration, corpus quality, latency, or device rows.
        details = receipt.get("details") if isinstance(receipt.get("details"), dict) else {}
        simulated = receipt.get("execution_lane") == "simulated" or details.get("execution_lane") == "simulated"
        supported_rows = receipt.get("supported_rows", details.get("supported_rows"))
        if simulated:
            # The composite kind is the canonical persisted envelope for this
            # deterministic lane. Legacy interval/fault receipts predate
            # target-bound claims and cannot be promoted by supported_rows.
            if (item.get("kind") != SIMULATED_ACCEPTANCE_KIND or
                    receipt.get("execution_lane") != "simulated" or
                    receipt.get("simulated_as_native") is not False):
                continue
            if not isinstance(supported_rows, list) or not supported_rows or any(
                    not isinstance(name, str) or name not in required for name in supported_rows):
                continue
            supported_rows = set(supported_rows)
        else:
            supported_rows = None
        claim_lanes = details.get("claim_execution_lanes") if isinstance(details.get("claim_execution_lanes"), dict) else {}
        source_artifacts = details.get("source_artifacts") if isinstance(details.get("source_artifacts"), list) else []
        key = (RECEIPT_METHOD, cohort.strip(), build.strip(), device.strip(), policy.strip(), corpus.strip())
        for name, claim in acceptance.items():
            if name not in required:
                continue
            claim_simulated = (simulated or claim.get("execution_lane") == "simulated" or
                               claim_lanes.get(name) == "simulated") if isinstance(claim, dict) else simulated
            # The deterministic composite exercises all three route tiers
            # within one artifact. Its exact target-aware route checks are the
            # provenance; a single live tier cannot represent that fixture.
            # Native/live route claims must match independently persisted tier.
            if name == "explicit-routing" and not claim_simulated:
                item_tier = _provider_tier(item.get("providerTier", item.get("provider_tier")))
                receipt_tier = _provider_tier(provider_cell.get("provider_tier"))
                if item_tier not in {"local", "byok", "vrooli"} or receipt_tier not in {"local", "byok", "vrooli"}:
                    continue
                if item_tier != receipt_tier:
                    rejected[name].append("route_provenance_mismatch")
                    continue
            if simulated and name not in supported_rows:
                rejected[name].append("unsupported_simulation_claim")
                continue
            if not isinstance(claim, dict):
                rejected[name].append("invalid_claim")
                continue
            if claim_simulated and (item.get("kind") != SIMULATED_ACCEPTANCE_KIND or
                              claim.get("target_id") != name or
                              claim.get("target_revision") != TARGET_REVISION or
                              claim.get("method") != RECEIPT_METHOD or
                              (claim.get("receipt_reference") != artifact and claim.get("receipt_reference") not in source_artifacts) or
                              claim.get("execution_lane") != "simulated" or
                              claim.get("simulated_as_native") is not False or
                              claim.get("checks") != SIMULATED_ROW_CHECKS.get(name)):
                rejected[name].append("invalid_target_aware_simulation_claim")
                continue
            quality_complete = name != "corpus-quality" or quality_subgroup_metrics_complete(receipt)
            quality_in_band = name != "corpus-quality" or quality_subgroup_metrics_in_band(receipt)
            if not quality_complete:
                rejected[name].append("quality_subgroup_metrics_incomplete")
            elif not quality_in_band:
                measurement = receipt.get("quality_measurement", {})
                measurement_reasons = measurement.get("reasons", []) if isinstance(measurement, dict) else []
                if any(isinstance(reason, str) and "no human-reviewed reference" in reason
                       for reason in measurement_reasons):
                    rejected[name].append("quality_reference_provenance_unverified")
                else:
                    rejected[name].append("claim_out_of_band")
            # OT-P0-008 is a matrix, not a scalar assertion. A passing claim
            # must identify every required cell as available; unavailable or
            # omitted cells keep the row unqualified.
            host_matrix_complete = True
            if name == "host-portability":
                host_matrix_complete = host_portability_matrix_status(details, item, artifact, source_artifacts)
            if claim.get("stale") is True or claim.get("simulated_as_native") is True:
                rejected[name].append("rejected_negative_control")
                continue
            # A failed persisted artifact is itself negative evidence even if
            # its payload contains an optimistic claim. Never let that row
            # participate in a passing candidate tuple.
            claim_passed = (host_matrix_complete and quality_complete and quality_in_band and item.get("passed") is True and
                            claim.get("in_band") is True and
                            claim.get("negative_controls_passed") is True)
            if not claim_passed:
                if name == "host-portability" and not host_matrix_complete:
                    rejected[name].append("host_matrix_incomplete")
                elif name != "corpus-quality" or quality_complete and quality_in_band:
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
                resolved[name]["reason"] = ("failed:host_matrix_incomplete" if "host_matrix_incomplete" in rejected[name]
                                         else "failed:quality_subgroup_metrics_incomplete" if "quality_subgroup_metrics_incomplete" in rejected[name]
                                         else "failed:quality_reference_provenance_unverified" if "quality_reference_provenance_unverified" in rejected[name]
                                         else "failed:receipt_claim_out_of_band" if "claim_out_of_band" in rejected[name]
                                         else "failed:receipt_negative_control")
    complete = [key for key, claims in candidates.items()
                if all(name in claims and all(passed for passed, _ in claims[name]) for name in required)]
    return resolved, candidates, inspected, bool(complete)


def latest_model_resolution_observation(evidence):
    """Expose owner model-resolution measurements without treating them as acceptance."""
    if not isinstance(evidence, list) or len(evidence) > MAX_EVIDENCE_ROWS:
        return None
    candidates = []
    expected_profiles = {"D1-linux-amd64-cpu", "D2-linux-amd64-constrained"}
    expected_capabilities = {"stt", "tts", "summarize", "speaker"}
    for item in evidence:
        if not isinstance(item, dict):
            continue
        artifact = item.get("artifactRef", item.get("artifact_ref"))
        if not isinstance(artifact, str) or not artifact.startswith("audio-tools:host-portability:"):
            continue
        receipt = _json_object(item.get("machineJson", item.get("machine_json")))
        if (not receipt or receipt.get("target_revision") != TARGET_REVISION or
                receipt.get("method") != RECEIPT_METHOD or
                receipt.get("cohort") != "voice-pilot-en-v1" or
                receipt.get("device_profile") != "D1" or
                receipt.get("policy_revision") != "policy-v1" or
                receipt.get("corpus_revision") != "voice-pilot-en-v1"):
            continue
        provider_cell = receipt.get("provider_cell")
        if (not isinstance(provider_cell, dict) or provider_cell.get("device_profile") != "D1" or
                provider_cell.get("engine_id") != item.get("engineId", item.get("engine_id")) or
                provider_cell.get("model_id") != item.get("modelId", item.get("model_id")) or
                provider_cell.get("strategy") != item.get("strategy")):
            continue
        details = receipt.get("details")
        observations = details.get("model_resolution_receipts") if isinstance(details, dict) else None
        if not isinstance(observations, dict) or observations.get("schema_version") != 1:
            continue
        profiles = []
        valid = True
        for profile_id in sorted(expected_profiles):
            profile = observations.get(profile_id)
            rows = profile.get("observations") if isinstance(profile, dict) else None
            constraints = profile.get("platform_constraints") if isinstance(profile, dict) else None
            if (not isinstance(profile, dict) or profile.get("profile") != profile_id or
                    not isinstance(rows, list) or len(rows) != len(expected_capabilities) or
                    not isinstance(constraints, dict) or constraints.get("os") != "linux" or
                    constraints.get("arch") != "amd64"):
                valid = False
                break
            projected = []
            seen = set()
            for model in rows:
                if not isinstance(model, dict):
                    valid = False
                    break
                capability = model.get("capability")
                state = model.get("state")
                if capability not in expected_capabilities or capability in seen or state not in {"resolved", "unavailable"}:
                    valid = False
                    break
                seen.add(capability)
                projected_model = {"capability": capability, "engine_id": str(model.get("engine_id", ""))[:80], "state": state}
                for key, limit in (("model_id", 120), ("readiness", 80), ("commercial_use", 80), ("license", 100)):
                    value = model.get(key)
                    if isinstance(value, str):
                        projected_model[key] = value[:limit]
                size = model.get("declared_size_mb")
                if finite(size) and size >= 0:
                    projected_model["declared_size_mb"] = size
                reason = model.get("reason")
                if isinstance(reason, str):
                    projected_model["reason"] = reason[:180]
                projected.append(projected_model)
            if not valid or seen != expected_capabilities:
                valid = False
                break
            profiles.append({"profile": profile_id, "cpu_limit": profile.get("cpu_limit", 0),
                             "platform_constraints": constraints, "observations": projected})
        if not valid:
            continue
        memory = details.get("model_memory_evidence", {})
        memory_observations = []
        if isinstance(memory, dict):
            for model_id, value in sorted(memory.items()):
                if not isinstance(value, dict):
                    continue
                memory_observations.append({
                    "model_id": str(model_id)[:120], "state": str(value.get("state", ""))[:40],
                    "passed": value.get("passed") is True,
                    "peak_memory_bytes": value.get("peak_memory_bytes") if finite(value.get("peak_memory_bytes"), 1) else None,
                    "memory_budget_bytes": value.get("memory_budget_bytes") if finite(value.get("memory_budget_bytes"), 1) else None,
                    "receipt_id": str(value.get("receipt_id", ""))[:100] or None,
                })
        candidates.append({"observed_at": str(item.get("observedAt", item.get("observed_at", "")))[:40],
                           "artifact_ref": artifact[:200], "profiles": profiles,
                           "model_memory_evidence": memory_observations})
    if not candidates:
        return None
    candidates.sort(key=lambda item: (item["observed_at"], item["artifact_ref"]))
    latest = candidates[-1]
    return {"state": "observed", "evidence_ref": latest["artifact_ref"],
            "observed_at": latest["observed_at"] or None, "profiles": latest["profiles"],
            "model_memory_evidence": latest["model_memory_evidence"],
            "quality_evidence": "not_established_by_model_resolution",
            "capacity_evidence": "partial; D2 CPU limit is declared but not enforced by the model resolver",
            "acceptance": "unknown until matching quality, capacity, and route/fallback evidence passes"}


def classify_phase(selected, engines, health, evidence, meta):
    outcomes = [
        row("dictation-trust", "OT-P0-001: processed, retained, or explicitly recoverable intervals"),
        row("provider-parity", "OT-P0-002: common provider trust floor"),
        row("speaker-policy", "OT-P0-003: applied/degraded and required-policy fail-closed"),
        row("explicit-routing", "OT-P0-004: explicit local/BYOK/owned route and prior fallback consent"),
        row("interactive-latency", "OT-P0-005: warm click-to-ready p95 ≤ 500 ms; speech-to-visible-partial p95 ≤ 1,000 ms; stop-to-final-tail p95 ≤ 1,000 ms; report cold readiness separately; batch-only is labeled"),
        row("corpus-quality", "OT-P0-006: held-out voice-pilot-en-v1 WER ≤ 10% clean and ≤ 20% at 20 dB and 10 dB SNR; report every accent/noise subgroup and retain failed attempts"),
        row("owned-settlement", "OT-P0-007: adopted voice-stt-delivery-v1 required proof cells pass through shared owners; local/BYOK debit is zero; fixture evidence remains simulated"),
        row("host-portability", "OT-P0-008: qualify the required D1–D6 simulation matrix; D1 CPU/accelerator and constrained D2 select a compatible path or expose explicit unavailability; native claims need matching device evidence"),
        row("private-lifecycle", "OT-P0-009: authorized bounded retention, recovery and metadata-only diagnostics"),
        row("acceptance-integrity", "OT-P0-010: complete current owner receipts; simulation and duration integrity"),
        row("desktop-dictation-parity", "OT-P0-011: supported desktop dictation task completion, insertion, onboarding, permissions, offline use, and recovery parity"),
        row("provider-evidence", "OT-P1-001: provider-neutral persisted evidence"),
        row("mobile-recovery", "OT-P1-002: accessible recovery and diagnostics"),
        row("adapter-maintainability", "OT-P1-003: replaceable shared adapters without consumer duplication"),
        row("shared-voice-capabilities", "OT-P1-004: operation-specific TTS and summarization qualification"),
        row("efficient-model-selection", "OT-P1-005: declared models selected by language, rights, quality, capacity, cost, and fallback"),
        row("advanced-voice-workflows", "OT-P1-006: reusable synthesis, customization, cloning, conversion, separation, wake-word, and duplex workflows"),
        row("standalone-offer-upgrade", "OT-P1-007: standalone $10/month offer, entitlement, payment recovery, cancellation, and bundle upgrade"),
        row("offline-install-lifecycle", "OT-P1-008: supported desktop install/update with offline model provisioning and clean recovery"),
        row("device-qualification", "OT-P2-001: native device qualification evidence"),
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
    model_resolution = latest_model_resolution_observation(evidence)
    for outcome in outcomes:
        resolved = outcome_map[outcome["row"]]
        outcome.update({key: resolved[key] for key in ("reading", "in_band", "unavailable", "reason")})
        if outcome["row"] == "efficient-model-selection" and model_resolution is not None and outcome["in_band"] is not True:
            outcome["reading"] = {
                "state": model_resolution["state"],
                "evidence_ref": model_resolution["evidence_ref"],
                "observed_at": model_resolution["observed_at"],
                "profiles": [profile["profile"] for profile in model_resolution["profiles"]],
                "model_memory_receipts": len(model_resolution["model_memory_evidence"]),
                "quality_evidence": model_resolution["quality_evidence"],
                "capacity_evidence": model_resolution["capacity_evidence"],
                "acceptance": model_resolution["acceptance"],
            }
            outcome["in_band"] = None
            outcome["unavailable"] = True
            outcome["reason"] = "pending:matching quality, capacity, and route/fallback evidence"
    # Full acceptance is stricter than per-row visibility: all required rows
    # must resolve from one shared identity tuple. Partial receipts can make
    # their exercised rows green without allowing mixed cohorts to combine.
    accepted = acceptance_complete and all(outcome_map[name]["in_band"] is True for name in REQUIRED_ROWS)
    resolved_row_count = sum(v["in_band"] is True for v in outcome_map.values())
    envelope["signals"] = {"method": "audio-tools-evidence-board-v3", "target_revision": TARGET_REVISION,
                           "acceptance": "in_band" if accepted else "unknown",
                           "rows": outcomes, "model_resolution": model_resolution,
                           "diagnostics": diagnostics, "readable": readable,
                           "unavailable": sum(r["unavailable"] for r in outcomes + diagnostics),
                           "experiment": experiment, "replay_metrics": metrics,
                           "evidence_integrity": integrity,
                           "acceptance_join": {"required_rows": len(REQUIRED_ROWS),
                                               "resolved_rows": resolved_row_count,
                                               "receipt_cohorts": len(receipt_candidates),
                                               "inspected_receipts": inspected_receipts if inspected_receipts <= MAX_EVIDENCE_ROWS else None,
                                               "in_band": accepted,
                                               "reason": None if accepted else (
                                                   "pending:required_owner_receipts" if resolved_row_count else
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
program.report(envelope)

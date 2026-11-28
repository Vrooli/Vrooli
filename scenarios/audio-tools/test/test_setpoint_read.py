"""Board behavior with governed bindings replaced, not audio qualification."""
import contextlib
import io
import importlib.util
import json
import re
from pathlib import Path
from types import SimpleNamespace
import unittest


SOURCE = Path(__file__).resolve().parents[1] / ".vrooli/program-runtime/setpoint-read.py"
HELPER_SOURCE = SOURCE.parents[3] / "program-runtime/kernel/host/program_helper.py"
EXPECTED_TARGET_BY_ROW = {
    "dictation-trust": "OT-P0-001",
    "provider-parity": "OT-P0-002",
    "speaker-policy": "OT-P0-003",
    "explicit-routing": "OT-P0-004",
    "interactive-latency": "OT-P0-005",
    "corpus-quality": "OT-P0-006",
    "owned-settlement": "OT-P0-007",
    "host-portability": "OT-P0-008",
    "private-lifecycle": "OT-P0-009",
    "acceptance-integrity": "OT-P0-010",
    "desktop-dictation-parity": "OT-P0-011",
    "provider-evidence": "OT-P1-001",
    "mobile-recovery": "OT-P1-002",
    "adapter-maintainability": "OT-P1-003",
    "shared-voice-capabilities": "OT-P1-004",
    "efficient-model-selection": "OT-P1-005",
    "advanced-voice-workflows": "OT-P1-006",
    "standalone-offer-upgrade": "OT-P1-007",
    "offline-install-lifecycle": "OT-P1-008",
    "device-qualification": "OT-P2-001",
}
REQUIRED_ROW_IDS = set(EXPECTED_TARGET_BY_ROW) - {"shared-voice-capabilities", "device-qualification"}


def passing_quality_measurement():
    """Complete synthetic subgroup metrics for receipt-join behavior tests."""
    names = ["clean", "noisy_20db", "noisy_10db"]
    names.extend(f"accent_noise:{accent}:{noise}"
                 for accent in ("uk", "us", "south_african")
                 for noise in ("clean", "20db", "10db"))
    return {"schema_version": "corpus-quality-subgroups-v1", "passed": True,
            "metrics": [{"name": name, "available": True, "wer": 0.01,
                         "max_wer": 0.10 if name.endswith("clean") or name == "clean" else 0.20,
                         "ref_words": 100} for name in names]}
helper_spec = importlib.util.spec_from_file_location("audio_board_program_helper", HELPER_SOURCE)
program_helper = importlib.util.module_from_spec(helper_spec)
helper_spec.loader.exec_module(program_helper)


class Handle:
    def __init__(self, value):
        self.value = value

    def head(self, limit):
        return self.value[:limit]

    def meta(self):
        return self.value

    def raw(self):
        return self.value


def run_board(inputs=None, engines=None, health=None, evidence=None, report=None, errors=None):
    calls = []
    errors = errors or {}

    def binding(name, value):
        def invoke(**kwargs):
            calls.append((name, kwargs))
            if name in errors:
                raise errors[name]
            return Handle(value)
        return invoke

    api = SimpleNamespace(
        stt=SimpleNamespace(engines=binding("engines", engines if engines is not None else [{"id": "local", "available": True, "nativeStreaming": True}])),
        health=SimpleNamespace(show=binding("health", health if health is not None else [{"capability": "CAPABILITY_STT", "effectiveState": "PROVIDER_STATE_AVAILABLE"}])),
        experiment=SimpleNamespace(
            list_evidence=binding("evidence", evidence if evidence is not None else []),
            report=binding("report", report)),
    )
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), {
            # Inject the declared runtime interface. Keep malformed values for
            # domain validation tests; this fixture does not certify the
            # kernel's separate input-schema gate or session lifecycle.
            "program": SimpleNamespace(
                inputs=lambda: {} if inputs is None else inputs,
                classify=program_helper.ProgramHelper().classify,
                report=lambda envelope: print(json.dumps(envelope)),
            ),
            "audio_tools": api,
            "gather": lambda *jobs: [job() for job in jobs],
        })
    lines = output.getvalue().splitlines()
    if len(lines) != 1:
        raise AssertionError("The board must print exactly one envelope")
    return json.loads(lines[0]), calls


def completed_report(**cell_fields):
    cell = {"engineId": "local", "strategy": "native", "refWords": 100,
            "wer": 0.12, "rtf": 0.4, "finalizationLatencyP95Ms": 800,
            "replayLane": "REPLAY_LANE_REALTIME"}
    cell.update(cell_fields)
    return {"experiment": {"id": "experiment-1", "status": "EXPERIMENT_STATUS_SUCCEEDED", "finishedAt": "2020-01-01T00:00:00Z"},
            "report": {"perStrategy": [cell], "latencyMeasured": True}}


def host_matrix(artifact="artifact-1", engine="local", model="model-1", accelerator_state="unavailable"):
    digest = "a" * 64
    def cell(profile, engine_id, model_id, route, accelerator="none", constrained=False):
        product = {"outcome": "passed", "engine_id": engine_id, "model_id": model_id,
                   "selected_route": route, "artifact_ref": artifact, "artifact_sha256": digest,
                   "visible_transcript_ref": "transcript-1", "model_revision": "test-model-revision",
                   "engine_version": "test-engine-version"}
        result = {"profile_id": profile, "os": "linux", "os_version": "Linux 6.8",
                  "architecture": "amd64", "browser": "Chromium", "browser_version": "150.0",
                  "device_identity": "host-1", "accelerator_identity": accelerator,
                  "state": "available", "inference": "passed", "engine_id": engine_id,
                  "model_id": model_id, "selected_route": route, "model_revision": "test-model-revision",
                  "engine_version": "test-engine-version", "artifact_ref": artifact,
                  "artifact_sha256": digest, "product_path": "browser Dictation Studio route",
                  "product_path_status": "passed", "product_path_receipt": product,
                  "cpu_count": 8, "memory_total_bytes": 8 * 1024 ** 3}
        if constrained:
            result.update(resource_cpu_limit=2, resource_memory_limit_bytes=4 * 1024 ** 3)
        return result
    sherpa = "sherpa-onnx-streaming-zipformer-en-2023-06-26"
    cpu = cell("D1-linux-amd64-cpu", "whisper-local", "whisper-small", "whisper.cpp-cpu")
    d2 = cell("D2-linux-amd64-constrained", "sherpa-streaming", sherpa,
              "sherpa-streaming-zipformer-en-2023-06-26", constrained=True)
    if accelerator_state == "unavailable":
        fallback = {"outcome": "passed", "engine_id": "sherpa-streaming", "model_id": sherpa,
                    "selected_route": "sherpa-streaming-zipformer-en-2023-06-26",
                    "artifact_ref": artifact, "artifact_sha256": digest,
                    "visible_transcript_ref": "fallback-transcript-1", "visible_fallback": True,
                    "model_revision": "test-model-revision", "engine_version": "test-engine-version"}
        accelerator = cell("D1-linux-amd64-accelerator", "kyutai", "kyutai/stt-1b-en_fr", "kyutai")
        accelerator.update(state="unavailable", inference="not_run", reason="Kyutai accelerator device not assigned",
                           fallback_product_path=fallback)
    else:
        accelerator = cell("D1-linux-amd64-accelerator", "kyutai", "kyutai/stt-1b-en_fr", "kyutai", accelerator="NVIDIA test GPU")
    return {"d1_cpu": cpu, "d1_accelerator": accelerator, "d2": d2}


def claim_cpu_as_accelerator(cells):
    cells["d1_accelerator"].update(state="available", accelerator_identity="none")


class EvidenceBoardTests(unittest.TestCase):
    def test_every_published_target_remains_visible_in_the_board(self):
        # Structural traceability only; this does not validate product behavior.
        prd = (SOURCE.parents[2] / "PRD.md").read_text()
        testing = (SOURCE.parents[2] / "docs/internal/TESTING.md").read_text()
        target_ids = set(re.findall(r"^- \[[ x]\] (OT-P\d-\d+)", prd, re.MULTILINE))
        board, _ = run_board()
        exposed = set()
        actual_mapping = {}
        for row in board["signals"]["rows"]:
            matches = re.findall(r"OT-P\d-\d+", row["target"] or "")
            exposed.update(matches)
            self.assertEqual(len(matches), 1, row)
            actual_mapping[row["row"]] = matches[0]
        self.assertTrue(target_ids)
        self.assertEqual(exposed, target_ids)
        self.assertEqual(actual_mapping, EXPECTED_TARGET_BY_ROW)
        inventory = testing.split("| Row | Outcome | Band | Validity |", 1)[1].split("Evidence audit:", 1)[0]
        inventoried_targets = set(re.findall(r"^\| `[^`]+` \| (OT-P\d-\d+) \|", inventory, re.MULTILINE))
        self.assertEqual(inventoried_targets, target_ids)

    def test_successful_inventory_keeps_every_outcome_unknown(self):
        board, calls = run_board()
        self.assertEqual(board["status"], "ok")
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual({c[0] for c in calls}, {"engines", "health", "evidence"})
        self.assertEqual(board["signals"]["target_revision"], "portable-voice-v2")
        self.assertEqual(len(board["signals"]["rows"]), 20)
        self.assertEqual(board["signals"]["acceptance_join"]["required_rows"], 18)
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], 0)
        for row in board["signals"]["rows"]:
            if row["row"] in {"shared-voice-capabilities", "device-qualification"}:
                self.assertEqual(row["reason"], "excluded:mandate_scope")
                self.assertFalse(row["unavailable"])
            else:
                self.assertTrue(row["unavailable"])
                self.assertIsNone(row["reading"])
                self.assertIsNone(row["in_band"])
        self.assertIsNotNone(board["signals"]["rows"][0]["target"])
        self.assertTrue(all(row["target"] for row in board["signals"]["rows"]))

    def test_board_reports_adopted_bands_without_implying_acceptance(self):
        board, _ = run_board()
        targets = {item["row"]: item["target"] for item in board["signals"]["rows"]}
        self.assertIn("p95 ≤ 500 ms", targets["interactive-latency"])
        self.assertIn("p95 ≤ 1,000 ms", targets["interactive-latency"])
        self.assertIn("WER ≤ 10% clean", targets["corpus-quality"])
        self.assertIn("≤ 20% at 20 dB and 10 dB SNR", targets["corpus-quality"])
        self.assertIn("voice-stt-delivery-v1", targets["owned-settlement"])
        self.assertIn("D1–D6 simulation matrix", targets["host-portability"])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertTrue(all(item["in_band"] is None for item in board["signals"]["rows"]
                            if item["row"] not in {"shared-voice-capabilities", "device-qualification"}))

    def test_owner_receipt_join_resolves_all_required_rows_from_one_build(self):
        required = REQUIRED_ROW_IDS
        evidence = [{
            "engineId": "local", "modelId": "model-1", "strategy": "native", "kind": "fault",
            "artifactRef": "qualification:portable-voice-v2:build-1", "passed": True,
            "machineJson": json.dumps({
                "host": "runner", "goos": "linux", "goarch": "amd64",
                "target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                "cohort": "voice-pilot-en-v1", "build": "build-1", "device_profile": "D1",
                "policy_revision": "policy-v1", "corpus_revision": "corpus-v1",
                "provider_cell": {"engine_id": "local", "model_id": "model-1", "strategy": "native", "device_profile": "D1"},
                "acceptance": {name: {"in_band": True, "negative_controls_passed": True} for name in required},
                "quality_measurement": passing_quality_measurement(),
                "details": host_matrix("qualification:portable-voice-v2:build-1"),
            }),
        }]
        board, _ = run_board(evidence=evidence)
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertFalse(board["signals"]["acceptance_join"]["in_band"])
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], len(required) - 1)
        for row in board["signals"]["rows"]:
            if row["row"] in required - {"explicit-routing"}:
                self.assertTrue(row["in_band"], row)
                self.assertIsInstance(row["reading"], dict)
            elif row["row"] == "explicit-routing":
                self.assertIsNone(row["in_band"], row)
            else:
                self.assertEqual(row["reason"], "excluded:mandate_scope")

    def test_owner_receipt_join_rejects_stale_and_mixed_build_claims(self):
        claim = {"in_band": True, "negative_controls_passed": True}
        stale = {"target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v2",
                 "cohort": "c", "build": "old", "device_profile": "D1", "policy_revision": "p", "corpus_revision": "c1",
                 "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "s", "device_profile": "D1"},
                 "stale": True, "acceptance": {"dictation-trust": claim}}
        mixed = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                 "cohort": "c", "build": "new", "device_profile": "D1", "policy_revision": "p", "corpus_revision": "c1",
                 "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "s", "device_profile": "D1"},
                 "acceptance": {"dictation-trust": claim}}
        evidence = []
        for i, machine in enumerate((stale, mixed)):
            evidence.append({"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                              "artifactRef": f"a-{i}", "passed": True, "machineJson": json.dumps(machine)})
        board, _ = run_board(evidence=evidence)
        trust = next(item for item in board["signals"]["rows"] if item["row"] == "dictation-trust")
        self.assertTrue(trust["in_band"])
        self.assertEqual(trust["reading"]["build"], "new")
        self.assertFalse(board["signals"]["acceptance_join"]["in_band"])

    def test_v1_receipt_is_not_reinterpreted_under_v2_contract(self):
        receipt = {"target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                   "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                   "corpus_revision": "c1", "acceptance": {
                       "dictation-trust": {"in_band": True, "negative_controls_passed": True}}}
        item = {"engineId": "local", "modelId": "m", "strategy": "native", "kind": "fault",
                "artifactRef": "old-v1", "passed": True, "machineJson": json.dumps(receipt)}
        board, _ = run_board(evidence=[item])
        trust = next(row for row in board["signals"]["rows"] if row["row"] == "dictation-trust")
        self.assertIsNone(trust["in_band"])
        self.assertEqual(trust["reason"], "pending_telemetry:no_matching_receipt")

    def test_owner_receipt_join_binds_strategy_device_and_observed_route_tier(self):
        base = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                "corpus_revision": "c1", "acceptance": {
                    "dictation-trust": {"in_band": True, "negative_controls_passed": True},
                    "explicit-routing": {"in_band": True, "negative_controls_passed": True}},
                "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "native",
                                  "device_profile": "D1", "provider_tier": "local"}}

        def evaluate(receipt, provider_tier=None):
            item = {"engineId": "local", "modelId": "m", "strategy": "native", "kind": "fault",
                    "artifactRef": "a", "passed": True, "machineJson": json.dumps(receipt)}
            if provider_tier is not None:
                item["providerTier"] = provider_tier
            board, _ = run_board(evidence=[item])
            return {row["row"]: row for row in board["signals"]["rows"]}

        wrong_strategy = json.loads(json.dumps(base))
        wrong_strategy["provider_cell"]["strategy"] = "other-strategy"
        self.assertIsNone(evaluate(wrong_strategy)["dictation-trust"]["in_band"])
        wrong_device = json.loads(json.dumps(base))
        wrong_device["provider_cell"]["device_profile"] = "D2"
        self.assertIsNone(evaluate(wrong_device)["dictation-trust"]["in_band"])
        missing_provenance = json.loads(json.dumps(base))
        missing_provenance["provider_cell"] = None
        self.assertIsNone(evaluate(missing_provenance)["dictation-trust"]["in_band"])

        # The persisted evidence row observed local; the receipt cannot claim
        # BYOK or owned routing for that same artifact.
        for claimed_tier in ("byok", "vrooli"):
            wrong_route = json.loads(json.dumps(base))
            wrong_route["provider_cell"]["provider_tier"] = claimed_tier
            self.assertIsNone(evaluate(wrong_route, "PROVIDER_TIER_LOCAL")["explicit-routing"]["in_band"])
            self.assertEqual(evaluate(wrong_route, "PROVIDER_TIER_LOCAL")["explicit-routing"]["reason"],
                             "failed:receipt_negative_control")

        matching = evaluate(base, "PROVIDER_TIER_LOCAL")
        self.assertTrue(matching["explicit-routing"]["in_band"])
        self.assertIsNone(evaluate(base)["explicit-routing"]["in_band"], "missing observed tier must remain unknown")
        missing_receipt_tier = json.loads(json.dumps(base))
        del missing_receipt_tier["provider_cell"]["provider_tier"]
        self.assertIsNone(evaluate(missing_receipt_tier, "local")["explicit-routing"]["in_band"])

    def test_simulated_receipt_cannot_certify_unmeasured_rows(self):
        required = REQUIRED_ROW_IDS
        broad_simulation = {
            "target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
            "cohort": "voice-pilot-en-v1", "build": "simulated-policy-v1", "device_profile": "SIMULATED",
            "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
            "execution_lane": "simulated",
            "details": {"execution_lane": "simulated"},
            "acceptance": {name: {"in_band": True, "negative_controls_passed": True} for name in required},
        }
        board, _ = run_board(evidence=[{"engineId": "simulated", "modelId": "v1", "strategy": "simulation",
                                        "kind": "fault", "artifactRef": "simulated-broad", "passed": True,
                                        "machineJson": json.dumps(broad_simulation)}])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], 0)
        self.assertEqual(board["signals"]["acceptance_join"]["reason"], "pending:no_target_aware_acceptance_claims")

    def test_legacy_simulated_supported_rows_cannot_resolve_without_target_aware_claims(self):
        supported = {"explicit-routing", "owned-settlement", "private-lifecycle",
                     "acceptance-integrity", "adapter-maintainability"}
        receipt = {
            "target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
            "cohort": "voice-pilot-en-v1", "build": "simulated-policy-v1", "device_profile": "D1",
            "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
            "execution_lane": "simulated", "supported_rows": sorted(supported),
            "acceptance": {name: {"in_band": True, "negative_controls_passed": True}
                           for name in supported},
        }
        board, _ = run_board(evidence=[{"engineId": "simulated", "modelId": "v1", "strategy": "simulation",
                                        "kind": "QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE", "artifactRef": "simulated-policy", "passed": True,
                                        "machineJson": json.dumps(receipt)}])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], 0)

    def test_target_aware_acceptance_composite_resolves_only_named_simulated_rows(self):
        checks = {
            "explicit-routing": ["route_local_zero_debit", "route_byok_zero_debit", "route_owned_requires_consent"],
            "owned-settlement": ["settlement_reserve_and_partial_finalize", "settlement_retry_is_idempotent", "settlement_local_byok_zero_debit"],
            "private-lifecycle": ["lifecycle_authorized_recoverable_metadata_only", "lifecycle_delete_removes_retained_item"],
            "acceptance-integrity": ["acceptance_rejects_missing_target", "acceptance_rejects_failed_artifact", "acceptance_rejects_simulated_as_native"],
            "adapter-maintainability": ["adapter_versioned_route_metadata", "adapter_metadata_only"],
        }
        supported = set(checks)
        artifact = "simulated-policy-new"
        claims = {name: {"target_id": name, "target_revision": "portable-voice-v2",
                         "method": "audio-tools-acceptance-v2", "receipt_reference": artifact,
                         "execution_lane": "simulated", "simulated_as_native": False,
                         "checks": row_checks, "in_band": True, "negative_controls_passed": True}
                  for name, row_checks in checks.items()}
        # An unsupported required row is deliberately included but has no
        # accepted target contract and must remain unknown.
        claims["dictation-trust"] = {"target_id": "dictation-trust", "target_revision": "portable-voice-v2",
                                     "method": "audio-tools-acceptance-v2", "receipt_reference": artifact,
                                     "execution_lane": "simulated", "simulated_as_native": False,
                                     "checks": ["route_local_zero_debit"], "in_band": True,
                                     "negative_controls_passed": True}
        receipt = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                   "cohort": "voice-pilot-en-v1", "build": "simulated-policy-v1", "device_profile": "D1",
                   "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
                   "provider_cell": {"engine_id": "simulated", "model_id": "v1", "strategy": "simulation", "device_profile": "D1"},
                   "execution_lane": "simulated", "simulated_as_native": False,
                   "supported_rows": sorted(supported), "acceptance": claims}
        board, _ = run_board(evidence=[{"engineId": "simulated", "modelId": "v1", "strategy": "simulation",
                                        "kind": "QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE",
                                        "artifactRef": artifact, "passed": True, "machineJson": json.dumps(receipt)}])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], len(supported))
        self.assertEqual(board["signals"]["acceptance_join"]["reason"], "pending:required_owner_receipts")
        routing = next(item for item in board["signals"]["rows"] if item["row"] == "explicit-routing")
        self.assertTrue(routing["in_band"], "route simulation is backed by its exact target-aware checks")
        for item in board["signals"]["rows"]:
            if item["row"] in supported:
                self.assertTrue(item["in_band"], item)
            elif item["row"] not in {"shared-voice-capabilities", "device-qualification"}:
                self.assertIsNone(item["in_band"], item)

        # Each target binding is required; a success-shaped claim cannot
        # survive a mismatched revision, method, source ref, lane, or checks.
        for field, value in (("target_id", "owned-settlement"),
                             ("target_revision", "future-revision"),
                             ("method", "other-method"),
                             ("receipt_reference", "another-artifact"),
                             ("execution_lane", "native"),
                             ("simulated_as_native", True),
                             ("checks", ["some-other-check"])):
            mutated = json.loads(json.dumps(receipt))
            mutated["acceptance"]["private-lifecycle"][field] = value
            bad_board, _ = run_board(evidence=[{"engineId": "simulated", "modelId": "v1", "strategy": "simulation",
                                                "kind": "QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE",
                                                "artifactRef": artifact, "passed": True,
                                                "machineJson": json.dumps(mutated)}])
            self.assertEqual(bad_board["signals"]["acceptance_join"]["resolved_rows"], len(supported) - 1, field)

        wrong_route_checks = json.loads(json.dumps(receipt))
        wrong_route_checks["acceptance"]["explicit-routing"]["checks"] = ["route_local_zero_debit"]
        wrong_route_board, _ = run_board(evidence=[{"engineId": "simulated", "modelId": "v1", "strategy": "simulation",
                                                     "kind": "QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE",
                                                     "artifactRef": artifact, "passed": True,
                                                     "machineJson": json.dumps(wrong_route_checks)}])
        self.assertEqual(wrong_route_board["signals"]["acceptance_join"]["resolved_rows"], len(supported) - 1)
        self.assertIsNone(next(row for row in wrong_route_board["signals"]["rows"]
                               if row["row"] == "explicit-routing")["in_band"])

        composite = json.loads(json.dumps(receipt))
        composite["execution_lane"] = "composite"
        composite["supported_rows"] = None
        composite["provider_cell"] = {"engine_id": "acceptance-join", "model_id": "v1", "strategy": "composite", "device_profile": "D1"}
        composite["details"] = {"claim_execution_lanes": {name: "simulated" for name in supported},
                                "source_artifacts": [artifact], "acceptance_complete": False}
        composite_board, _ = run_board(evidence=[{"engineId": "acceptance-join", "modelId": "v1",
                                                   "strategy": "composite",
                                                   "kind": "QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE",
                                                   "artifactRef": "composite-new", "passed": True,
                                                   "machineJson": json.dumps(composite)}])
        self.assertEqual(composite_board["signals"]["acceptance_join"]["resolved_rows"], len(supported))

    def test_owner_receipts_with_mixed_context_cannot_be_combined(self):
        required = REQUIRED_ROW_IDS
        def receipt(context, names):
            return json.dumps({"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                               "cohort": "c", "build": "b", "device_profile": "D1",
                               "policy_revision": context, "corpus_revision": "corpus",
                               "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "s", "device_profile": "D1"},
                               "acceptance": {name: {"in_band": True, "negative_controls_passed": True} for name in names},
                               "quality_measurement": passing_quality_measurement()})
        evidence = [
            {"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault", "artifactRef": "a1",
             "passed": True, "machineJson": receipt("policy-a", required - {"corpus-quality"})},
            {"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault", "artifactRef": "a2",
             "passed": True, "machineJson": receipt("policy-b", {"corpus-quality"})},
        ]
        board, _ = run_board(evidence=evidence)
        self.assertFalse(board["signals"]["acceptance_join"]["in_band"])
        self.assertTrue(next(item for item in board["signals"]["rows"] if item["row"] == "corpus-quality")["in_band"])

    def test_out_of_band_owner_claim_is_visible_as_failure(self):
        evidence = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                     "artifactRef": "a", "passed": True, "machineJson": json.dumps({
                         "target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                         "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                         "corpus_revision": "c1", "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "s", "device_profile": "D1"}, "acceptance": {
                             "dictation-trust": {"in_band": False, "negative_controls_passed": True}}})}]
        board, _ = run_board(evidence=evidence)
        trust = next(item for item in board["signals"]["rows"] if item["row"] == "dictation-trust")
        self.assertEqual(trust["reason"], "failed:receipt_claim_out_of_band")

    def test_host_portability_claim_rejects_incomplete_required_matrix(self):
        required = REQUIRED_ROW_IDS
        claims = {name: {"in_band": True, "negative_controls_passed": True} for name in required}
        receipt = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                   "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                   "corpus_revision": "c1", "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "s", "device_profile": "D1"}, "acceptance": claims,
                   "details": {"d1_cpu": {"available": True}}}
        board, _ = run_board(evidence=[{"engineId": "local", "modelId": "m", "strategy": "s",
                                        "kind": "fault", "artifactRef": "a", "passed": True,
                                        "machineJson": json.dumps(receipt)}])
        portability = next(item for item in board["signals"]["rows"] if item["row"] == "host-portability")
        self.assertIsNone(portability["in_band"])
        self.assertEqual(portability["reason"], "failed:host_matrix_incomplete")
        self.assertFalse(board["signals"]["acceptance_join"]["in_band"])

    def test_host_portability_requires_pinned_os_browser_and_product_provenance(self):
        required = REQUIRED_ROW_IDS
        cases = {
            "missing browser version": lambda cells: cells["d1_cpu"].pop("browser_version"),
            "wrong operating system": lambda cells: cells["d1_cpu"].update(os="windows"),
            "missing cell artifact": lambda cells: cells["d1_cpu"].update(artifact_ref="other-run"),
            "missing visible product path": lambda cells: cells["d1_cpu"].update(product_path_receipt={"outcome": "passed"}),
            "split device and product artifacts": lambda cells: cells["d1_cpu"]["product_path_receipt"].update(artifact_ref="other-run"),
            "wrong model revision omitted": lambda cells: cells["d1_cpu"].pop("model_revision"),
            "cpu-only cannot claim accelerator": claim_cpu_as_accelerator,
            "D2 resource envelope must be exact": lambda cells: cells["d2"].update(resource_cpu_limit=True),
        }
        for name, mutate in cases.items():
            with self.subTest(name=name):
                cells = host_matrix("bad-matrix")
                mutate(cells)
                receipt = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                           "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                           "corpus_revision": "c1", "provider_cell": {"engine_id": "local", "model_id": "model-1", "strategy": "s", "device_profile": "D1"}, "acceptance": {
                               row: {"in_band": True, "negative_controls_passed": True} for row in required},
                           "details": cells}
                board, _ = run_board(evidence=[{"engineId": "local", "modelId": "model-1", "strategy": "s",
                                                "kind": "fault", "artifactRef": "bad-matrix", "passed": True,
                                                "machineJson": json.dumps(receipt)}])
                portability = next(item for item in board["signals"]["rows"] if item["row"] == "host-portability")
                self.assertIsNone(portability["in_band"])
                self.assertEqual(portability["reason"], "failed:host_matrix_incomplete")

    def test_explicit_unavailable_accelerator_is_an_evidenced_cell_not_a_passed_inference(self):
        required = REQUIRED_ROW_IDS
        receipt = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                   "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                   "corpus_revision": "c1", "provider_cell": {"engine_id": "local", "model_id": "model-1", "strategy": "s", "device_profile": "D1"}, "acceptance": {
                       row: {"in_band": True, "negative_controls_passed": True} for row in required},
                   "details": host_matrix("accelerator-unavailable")}
        board, _ = run_board(evidence=[{"engineId": "local", "modelId": "model-1", "strategy": "s",
                                        "kind": "fault", "artifactRef": "accelerator-unavailable", "passed": True,
                                        "machineJson": json.dumps(receipt)}])
        portability = next(item for item in board["signals"]["rows"] if item["row"] == "host-portability")
        self.assertTrue(portability["in_band"], portability)
        self.assertEqual(portability["reading"]["evidence_refs"], ["accelerator-unavailable"])

    def test_host_matrix_accepts_distinct_fixed_cell_engines_and_models(self):
        cells = host_matrix("accelerator-passed", accelerator_state="available")
        receipt = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                   "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                   "corpus_revision": "c1", "provider_cell": {"engine_id": "aggregate", "model_id": "matrix", "strategy": "composite", "device_profile": "D1"},
                   "acceptance": {"host-portability": {"in_band": True, "negative_controls_passed": True}},
                   "details": cells}
        board, _ = run_board(evidence=[{"engineId": "aggregate", "modelId": "matrix", "strategy": "composite",
                                        "kind": "fault", "artifactRef": "accelerator-passed", "passed": True,
                                        "machineJson": json.dumps(receipt)}])
        portability = next(item for item in board["signals"]["rows"] if item["row"] == "host-portability")
        self.assertTrue(portability["in_band"], portability)

    def test_engine_inventory_does_not_count_as_host_inference(self):
        receipt = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                   "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                   "corpus_revision": "c1", "provider_cell": {"engine_id": "host-portability", "model_id": "portable-voice-v2", "strategy": "inventory-only", "device_profile": "D1"},
                   "acceptance": {"host-portability": {"in_band": True, "negative_controls_passed": True}},
                   "details": {"engine_inventory": {"whisper-local": {"available": True},
                                                        "kyutai": {"available": True}},
                               "d1_cpu": {"state": "unqualified", "inference": "not_run"},
                               "d1_accelerator": {"state": "unqualified", "inference": "not_run"},
                               "d2": {"state": "unqualified", "inference": "not_run"}}}
        board, _ = run_board(evidence=[{"engineId": "host-portability", "modelId": "portable-voice-v2",
                                        "strategy": "inventory-only", "kind": "fault", "artifactRef": "inventory",
                                        "passed": True, "machineJson": json.dumps(receipt)}])
        portability = next(item for item in board["signals"]["rows"] if item["row"] == "host-portability")
        self.assertIsNone(portability["in_band"])
        self.assertEqual(portability["reason"], "failed:host_matrix_incomplete")

    def test_failed_target_aware_receipt_is_not_ignored_by_the_join(self):
        evidence = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                     "artifactRef": "failed-artifact", "passed": False, "machineJson": json.dumps({
                         "target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                         "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                         "corpus_revision": "c1", "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "s", "device_profile": "D1"}, "acceptance": {
                             "dictation-trust": {"in_band": True, "negative_controls_passed": True}}})}]
        board, _ = run_board(evidence=evidence)
        trust = next(item for item in board["signals"]["rows"] if item["row"] == "dictation-trust")
        self.assertEqual(trust["reason"], "failed:receipt_claim_out_of_band")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], 0)

    def test_repeated_evidence_rows_use_raw_response_projection(self):
        evidence = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                     "artifactRef": "artifact", "passed": False}]
        board, calls = run_board(evidence=evidence)
        self.assertEqual(board["signals"]["evidence_integrity"]["receipt_count"], 1)
        self.assertIn(("evidence", {}), calls)

    def test_acceptance_claim_after_legacy_history_is_not_lost_at_old_bound(self):
        legacy = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                   "artifactRef": f"legacy-{i}", "passed": True} for i in range(64)]
        required = REQUIRED_ROW_IDS
        receipt = {"target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
                   "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                   "corpus_revision": "c1", "provider_cell": {"engine_id": "local", "model_id": "m", "strategy": "s", "device_profile": "D1"}, "acceptance": {
                       name: {"in_band": True, "negative_controls_passed": True} for name in required},
                   "quality_measurement": passing_quality_measurement(),
                   "details": host_matrix("current", model="m")}
        legacy.append({"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                       "artifactRef": "current", "passed": True, "machineJson": json.dumps(receipt)})
        board, _ = run_board(evidence=legacy)
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], len(required) - 1)
        self.assertEqual(board["signals"]["acceptance_join"]["inspected_receipts"], 65)

    def test_invalid_input_never_invokes_bindings(self):
        for values in ({"experiment_id": "../secret"}, {"experiment_id": 4}, {"experiment_id": "x" * 129}, {"unknown": True}, []):
            with self.subTest(values=values):
                board, calls = run_board(inputs=values)
                self.assertEqual(board["status"], "failed")
                self.assertEqual(board["errors"][0]["class"], "invalid_input")
                self.assertEqual(calls, [])

    def test_empty_owner_receipts_remain_unqualified(self):
        board, _ = run_board(evidence=[])
        self.assertEqual(board["signals"]["evidence_integrity"]["receipt_count"], 0)
        self.assertFalse(board["signals"]["evidence_integrity"]["in_band"])
        self.assertIn("no_owner_receipts", board["signals"]["evidence_integrity"]["reason"])
        self.assertEqual(board["signals"]["acceptance_join"]["required_rows"], 18)
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], 0)

    def test_omitted_empty_proto_repeated_field_is_measured_as_zero_receipts(self):
        board, _ = run_board(evidence={"metadata": {"source": "owner"}})
        integrity = board["signals"]["evidence_integrity"]
        self.assertEqual(integrity["receipt_count"], 0)
        self.assertIn("no_owner_receipts", integrity["reason"])
        self.assertFalse(integrity["in_band"])
        self.assertTrue(all(item["in_band"] is None for item in board["signals"]["rows"]
                            if item["reason"] != "excluded:mandate_scope"))

    def test_retyped_repeated_evidence_field_remains_unreadable(self):
        board, _ = run_board(evidence={"evidence": {"unexpected": "shape"}})
        integrity = board["signals"]["evidence_integrity"]
        self.assertIsNone(integrity["receipt_count"])
        self.assertIn("shape_or_bound", integrity["reason"])
        self.assertFalse(integrity["in_band"])

    def test_failed_owner_receipt_is_negative_evidence(self):
        evidence = [{"engineId": "local", "modelId": "model-1", "strategy": "native",
                     "kind": "fault", "artifactRef": "bas:run-1", "passed": False}]
        board, _ = run_board(evidence=evidence)
        integrity = board["signals"]["evidence_integrity"]
        self.assertEqual(integrity["failed_receipts"], 1)
        self.assertFalse(integrity["in_band"])
        self.assertIn("negative_control", integrity["reason"])

    def test_retained_failed_attempt_does_not_corrupt_valid_owner_audit(self):
        positive = {"engineId": "local", "modelId": "model-1", "strategy": "native",
                    "kind": "fault", "artifactRef": "good-artifact", "passed": True}
        failed = {"engineId": "local", "modelId": "model-1", "strategy": "native",
                  "kind": "fault", "artifactRef": "failed-artifact", "passed": False}
        board, _ = run_board(evidence=[positive, failed])
        integrity = board["signals"]["evidence_integrity"]
        self.assertTrue(integrity["in_band"])
        self.assertEqual(integrity["failed_receipts"], 1)
        self.assertEqual(integrity["reason"], "retained_failed_attempts")

    def test_partial_failure_preserves_other_measurements_and_redacts_error(self):
        board, _ = run_board(errors={"engines": RuntimeError("connection refused secret-transcript")})
        self.assertEqual(board["status"], "partial")
        self.assertEqual(board["errors"][0]["class"], "scenario_unreachable")
        self.assertEqual(board["signals"]["diagnostics"][-1]["reading"], 1)
        self.assertNotIn("secret-transcript", json.dumps(board))

    def test_all_failed_reads_are_unavailable_not_zero(self):
        board, _ = run_board(errors={key: RuntimeError("connection refused") for key in ("engines", "health")})
        self.assertEqual(board["status"], "unavailable")
        self.assertEqual(board["signals"]["readable"], 0)
        self.assertTrue(all(d["reading"] is None for d in board["signals"]["diagnostics"]))

    def test_empty_inventory_is_a_measured_zero_not_acceptance(self):
        board, _ = run_board(engines=[], health=[])
        self.assertEqual(board["signals"]["diagnostics"][0]["reading"], 0)
        self.assertIsNone(board["signals"]["diagnostics"][0]["in_band"])

    def test_oversized_inventory_does_not_claim_exact_count(self):
        board, _ = run_board(engines=[{"id": str(i), "available": True} for i in range(17)])
        self.assertIsNone(board["signals"]["diagnostics"][0]["reading"])
        self.assertIn("bound", board["signals"]["diagnostics"][0]["reason"])

    def test_selected_old_report_is_diagnostic_and_never_browser_latency(self):
        report = completed_report()
        report["experiment"]["recipe"] = {"realizedReference": "private speech"}
        board, calls = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
        self.assertIn(("report", {"id": "experiment-1"}), calls)
        self.assertEqual(board["signals"]["replay_metrics"][0]["wer"], 0.12)
        self.assertEqual(board["signals"]["replay_metrics"][0]["replay_finalization_p95_ms"], 800)
        self.assertFalse(board["signals"]["experiment"]["acceptance_eligible"])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertNotIn("private speech", json.dumps(board))

    def test_failed_canceled_or_wrong_experiment_cannot_supply_quality(self):
        for status, identity in (("EXPERIMENT_STATUS_FAILED", "experiment-1"), ("EXPERIMENT_STATUS_CANCELED", "experiment-1"), ("EXPERIMENT_STATUS_SUCCEEDED", "wrong")):
            report = completed_report()
            report["experiment"].update(status=status, id=identity)
            board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
            self.assertEqual(board["signals"]["replay_metrics"], [])
            self.assertFalse(board["signals"]["experiment"]["acceptance_eligible"])

    def test_missing_denominator_nan_and_absent_metrics_stay_unknown(self):
        for fields in ({"refWords": 0}, {"refWords": None}, {"wer": float("nan")}, {"wer": None}):
            board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=completed_report(**fields))
            self.assertIsNone(board["signals"]["replay_metrics"][0]["wer"])
        report = completed_report()
        del report["report"]["perStrategy"][0]["wer"]
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
        self.assertIsNone(board["signals"]["replay_metrics"][0]["wer"])

    def test_measured_zero_wer_is_preserved_and_wer_above_one_not_clamped(self):
        for wer in (0, 1.5):
            board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=completed_report(wer=wer))
            self.assertEqual(board["signals"]["replay_metrics"][0]["wer"], wer)

    def test_unmeasured_latency_is_not_imputed_from_quality(self):
        report = completed_report()
        report["report"]["latencyMeasured"] = False
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
        self.assertIsNone(board["signals"]["replay_metrics"][0]["replay_finalization_p95_ms"])

    def test_failed_child_report_does_not_inherit_inventory_success(self):
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, errors={"report": RuntimeError("deadline exceeded")})
        self.assertEqual(board["status"], "partial")
        self.assertEqual(board["signals"]["replay_metrics"], [])
        self.assertEqual(board["errors"][0]["class"], "deadline_exceeded")

    def test_successful_job_preserves_failing_cell_verdict(self):
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=completed_report(verdict="quarantined"))
        self.assertEqual(board["signals"]["replay_metrics"][0]["verdict"], "quarantined")
        self.assertEqual(board["signals"]["acceptance"], "unknown")

    def test_programming_error_is_not_reported_as_down_scenario(self):
        board, _ = run_board(errors={"engines": AttributeError("wrong binding")})
        self.assertEqual(board["status"], "failed")
        self.assertEqual(board["errors"][0]["class"], "kernel_runtime")


if __name__ == "__main__":
    unittest.main()

"""Faithful mocked protocol tests for BAS owner-bound qualification."""
import contextlib
import io
import json
from pathlib import Path
import re
import sys
import unittest
from types import SimpleNamespace as NS

ROOT = Path(__file__).resolve().parents[1]
KERNEL_HOST = ROOT.parents[2] / "program-runtime" / "kernel" / "host"
if str(KERNEL_HOST) not in sys.path:
    sys.path.insert(0, str(KERNEL_HOST))
from program_helper import ProgramHelper


class Handle:
    """Runtime Handle: rows and metadata are distinct protocol surfaces."""
    def __init__(self, rows=None, metadata=None):
        self.rows = rows or []
        self.metadata = metadata or {}

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return self.metadata


EXPECTED_IDENTITY = {"schemaVersion": 1, "identity": "tg-source", "roots": []}
BAS_ROWS = ["preservation", "interactive-feedback", "motion", "readiness", "capture",
            "passive-fidelity", "profile-durability", "known-flow-reliability", "cancellation-recovery",
            "resource-budget", "soak-stability", "evidence-completeness", "desktop-portability",
            "agent-usefulness", "structural-debt", "workspace-usability", "adversarial-review"]


def refs():
    return [{"producer": "browser-automation-studio", "artifactId": "artifact-receipt",
             "kind": "generic.file", "checksum": "c" * 64, "sizeBytes": "128"},
            {"producer": "browser-automation-studio", "artifactId": "artifact-unit-log",
             "kind": "command.output", "checksum": "d" * 64, "sizeBytes": "4096"},
            {"producer": "browser-automation-studio", "artifactId": "artifact-api-log",
             "kind": "command.output", "checksum": "e" * 64, "sizeBytes": "2048"}]


def candidate():
    return {"reviewAttemptId": "review-attempt-7", "verdictSha256": "1" * 64,
            "reviewerRunId": "reviewer-run-1", "sourceRunId": "source-run-1",
            "sandboxId": "sandbox-1", "reviewRequestId": "review-request-1",
            "sha256": "a" * 64}


def program_inputs(**changes):
    values = {"candidate": candidate(), "request_key": "am-request-42",
              "invalidated_producers": ["evidence-completeness"],
              "required_rows": ["evidence-completeness"], "retained_evidence_sets": []}
    values.update(changes)
    return values


class Owners:
    def __init__(self, *, stale=False, stale_after_refresh=False, promote_error=None,
                 promote_success=True, producer_state="RECEIPT_STATE_SUCCEEDED",
                 producer_identity="tg-source", validation_state="RECEIPT_STATE_SUCCEEDED",
                 validation_create_error=None, setpoint_status="ok", setpoint_bound=True,
                 setpoint_qualified=True, producer_admit_error=None, producer_wait_error=None,
                 validation_wait_error=None, start_error=None, uncertain_wss=False, usage=None,
                 selected_row_pass=True, missing_selected_row=False, produced_artifacts=None,
                 resolved_identity=None, identity_error=None):
        self.calls = []
        self.produced_sets = []
        self.usage = usage
        self.selected_row_pass = selected_row_pass
        self.missing_selected_row = missing_selected_row
        self.produced_artifacts = produced_artifacts
        self.promote_error = promote_error
        self.promote_success = promote_success
        self.producer_state = producer_state
        self.producer_identity = producer_identity
        self.validation_state = validation_state
        self.validation_create_error = validation_create_error
        self.setpoint_status = setpoint_status
        self.setpoint_bound = setpoint_bound
        self.setpoint_qualified = setpoint_qualified
        self.producer_admit_error = producer_admit_error
        self.producer_wait_error = producer_wait_error
        self.validation_wait_error = validation_wait_error
        self.uncertain_wss = uncertain_wss
        self.start_error = start_error
        self.resolved_identity = resolved_identity or EXPECTED_IDENTITY
        self.identity_error = identity_error

        def promote(**request):
            self.calls.append(("promote", request))
            if self.promote_error:
                raise self.promote_error
            if self.uncertain_wss:
                raise RuntimeError("response lost after promotion dispatch")
            # WSS PromoteSandboxResponse protojson, directly in Handle metadata.
            return Handle(metadata={"success": self.promote_success, "sandboxId": "sandbox-1",
                                    "applied": 1, "failed": 0, "remaining": 0,
                                    "isPartial": False, "commitHash": ""})

        def freshness(**request):
            self.calls.append(("freshness", request))
            refreshed = any(name == "start" for name, _ in self.calls)
            is_stale = stale_after_refresh if refreshed else stale
            return NS(raw=lambda: {"success": True, "stale": is_stale,
                                   "checks": [{"target": "api/api", "stale": is_stale},
                                              {"target": "ui/dist", "stale": False}]})

        def start(**request):
            self.calls.append(("start", request))
            if self.start_error:
                raise self.start_error
            return NS(raw=lambda: {"success": True, "name": "browser-automation-studio"})

        self.workspace_sandbox = NS(change=NS(promote=promote))
        self.vrooli = NS(scenario=NS(freshness=freshness, start=start))

        def resolve_identity(**request):
            self.calls.append(("resolve_identity", request))
            if self.identity_error:
                raise self.identity_error
            return Handle(metadata={"identity": self.resolved_identity})

        def produce_evidence(**request):
            self.calls.append(("produce_evidence", request))
            if self.producer_admit_error:
                raise self.producer_admit_error
            # CreateEvidenceProductionResponse { receipt: ValidationReceipt }.
            producer_receipt = "producer-receipt-" + request["producer"]
            return Handle(metadata={"receipt": {"receiptId": producer_receipt,
                                                  "state": "RECEIPT_STATE_QUEUED"}})

        def wait(**request):
            self.calls.append(("wait", request))
            receipt_id = request["receipt_id"]
            if receipt_id.startswith("producer-receipt-"):
                if self.producer_wait_error:
                    raise self.producer_wait_error
                producer = receipt_id.removeprefix("producer-receipt-")
                produced = {"producerReceiptId": receipt_id, "producer": producer,
                            "target": "browser-automation-studio", "runId": "tg-run-" + producer,
                            "candidateIdentity": self.producer_identity,
                            "catalogDigest": "ac:sha256:" + "b" * 64,
                            "artifacts": self.produced_artifacts if self.produced_artifacts is not None else refs()}
                self.produced_sets.append(produced)
                return Handle(metadata={"receipt": {"receiptId": receipt_id,
                    "state": self.producer_state, "producedEvidenceSet": produced}})
            if self.validation_wait_error:
                raise self.validation_wait_error
            return Handle(metadata={"receipt": {"receiptId": receipt_id,
                "state": self.validation_state}})

        def create(**request):
            self.calls.append(("create", request))
            if self.validation_create_error:
                raise self.validation_create_error
            return Handle(metadata={"receipt": {"receiptId": "rehabilitation-receipt",
                                                  "state": "RECEIPT_STATE_QUEUED"}})

        self.test_genie = NS(validation=NS(resolve_identity=resolve_identity,
                                          produce_evidence=produce_evidence, wait=wait, create=create))

        def setpoint_read(**request):
            self.calls.append(("setpoint_read", request))
            # lib child result is an envelope row. Metadata carries program artifact identity only.
            return Handle(rows=[{"status": self.setpoint_status,
                                 "signals": {"qualification_bound": self.setpoint_bound,
                                             "product_qualified": self.setpoint_qualified,
                                             "unmet": 1 if not self.setpoint_qualified else 0,
                                             "qualification_candidate": {"receipt_state": self.validation_state},
                                             "rows": ([{"row": name, "in_band": self.selected_row_pass if name == "evidence-completeness" else False,
                                                        "unavailable": name != "evidence-completeness"}
                                                       for name in BAS_ROWS if not (self.missing_selected_row and name == "evidence-completeness")])}}],
                          metadata={"digest": "sha256:" + "d" * 64})

        self.lib = NS(browser_automation_studio=NS(setpoint_read=setpoint_read))


def execute(values, owners):
    helper = ProgramHelper()
    scope = {"inputs": values, "program": helper,
             "workspace_sandbox": owners.workspace_sandbox,
             "test_genie": owners.test_genie, "vrooli": owners.vrooli,
             "lib": owners.lib}
    helper._bind(scope)
    stream = io.StringIO()
    with contextlib.redirect_stdout(stream):
        exec(compile((ROOT / "qualify-candidate.py").read_text(), "qualify-candidate.py", "exec"), scope)
    lines = stream.getvalue().splitlines()
    if len(lines) != 1:
        raise AssertionError(f"expected exactly one program.report envelope, got {len(lines)}")
    return json.loads(lines[0])


def execute_bound_setpoint_without_capture(*, failed_global=False):
    helper = ProgramHelper()
    receipt_state = "RECEIPT_STATE_FAILED" if failed_global else "RECEIPT_STATE_SUCCEEDED"
    child_state = "CHILD_OPERATION_STATE_FAILED" if failed_global else "CHILD_OPERATION_STATE_SUCCEEDED"
    phase_status = "failed" if failed_global else "passed"
    receipt = {"receiptId": "rehab-1", "state": receipt_state,
               "admittedIdentity": EXPECTED_IDENTITY, "observedIdentity": EXPECTED_IDENTITY,
               "children": [{"kind": "CHILD_OPERATION_KIND_TEST_RUN", "owner": "test-genie",
                             "state": child_state, "operationId": "run-1"}]}
    run = {"runId": "run-1", "target": "browser-automation-studio", "status": phase_status,
           "plannedPhases": ["rehabilitation-evidence"], "completedAt": "2026-01-02T00:00:00Z",
           "phases": [{"name": "rehabilitation-evidence", "status": phase_status}], "evidenceTier": "targeted"}
    findings = [{"name": "rehabilitation-evidence", "status": phase_status, "phasePresentation": {
        "capabilities": [{"id": "evidence-completeness", "currentLevel": "L1",
                          "clean": True, "currentLevelLabel": "Ready"}]}}]
    tg = NS(validation=NS(get=lambda **_: Handle(metadata={"receipt": receipt})),
            runs=NS(show=lambda **_: Handle(metadata={"run": run}),
                    findings=lambda **_: Handle(rows=findings, metadata={"target": "browser-automation-studio", "runId": "run-1"})))
    vrooli = NS(scenario=NS(status=lambda **_: NS(raw=lambda: {"runtime": {"buildIdentity": "build-1"}}),
                           freshness=lambda **_: NS(raw=lambda: {"success": True, "stale": False,
                                                                  "checks": [{"target": "ui", "stale": False}]})))
    performance_health = NS(sweep=NS(workload_get=lambda **_: Handle()))
    scope = {"inputs": {"profile": "rehabilitation", "validation_receipt_id": "rehab-1",
                         "expected_source_identity": "tg-source"}, "program": helper,
             "test_genie": tg, "vrooli": vrooli, "performance_health": performance_health}
    helper._bind(scope)
    stream = io.StringIO()
    with contextlib.redirect_stdout(stream):
        exec(compile((ROOT / "setpoint-read.py").read_text(), "setpoint-read.py", "exec"), scope)
    output = stream.getvalue().splitlines()
    if len(output) != 1:
        raise AssertionError(f"expected one setpoint envelope, got {len(output)}")
    return json.loads(output[0])


class QualificationComposition(unittest.TestCase):
    def test_success_order_fixed_keys_confirmations_and_unchanged_typed_sets(self):
        owners = Owners()
        result = execute(program_inputs(), owners)
        self.assertEqual("ok", result["status"])
        self.assertEqual(["promote", "resolve_identity", "freshness", "produce_evidence", "wait", "create", "wait", "setpoint_read"], [name for name, _ in owners.calls])
        wss = owners.calls[0][1]
        self.assertTrue(wss["_confirm"] and wss["confirm"])
        self.assertEqual("review-request-1", wss["review_request_id"])
        self.assertEqual("a" * 64, wss["expected_review_sha256"])
        self.assertNotIn("expected_patch_sha256", wss)
        self.assertFalse(wss["create_commit"] or wss["force"] or wss["override_acceptance"])
        identity_request = owners.calls[1][1]
        self.assertEqual("browser-automation-studio", identity_request["targets"][0]["id"])
        self.assertEqual("scenarios/browser-automation-studio", identity_request["content_inputs"][0]["root"])
        producer_requests = [args for name, args in owners.calls if name == "produce_evidence"]
        self.assertEqual(["am-request-42/evidence/evidence-completeness/v1"],
                         [request["idempotency_key"] for request in producer_requests])
        self.assertTrue(all(request["_confirm"] is True for request in producer_requests))
        self.assertTrue(all(request["_confirm"] is True for request in producer_requests))
        self.assertEqual(EXPECTED_IDENTITY, producer_requests[0]["expected_candidate_identity"])
        validation = next(args["intent"] for name, args in owners.calls if name == "create")
        self.assertEqual(EXPECTED_IDENTITY, validation["expected_identity"])
        self.assertEqual(["rehabilitation-evidence"], validation["phases"])
        self.assertEqual(owners.produced_sets, validation["retained_evidence_sets"])
        self.assertIs(owners.produced_sets[0], validation["retained_evidence_sets"][0])
        self.assertEqual("30s", validation["deadline_policy"]["queue_budget"])
        self.assertEqual("30s", validation["deadline_policy"]["execution_budget"])
        wait_requests = [args for name, args in owners.calls if name == "wait"]
        self.assertEqual(["660s", "120s"], [args["timeout"] for args in wait_requests])
        wait_seconds = [int(args["timeout"][:-1]) for args in wait_requests]
        self.assertLessEqual(max(wait_seconds), 660)  # PRT terminal-wait ceiling
        self.assertLessEqual(max(wait_seconds), 720)  # owner bridge ceiling per receipt
        self.assertLessEqual(max(wait_seconds), 780)  # kernel ceiling per receipt
        self.assertEqual(1800000, json.loads((ROOT / "qualify-candidate.json").read_text())[
            "budget"]["wall_ms"])
        self.assertEqual({"generic.file", "command.output"},
                         {artifact["kind"] for artifact in validation["retained_evidence_sets"][0]["artifacts"]})
        self.assertEqual("tg-source", owners.calls[-1][1]["expected_source_identity"])
        self.assertEqual({"promotion": 1, "refresh": 0, "producer_admissions": 1,
                          "producer_waits": 1, "validation_admissions": 1,
                          "validation_waits": 1, "setpoint_reads": 1}, result["signals"]["effects"])
        self.assertEqual("a" * 64, result["signals"]["candidate_sha256"])
        self.assertIsNone(result["usage"])

    def test_refresh_is_forced_once_only_when_stale_and_owner_grant_is_admitted(self):
        owners = Owners(stale=True)
        result = execute(program_inputs(), owners)
        self.assertEqual("ok", result["status"])
        start = next(args for name, args in owners.calls if name == "start")
        self.assertTrue(start["force"] and start["_confirm"])
        self.assertEqual(1, result["signals"]["effects"]["refresh"])
        self.assertEqual(1, len([1 for name, _ in owners.calls if name == "start"]))
        owners = Owners(stale=True, stale_after_refresh=True)
        result = execute(program_inputs(), owners)
        self.assertEqual("failed", result["status"])
        self.assertEqual(0, result["signals"]["effects"]["producer_admissions"])
        owners = Owners(stale=True, start_error=ProgramHelper.Refused("lifecycle grant denied"))
        result = execute(program_inputs(), owners)
        self.assertEqual("refused", result["status"])
        self.assertIsNone(result["signals"]["effects"]["refresh"])

    def test_input_refusal_and_malformed_candidate_precede_effects(self):
        owners = Owners()
        result = execute(program_inputs(candidate={}), owners)
        self.assertEqual("failed", result["status"])
        self.assertEqual([], owners.calls)
        result = execute(program_inputs(invalidated_producers=["producer-" + str(i) for i in range(9)]), owners)
        self.assertEqual("failed", result["status"])
        self.assertEqual([], owners.calls)
        owners = Owners(promote_error=ProgramHelper.Refused("grant denied"))
        result = execute(program_inputs(), owners)
        self.assertEqual("refused", result["status"])
        self.assertEqual("no_grant", result["errors"][0]["class"])
        owners = Owners(identity_error=ProgramHelper.Refused("identity resolver unavailable"))
        result = execute(program_inputs(), owners)
        self.assertEqual("refused", result["status"])
        self.assertEqual(["promote", "resolve_identity"], [name for name, _ in owners.calls])
        self.assertEqual(0, result["signals"]["effects"]["producer_admissions"])

    def test_tg_destructive_grant_refusal_keeps_wss_owner_handle(self):
        owners = Owners(producer_admit_error=ProgramHelper.Refused("TG producer grant denied"))
        result = execute(program_inputs(), owners)
        self.assertEqual("refused", result["status"])
        self.assertEqual(1, result["signals"]["effects"]["promotion"])
        self.assertEqual("review-request-1", result["signals"]["owner_handles"]["wss_review_request_id"])
        self.assertEqual([], result["signals"]["owner_handles"]["producer_receipts"])

    def test_mismatched_and_failed_owner_receipts_fail_without_fallback(self):
        for opts in ({"producer_identity": "different-source"},
                     {"producer_state": "RECEIPT_STATE_FAILED"},
                     {"setpoint_bound": False}, {"promote_success": False}):
            with self.subTest(opts=opts):
                owners = Owners(**opts)
                result = execute(program_inputs(), owners)
                self.assertNotEqual("ok", result["status"])
                self.assertTrue(result["errors"])
        owners = Owners(producer_state="RECEIPT_STATE_FAILED")
        result = execute(program_inputs(), owners)
        self.assertEqual("producer-receipt-evidence-completeness",
                         result["signals"]["owner_handles"]["producer_receipts"][0]["receipt_id"])
        self.assertFalse(any(name == "create" for name, _ in owners.calls))

    def test_owner_exceptions_report_once_and_keep_uncertain_usage_and_counts_unknown(self):
        for owners, counter in ((Owners(uncertain_wss=True), "promotion"),
                                (Owners(producer_admit_error=RuntimeError("lost response")), "producer_admissions"),
                                (Owners(producer_wait_error=RuntimeError("lost wait")), "producer_waits"),
                                (Owners(validation_create_error=RuntimeError("lost create")), "validation_admissions"),
                                (Owners(validation_wait_error=RuntimeError("lost wait")), "validation_waits")):
            with self.subTest(counter=counter):
                result = execute(program_inputs(), owners)
                self.assertNotEqual("ok", result["status"])
                self.assertIsNone(result["signals"]["effects"][counter])
                self.assertIsNone(result["usage"])

    def test_partial_product_can_accept_selected_boundary_and_preserves_failed_receipt(self):
        owners = Owners(validation_state="RECEIPT_STATE_FAILED", setpoint_status="partial",
                        setpoint_qualified=False)
        result = execute(program_inputs(), owners)
        self.assertEqual("ok", result["status"])
        self.assertTrue(result["signals"]["qualification"]["boundary_accepted"])
        self.assertFalse(result["signals"]["qualification"]["product_qualified"])
        self.assertEqual("RECEIPT_STATE_FAILED", result["signals"]["qualification"]["validation_receipt_state"])

    def test_selected_failure_missing_row_and_invalid_bounds_refuse(self):
        for owners in (Owners(selected_row_pass=False), Owners(missing_selected_row=True)):
            result = execute(program_inputs(), owners)
            self.assertNotEqual("ok", result["status"])
            self.assertFalse(result["signals"]["qualification"]["boundary_accepted"])
        for bad in ({"required_rows": []}, {"required_rows": ["made-up"]},
                    {"retained_evidence_sets": [{"producer": "unsupported"}]}):
            owners = Owners()
            result = execute(program_inputs(**bad), owners)
            self.assertEqual("failed", result["status"])
            self.assertEqual([], owners.calls)

    def test_zero_refresh_reuses_exact_retained_cohort(self):
        retained = {"producerReceiptId": "kept-receipt", "producer": "evidence-completeness",
                    "target": "browser-automation-studio", "runId": "kept-run",
                    "candidateIdentity": "tg-source", "catalogDigest": "ac:sha256:" + "b" * 64, "artifacts": refs()}
        owners = Owners()
        result = execute(program_inputs(invalidated_producers=[], retained_evidence_sets=[retained]), owners)
        self.assertEqual("ok", result["status"])
        self.assertFalse(any(name == "produce_evidence" for name, _ in owners.calls))
        intent = next(args["intent"] for name, args in owners.calls if name == "create")
        self.assertIs(intent["retained_evidence_sets"][0], retained)

    def test_evidence_contract_budget_and_retained_refs_are_checked_before_promotion(self):
        contract = json.loads((ROOT / "qualify-candidate.json").read_text())
        descriptor_path = ROOT.parents[1] / ".vrooli" / "test-genie.json"
        descriptor = json.loads(descriptor_path.read_text())
        descriptor_seconds = int(descriptor["timeout"].removesuffix("s"))
        self.assertEqual("rehabilitation-evidence", descriptor["phase"])
        self.assertEqual(30, descriptor_seconds)
        source = (ROOT / "qualify-candidate.py").read_text()
        self.assertIn('timeout="660s"', source)
        self.assertIn('timeout="120s"', source)
        intent_budgets = re.search(r'"queue_budget": "(\d+)s", "execution_budget": "(\d+)s"', source)
        self.assertIsNotNone(intent_budgets)
        queue_seconds, execution_seconds = map(int, intent_budgets.groups())
        self.assertEqual(descriptor_seconds, execution_seconds)
        self.assertGreaterEqual(120, queue_seconds + execution_seconds)
        self.assertGreaterEqual(660, 600)  # declared producer-command headroom
        self.assertEqual(1800000, contract["budget"]["wall_ms"])
        self.assertTrue(contract["budget"]["async"])
        self.assertGreaterEqual(contract["budget"]["wall_ms"], (660 + 120 + 90) * 1000)
        self.assertIs(contract["budget"]["async"], True)
        self.assertIn("600s", contract["budget"]["async_reason"])
        for artifacts in ([], [{"producer": "browser-automation-studio", "artifactId": "x",
                                "kind": "generic.file", "sizeBytes": "10"}],
                          [{"producer": "browser-automation-studio", "artifactId": "x",
                            "kind": "generic.file", "checksum": "f" * 64, "sizeBytes": "-1"}],
                          [{"producer": "not-browser-automation-studio", "artifactId": "x",
                            "kind": "generic.file", "checksum": "f" * 64, "sizeBytes": "1"}],
                          [{"producer": "browser-automation-studio", "artifactId": "x",
                            "kind": "trace", "checksum": "f" * 64, "sizeBytes": "1"}],
                          [{"producer": "browser-automation-studio", "artifactId": "x",
                            "kind": "generic.file", "checksum": "sha256:" + "f" * 64, "sizeBytes": "1"}],
                          [{"producer": "browser-automation-studio", "artifactId": "x",
                            "kind": "generic.file", "checksum": "f" * 64, "sizeBytes": "8388609"},
                           {"producer": "browser-automation-studio", "artifactId": "y",
                            "kind": "command.output", "checksum": "e" * 64, "sizeBytes": "8388608"}]):
            retained = {"producerReceiptId": "kept", "producer": "evidence-completeness",
                        "target": "browser-automation-studio", "runId": "kept-run",
                        "candidateIdentity": "tg-source", "catalogDigest": "ac:sha256:" + "b" * 64,
                        "artifacts": artifacts}
            owners = Owners()
            result = execute(program_inputs(invalidated_producers=[], retained_evidence_sets=[retained]), owners)
            self.assertEqual("failed", result["status"])
            self.assertEqual([], owners.calls)

    def test_produced_reference_set_is_validated_before_forwarding_unchanged(self):
        cases = []
        missing_checksum = refs()
        missing_checksum[0] = {"producer": "browser-automation-studio", "artifactId": "receipt",
                               "kind": "generic.file", "checksum": "", "sizeBytes": "128"}
        cases.append(missing_checksum)
        foreign_producer = refs()
        foreign_producer[0]["producer"] = "test-genie"
        cases.append(foreign_producer)
        unsupported_kind = refs()
        unsupported_kind[0]["kind"] = "trace"
        cases.append(unsupported_kind)
        oversized = refs()
        oversized[0]["sizeBytes"] = "16777217"
        cases.append(oversized)
        for artifacts in cases:
            with self.subTest(artifacts=artifacts):
                owners = Owners(produced_artifacts=artifacts)
                result = execute(program_inputs(), owners)
                self.assertEqual("failed", result["status"])
                self.assertFalse(any(name == "create" for name, _ in owners.calls))

    def test_interrupted_wait_is_single_unknown_attempt(self):
        owners = Owners(validation_wait_error=InterruptedError("owner wait interrupted"))
        result = execute(program_inputs(), owners)
        self.assertNotEqual("ok", result["status"])
        self.assertEqual(1, len([1 for name, _ in owners.calls if name == "wait" and
                                  _.get("receipt_id") == "rehabilitation-receipt"]))
        self.assertIsNone(result["signals"]["effects"]["validation_waits"])

    def test_bound_owner_row_does_not_borrow_capture_applicability(self):
        result = execute_bound_setpoint_without_capture()
        self.assertEqual(17, len(result["signals"]["rows"]))
        by_name = {row["row"]: row for row in result["signals"]["rows"]}
        self.assertTrue(result["signals"]["qualification_bound"])
        self.assertTrue(by_name["evidence-completeness"]["in_band"])
        self.assertTrue(by_name["capture"]["unavailable"])
        self.assertFalse(result["signals"]["product_qualified"])

    def test_failed_global_run_keeps_selected_passing_row_and_actual_status(self):
        result = execute_bound_setpoint_without_capture(failed_global=True)
        by_name = {row["row"]: row for row in result["signals"]["rows"]}
        self.assertEqual("partial", result["status"])
        self.assertEqual("RECEIPT_STATE_FAILED", result["signals"]["qualification_candidate"]["receipt_state"])
        self.assertEqual("failed", result["signals"]["owner_evidence"]["phase_status"])
        self.assertTrue(by_name["evidence-completeness"]["in_band"])
        self.assertFalse(result["signals"]["product_qualified"])


if __name__ == "__main__":
    unittest.main()

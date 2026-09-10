"""Focused offline behavior tests; no lifecycle, live ledger writes, or suite run."""
import contextlib
import copy
import io
import json
from pathlib import Path
from types import SimpleNamespace
import unittest

ROOT = Path(__file__).parent.parent


class Handle:
    def __init__(self, rows=(), meta=None):
        self.rows = list(rows)
        self.metadata = meta or {}

    def head(self, limit):
        assert 0 <= limit <= 10
        return copy.deepcopy(self.rows[:limit])

    def count(self):
        return len(self.rows)

    def map(self, transform):
        return Handle([transform(row) for row in self.rows], self.metadata)

    def agg(self, key, operation):
        values = [row[key] for row in self.rows]
        if operation != "sum":
            raise ValueError("test handle only supports sum")
        return sum(values)

    def meta(self):
        return copy.deepcopy(self.metadata)


class Memory:
    """Immutable-ID fake with commit-before-disconnect and receipt failures."""
    def __init__(self):
        self.calls = []
        self.stored = {}
        self.fail_call = None
        self.failure = RuntimeError("bridge unavailable")
        self.commit_before_failure = False
        self.missing_receipt = False
        self.hits = []
        self.cohorts = []
        self.metadata = {"reliable": True, "from": "2026-01-01T00:00:00Z",
                         "to": "2026-01-02T00:00:00Z", "eligibleAttempts": 2}
        self.recall = SimpleNamespace(recall=self.read_recall)
        self.learning = SimpleNamespace(record=self.record, observe=self.observe, measure=self.measure)

    def check(self, kind, body):
        self.calls.append((kind, copy.deepcopy(body)))
        if len(self.calls) == self.fail_call:
            raise self.failure

    def read_recall(self, **body):
        if "rows" in body:
            raise ValueError("Recall has a declared primary rows field; no ambiguous-field override is accepted")
        self.check("recall", body)
        return Handle(self.hits)

    def measure(self, **body):
        self.check("measure", body)
        return Handle(self.cohorts, self.metadata)

    def write(self, kind, body):
        item = body["attempt" if kind == "record" else "observation"]
        identity = item["attempt_id" if kind == "record" else "observation_id"]
        key = (body["scope"], kind, identity)
        existing = key in self.stored
        if existing and self.stored[key] != body:
            self.check(kind, body)
            raise RuntimeError("already_exists: immutable ID has different body")
        if self.commit_before_failure:
            self.stored[key] = copy.deepcopy(body)
        self.check(kind, body)
        self.stored[key] = copy.deepcopy(body)
        return Handle([{} if self.missing_receipt else {"entryId": identity, "existing": existing}])

    def record(self, **body):
        return self.write("record", body)

    def observe(self, **body):
        return self.write("observe", body)


class ProgramFixture:
    """Minimal program helper projection used by the offline script harness."""
    def __init__(self, namespace):
        self.namespace = namespace

    def inputs(self):
        return self.namespace["inputs"]

    def fail(self, status, klass, detail, where):
        envelope = self.namespace.get("envelope")
        if isinstance(envelope, dict):
            envelope["status"] = status
            envelope.setdefault("errors", []).append({"class": str(klass), "detail": str(detail)[:240], "where": str(where)})
        return "report"

    def classify(self, exc):
        if isinstance(exc, (NameError, AttributeError)):
            return ("failed", "kernel_runtime")
        return ("unavailable", "scenario_unreachable") if "unavailable" in str(exc) else ("failed", "binding_error")


def run(name, values, memory=None, bind=True):
    namespace = {"inputs": copy.deepcopy(values)}
    namespace["program"] = ProgramFixture(namespace)
    if bind:
        namespace["vrooli_memory"] = memory or Memory()
    stream = io.StringIO()
    with contextlib.redirect_stdout(stream):
        exec(compile((ROOT / (name + ".py")).read_text(), name, "exec"), namespace)
    output = stream.getvalue()
    assert len(output.encode()) <= 65536
    assert len(output.splitlines()) == 1
    return json.loads(output)


def prepare_inputs(**changes):
    values = {"scope": "learning-test", "task_id": "task-1", "operation": "inspect",
              "context_key": "linux-v1", "started_at": "2026-01-01T00:00:00Z",
              "task_started_at": "2026-01-01T00:00:00Z", "attempt_number": 1,
              "trigger": "test request", "approach": "inspect evidence", "provenance": "test"}
    values.update(changes)
    return values


def attempt(**changes):
    values = prepare_inputs()
    values.pop("scope")
    values.update({"finished_at": "2026-01-01T00:01:00Z", "outcome": "verified_success",
                   "evidence_refs": ["test:receipt"], "recall_status": "no_match"})
    values.update(changes)
    return values


def observation(**changes):
    values = {"disposition": "supported", "evidence_refs": ["test:feedback"],
              "provenance": "test", "observed_at": "2026-01-01T00:02:00Z"}
    values.update(changes)
    return values


class PrepareTests(unittest.TestCase):
    def test_scoped_candidates_are_not_advice_uses(self):
        mem = Memory()
        mem.hits = [{"entryId": "advice-1", "text": "Try bounded reads"},
                    {"entryId": "summary", "summary": True, "text": "summary"},
                    {"entryId": "advice-1", "text": "duplicate"}]
        result = run("prepare-attempt", prepare_inputs(), mem)
        signals = result["signals"]
        self.assertEqual(signals["recall_status"], "matched")
        self.assertEqual(signals["attempt"]["advice"], [])
        self.assertNotIn("recall_status", signals["attempt"])
        self.assertTrue(signals["decision_required"])
        self.assertEqual(signals["discarded_hits"], 1)
        self.assertEqual(signals["advice_candidates"][1]["summary"], True)
        self.assertEqual(mem.calls[0][1]["scope"], "learning-test")
        self.assertEqual(len(mem.calls), 1)

    def test_no_match_differs_from_unavailable(self):
        no_match = run("prepare-attempt", prepare_inputs())
        self.assertEqual(no_match["signals"]["attempt"]["recall_status"], "no_match")
        mem = Memory()
        mem.fail_call = 1
        unavailable = run("prepare-attempt", prepare_inputs(), mem)
        self.assertEqual(unavailable["status"], "unavailable")
        self.assertEqual(unavailable["signals"]["attempt"]["recall_status"], "unavailable")
        self.assertEqual(unavailable["signals"]["advice_candidates"], [])

    def test_identity_stable_but_separates_scope_and_attempt(self):
        def identity(**changes):
            return run("prepare-attempt", prepare_inputs(**changes))["signals"]["attempt"]["attempt_id"]
        self.assertEqual(identity(), identity(query="different retrieval query"))
        self.assertNotEqual(identity(), identity(scope="other-scope"))
        self.assertNotEqual(identity(), identity(attempt_number=2))

    def test_invalid_inputs_never_read(self):
        for changes in ({"scope": ""}, {"advice_limit": 11}, {"attempt_number": True}, {"started_at": "yesterday"}):
            mem = Memory()
            self.assertEqual(run("prepare-attempt", prepare_inputs(**changes), mem)["status"], "failed")
            self.assertEqual(mem.calls, [])

    def test_missing_runtime_is_not_transport_unavailability(self):
        result = run("prepare-attempt", prepare_inputs(), bind=False)
        self.assertEqual(result["errors"][0]["class"], "kernel_runtime")


class FinishTests(unittest.TestCase):
    def test_agent_provenance_is_accepted_and_other_values_are_rejected(self):
        result = run("finish-attempt", {"scope": "learning-test", "attempt": attempt(provenance="agent"), "observations": [observation(provenance="agent")]})
        self.assertEqual(result["status"], "ok")
        bad = attempt(provenance="service")
        self.assertEqual(run("finish-attempt", {"scope": "learning-test", "attempt": bad})["status"], "failed")

    def test_unassessed_exposure_is_valid_but_cannot_claim_adoption(self):
        value = attempt()
        value.update(recall_status="matched", advice=[{
            "entry_id": "prior", "decision": "unassessed", "verdict": "unknown"}])
        result = run("finish-attempt", {"scope": "learning-test", "attempt": value})
        self.assertEqual(result["status"], "ok")
        for change in ({"decision_change": "used it"}, {"verdict": "supported"}, {"evidence_refs": ["run:1"]}):
            bad = copy.deepcopy(value)
            bad["advice"][0].update(change)
            mem = Memory()
            self.assertEqual(run("finish-attempt", {"scope": "learning-test", "attempt": bad}, mem)["status"], "failed")
            self.assertEqual(mem.calls, [])

    def test_receipts_and_optional_absence(self):
        mem = Memory()
        result = run("finish-attempt", {"scope": "learning-test", "attempt": attempt(), "observations": [observation()]}, mem)
        self.assertEqual(result["status"], "ok")
        signals = result["signals"]
        self.assertEqual(signals["capture_status"], "complete")
        self.assertEqual(signals["observations"][0]["entry_id"], signals["observations"][0]["observation_id"])
        self.assertNotIn("tool_round_trips", signals["retry_inputs"]["attempt"])
        retry = run("finish-attempt", signals["retry_inputs"], mem)
        self.assertTrue(retry["signals"]["existing"])
        self.assertTrue(retry["signals"]["observations"][0]["existing"])
        self.assertEqual(mem.calls[0], mem.calls[2])
        self.assertEqual(mem.calls[1], mem.calls[3])

    def test_partial_observation_capture_keeps_success_and_receipts(self):
        mem = Memory()
        mem.fail_call = 3
        values = {"scope": "learning-test", "attempt": attempt(), "observations": [observation(), observation(disposition="unknown", evidence_refs=[])]}
        result = run("finish-attempt", values, mem)
        signals = result["signals"]
        self.assertEqual(result["status"], "partial")
        self.assertEqual(signals["task_outcome"], "verified_success")
        self.assertEqual(signals["capture_status"], "capture_failed")
        self.assertTrue(signals["entry_id"])
        self.assertEqual(len(signals["observations"]), 1)
        self.assertEqual(result["errors"][0]["class"], "capture_failed")
        retry = run("finish-attempt", signals["retry_inputs"], mem)
        self.assertEqual(retry["status"], "ok")
        self.assertEqual(mem.calls[:3], mem.calls[3:])

    def test_lost_receipt_after_commit_exact_retry(self):
        mem = Memory()
        mem.fail_call = 1
        mem.commit_before_failure = True
        result = run("finish-attempt", {"scope": "learning-test", "attempt": attempt()}, mem)
        self.assertIsNone(result["signals"]["entry_id"])
        self.assertEqual(result["status"], "partial")
        self.assertEqual(len(mem.stored), 1)
        retry = run("finish-attempt", result["signals"]["retry_inputs"], mem)
        self.assertEqual(retry["status"], "ok")
        self.assertTrue(retry["signals"]["existing"])
        self.assertEqual(mem.calls[0], mem.calls[1])

    def test_changed_body_keeps_id_and_surfaces_conflict(self):
        mem = Memory()
        result = run("finish-attempt", {"scope": "learning-test", "attempt": attempt()}, mem)
        changed = result["signals"]["retry_inputs"]
        changed["attempt"]["outcome"] = "unknown"
        result = run("finish-attempt", changed, mem)
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["task_outcome"], "unknown")
        self.assertEqual(len(mem.stored), 1)

    def test_validate_all_observations_before_any_write(self):
        for bad in (observation(attempt_id="other"), observation(provenance="service"), observation(evidence_refs=[])):
            mem = Memory()
            result = run("finish-attempt", {"scope": "learning-test", "attempt": attempt(), "observations": [observation(), bad]}, mem)
            self.assertEqual(result["errors"][0]["class"], "invalid_input")
            self.assertEqual(mem.calls, [])

    def test_no_invented_advice_or_unverified_success(self):
        for a in (attempt(recall_status="matched"), attempt(evidence_refs=[]), attempt(outcome="failed"), attempt(tool_round_trips=None)):
            mem = Memory()
            self.assertEqual(run("finish-attempt", {"scope": "learning-test", "attempt": a}, mem)["status"], "failed")
            self.assertEqual(mem.calls, [])

    def test_failed_task_can_be_captured_successfully(self):
        result = run("finish-attempt", {"scope": "learning-test", "attempt": attempt(outcome="failed", failure_fingerprint="timeout")})
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["signals"]["task_outcome"], "failed")

    def test_missing_receipt_and_refusal_stop_without_observing(self):
        for missing in (True, False):
            mem = Memory()
            mem.missing_receipt = missing
            if not missing:
                mem.fail_call = 1
                mem.failure = RuntimeError("requires an explicit grant")
            result = run("finish-attempt", {"scope": "learning-test", "attempt": attempt(), "observations": [observation()]}, mem)
            self.assertEqual(result["status"], "partial")
            self.assertEqual(len(mem.calls), 1)
            self.assertIsNone(result["signals"]["entry_id"])


class CompareTests(unittest.TestCase):
    def test_resolved_default_window_is_replayable(self):
        mem = Memory()
        first = run("compare-outcomes", {"scope": "test"}, mem)
        self.assertTrue(first["inputs"]["from"])
        self.assertTrue(first["inputs"]["to"])
        second = run("compare-outcomes", first["inputs"], mem)
        self.assertEqual(first["inputs"], second["inputs"])
        self.assertEqual(mem.calls[0], mem.calls[1])
        self.assertEqual(mem.calls[0][1]["from"], first["inputs"]["from"])
        self.assertEqual(mem.calls[0][1]["to"], first["inputs"]["to"])

    def test_seven_rows_multiple_cohorts_zero_and_unknown(self):
        mem = Memory()
        mem.cohorts = [{"operation": "inspect", "contextKey": "a", "attempts": 1,
                        "appliedAdvice": 2},
                       {"operation": "inspect", "contextKey": "b", "attempts": 1,
                        "medianToolRoundTrips": 0.0, "rejectedAdvice": 1}]
        result = run("compare-outcomes", {"scope": "test", "from": "2026-01-01T00:00:00Z", "to": "2026-01-02T00:00:00Z", "operation": "inspect", "context_key": ""}, mem)
        self.assertEqual(result["status"], "ok")
        rows = {row["row"]: row for row in result["signals"]["rows"]}
        self.assertEqual(len(rows), 7)
        self.assertEqual(rows["failure-recurrence"]["reading"]["cohorts"][0]["failed"], 0)
        cohorts = rows["agent-round-trips"]["reading"]["cohorts"]
        self.assertIsNone(cohorts[0]["medianToolRoundTrips"])
        self.assertEqual(cohorts[1]["medianToolRoundTrips"], 0.0)
        self.assertEqual([c["context"] for c in cohorts], ["a", "b"])
        self.assertEqual(result["signals"]["aggregate"]["advice"]["applied"], 2)
        self.assertEqual(result["signals"]["aggregate"]["advice"]["rejected"], 1)
        self.assertEqual(mem.calls[0][1]["rows"], "cohorts")
        self.assertEqual(mem.calls[0][1]["operation"], "inspect")

    def test_scan_unreliability_survives(self):
        mem = Memory()
        mem.cohorts = [{"operation": "inspect", "contextKey": "a"}]
        mem.metadata.update(reliable=False, reason="unreliable:invalid_records", invalidRecords=2)
        result = run("compare-outcomes", {"scope": "test"}, mem)
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["reliability"]["invalidRecords"], 2)
        self.assertTrue(all(row["unavailable"] for row in result["signals"]["rows"]))
        self.assertEqual(result["signals"]["rows"][0]["reason"], "unreliable:invalid_records")

    def test_cohort_limit_does_not_claim_complete_sample(self):
        mem = Memory()
        mem.cohorts = [{"operation": "inspect", "contextKey": str(n)} for n in range(6)]
        result = run("compare-outcomes", {"scope": "test", "cohort_limit": 2}, mem)
        reliability = result["signals"]["reliability"]
        self.assertEqual(reliability["cohort_count"], 6)
        self.assertEqual(reliability["returned_cohorts"], 2)
        self.assertEqual(reliability["reason"], "unreliable:cohort_sample")
        self.assertFalse(reliability["reliable"])

    def test_unavailable_is_not_zero_or_no_match(self):
        mem = Memory()
        mem.fail_call = 1
        result = run("compare-outcomes", {"scope": "test"}, mem)
        self.assertEqual(result["status"], "unavailable")
        self.assertIsNone(result["signals"]["reliability"]["eligibleAttempts"])
        self.assertTrue(all(row["reading"] is None for row in result["signals"]["rows"]))

    def test_no_cohorts_does_not_establish_health(self):
        result = run("compare-outcomes", {"scope": "test"})
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["reliability"]["reason"], "unreliable:no_eligible_attempts")

    def test_output_bound_preserves_full_id_or_explicit_unknown(self):
        mem = Memory()
        mem.cohorts = [{"operation": "x" * 1024, "contextKey": str(i) + "y" * 1023} for i in range(10)]
        result = run("compare-outcomes", {"scope": "test", "cohort_limit": 10}, mem)
        self.assertEqual(result["signals"]["reliability"]["reason"], "unreliable:output_bound")
        self.assertTrue(all(row["reading"] is None for row in result["signals"]["rows"]))


class RuntimeWrapperTests(unittest.TestCase):
    def invoke(self, name, values, tasks):
        stream = io.StringIO()
        namespace = {"inputs": values, "tasks": tasks}
        namespace["program"] = ProgramFixture(namespace)
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / (name + ".py")).read_text(), name, "exec"), namespace)
        return json.loads(stream.getvalue())

    def test_repeat_task_uses_scoped_advice_with_equal_outcome_and_less_work(self):
        mem = Memory()
        context = {"operation": "fixture.inspect", "context_key": "fixture-v1", "prepare": {"signals": {}}}
        options = {"options": ["scan", "indexed"], "default_id": "scan"}
        def execute(number):
            prepared = run("prepare-attempt", prepare_inputs(operation="fixture.inspect", context_key="fixture-v1", attempt_number=number), mem)
            context["prepare"] = prepared
            selected = self.invoke("choose-option", options, SimpleNamespace(current=lambda: context))
            route = selected["signals"]["selected_id"]
            # Both routes must establish the same owner assertion. Count actual
            # fixture reads, rather than claiming retrieval itself is improvement.
            records = ["unrelated", "older", "desired"]
            reads = []
            for index in ([2] if route == "indexed" else [0, 1, 2]):
                reads.append(index)
                if records[index] == "desired":
                    break
            self.assertEqual(records[reads[-1]], "desired")
            completed = prepared["signals"]["attempt"]
            completed.update(finished_at="2026-01-01T00:01:00Z", outcome="verified_success",
                             evidence_refs=["fixture:desired"], tool_round_trips=len(reads),
                             recall_status=prepared["signals"]["recall_status"],
                             advice=selected["signals"]["learning"]["advice"])
            captured = run("finish-attempt", {"scope": "learning-test", "attempt": completed}, mem)
            self.assertEqual(captured["status"], "ok")
            return selected, len(reads)
        first, first_reads = execute(1)
        self.assertEqual(first["signals"]["source"], "default")
        # An owner-validated preference from the first investigation is memory
        # input for the second attempt, with exact operation and context identity.
        mem.hits = [{"entryId": "verified-index", "text": "option-preference/v1 " + json.dumps({
            "operation": "fixture.inspect", "context_key": "fixture-v1", "option_id": "indexed"})}]
        second, second_reads = execute(2)
        self.assertEqual(second["signals"]["source"], "advice")
        self.assertEqual(second["signals"]["learning"]["advice"][0]["decision"], "applied")
        self.assertLess(second_reads, first_reads)
        self.assertEqual((first_reads, second_reads), (3, 1))

    def test_advice_cannot_expand_options_cross_context_or_resolve_conflicts(self):
        def choose(preferences):
            candidates = [{"entry_id": str(i), "text": "option-preference/v1 " + json.dumps(p)}
                          for i, p in enumerate(preferences)]
            current = {"operation": "fixture.inspect", "context_key": "ctx", "prepare": {"signals": {"advice_candidates": candidates}}}
            return self.invoke("choose-option", {"options": ["safe", "faster"], "default_id": "safe"}, SimpleNamespace(current=lambda: current))
        pref = {"operation": "fixture.inspect", "context_key": "ctx", "option_id": "faster"}
        for altered in (dict(pref, option_id="unapproved"), dict(pref, context_key="other"), dict(pref, operation="other")):
            self.assertEqual(choose([altered])["signals"]["selected_id"], "safe")
        conflict = choose([pref, dict(pref, option_id="safe")])
        self.assertTrue(conflict["signals"]["conflicting"])
        self.assertEqual(conflict["signals"]["learning"]["advice"], [])

    def test_run_keeps_failed_outcome_separate_from_pending_delivery(self):
        tasks = SimpleNamespace(run=lambda **kw: Handle([{"status": "partial", "signals": {"outcome": "failed"}, "learning": {"delivery": "pending"}, "evidence": ["run:1"]}]))
        result = self.invoke("run-task", {"operation": "owner.task", "inputs": {}}, tasks)
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["result"]["outcome"], "failed")
        self.assertEqual(result["signals"]["learning"]["delivery"], "pending")

    def test_inspection_forwards_capability_but_does_not_leak_payload(self):
        calls = []
        def get(**kw):
            calls.append(kw)
            return Handle([{"attempt_id": "a", "delivery": "delivered", "finish_inputs": {"private": "payload"}, "resume_token": "secret"}])
        result = self.invoke("inspect-task", {"attempt_id": "a", "resume_token": "secret"}, SimpleNamespace(get=get))
        self.assertEqual(calls, [{"attempt_id": "a", "resume_token": "secret"}])
        self.assertNotIn("secret", json.dumps(result))
        self.assertNotIn("payload", json.dumps(result))

    def test_resume_only_calls_capture_recovery(self):
        calls = []
        def resume(**kw):
            calls.append(kw)
            return Handle([{"attempt_id": "a", "outcome": "unknown", "delivery": "pending"}])
        result = self.invoke("resume-task", {"attempt_id": "a", "resume_token": "secret"}, SimpleNamespace(resume=resume))
        self.assertEqual(len(calls), 1)
        self.assertEqual(result["signals"]["task"]["delivery"], "pending")
        self.assertEqual(result["signals"]["task"]["outcome"], "unknown")


class ContractTests(unittest.TestCase):
    def test_contracts_validate_against_owner_schema(self):
        import jsonschema
        schema = json.loads((ROOT.parents[2] / "program-runtime/schemas/program-contract.schema.json").read_text())
        for name in ("prepare-attempt", "finish-attempt", "compare-outcomes", "run-task", "inspect-task", "resume-task", "choose-option"):
            with self.subTest(name=name):
                contract = json.loads((ROOT / (name + ".json")).read_text())
                jsonschema.validate(contract, schema)
                self.assertEqual(contract["name"], "vrooli-memory." + name)


if __name__ == "__main__":
    unittest.main()


def test_later_feedback_is_delivered_idempotently_with_the_finish_outbox():
    """[REQ:LV-08] Later-run feedback survives disconnect without repeating domain work."""
    memory = Memory()
    values = {"scope": "learning-test", "attempt": attempt(attempt_id="current"),
              "observations": [observation(attempt_id="earlier", method_revision="learn.feedback")]}
    memory.fail_call = 2
    memory.commit_before_failure = True
    failed = run("finish-attempt", values, memory)
    assert failed["status"] == "partial"
    memory.fail_call = None
    retried = run("finish-attempt", failed["signals"]["retry_inputs"], memory)
    assert retried["status"] == "ok"
    assert retried["signals"]["observations"][0]["existing"] is True
    assert len(memory.stored) == 2

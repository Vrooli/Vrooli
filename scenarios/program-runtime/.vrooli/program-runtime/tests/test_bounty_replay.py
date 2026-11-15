"""Owner-boundary tests for the governed F9 bounty replay program."""
import contextlib
import copy
import hashlib
import io
import json
import threading
import unittest
from pathlib import Path
from types import SimpleNamespace as NS

ROOT = Path(__file__).parents[1]
CONTRACT = json.loads((ROOT / "bounty-replay.json").read_text())
SOURCE = (ROOT / "bounty-replay.py").read_text()
VARIANTS = (
    "FROZEN_REPLAY_VARIANT_PREIMAGE",
    "FROZEN_REPLAY_VARIANT_CANDIDATE",
    "FROZEN_REPLAY_VARIANT_REFERENCE",
)
ROLES = ("source", "candidate", "reference")
def seal_digests(case_id):
    return {role: hashlib.sha256((case_id + ":" + role).encode()).hexdigest() for role in ROLES}


class Handle:
    def __init__(self, rows):
        self.rows = rows

    def head(self, _count):
        return copy.deepcopy(self.rows)


class ReplayOwners:
    """Small in-memory owner fakes; they retain durable records across invocations."""
    def __init__(self, *, case_id="case-1", interrupt_wait=False, tamper_role=None):
        self.case_id = case_id
        self.seals = seal_digests(case_id)
        self.interrupt_wait = interrupt_wait
        self.tamper_role = tamper_role
        self.calls = []
        self.jobs = {}
        self.sandboxes = {}
        self.validation = {}
        self.frozen = {}
        self._lock = threading.Lock()

        self.client_rehearsal = NS(
            bounty_cases=NS(resolve=self.resolve_case),
            coding_jobs=NS(create=self.create_job, record_stage=self.record_stage,
                           record_evaluation=self.record_evaluation),
        )
        self.workspace_sandbox = NS(
            sandbox=NS(create=self.create_sandbox),
            replay_seal=NS(create=self.create_seal, resolve=self.resolve_seal,
                           verify=self.verify_seal),
        )
        self.test_genie = NS(
            validation=NS(resolve_identity=self.resolve_identity, create=self.create_validation,
                          wait=self.wait_validation),
            frozen_replay=NS(create=self.create_frozen, get=self.get_frozen),
        )
        self.agent = NS(start=self.agent_start, collect=self.agent_collect)

    def record(self, operation, **kwargs):
        with self._lock:
            self.calls.append((operation, copy.deepcopy(kwargs)))

    def resolve_case(self, **kwargs):
        self.record("case.resolve", **kwargs)
        return Handle([{"case": {
            "caseId": self.case_id, "publicUrl": "https://example.invalid/issues/1", "project": "fixture",
            "title": "Synthetic bounty", "task": "Fix the deterministic fixture bug.", "repository": "fixture/repo",
            "issueNumber": 1, "pullRequestUrl": "https://example.invalid/pulls/2", "pullRequestNumber": 2,
            "sourceCommit": "source-commit", "referenceCommit": "reference-commit",
            "bundleDigest": "d" * 64, "sourceProjectRoot": "/corpus/source", "referenceProjectRoot": "/corpus/reference",
            "frozenEvaluatorDigest": "e" * 64, "frozenEvaluatorEntrypoint": "tests/replay.py",
            "evaluatorBundleId": self.case_id + "@1:" + "d" * 64, "frozenEvaluatorBundleDigest": "f" * 64,
            "testGeniePlan": {
                "validationPurpose": "VALIDATION_PURPOSE_REGRESSION_CURRENT",
                "requiredStrength": "VALIDATION_STRENGTH_TARGETED",
                "phases": [],
                "evidencePolicy": {"requireSourceSnapshot": True, "allowAuthorizedDegradation": False,
                                   "requiredEvidenceKinds": ["test-genie-run"]},
                "deadlinePolicy": {"queueBudgetMilliseconds": 300000, "executionBudgetMilliseconds": 1200000,
                                   "maximumAttempts": 1},
            },
        }}])

    def create_sandbox(self, **kwargs):
        self.record("sandbox.create", **kwargs)
        name = kwargs["name"]
        role = name.rsplit("-", 1)[-1]
        sandbox_id = "sandbox-" + self.case_id + "-" + role
        root = "/sandbox/" + self.case_id + "/" + role
        self.sandboxes[name] = {"sandboxId": sandbox_id, "workspaceRoot": root}
        return Handle([{"sandbox": self.sandboxes[name]}])

    def create_seal(self, **kwargs):
        self.record("seal.create", **kwargs)
        role = kwargs["role"].removeprefix("REPLAY_SEAL_ROLE_").lower()
        return Handle([{"success": True, "sealDigest": self.seals[role]}])

    def resolve_seal(self, **kwargs):
        self.record("seal.resolve", **kwargs)
        role = kwargs["role"].removeprefix("REPLAY_SEAL_ROLE_").lower()
        return Handle([{"success": True, "caseId": self.case_id, "role": kwargs["role"],
                        "sealDigest": kwargs["seal_digest"], "workspaceRoot": "/sandbox/" + self.case_id + "/" + role}])

    def verify_seal(self, **kwargs):
        self.record("seal.verify", **kwargs)
        role = kwargs["role"].removeprefix("REPLAY_SEAL_ROLE_").lower()
        intact = role != self.tamper_role
        return Handle([{"success": True, "caseId": self.case_id, "role": kwargs["role"],
                        "sealDigest": kwargs["seal_digest"], "intact": intact,
                        "tamperDetected": not intact}])

    def create_job(self, **kwargs):
        self.record("coding_jobs.create", **kwargs)
        key = kwargs["replay_key"]
        if key not in self.jobs:
            self.jobs[key] = {"id": "job-" + self.case_id, "caseId": self.case_id, "project": "fixture",
                "sourceSealRef": self.seals["source"], "state": "CODING_JOB_STATE_READY", "revision": 1,
                "events": [], "solverRunRef": "", "evaluatorReceiptRef": "", "evaluatorReceiptSha256": "", "score": ""}
        return Handle([{"job": self.jobs[key]}])

    def record_stage(self, **kwargs):
        self.record("coding_jobs.record_stage", **kwargs)
        job = next(iter(self.jobs.values()))
        job["revision"] += 1
        job["events"].append({"stage": kwargs["stage"], "evidenceRef": kwargs.get("evidence_ref", ""),
                              "evidenceSha256": kwargs.get("evidence_sha256", "")})
        if kwargs["stage"] == "solver_started": job["state"] = "CODING_JOB_STATE_SOLVING"
        if kwargs["stage"] == "solver_completed":
            job["state"] = "CODING_JOB_STATE_SCORING"
            job["solverRunRef"] = kwargs["evidence_ref"]
        return Handle([{"job": job}])

    def record_evaluation(self, **kwargs):
        self.record("coding_jobs.record_evaluation", **kwargs)
        job = next(iter(self.jobs.values()))
        job.update({"revision": job["revision"] + 1, "state": "CODING_JOB_STATE_REVIEWABLE",
                    "evaluatorReceiptRef": kwargs["receipt_ref"],
                    "evaluatorReceiptSha256": kwargs["receipt_sha256"], "score": kwargs["score"]})
        return Handle([{"job": job}])

    def resolve_identity(self, **kwargs):
        self.record("validation.resolve_identity", **kwargs)
        content = kwargs["content_inputs"]
        seal = content[0]["workspaceSandboxSeal"]
        role = next(role for role in ROLES if role == seal["role"].split("_")[-1].lower())
        identity = {"identity": "sha256:" + seal["sealDigest"], "roots": [
            {"name": "frozen-evaluator", "identity": "sha256:" + "f" * 64,
             "files": [{"path": "tests/replay.py", "digest": "e" * 64, "size": 10}]},
        ]}
        return Handle([{"identity": identity}])

    def create_validation(self, **kwargs):
        self.record("validation.create", **kwargs)
        intent = kwargs["intent"]
        role = intent["frozenReplayExecution"]["role"].split("_")[-1].lower()
        receipt_id = "validation-" + role
        self.validation[receipt_id] = role
        return Handle([{"receipt": {"receiptId": receipt_id}}])

    def wait_validation(self, **kwargs):
        self.record("validation.wait", **kwargs)
        if self.interrupt_wait:
            self.interrupt_wait = False
            raise TimeoutError("synthetic wait interruption")
        receipt_id = kwargs["receipt_id"]
        role = self.validation[receipt_id]
        identity = "sha256:" + self.seals[role]
        children = [{"kind": "CHILD_OPERATION_KIND_TEST_RUN", "owner": "test-genie", "operationId": "run-" + role}]
        base = {"receiptId": receipt_id, "children": children, "admittedIdentity": {"identity": identity},
                "observedIdentity": {"identity": identity}}
        base["state"] = "RECEIPT_STATE_FAILED" if role == "source" else "RECEIPT_STATE_SUCCEEDED"
        return Handle([{"receipt": base}])

    def create_frozen(self, **kwargs):
        self.record("frozen_replay.create", **kwargs)
        adjudication = copy.deepcopy(kwargs["adjudication"])
        for item in adjudication["evidence"]:
            item["artifactDigest"] = "sha256:" + hashlib.sha256(item["producerReceiptId"].encode()).hexdigest()
        disposition = "FROZEN_REPLAY_DISPOSITION_QUALIFIED"
        receipt = {"receiptId": "frozen-1", "receiptDigest": "sha256:" + "9" * 64,
                   "adjudication": adjudication, "disposition": disposition}
        self.frozen[receipt["receiptId"]] = receipt
        return Handle([{"receipt": receipt}])

    def get_frozen(self, **kwargs):
        self.record("frozen_replay.get", **kwargs)
        return Handle([{"receipt": self.frozen[kwargs["receipt_id"]]}])

    def agent_start(self, **kwargs):
        self.record("agent.start", **kwargs)
        return "solver-handle" if kwargs["workflow_key"].endswith("bounty-solver") else "scorer-handle"

    def agent_collect(self, handle, **kwargs):
        self.record("agent.collect", handle=handle, **kwargs)
        if handle == "solver-handle": return Handle([{"execution_id": "solver-run", "result": {"outcome": "solved"}}])
        return Handle([{"result": {"verdict": "qualifies", "reason": "three immutable cases"}}])


class BountyReplayTests(unittest.TestCase):
    def invoke(self, owners, *, key="run-1", case_id="case-1"):
        envelope = {"status": "ok", "errors": [], "evidence": [], "signals": {"tamper_detected": False}}
        program = NS(
            inputs=lambda: {"case_id": case_id, "idempotency_key": key},
            envelope=lambda _name, _version: envelope,
            classify=lambda exc: ("unavailable", "remote_error"),
            fail=lambda status, klass, detail, where: self._fail(envelope, status, klass, detail, where),
            run=lambda states, initial: self._run(states, initial),
            report=lambda: print(json.dumps(envelope)),
        )
        stream = io.StringIO()
        context = {"program": program, "client_rehearsal": owners.client_rehearsal,
                   "workspace_sandbox": owners.workspace_sandbox, "test_genie": owners.test_genie,
                   "agent": owners.agent}
        with contextlib.redirect_stdout(stream):
            exec(compile(SOURCE, "bounty-replay", "exec"), context)
        return json.loads(stream.getvalue())

    @staticmethod
    def _fail(envelope, status, klass, detail, where):
        envelope["status"] = status
        envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
        return "report"

    @staticmethod
    def _run(states, initial):
        state = initial
        for _ in range(40):
            if state is None: return
            step = states[state]
            state = step()
        raise AssertionError("program did not terminate")

    def test_full_flow_uses_typed_owner_identity_and_never_sends_reference_to_solver(self):
        owners = ReplayOwners()
        result = self.invoke(owners)
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["signals"]["score"], "strict_pass")
        self.assertEqual(result["signals"]["review_packet"]["receipt_id"], "frozen-1")
        solve = next(kwargs for name, kwargs in owners.calls if name == "agent.start" and kwargs["workflow_key"].endswith("bounty-solver"))
        self.assertEqual(set(solve["input"]), {"caseId", "task", "checkout"})
        self.assertNotIn("referenceProjectRoot", solve["input"])
        scorer = next(kwargs for name, kwargs in owners.calls if name == "agent.start" and kwargs["workflow_key"].endswith("bounty-scorer"))
        self.assertNotIn("checkout", scorer["input"])
        self.assertNotIn("task", scorer["input"])
        self.assertNotIn("referenceProjectRoot", scorer["input"])
        identity = next(kwargs for name, kwargs in owners.calls if name == "validation.resolve_identity")
        self.assertIn("workspaceSandboxSeal", identity["content_inputs"][0])
        self.assertEqual(identity["targets"][0]["id"], "case-1")
        self.assertNotIn("workspaceRoot", json.dumps(result))
        self.assertNotIn("/sandbox/", json.dumps(result))
        self.assertEqual(len([1 for name, _ in owners.calls if name == "validation.create"]), 3)

    def test_tamper_stops_before_validation_or_score_mutation(self):
        owners = ReplayOwners(tamper_role="reference")
        result = self.invoke(owners)
        self.assertEqual(result["status"], "failed")
        self.assertTrue(result["signals"]["tamper_detected"])
        self.assertFalse(any(name == "validation.create" for name, _ in owners.calls))
        self.assertFalse(any(name == "coding_jobs.record_evaluation" for name, _ in owners.calls))

    def test_interrupted_validation_reattaches_same_job_and_receipts(self):
        owners = ReplayOwners(interrupt_wait=True)
        first = self.invoke(owners)
        self.assertEqual(first["status"], "unavailable")
        second = self.invoke(owners)
        self.assertEqual(second["status"], "ok")
        self.assertEqual(len([1 for name, _ in owners.calls if name == "coding_jobs.create"]), 2)
        self.assertEqual(len([1 for name, _ in owners.calls if name == "coding_jobs.record_evaluation"]), 1)
        self.assertEqual(len([1 for name, _ in owners.calls if name == "frozen_replay.create"]), 1)

    def test_parallel_independent_runs_have_distinct_owner_idempotency_keys(self):
        owners_a, owners_b = ReplayOwners(case_id="case-a"), ReplayOwners(case_id="case-b")
        results = {}
        workers = [threading.Thread(target=lambda key, owner: results.setdefault(key, self.invoke(owner, key=key, case_id=owner.case_id)), args=(key, owner))
                   for key, owner in (("parallel-a", owners_a), ("parallel-b", owners_b))]
        for worker in workers: worker.start()
        for worker in workers: worker.join(timeout=5)
        self.assertTrue(all(not worker.is_alive() for worker in workers))
        self.assertEqual({result["status"] for result in results.values()}, {"ok"}, results)
        keys = {kwargs["intent"]["idempotencyKey"] for owner in (owners_a, owners_b)
                for name, kwargs in owner.calls if name == "validation.create"}
        self.assertEqual(len(keys), 6)
        job_case_ids = {owner.jobs[key]["caseId"] for owner in (owners_a, owners_b) for key in owner.jobs}
        self.assertEqual(job_case_ids, {"case-a", "case-b"})

    def test_contract_schema(self):
        import jsonschema
        schema = json.loads((ROOT.parents[1] / "schemas/program-contract.schema.json").read_text())
        jsonschema.validate(CONTRACT, schema)


if __name__ == "__main__":
    unittest.main()

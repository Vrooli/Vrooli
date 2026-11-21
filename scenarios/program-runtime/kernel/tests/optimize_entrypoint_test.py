import contextlib
import hashlib
import io
import json
import runpy
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / ".vrooli" / "program-runtime" / "optimize.py"


def run_program(inputs, runtime=None):
    class Program:
        @staticmethod
        def inputs():
            return inputs

        @staticmethod
        def classify(exc):
            return "failed", "remote_error"

    namespace = {"program": Program()}
    if runtime is not None:
        namespace["program_runtime"] = runtime
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        runpy.run_path(str(SOURCE), init_globals=namespace)
    lines = output.getvalue().splitlines()
    if len(lines) != 1:
        raise AssertionError(f"expected exactly one envelope, got {len(lines)} lines")
    return json.loads(lines[0])


def make_record(run_id, stage, revision=1, request=None):
    request = request if request is not None else role_request(run_id)
    case_ids = request["case_ids"]
    case_digest = hashlib.sha256(json.dumps(case_ids, separators=(",", ":")).encode("utf-8")).hexdigest()
    candidates = request["candidates"]
    acquired_digests = {item["id"]: item.get("artifact_digest") or "acquired:" + item["id"] for item in candidates}
    evidence = []
    if stage in ("preflight_disk", "acquire", "evaluate", "rights", "disk", "promote", "cleanup", "complete"):
        evidence.append({"stage": "preflight_rights", "receipt_id": "owner-preflight-rights"})
    if stage in ("acquire", "evaluate", "rights", "disk", "promote", "cleanup", "complete"):
        evidence.append({"stage": "preflight_disk", "receipt_id": "owner-preflight-disk"})
    if stage in ("evaluate", "rights", "disk", "promote", "cleanup", "complete"):
        evidence.append({
            "stage": "acquire",
            "receipt_id": "owner-acquire",
            "details": {"candidate_artifact_digests": acquired_digests},
        })
    if stage in ("rights", "disk", "promote", "cleanup", "complete"):
        evidence.append({
            "stage": "evaluate",
            "receipt_id": "owner-evaluate",
            "details": {
                "suite_id": request["suite_id"],
                "case_count": len(case_ids),
                "case_digest": case_digest,
                "candidate_results": {
                    item["id"]: {"case_count": len(case_ids), "case_digest": case_digest, "score": 1.0}
                    for item in candidates
                },
            },
        })
    for gate in ("rights", "disk"):
        if stage in ("promote", "cleanup", "complete") or stage == gate:
            evidence.append({"stage": gate, "receipt_id": "owner-" + gate})
    if stage in ("cleanup", "complete"):
        winner = candidates[0]
        evidence.extend([
            {
                "stage": "promote",
                "receipt_id": "owner-promote",
                "details": {"winner_id": winner["id"], "artifact_digest": acquired_digests[winner["id"]]},
            },
            {
                "stage": "cleanup",
                "receipt_id": "owner-cleanup",
                "details": {"removal_receipts": {item["id"]: "remove-" + item["id"] for item in candidates[1:]}},
            },
        ])
    return {
        "request": request,
        "stage": stage,
        "revision": revision,
        "winner_id": candidates[0]["id"] if stage in ("cleanup", "complete") else "",
        "evidence": evidence,
    }


def role_request(run_id):
    return {
        "run_id": run_id,
        "target": "ollama-role",
        "subject": {"role": "chat.default"},
        "candidates": [
            {"id": "candidate-1", "artifact_digest": "sha256:one"},
            {"id": "candidate-2", "artifact_digest": "sha256:two"},
            {"id": "candidate-3", "artifact_digest": "sha256:three"},
        ],
        "suite_id": "fixed-role-v1",
        "case_ids": [f"case-{index:02d}" for index in range(20)],
        "minimum_cases": 20,
    }


def test_invalid_request_fails_before_owner_call():
    class NeverCalled:
        def submit(self, **kwargs):
            raise AssertionError("submit called for malformed JSON")

        def wait(self, **kwargs):
            raise AssertionError("wait called for malformed JSON")

    result = run_program({"request": {"run_id": ""}}, type("Runtime", (), {"optimization": NeverCalled()})())
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"
    assert result["signals"]["run_id"] == ""


def test_engine_identity_and_size_limits_fail_before_owner_call():
    class NeverCalled:
        def submit(self, **kwargs):
            raise AssertionError("submit called for a non-canonical or oversized request")

        def wait(self, **kwargs):
            raise AssertionError("wait called for a non-canonical or oversized request")

    mutations = {
        "padded suite": lambda request: request.update(suite_id=" fixed-role-v1"),
        "padded case": lambda request: request["case_ids"].__setitem__(0, " case-00"),
        "padded artifact digest": lambda request: request["candidates"][0].update(artifact_digest=" sha256:one"),
        "padded predecessor digest": lambda request: request["candidates"][0].update(predecessor_digest=" sha256:old"),
        "long suite ID": lambda request: request.update(suite_id="s" * 257),
        "long candidate ID": lambda request: request["candidates"][0].update(id="c" * 257),
        "long artifact digest": lambda request: request["candidates"][0].update(artifact_digest="d" * 257),
        "long predecessor ID": lambda request: request.update(predecessor_id="p" * 257),
        "long case id": lambda request: request["case_ids"].__setitem__(0, "x" * 257),
        "multibyte run ID exceeds UTF-8 limit": lambda request: request.update(run_id="é" * 65),
        "multibyte case ID exceeds UTF-8 limit": lambda request: request["case_ids"].__setitem__(0, "é" * 129),
        "too many cases": lambda request: request.update(case_ids=[f"case-{index}" for index in range(2001)]),
        "too many candidates": lambda request: request.update(candidates=request["candidates"] * 6),
        "too many subject fields": lambda request: request.update(subject={f"field-{index}": index for index in range(33)}),
        "long run ID": lambda request: request.update(run_id="r" * 129),
    }
    runtime = type("Runtime", (), {"optimization": NeverCalled()})()
    for name, mutate in mutations.items():
        request = role_request("stable-run-validation")
        mutate(request)
        result = run_program({"request": request, "wait_ms": 1000}, runtime)
        assert result["status"] == "failed", (name, result)
        assert result["errors"][0]["class"] == "invalid_input", (name, result)


def test_wait_budget_over_limit_fails_before_owner_call():
    class NeverCalled:
        def submit(self, **kwargs):
            raise AssertionError("submit called for out-of-range wait budget")

        def wait(self, **kwargs):
            raise AssertionError("wait called for out-of-range wait budget")

    request = role_request("stable-run-over-limit")
    runtime = type("Runtime", (), {"optimization": NeverCalled()})()
    result = run_program({"request": request, "wait_ms": 60001}, runtime)
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"
    assert result["signals"]["run_id"] == ""


def test_wait_budget_rejects_non_integer_types_before_owner_call():
    class NeverCalled:
        def submit(self, **kwargs):
            raise AssertionError("submit called for a non-integer wait budget")

        def wait(self, **kwargs):
            raise AssertionError("wait called for a non-integer wait budget")

    request = role_request("stable-run-invalid-wait-type")
    runtime = type("Runtime", (), {"optimization": NeverCalled()})()
    for value in ("30000", 1.5, True, None):
        result = run_program({"request": request, "wait_ms": value}, runtime)
        assert result["status"] == "failed", (value, result)
        assert result["errors"][0]["class"] == "invalid_input", (value, result)
        assert len(result["errors"]) == 1, (value, result)


def test_role_denominator_rejects_fewer_than_twenty_fixed_cases():
    class NeverCalled:
        def submit(self, **kwargs):
            raise AssertionError("submit called for an undersized role denominator")

        def wait(self, **kwargs):
            raise AssertionError("wait called for an undersized role denominator")

    request = role_request("stable-run-short-denominator")
    request["case_ids"] = request["case_ids"][:-1]
    runtime = type("Runtime", (), {"optimization": NeverCalled()})()
    result = run_program({"request": request, "wait_ms": 1000}, runtime)
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"


def test_future_target_is_delegated_to_registered_owner_boundary():
    request = role_request("stable-run-future-target")
    request["target"] = "retrieval-embedding"
    request["subject"] = {"store_id": "knowledge-base", "recipe_id": "embedding-v2"}

    class Optimization:
        def __init__(self):
            self.submitted = []

        def submit(self, **kwargs):
            self.submitted.append(kwargs)
            raise RuntimeError("target adapter unavailable")

        def wait(self, **kwargs):
            raise AssertionError("wait must not run when the owner rejects submit")

    optimization = Optimization()
    runtime = type("Runtime", (), {"optimization": optimization})()
    result = run_program({"request": request, "wait_ms": 1000}, runtime)

    assert len(optimization.submitted) == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["where"] == "act"


def test_non_default_role_is_delegated_to_authoritative_role_adapter():
    request = role_request("stable-run-specialist-role")
    request["subject"] = {"role": "summarize.default"}

    class Optimization:
        def __init__(self):
            self.submitted = []

        def submit(self, **kwargs):
            self.submitted.append(kwargs)
            raise RuntimeError("owner adapter is not registered")

        def wait(self, **kwargs):
            raise AssertionError("wait must not run when owner registration is absent")

    optimization = Optimization()
    runtime = type("Runtime", (), {"optimization": optimization})()
    result = run_program({"request": request, "wait_ms": 1000}, runtime)

    assert len(optimization.submitted) == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["where"] == "act"


def test_receipt_digest_mismatch_never_reports_success():
    run_id = "stable-run-bad-digest"

    class Optimization:
        def submit(self, **kwargs):
            return {"run_id": run_id, "record_json": json.dumps(make_record(run_id, "acquire", 0))}

        def wait(self, **kwargs):
            record = make_record(run_id, "complete", 7)
            acquisition = next(item for item in record["evidence"] if item["stage"] == "acquire")
            acquisition["details"]["candidate_artifact_digests"]["candidate-1"] = "sha256:wrong"
            return {"record_json": json.dumps(record), "terminal": True, "waited_ms": 10}

    runtime = type("Runtime", (), {"optimization": Optimization()})()
    result = run_program({"request": role_request(run_id), "wait_ms": 1000}, runtime)
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "binding_error", result


def test_unknown_expected_digest_is_resolved_and_bound_by_acquisition_receipt():
    run_id = "stable-run-acquired-digest"
    request = role_request(run_id)
    request["target"] = "future-asset"
    request["subject"] = {"asset": "model-source"}
    for candidate in request["candidates"]:
        candidate.pop("artifact_digest")

    class Optimization:
        def submit(self, **kwargs):
            return {"run_id": run_id, "record_json": json.dumps(make_record(run_id, "acquire", 0, request))}

        def wait(self, **kwargs):
            return {"record_json": json.dumps(make_record(run_id, "complete", 7, request)), "terminal": True, "waited_ms": 10}

    runtime = type("Runtime", (), {"optimization": Optimization()})()
    result = run_program({"request": request, "wait_ms": 1000}, runtime)

    assert result["status"] == "ok", result
    assert result["signals"]["candidate_artifact_digests"] == {
        "candidate-1": "acquired:candidate-1",
        "candidate-2": "acquired:candidate-2",
        "candidate-3": "acquired:candidate-3",
    }
    assert result["signals"]["promoted_artifact_digest"] == "acquired:candidate-1"


def test_submit_once_wait_once_and_report_terminal_receipt():
    run_id = "stable-run-1"

    class Optimization:
        def __init__(self):
            self.submits = []
            self.waits = []

        def submit(self, **kwargs):
            self.submits.append(kwargs)
            return {"run_id": run_id, "record_json": json.dumps(make_record(run_id, "acquire", 0))}

        def wait(self, **kwargs):
            self.waits.append(kwargs)
            return {"record_json": json.dumps(make_record(run_id, "complete", 7)), "terminal": True, "waited_ms": 125}

    optimization = Optimization()
    runtime = type("Runtime", (), {"optimization": optimization})()
    request = role_request(run_id)
    request_json = json.dumps(request, separators=(",", ":"), ensure_ascii=False, allow_nan=False)
    result = run_program({"request": request, "wait_ms": 5000}, runtime)

    assert len(optimization.submits) == 1
    assert len(optimization.waits) == 1
    assert optimization.submits[0] == {"request_json": request_json}
    assert optimization.waits[0] == {"run_id": run_id, "wait_ms": 5000}
    assert result["status"] == "ok", result
    assert result["signals"]["run_id"] == run_id
    assert result["signals"]["state"] == "terminal"
    assert result["signals"]["stage"] == "complete"
    assert result["signals"]["candidate_artifact_digests"] == {"candidate-1": "sha256:one", "candidate-2": "sha256:two", "candidate-3": "sha256:three"}
    assert result["signals"]["candidate_results"]["candidate-1"]["case_count"] == 20
    assert result["signals"]["promoted_artifact_digest"] == "sha256:one"
    assert result["signals"]["removal_receipts"] == {"candidate-2": "remove-candidate-2", "candidate-3": "remove-candidate-3"}
    assert result["evidence"] == ["owner-preflight-rights", "owner-preflight-disk", "owner-acquire", "owner-evaluate", "owner-rights", "owner-disk", "owner-promote", "owner-cleanup"]


def test_submit_pending_is_reported_without_wait_polling():
    run_id = "stable-run-2"

    class Optimization:
        def __init__(self):
            self.submits = 0
            self.waits = 0

        def submit(self, **kwargs):
            self.submits += 1
            return {"run_id": run_id, "record_json": json.dumps(make_record(run_id, "evaluate", 2))}

        def wait(self, **kwargs):
            self.waits += 1
            return {"record_json": json.dumps(make_record(run_id, "evaluate", 2)), "terminal": False, "waited_ms": 1000}

    optimization = Optimization()
    runtime = type("Runtime", (), {"optimization": optimization})()
    result = run_program({"request": role_request(run_id), "wait_ms": 1000}, runtime)
    assert optimization.submits == 1
    assert optimization.waits == 1
    assert result["status"] == "partial"
    assert result["signals"]["state"] == "pending"


def test_wait_failure_preserves_submitted_pending_record():
    run_id = "stable-run-wait-failure"

    class Optimization:
        def __init__(self):
            self.submits = 0
            self.waits = 0

        def submit(self, **kwargs):
            self.submits += 1
            return {"run_id": run_id, "record_json": json.dumps(make_record(run_id, "evaluate", 2))}

        def wait(self, **kwargs):
            self.waits += 1
            raise RuntimeError("temporary wait binding outage")

    optimization = Optimization()
    runtime = type("Runtime", (), {"optimization": optimization})()
    result = run_program({"request": role_request(run_id), "wait_ms": 1000}, runtime)
    assert optimization.submits == 1
    assert optimization.waits == 1
    assert result["status"] == "partial"
    assert result["phase"] == "report"
    assert result["signals"]["state"] == "pending"
    assert result["signals"]["run_id"] == run_id
    assert result["errors"][0]["where"] == "collect"


test_invalid_request_fails_before_owner_call()
test_wait_budget_over_limit_fails_before_owner_call()
test_role_denominator_rejects_fewer_than_twenty_fixed_cases()
test_submit_once_wait_once_and_report_terminal_receipt()
test_submit_pending_is_reported_without_wait_polling()
test_wait_failure_preserves_submitted_pending_record()
test_receipt_digest_mismatch_never_reports_success()

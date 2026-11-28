"""Governed F9 bounty replay. All paths stay inside owner calls or transient work values."""
import json

inputs = program.inputs()
envelope = program.envelope("program-runtime.bounty-replay", "1")
envelope["phase"] = "validate"
envelope["signals"] = {"coding_job_id": None, "case_id": None, "project_id": None,
    "job_state": None, "solver_run_ref": None, "score": None, "receipt_id": None,
    "receipt_digest": None, "disposition": None, "tamper_detected": False,
    "review_packet": None}
envelope["errors"] = []
envelope["evidence"] = []
case_id_input = inputs.get("case_id")
public_url_input = inputs.get("public_url")
key = inputs.get("idempotency_key")
case = {}
work = {}
ROLES = {"source": "REPLAY_SEAL_ROLE_SOURCE", "candidate": "REPLAY_SEAL_ROLE_CANDIDATE", "reference": "REPLAY_SEAL_ROLE_REFERENCE"}
TG_ROLES = {"source": "WORKSPACE_SANDBOX_SEAL_ROLE_SOURCE", "candidate": "WORKSPACE_SANDBOX_SEAL_ROLE_CANDIDATE", "reference": "WORKSPACE_SANDBOX_SEAL_ROLE_REFERENCE"}
VARIANTS = ("FROZEN_REPLAY_VARIANT_PREIMAGE", "FROZEN_REPLAY_VARIANT_CANDIDATE", "FROZEN_REPLAY_VARIANT_REFERENCE")


def sha256_digest(value):
    return value if value.startswith("sha256:") else "sha256:" + value


def is_sha256(value):
    return isinstance(value, str) and len(value) == 71 and value.startswith("sha256:") and all(char in "0123456789abcdef" for char in value[7:])


def rows_one(handle, field, where):
    try:
        rows = handle if isinstance(handle, list) else handle.head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, where)
    if not rows or not isinstance(rows[0], dict) or not isinstance(rows[0].get(field), dict):
        return program.fail("failed", "invalid_response", "owner response omitted " + field, where)
    return rows[0][field]


def step_validate():
    if bool(case_id_input) == bool(public_url_input):
        return program.fail("failed", "invalid_input", "supply exactly one registered case_id or public_url", "validate")
    if not isinstance(key, str) or not key.strip() or len(key) > 128:
        return program.fail("failed", "invalid_input", "idempotency_key must be stable and at most 128 characters", "validate")
    return "resolve_case"


def step_resolve_case():
    """Only the private owner catalog may supply roots, task, evaluator, or plan."""
    envelope["phase"] = "collect"
    request = {"case_id": case_id_input} if case_id_input else {"public_url": public_url_input}
    try:
        resolved = rows_one(client_rehearsal.bounty_cases.resolve(**request).head(1), "case", "case_catalog:resolve")
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "case_catalog:resolve")
    if resolved == "report": return "report"
    required = ("caseId", "project", "title", "task", "sourceProjectRoot", "referenceProjectRoot",
                "frozenEvaluatorDigest", "frozenEvaluatorEntrypoint", "evaluatorBundleId",
                "frozenEvaluatorBundleDigest", "testGeniePlan", "bundleDigest")
    if any(not resolved.get(name) for name in required):
        return program.fail("failed", "invalid_response", "registered case bundle omitted trusted replay fields", "case_catalog:resolve")
    if case_id_input and resolved["caseId"] != case_id_input:
        return program.fail("failed", "case_mismatch", "case catalog returned a different case id", "case_catalog:resolve")
    if public_url_input and resolved.get("publicUrl") != public_url_input:
        return program.fail("failed", "case_mismatch", "case catalog returned a different registered URL", "case_catalog:resolve")
    case.clear()
    case.update(resolved)
    envelope["signals"]["case_id"] = case["caseId"]
    envelope["signals"]["project_id"] = case["project"]
    envelope["evidence"].append("registered-bounty-case:" + case["caseId"] + ":" + case["bundleDigest"])
    return "source_sandbox"


def make_sandbox(suffix, project_root):
    result = rows_one(workspace_sandbox.sandbox.create(
        name="bounty-replay-" + case["caseId"] + "-" + suffix,
        scope_path=".", project_root=project_root, owner="program-runtime:bounty-replay:" + case["caseId"],
        idempotency_key=key + ":" + suffix).head(1), "sandbox", "sandbox:create:" + suffix)
    if isinstance(result, str):
        return result
    sandbox = result
    if not sandbox.get("sandboxId"):
        return program.fail("failed", "invalid_response", "Workspace Sandbox returned no sandbox id", "sandbox:create:" + suffix)
    return sandbox


def seal(sandbox_id, role):
    try:
        rows = workspace_sandbox.replay_seal.create(sandbox_id=sandbox_id, case_id=case["caseId"], role=ROLES[role]).head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "seal:create:" + role)
    data = rows[0] if rows else {}
    if data.get("success") is not True or not data.get("sealDigest"):
        return program.fail("failed", "invalid_response", "Sandbox did not create the " + role + " seal", "seal:create:" + role)
    return data["sealDigest"]


def resolve(role, digest):
    try:
        rows = workspace_sandbox.replay_seal.resolve(case_id=case["caseId"], role=ROLES[role], seal_digest=digest).head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "seal:resolve:" + role)
    data = rows[0] if rows else {}
    if data.get("success") is not True or data.get("sealDigest") != digest or not data.get("workspaceRoot"):
        return program.fail("failed", "invalid_response", "Sandbox returned an incomplete or different " + role + " seal", "seal:resolve:" + role)
    return data["workspaceRoot"]


def verify_all(where):
    for role, digest in (("source", work["source_digest"]), ("candidate", work.get("candidate_digest")), ("reference", work["reference_digest"])):
        if not digest:
            continue
        try:
            rows = workspace_sandbox.replay_seal.verify(case_id=case["caseId"], role=ROLES[role], seal_digest=digest).head(1)
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, where + ":" + role)
        data = rows[0] if rows else {}
        if data.get("success") is not True:
            return program.fail("failed", "invalid_response", "Sandbox omitted seal verification", where + ":" + role)
        if data.get("tamperDetected") is True or data.get("intact") is not True:
            envelope["signals"]["tamper_detected"] = data.get("tamperDetected") is True
            return program.fail("failed", "receipt_mismatch", "Workspace Sandbox detected replay seal tampering", where + ":" + role)
        if data.get("caseId") != case["caseId"] or data.get("role") != ROLES[role] or data.get("sealDigest") != digest:
            return program.fail("failed", "receipt_mismatch", "Sandbox verified a different seal identity", where + ":" + role)
        envelope["evidence"].append("workspace-sandbox-seal:" + role + ":" + digest)
    return None


def step_source_sandbox():
    envelope["phase"] = "collect"
    source = make_sandbox("source", case["sourceProjectRoot"])
    if isinstance(source, str): return "report"
    work["source_sandbox_id"] = source["sandboxId"]
    source_digest = seal(source["sandboxId"], "source")
    if source_digest == "report" or not isinstance(source_digest, str) or len(source_digest) != 64: return "report"
    work["source_digest"] = source_digest
    reference = make_sandbox("reference", case["referenceProjectRoot"])
    if isinstance(reference, str): return "report"
    work["reference_sandbox_id"] = reference["sandboxId"]
    reference_digest = seal(reference["sandboxId"], "reference")
    if reference_digest == "report" or not isinstance(reference_digest, str) or len(reference_digest) != 64: return "report"
    work["reference_digest"] = reference_digest
    envelope["evidence"].extend(["source-seal:" + source_digest, "reference-seal:" + reference_digest])
    return "candidate_sandbox"


def step_candidate_sandbox():
    envelope["phase"] = "collect"
    source_root = resolve("source", work["source_digest"])
    if source_root == "report": return "report"
    candidate = make_sandbox("candidate", source_root)
    if candidate == "report": return "report"
    work["candidate_sandbox_id"] = candidate["sandboxId"]
    work["candidate_root"] = candidate.get("workspaceRoot")
    if not isinstance(work["candidate_root"], str) or not work["candidate_root"]:
        return program.fail("failed", "invalid_response", "candidate sandbox has no workspace root", "candidate_sandbox")
    return "coding_job"


def step_coding_job():
    envelope["phase"] = "act"
    try:
        job = rows_one(client_rehearsal.coding_jobs.create(
            replay_key=key + ":job", case_id=case["caseId"], project=case["project"], title=case["title"], task=case["task"],
            source_seal_ref=work["source_digest"]).head(1), "job", "coding_job:create")
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "coding_job:create")
    if isinstance(job, str): return "report"
    if job.get("caseId") != case["caseId"] or job.get("project") != case["project"] or job.get("sourceSealRef") != work["source_digest"]:
        return program.fail("failed", "case_mismatch", "coding job differs from the sealed case record", "coding_job")
    work["job"] = job
    envelope["signals"]["coding_job_id"] = job["id"]
    envelope["signals"]["job_state"] = job.get("state")
    if job.get("evaluatorReceiptRef"):
        work["receipt_ref"] = job["evaluatorReceiptRef"]
        work["receipt_digest"] = job.get("evaluatorReceiptSha256")
        return "review_packet"
    if job.get("solverRunRef"):
        work["solver_run_ref"] = job["solverRunRef"]
        envelope["signals"]["solver_run_ref"] = job["solverRunRef"]
    if job.get("state") == "CODING_JOB_STATE_SCORING":
        # The solver_completed event atomically persisted the candidate seal
        # digest. Never mint a new seal during recovery: that could bless a
        # changed tree after interruption.
        candidate_digest = ""
        for event in job.get("events", []):
            if event.get("stage") == "solver_completed":
                candidate_digest = event.get("evidenceSha256", "")
        if len(candidate_digest) != 64:
            return program.fail("partial", "invalid_response", "scoring job has no persisted candidate seal digest", "resume:solver_completed")
        work["candidate_digest"] = candidate_digest
        return "test_genie"
    return "solve"


def step_solve():
    envelope["phase"] = "delegate"
    job = work["job"]
    if job.get("state") == "CODING_JOB_STATE_READY":
        try:
            row = rows_one(client_rehearsal.coding_jobs.record_stage(
                id=job["id"], stage="solver_started", actor="agent-manager-solver",
                evidence_ref="case:" + case["caseId"] + ":" + key, expected_revision=job["revision"],
                note="blind bounty solver dispatched to the sealed candidate checkout").head(1), "job", "solver:start")
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, "solver:start")
        if isinstance(row, str): return "report"
        work["job"] = row
    elif job.get("state") != "CODING_JOB_STATE_SOLVING":
        return program.fail("failed", "wrong_job_state", "coding job cannot resume solver from its current state", "solver")
    envelope["signals"]["job_state"] = work["job"].get("state")
    try:
        handle = agent.start(owner="agent-manager", workflow_key="agent-manager/bounty-solver", idempotency_key=key + ":solver",
            input={"caseId": case["caseId"], "task": case["task"], "checkout": work["candidate_root"]})
    except Exception as exc:
        status, klass = program.classify(exc)
        # Keep the durable job in SOLVING for a same-profile resume. Never select
        # a generic workflow as fallback when the approved profile is absent.
        return program.fail("unavailable", "workflow_rejected" if klass == "binding_error" else klass, exc, "solver:start")
    try:
        rows = agent.collect(handle, wait_seconds=300).head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail("partial", klass, exc, "solver:collect")
    result = rows[0] if rows else {}
    run_ref = result.get("execution_id")
    solver_result = result.get("result")
    if not run_ref or not isinstance(solver_result, dict) or solver_result.get("outcome") not in ("solved", "blocked", "incomplete"):
        return program.fail("failed", "invalid_response", "Agent Manager solver returned no typed run result", "solver:collect")
    work["solver_run_ref"] = run_ref
    envelope["signals"]["solver_run_ref"] = run_ref
    candidate_digest = seal(work["candidate_sandbox_id"], "candidate")
    if candidate_digest == "report" or not isinstance(candidate_digest, str) or len(candidate_digest) != 64: return "report"
    work["candidate_digest"] = candidate_digest
    # The immutable candidate tree digest is the solver completion evidence.
    try:
        job = rows_one(client_rehearsal.coding_jobs.record_stage(
            id=work["job"]["id"], stage="solver_completed", actor="agent-manager-solver",
            evidence_ref=run_ref, evidence_sha256=candidate_digest, expected_revision=work["job"]["revision"],
            note="solver run completed; candidate state sealed").head(1), "job", "solver:complete")
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "solver:complete")
    if isinstance(job, str): return "report"
    work["job"] = job
    return "test_genie"


def outcome_from_validation(receipt, index):
    if receipt.get("state") == "RECEIPT_STATE_SUCCEEDED": status = "FROZEN_REPLAY_TEST_STATUS_PASSED"
    elif receipt.get("state") == "RECEIPT_STATE_FAILED": status = "FROZEN_REPLAY_TEST_STATUS_FAILED"
    else: return None
    run_id = ""
    for child in receipt.get("children", []):
        if child.get("kind") == "CHILD_OPERATION_KIND_TEST_RUN" and child.get("owner") == "test-genie":
            run_id = child.get("operationId")
            break
    identity = receipt.get("observedIdentity") if status.endswith("PASSED") else receipt.get("admittedIdentity")
    identity = identity or {}
    if not receipt.get("receiptId") or not run_id or not identity.get("identity"):
        return None
    expected_identity = work["expected_identities"][VARIANTS[index]]
    if identity.get("identity") != expected_identity.get("identity"):
        return None
    return {"variant": VARIANTS[index], "runId": run_id, "producerReceiptId": receipt["receiptId"],
        "sourceIdentity": expected_identity["identity"], "evaluatorDigest": sha256_digest(case["frozenEvaluatorDigest"]),
        "status": status}


def step_test_genie():
    envelope["phase"] = "collect"
    failed = verify_all("seal:before_test_genie")
    if failed: return "report"
    outcomes = []
    work["expected_identities"] = {}
    for index, variant in enumerate(VARIANTS):
        role = ("source", "candidate", "reference")[index]
        seal_digest = (work["source_digest"], work["candidate_digest"], work["reference_digest"])[index]
        content_inputs = [
            {"name": "bounty-" + role, "workspaceSandboxSeal": {"caseId": case["caseId"], "role": TG_ROLES[role], "sealDigest": seal_digest}},
            {"name": "frozen-evaluator", "dependency": True, "frozenEvaluatorBundle": {
                "evaluatorBundleId": case["evaluatorBundleId"], "evaluatorDigest": case["frozenEvaluatorDigest"],
                "entryPoint": case["frozenEvaluatorEntrypoint"]}},
        ]
        target = {"kind": "VALIDATION_TARGET_KIND_PROJECT", "id": case["caseId"]}
        try:
            identity_rows = test_genie.validation.resolve_identity(targets=[target], content_inputs=content_inputs).head(1)
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, "validation:resolve_identity:" + role)
        identity = identity_rows[0].get("identity") if identity_rows else None
        if not isinstance(identity, dict) or not identity.get("identity"):
            return program.fail("failed", "invalid_response", "Test Genie returned no owner-resolved source identity", "validation:resolve_identity:" + role)
        evaluator_root = next((root for root in identity.get("roots", []) if root.get("name") == "frozen-evaluator"), {})
        if evaluator_root.get("identity") != sha256_digest(case["frozenEvaluatorBundleDigest"]):
            return program.fail("failed", "evaluator_mismatch", "resolved Test Genie evaluator bundle identity differs from the registered case bundle", "validation:resolve_identity:" + role)
        entrypoint_file = next((item for item in evaluator_root.get("files", []) if item.get("path") == case["frozenEvaluatorEntrypoint"]), {})
        if entrypoint_file.get("digest") != case["frozenEvaluatorDigest"]:
            return program.fail("failed", "evaluator_mismatch", "resolved Test Genie identity does not contain the registered frozen evaluator", "validation:resolve_identity:" + role)
        if identity["identity"] != sha256_digest(seal_digest):
            return program.fail("failed", "receipt_mismatch", "Test Genie source identity is not the exact Workspace Sandbox seal identity", "validation:resolve_identity:" + role)
        work["expected_identities"][variant] = identity
        plan = case["testGeniePlan"]
        execution = {"caseId": case["caseId"], "role": TG_ROLES[role], "sealDigest": seal_digest,
            "evaluatorBundleId": case["evaluatorBundleId"], "evaluatorDigest": case["frozenEvaluatorDigest"],
            "evaluatorEntryPoint": case["frozenEvaluatorEntrypoint"]}
        deadline = plan["deadlinePolicy"]
        deadline_policy = {"queueBudget": str(float(deadline["queueBudgetMilliseconds"]) / 1000) + "s",
            "executionBudget": str(float(deadline["executionBudgetMilliseconds"]) / 1000) + "s",
            "maximumAttempts": deadline["maximumAttempts"]}
        intent = {"schemaVersion": 1, "intentId": key + ":" + role, "idempotencyKey": key + ":validation:" + role,
            "callerScenario": "program-runtime", "callerExecutionId": key, "planId": "bounty-replay:" + case["caseId"],
            "phaseId": role, "targets": [target], "purpose": plan["validationPurpose"],
            "requiredStrength": plan["requiredStrength"], "reusePolicy": {"mode": "REUSE_MODE_NEVER"},
            "concurrencyPolicy": {"mode": "CONCURRENCY_MODE_EXCLUSIVE", "maximumParallelism": 1},
            "expectedIdentity": identity, "evidencePolicy": plan["evidencePolicy"],
            "deadlinePolicy": deadline_policy, "contentInputs": content_inputs,
            "phases": plan["phases"], "frozenReplayExecution": execution}
        try:
            created = rows_one(test_genie.validation.create(intent=intent).head(1), "receipt", "validation:create:" + str(index))
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, "validation:create:" + str(index))
        if isinstance(created, str): return "report"
        receipt_id = created.get("receiptId")
        if not receipt_id: return program.fail("failed", "invalid_response", "Test Genie create returned no receipt id", "validation:create")
        try:
            waited = rows_one(test_genie.validation.wait(receipt_id=receipt_id, wait_id=key + ":wait:" + str(index), timeout="1800s").head(1), "receipt", "validation:wait:" + str(index))
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, "validation:wait:" + str(index))
        if isinstance(waited, str): return "report"
        evidence = outcome_from_validation(waited, index)
        if evidence is None: return program.fail("partial", "invalid_response", "validation receipt has not reached a linked terminal Test Genie run", "validation:wait")
        outcomes.append(evidence)
        envelope["evidence"].append("test-genie-validation:" + receipt_id + ":" + evidence["runId"])
    source_identity = sha256_digest(work["source_digest"])
    candidate_seal_digest = sha256_digest(work["candidate_digest"])
    reference_identity = sha256_digest(work["reference_digest"])
    if outcomes[0]["sourceIdentity"] != source_identity or outcomes[1]["sourceIdentity"] != candidate_seal_digest or outcomes[2]["sourceIdentity"] != reference_identity:
        return program.fail("partial", "receipt_mismatch", "Test Genie variant identity does not match its exact Workspace Sandbox seal", "frozen_replay:source_identity")
    adjudication = {"caseId": case["caseId"], "projectId": case["project"],
        "frozenEvaluatorDigest": sha256_digest(case["frozenEvaluatorDigest"]),
        "preimageIdentity": outcomes[0]["sourceIdentity"], "candidateSealDigest": candidate_seal_digest,
        "referenceIdentity": outcomes[2]["sourceIdentity"], "evidence": outcomes}
    if outcomes[0]["status"] == "FROZEN_REPLAY_TEST_STATUS_PASSED" or outcomes[1]["status"] != "FROZEN_REPLAY_TEST_STATUS_PASSED" or outcomes[2]["status"] != "FROZEN_REPLAY_TEST_STATUS_PASSED":
        expected_disposition = "FROZEN_REPLAY_DISPOSITION_NOT_QUALIFIED"
    else: expected_disposition = "FROZEN_REPLAY_DISPOSITION_QUALIFIED"
    try:
        created = rows_one(test_genie.frozen_replay.create(idempotency_key=key + ":frozen-replay", adjudication=adjudication).head(1), "receipt", "frozen_replay:create")
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "frozen_replay:create")
    if isinstance(created, str): return "report"
    receipt_id = created.get("receiptId")
    if not receipt_id:
        return program.fail("failed", "invalid_response", "Test Genie create returned no immutable receipt id", "frozen_replay:create")
    try:
        persisted = rows_one(test_genie.frozen_replay.get(receipt_id=receipt_id).head(1), "receipt", "frozen_replay:get_after_create")
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "frozen_replay:get_after_create")
    if persisted == "report": return "report"
    # Get is the persisted owner record. Require create and get to agree on the
    # receipt digest and every evidence item, including owner-derived artifacts.
    if (created.get("receiptId") != persisted.get("receiptId") or
            created.get("receiptDigest") != persisted.get("receiptDigest") or
            created.get("adjudication") != persisted.get("adjudication") or
            created.get("disposition") != persisted.get("disposition")):
        return program.fail("partial", "receipt_mismatch", "Test Genie create and persisted get returned different immutable receipt identities", "frozen_replay:get_after_create")
    receipt = persisted
    mismatch = verify_receipt(receipt, adjudication, expected_disposition)
    if mismatch: return program.fail("partial", "receipt_mismatch", mismatch, "frozen_replay:verify")
    work["receipt"] = receipt
    return "score_model"


def step_score_model():
    """Run the separate no-tools scorer on receipt facts; it cannot override Test Genie."""
    envelope["phase"] = "delegate"
    receipt = work["receipt"]
    job = work["job"]
    scoring_marker = "test-genie-receipt:" + receipt["receiptId"]
    if not any(event.get("stage") == "scoring_started" and event.get("evidenceRef") == scoring_marker for event in job.get("events", [])):
        try:
            staged = rows_one(client_rehearsal.coding_jobs.record_stage(
                id=job["id"], stage="scoring_started", actor="agent-manager-scorer",
                evidence_ref=scoring_marker, expected_revision=job["revision"],
                note="immutable Test Genie receipt verified; independent scorer may review receipt facts").head(1),
                "job", "scorer:stage")
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, "scorer:stage")
        if staged == "report": return "report"
        job = staged
        work["job"] = job
    actual = receipt["adjudication"]
    evidence = {item["variant"]: item for item in actual["evidence"]}
    score_input = {
        "caseId": actual["caseId"], "solverRunId": work["solver_run_ref"],
        "frozenEvaluatorDigest": sha256_digest(actual["frozenEvaluatorDigest"]),
        "preimageEvaluatorDigest": sha256_digest(evidence[VARIANTS[0]]["evaluatorDigest"]),
        "candidateEvaluatorDigest": sha256_digest(evidence[VARIANTS[1]]["evaluatorDigest"]),
        "referenceEvaluatorDigest": sha256_digest(evidence[VARIANTS[2]]["evaluatorDigest"]),
        "preimageReceiptDigest": sha256_digest(evidence[VARIANTS[0]]["artifactDigest"]),
        "candidateReceiptDigest": sha256_digest(evidence[VARIANTS[1]]["artifactDigest"]),
        "referenceReceiptDigest": sha256_digest(evidence[VARIANTS[2]]["artifactDigest"]),
        "preimagePassed": evidence[VARIANTS[0]]["status"] == "FROZEN_REPLAY_TEST_STATUS_PASSED",
        "candidatePassed": evidence[VARIANTS[1]]["status"] == "FROZEN_REPLAY_TEST_STATUS_PASSED",
        "referencePassed": evidence[VARIANTS[2]]["status"] == "FROZEN_REPLAY_TEST_STATUS_PASSED",
    }
    try:
        handle = agent.start(owner="agent-manager", workflow_key="agent-manager/bounty-scorer",
            idempotency_key=key + ":scorer", input=score_input)
        rows = agent.collect(handle, wait_seconds=300).head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        work["scorer"] = {"status": "unavailable", "error": str(exc)[:240]}
        envelope["errors"].append({"class": klass, "detail": str(exc)[:240], "where": "scorer"})
        envelope["status"] = "partial"
        return "persist_score"
    result = rows[0].get("result") if rows else None
    if not isinstance(result, dict) or result.get("verdict") not in ("qualifies", "does_not_qualify", "incomplete"):
        work["scorer"] = {"status": "partial", "error": "Agent Manager scorer returned no typed verdict"}
        envelope["status"] = "partial"
    else:
        expected = "qualifies" if receipt["disposition"] == "FROZEN_REPLAY_DISPOSITION_QUALIFIED" else "does_not_qualify"
        work["scorer"] = {"status": "ok", "verdict": result["verdict"], "reason": result["reason"], "consistent": result["verdict"] == expected}
        if result["verdict"] != expected:
            envelope["status"] = "partial"
            envelope["errors"].append({"class": "receipt_mismatch", "detail": "independent scorer verdict differs from Test Genie authority", "where": "scorer"})
    return "persist_score"


def verify_receipt(receipt, requested, disposition):
    if not receipt.get("receiptId") or not is_sha256(receipt.get("receiptDigest")):
        return "immutable Test Genie receipt identity is incomplete"
    actual = receipt.get("adjudication")
    if not isinstance(actual, dict): return "immutable receipt has no adjudication"
    for field in ("caseId", "projectId", "frozenEvaluatorDigest", "preimageIdentity", "candidateSealDigest", "referenceIdentity"):
        if actual.get(field) != requested.get(field): return "receipt identity mismatch: " + field
    expected = {item["variant"]: item for item in requested["evidence"]}
    returned = actual.get("evidence")
    if not isinstance(returned, list) or len(returned) != 3: return "receipt does not have three evidence records"
    seen = set()
    for item in returned:
        if not isinstance(item, dict) or item.get("variant") not in expected or item["variant"] in seen:
            return "receipt has missing, duplicate, or unexpected evidence variants"
        seen.add(item["variant"])
        source = expected[item["variant"]]
        for field in ("runId", "producerReceiptId", "sourceIdentity", "evaluatorDigest", "status"):
            if item.get(field) != source.get(field): return "receipt evidence mismatch: " + field
        # Test Genie derives artifactDigest from the immutable producer receipt;
        # because the create request intentionally omits it, require the owner
        # returned digest and carry it only from the immutable response.
        if not is_sha256(item.get("artifactDigest")): return "receipt omitted a valid owner-derived artifact digest"
    if receipt.get("disposition") != disposition: return "derived Test Genie disposition mismatch"
    return None


def step_persist_score():
    envelope["phase"] = "act"
    failed = verify_all("seal:before_score_persist")
    if failed: return "report"
    receipt = work["receipt"]
    verified = receipt["adjudication"]
    # Derive this label exclusively from verified immutable receipt evidence.
    outcomes = {item["variant"]: item["status"] for item in verified["evidence"]}
    if outcomes[VARIANTS[0]] == "FROZEN_REPLAY_TEST_STATUS_PASSED": score = "preimage_not_red"
    elif outcomes[VARIANTS[1]] == "FROZEN_REPLAY_TEST_STATUS_FAILED": score = "candidate_failed"
    elif outcomes[VARIANTS[2]] == "FROZEN_REPLAY_TEST_STATUS_FAILED": score = "reference_failed"
    elif receipt["disposition"] == "FROZEN_REPLAY_DISPOSITION_QUALIFIED": score = "strict_pass"
    else: score = "invalid"
    job = work["job"]
    try:
        persisted = rows_one(client_rehearsal.coding_jobs.record_evaluation(
            id=job["id"], receipt_ref=receipt["receiptId"],
            receipt_sha256=receipt["receiptDigest"].removeprefix("sha256:"), score=score,
            expected_revision=job["revision"]).head(1), "job", "coding_job:evaluate")
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, "revision_conflict" if klass == "remote_error" and "revision" in str(exc).lower() else klass, exc, "coding_job:evaluate")
    if isinstance(persisted, str): return "report"
    if persisted.get("evaluatorReceiptRef") != receipt["receiptId"] or persisted.get("evaluatorReceiptSha256") != receipt["receiptDigest"].removeprefix("sha256:") or persisted.get("score") != score:
        return program.fail("partial", "receipt_mismatch", "coding-job evaluation differs from immutable Test Genie receipt", "coding_job:evaluate")
    work["job"] = persisted
    work["receipt_ref"] = receipt["receiptId"]
    work["receipt_digest"] = receipt["receiptDigest"]
    envelope["signals"]["score"] = score
    envelope["signals"]["job_state"] = persisted.get("state")
    return "review_packet"


def step_review_packet():
    envelope["phase"] = "report"
    try:
        receipt = rows_one(test_genie.frozen_replay.get(receipt_id=work["receipt_ref"]).head(1), "receipt", "frozen_replay:get")
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "frozen_replay:get")
    if isinstance(receipt, str): return "report"
    if receipt.get("receiptId") != work["receipt_ref"] or receipt.get("receiptDigest") != work["receipt_digest"]:
        return program.fail("partial", "receipt_mismatch", "reloaded immutable receipt digest changed", "review_packet")
    job = work["job"]
    if job.get("evaluatorReceiptRef") != receipt["receiptId"] or job.get("evaluatorReceiptSha256") != receipt["receiptDigest"].removeprefix("sha256:"):
        return program.fail("partial", "receipt_mismatch", "coding-job receipt link differs from immutable receipt", "review_packet")
    adjudicated = receipt["adjudication"]
    envelope["signals"].update({"coding_job_id": job["id"], "case_id": adjudicated["caseId"], "project_id": adjudicated["projectId"],
        "job_state": job["state"], "score": job["score"], "receipt_id": receipt["receiptId"],
        "receipt_digest": receipt["receiptDigest"], "disposition": receipt["disposition"]})
    envelope["signals"]["review_packet"] = {"coding_job_id": job["id"], "case_id": adjudicated["caseId"],
        "project_id": adjudicated["projectId"], "job_state": job["state"], "score": job["score"],
        "public_url": case.get("publicUrl"), "repository": case.get("repository"),
        "issue_number": case.get("issueNumber"), "pull_request_url": case.get("pullRequestUrl"),
        "pull_request_number": case.get("pullRequestNumber"), "source_commit": case.get("sourceCommit"),
        "reference_commit": case.get("referenceCommit"), "case_bundle_digest": case.get("bundleDigest"),
        "frozen_evaluator_bundle_digest": case.get("frozenEvaluatorBundleDigest"),
        "solver_run_ref": job.get("solverRunRef"), "receipt_id": receipt["receiptId"],
        "receipt_digest": receipt["receiptDigest"], "disposition": receipt["disposition"],
        "preimage_identity": adjudicated["preimageIdentity"], "candidate_seal_digest": adjudicated["candidateSealDigest"],
        "reference_identity": adjudicated["referenceIdentity"], "frozen_evaluator_digest": adjudicated["frozenEvaluatorDigest"],
        "independent_scorer": work.get("scorer", {"status": "unavailable", "reason": "no scorer run was attached to this resumed job"})}
    if envelope["status"] != "partial": envelope["status"] = "ok"
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


STATES = {"validate": step_validate, "resolve_case": step_resolve_case,
    "source_sandbox": step_source_sandbox,
    "candidate_sandbox": step_candidate_sandbox, "coding_job": step_coding_job,
    "solve": step_solve, "test_genie": step_test_genie, "persist_score": step_persist_score,
    "score_model": step_score_model, "review_packet": step_review_packet, "report": step_report}
program.run(STATES, "validate")

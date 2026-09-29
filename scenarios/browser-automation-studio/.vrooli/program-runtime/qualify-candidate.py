"""Pinned BAS qualification composition over the declared WSS, TG and BAS owners."""
import json
import re

inputs = program.inputs()
candidate = inputs.get("candidate")
request_key = inputs.get("request_key")
expected_identity = None
producers = inputs.get("invalidated_producers", [])
required_rows = inputs.get("required_rows")
retained_input = inputs.get("retained_evidence_sets")
retained_sets = []

EFFECTS = {"promotion": 0, "refresh": 0, "producer_admissions": 0,
           "producer_waits": 0, "validation_admissions": 0,
           "validation_waits": 0, "setpoint_reads": 0}
envelope = {
    "program": "browser-automation-studio.qualify-candidate", "version": "1",
    "status": "failed", "phase": "validate",
    "signals": {"candidate_sha256": None,
                "owner_handles": {"review_attempt_id": None, "wss_review_request_id": None,
                                  "verdict_sha256": None, "reviewer_run_id": None,
                                  "source_run_id": None, "producer_receipts": [],
                                  "validation_receipt_id": None},
                "qualification": None, "effects": EFFECTS},
    "usage": None, "errors": [], "evidence": []}
program.attach(envelope)


def step_validate():
    required = ("reviewAttemptId", "verdictSha256", "reviewerRunId", "sourceRunId",
                "sandboxId", "reviewRequestId", "sha256")
    if not isinstance(candidate, dict) or any(not isinstance(candidate.get(k), str) or not candidate[k].strip() for k in required):
        return program.fail("failed", "invalid_input", "candidate must carry the accepted AM review identifiers and hashes", "validate")
    if any(not re.fullmatch(r"[0-9a-f]{64}", candidate[k]) for k in ("verdictSha256", "sha256")):
        return program.fail("failed", "invalid_input", "candidate verdictSha256 and sha256 must be bare 64-character hex digests", "validate")
    if not isinstance(request_key, str) or not re.fullmatch(r"[A-Za-z0-9._:/-]{1,51}", request_key):
        return program.fail("failed", "invalid_input", "request_key must be stable and at most 51 safe characters", "validate")
    supported = {"evidence-completeness"}
    if (not isinstance(producers, list) or len(producers) > 8
            or any(not isinstance(name, str) or name not in supported for name in producers)
            or len(set(producers)) != len(producers)):
        return program.fail("failed", "invalid_input", "invalidated_producers must select unique supported producers", "validate")
    product_rows = {"preservation", "interactive-feedback", "motion", "readiness", "capture", "passive-fidelity", "profile-durability", "known-flow-reliability", "cancellation-recovery", "resource-budget", "soak-stability", "evidence-completeness", "desktop-portability", "agent-usefulness", "structural-debt", "workspace-usability", "adversarial-review"}
    if (not isinstance(required_rows, list) or not required_rows or len(required_rows) > 17
            or any(not isinstance(row, str) or row not in product_rows for row in required_rows)
            or len(set(required_rows)) != len(required_rows)):
        return program.fail("failed", "invalid_input", "required_rows must be a nonempty unique BAS row selection", "validate")
    if not isinstance(retained_input, list) or len(retained_input) > 8:
        return program.fail("failed", "invalid_input", "retained_evidence_sets must be a bounded list", "validate")
    seen = set()
    retained_bytes = 0
    for evidence_set in retained_input:
        if (not isinstance(evidence_set, dict) or not isinstance(evidence_set.get("producer"), str)
                or evidence_set.get("producer") not in supported
                or evidence_set.get("target") != "browser-automation-studio"
                or not isinstance(evidence_set.get("candidateIdentity"), str)
                or not evidence_set.get("candidateIdentity")
                or not all(isinstance(evidence_set.get(k), str) and evidence_set[k] for k in
                           ("producerReceiptId", "runId", "catalogDigest"))
                or not isinstance(evidence_set.get("artifacts"), list)):
            return program.fail("failed", "invalid_input", "retained evidence set is incomplete or identity-mismatched", "validate")
        artifacts = evidence_set["artifacts"]
        if not artifacts:
            return program.fail("failed", "invalid_input", "retained evidence set has no artifact references", "validate")
        artifact_ids = set()
        for artifact in artifacts:
            if (not isinstance(artifact, dict)
                    or any(not isinstance(artifact.get(key), str) or not artifact[key].strip()
                           for key in ("producer", "artifactId", "kind", "checksum"))
                    or artifact.get("producer") != "browser-automation-studio"
                    or artifact.get("kind") not in ("generic.file", "command.output")
                    or not re.fullmatch(r"[0-9a-f]{64}", artifact.get("checksum", ""))
                    or artifact.get("artifactId") in artifact_ids):
                return program.fail("failed", "invalid_input", "retained evidence contains an invalid, duplicate or checksumless reference", "validate")
            size = artifact.get("sizeBytes")
            if (isinstance(size, bool) or not isinstance(size, (int, str)) or not str(size).isdigit()
                    or int(size) < 0):
                return program.fail("failed", "invalid_input", "retained evidence reference has invalid sizeBytes", "validate")
            artifact_ids.add(artifact["artifactId"])
            retained_bytes += int(size)
            if retained_bytes > 16777216:
                return program.fail("failed", "invalid_input", "retained evidence exceeds 16 MiB", "validate")
        producer = evidence_set["producer"]
        if producer in seen:
            return program.fail("failed", "invalid_input", "duplicate retained evidence producer", "validate")
        seen.add(producer)
    if sum(len(value["artifacts"]) for value in retained_input) > 32:
        return program.fail("failed", "invalid_input", "retained evidence sets exceed the bounded artifact selection", "validate")
    missing = supported - set(producers) - seen
    if missing:
        return program.fail("failed", "invalid_input", "required row has no retained or selected evidence cohort", "validate")
    if set(producers).intersection(seen):
        return program.fail("failed", "invalid_input", "invalidated producer must replace, not duplicate, its retained cohort", "validate")
    envelope["signals"]["candidate_sha256"] = candidate["sha256"]
    envelope["signals"]["required_rows"] = required_rows
    envelope["signals"]["owner_handles"]["review_attempt_id"] = candidate["reviewAttemptId"]
    envelope["signals"]["owner_handles"]["wss_review_request_id"] = candidate["reviewRequestId"]
    envelope["signals"]["owner_handles"]["verdict_sha256"] = candidate["verdictSha256"]
    envelope["signals"]["owner_handles"]["reviewer_run_id"] = candidate["reviewerRunId"]
    envelope["signals"]["owner_handles"]["source_run_id"] = candidate["sourceRunId"]
    envelope["evidence"].append("workspace-sandbox:review:" + candidate["reviewRequestId"])
    return "act"


def step_act():
    """Promote only the exact accepted WSS review request and SHA."""
    global expected_identity
    envelope["phase"] = "act"
    EFFECTS["promotion"] = None
    try:
        promoted = workspace_sandbox.change.promote(
            sandbox_id=candidate["sandboxId"], mode="all", actor="agent-manager",
            confirm=True, _confirm=True, create_commit=False, force=False,
            override_acceptance=False, review_request_id=candidate["reviewRequestId"],
            expected_review_sha256=candidate["sha256"]).meta()
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "act")
    if (promoted.get("success") is not True or promoted.get("commitHash")
            or promoted.get("sandboxId") != candidate["sandboxId"]
            or promoted.get("failed", 0) not in (0, "0")
            or promoted.get("remaining", 0) not in (0, "0")
            or promoted.get("isPartial") is True):
        return program.fail("failed", "owner_failed", "WSS did not promote the exact candidate without a commit", "act")
    EFFECTS["promotion"] = 1
    envelope["evidence"].append("workspace-sandbox:promoted:" + candidate["reviewRequestId"])
    try:
        resolved = test_genie.validation.resolve_identity(
            targets=[{"kind": "VALIDATION_TARGET_KIND_SCENARIO", "id": "browser-automation-studio"}],
            content_inputs=[{"name": "candidate", "root": "scenarios/browser-automation-studio",
                             "selections": [{"glob": "**", "required": True}]}]).meta()
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "identity")
    identity = resolved.get("identity") if isinstance(resolved, dict) else None
    if (not isinstance(identity, dict) or not isinstance(identity.get("identity"), str)
            or not identity["identity"]):
        return program.fail("failed", "owner_failed", "TG source identity resolver returned no identity", "identity")
    expected_identity = identity
    envelope["signals"]["source_identity"] = identity["identity"]
    envelope["evidence"].append("test-genie:source-identity:" + identity["identity"])
    for evidence_set in retained_input:
        if evidence_set["candidateIdentity"] != identity["identity"]:
            return program.fail("failed", "owner_failed", "retained evidence candidate identity differs from the promoted source", "identity")
    return "collect"


def step_collect():
    """Refresh only when this BAS lifecycle freshness verdict proves it is stale."""
    envelope["phase"] = "collect"
    try:
        freshness = vrooli.scenario.freshness(name="browser-automation-studio").raw()
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "collect")
    checks = freshness.get("checks") or []
    if freshness.get("success") is not True or not checks:
        return program.fail("unavailable", "binding_error", "managed BAS freshness is unavailable", "collect")
    stale = freshness.get("stale") is True or any(check.get("stale") is True for check in checks)
    return "refresh" if stale else "producers"


def step_refresh():
    """Use the managed lifecycle's forced start to rebuild stale artifacts once."""
    envelope["phase"] = "act"
    EFFECTS["refresh"] = None
    try:
        refreshed = vrooli.scenario.start(name="browser-automation-studio", force=True, _confirm=True).raw()
        if refreshed.get("success") is not True:
            return program.fail("failed", "owner_failed", "managed BAS artifact refresh failed", "refresh")
        EFFECTS["refresh"] = 1
        freshness = vrooli.scenario.freshness(name="browser-automation-studio").raw()
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "refresh")
    checks = freshness.get("checks") or []
    if (freshness.get("success") is not True or not checks or freshness.get("stale") is True
            or any(check.get("stale") is True for check in checks)):
        return program.fail("failed", "owner_failed", "managed BAS artifacts remain stale after one forced refresh", "refresh")
    return "producers"


def step_producers():
    """Admit each selected producer once, then wait once for its exact receipt."""
    envelope["phase"] = "act"
    retained_sets[:] = [value for value in retained_input if value["producer"] not in producers]
    artifact_count = 0
    for producer in producers:
        key = request_key + "/evidence/" + producer + "/v1"
        previous = EFFECTS["producer_admissions"]
        EFFECTS["producer_admissions"] = None
        try:
            admission = test_genie.validation.produce_evidence(
                idempotency_key=key, provider="browser-automation-studio", producer=producer,
                candidate_scenario="browser-automation-studio", expected_candidate_identity=expected_identity,
                caller_scenario="browser-automation-studio", caller_execution_id=request_key,
                _confirm=True).meta()
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, "producers")
        receipt = admission.get("receipt") or {}
        receipt_id = receipt.get("receiptId")
        if not isinstance(receipt_id, str) or not receipt_id:
            return program.fail("failed", "owner_failed", "TG producer admission returned no receipt ID", "producers")
        EFFECTS["producer_admissions"] = previous + 1
        envelope["signals"]["owner_handles"]["producer_receipts"].append(
            {"producer": producer, "receipt_id": receipt_id})
        envelope["evidence"].append("test-genie:validation:" + receipt_id)

        previous = EFFECTS["producer_waits"]
        EFFECTS["producer_waits"] = None
        try:
            waited = test_genie.validation.wait(
                receipt_id=receipt_id, wait_id=key + "/wait", timeout="660s").meta()
        except Exception as exc:
            status, klass = program.classify(exc)
            return program.fail(status, klass, exc, "producers")
        terminal = waited.get("receipt") or {}
        if terminal.get("state") != "RECEIPT_STATE_SUCCEEDED":
            return program.fail("failed", "owner_failed", "selected producer receipt did not succeed: " + receipt_id, "producers")
        EFFECTS["producer_waits"] = previous + 1
        produced = terminal.get("producedEvidenceSet")
        if not isinstance(produced, dict):
            return program.fail("failed", "owner_failed", "successful TG producer receipt lacks producedEvidenceSet", "producers")
        if (produced.get("producerReceiptId") != receipt_id or produced.get("producer") != producer
                or produced.get("target") != "browser-automation-studio"
                or not isinstance(produced.get("runId"), str) or not produced["runId"]
                or produced.get("candidateIdentity") != expected_identity.get("identity")
                or not isinstance(produced.get("catalogDigest"), str) or not produced["catalogDigest"]
                or not isinstance(produced.get("artifacts"), list)):
            return program.fail("failed", "owner_failed", "producedEvidenceSet does not match its exact producer receipt/candidate", "producers")
        if not produced["artifacts"]:
            return program.fail("failed", "owner_failed", "successful TG producer returned an empty evidence set", "producers")
        artifact_ids = set()
        artifact_bytes = 0
        for artifact in produced["artifacts"]:
            if (not isinstance(artifact, dict)
                    or any(not isinstance(artifact.get(key), str) or not artifact[key].strip()
                           for key in ("producer", "artifactId", "kind", "checksum"))
                    or artifact.get("producer") != "browser-automation-studio"
                    or artifact.get("kind") not in ("generic.file", "command.output")
                    or not re.fullmatch(r"[0-9a-f]{64}", artifact.get("checksum", ""))
                    or artifact.get("artifactId") in artifact_ids):
                return program.fail("failed", "owner_failed", "TG produced an invalid, duplicate or checksumless artifact reference", "producers")
            size = artifact.get("sizeBytes")
            if (isinstance(size, bool) or not isinstance(size, (int, str)) or not str(size).isdigit()
                    or int(size) < 0):
                return program.fail("failed", "owner_failed", "TG produced an invalid artifact sizeBytes", "producers")
            artifact_ids.add(artifact["artifactId"])
            artifact_bytes += int(size)
            if artifact_bytes > 16777216:
                return program.fail("failed", "owner_failed", "TG produced evidence exceeds 16 MiB", "producers")
        artifact_count += len(produced["artifacts"])
        if artifact_count > 32:
            return program.fail("failed", "owner_failed", "selected evidence sets exceed the retained artifact bound", "producers")
        retained_sets.append(produced)  # exact owner value; no edits or reconstruction
    return "admit"


def step_admit():
    """Create the one exact rehabilitation validation with retained sets unchanged."""
    envelope["phase"] = "act"
    intent = {
        "schema_version": 1, "idempotency_key": request_key,
        "caller_scenario": "browser-automation-studio", "caller_execution_id": request_key,
        "targets": [{"kind": "VALIDATION_TARGET_KIND_SCENARIO", "id": "browser-automation-studio",
                     "root": "scenarios/browser-automation-studio"}],
        "purpose": "VALIDATION_PURPOSE_INVESTIGATION",
        "required_strength": "VALIDATION_STRENGTH_TARGETED",
        "expected_identity": expected_identity,
        "evidence_policy": {"required_evidence_kinds": ["test-genie-run"]},
        "deadline_policy": {"queue_budget": "30s", "execution_budget": "30s", "maximum_attempts": 1},
        "reuse_policy": {"mode": "REUSE_MODE_ATTACH_OR_TERMINAL", "maximum_age": "3600s"},
        "concurrency_policy": {"mode": "CONCURRENCY_MODE_SHARED_COMPATIBLE", "maximum_parallelism": 1},
        "content_inputs": [{"name": "scenario", "root": "scenarios/browser-automation-studio",
                            "selections": [{"glob": "**", "required": True}]}],
        "phases": ["rehabilitation-evidence"], "retained_evidence_sets": retained_sets}
    EFFECTS["validation_admissions"] = None
    try:
        admission = test_genie.validation.create(intent=intent).meta()
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "admit")
    receipt_id = (admission.get("receipt") or {}).get("receiptId")
    if not isinstance(receipt_id, str) or not receipt_id:
        return program.fail("failed", "owner_failed", "TG validation admission returned no receipt ID", "admit")
    EFFECTS["validation_admissions"] = 1
    envelope["signals"]["owner_handles"]["validation_receipt_id"] = receipt_id
    envelope["evidence"].append("test-genie:validation:" + receipt_id)
    return "wait"


def step_wait():
    envelope["phase"] = "collect"
    receipt_id = envelope["signals"]["owner_handles"]["validation_receipt_id"]
    EFFECTS["validation_waits"] = None
    try:
        waited = test_genie.validation.wait(
            receipt_id=receipt_id, wait_id=request_key + "/rehabilitation/wait", timeout="120s").meta()
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "wait")
    terminal = waited.get("receipt") or {}
    if terminal.get("state") not in ("RECEIPT_STATE_SUCCEEDED", "RECEIPT_STATE_FAILED", "RECEIPT_STATE_DEGRADED"):
        return program.fail("failed", "owner_failed", "rehabilitation validation did not reach an evaluable terminal state: " + receipt_id, "wait")
    envelope["signals"]["qualification"] = {"validation_receipt_state": terminal.get("state")}
    EFFECTS["validation_waits"] = 1
    return "qualification"


def step_qualification():
    envelope["phase"] = "collect"
    EFFECTS["setpoint_reads"] = None
    try:
        child = lib.browser_automation_studio.setpoint_read(
            profile="rehabilitation",
            validation_receipt_id=envelope["signals"]["owner_handles"]["validation_receipt_id"],
            expected_source_identity=expected_identity["identity"])
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, exc, "qualification")
    rows = child.head(1)
    if not rows:
        return program.fail("failed", "owner_failed", "bound setpoint child returned no envelope row", "qualification")
    result = rows[0]  # child metadata is artifact identity, not the child result
    signals = result.get("signals") or {}
    envelope["signals"]["qualification"] = {
        "status": result.get("status"), "qualification_bound": signals.get("qualification_bound"),
        "product_qualified": signals.get("product_qualified"), "unmet": signals.get("unmet")}
    EFFECTS["setpoint_reads"] = 1
    product_rows = signals.get("rows", [])
    rows_by_name = {row.get("row"): row for row in product_rows if isinstance(row, dict)}
    rows_complete = (len(product_rows) == 17 and len(rows_by_name) == 17)
    selected_pass = all(name in rows_by_name and rows_by_name[name].get("in_band") is True
                        and rows_by_name[name].get("unavailable") is False for name in required_rows)
    receipt_state = (signals.get("qualification_candidate") or {}).get("receipt_state")
    actual_failed = receipt_state in ("RECEIPT_STATE_FAILED", "RECEIPT_STATE_DEGRADED")
    envelope["signals"]["qualification"].update({
        "status": result.get("status"), "qualification_bound": signals.get("qualification_bound"),
        "product_qualified": signals.get("product_qualified"), "unmet": signals.get("unmet"),
        "phase_status": (signals.get("owner_evidence") or {}).get("phase_status"),
        "required_rows": required_rows, "boundary_accepted": selected_pass and rows_complete
            and result.get("status") in ("ok", "partial") and signals.get("qualification_bound") is True,
        "validation_receipt_state": receipt_state, "selected_evidence_status": "passed" if selected_pass else "failed"})
    if (selected_pass and rows_complete and result.get("status") in ("ok", "partial")
            and signals.get("qualification_bound") is True):
        envelope["status"] = "ok"
        return "report"
    return program.fail("partial" if actual_failed else "failed", "owner_failed", "selected boundary rows lack exact passing bound evidence", "qualification")


def step_report():
    envelope["phase"] = "report"
    program.report()


STATES = {"validate": step_validate, "act": step_act, "collect": step_collect,
          "refresh": step_refresh, "producers": step_producers, "admit": step_admit,
          "wait": step_wait, "qualification": step_qualification, "report": step_report}
program.run(STATES, "validate")

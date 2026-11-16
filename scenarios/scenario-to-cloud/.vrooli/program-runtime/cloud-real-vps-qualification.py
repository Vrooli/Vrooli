import json
# scenario-to-cloud.cloud-real-vps-qualification: admit one real-VPS qualification
# request, bind it to the owned guest's deployment, deploy the fixture through
# governed operations (plan -> apply -> wait), observe health and edge state,
# and return the qualification standing with one next action. Admission
# refuses before any read or mutation; every effect goes through a governed
# scenario-to-cloud binding, never a shell string. The owned guest itself
# (vps instance create / wait-for-ssh / snapshot / reset / destroy) is the
# operator's documented REST pre-step and post-step.
inputs = program.inputs()
TARGET_GROUND = "real_vps"
MATRIX_REVISION = "cloud-launch-v2-two-grounds"
# Per-lane case sets are generated from the certification matrix and locked by api/certification/lanes_test.go.
REAL_VPS_LANE_CASES = {
    "package": ["AUTH-01", "AUTH-02", "AUTH-03", "AUTH-04", "AUTH-05", "AUTH-06", "AUTH-07", "AUTH-08", "PLAN-01", "PLAN-02", "PLAN-03", "PLAN-04", "PLAN-05", "PLAN-06", "PLAN-07", "PLAN-08", "RUN-01", "RUN-02", "RUN-03", "RUN-04", "RUN-05", "RUN-06", "RUN-07", "RUN-08", "RUN-09", "RUN-10", "RELEASE-01", "RELEASE-02", "RELEASE-03", "RELEASE-04", "RELEASE-05", "RELEASE-06", "RELEASE-07", "RELEASE-08", "RELEASE-09", "RELEASE-10", "DATA-01", "DATA-02", "DATA-03", "DATA-04", "DATA-05", "DATA-06", "DATA-07", "DATA-08", "DATA-09", "DATA-10", "SECRET-01", "SECRET-02", "SECRET-03", "SECRET-04", "SECRET-05", "SECRET-06", "SECRET-07", "SECRET-08"],
    "api": ["AUTH-01", "AUTH-02", "AUTH-03", "AUTH-04", "AUTH-05", "AUTH-06", "AUTH-07", "AUTH-08", "PLAN-01", "PLAN-02", "PLAN-03", "PLAN-04", "PLAN-05", "PLAN-06", "PLAN-07", "PLAN-08", "REACH-01", "REACH-02", "REACH-03", "REACH-04", "REACH-05", "REACH-06", "REACH-07", "REACH-08", "SECRET-01", "SECRET-02", "SECRET-03", "SECRET-04", "SECRET-05", "SECRET-06", "SECRET-07", "SECRET-08", "GOV-01", "GOV-02", "GOV-03", "GOV-04", "GOV-05", "GOV-06", "GOV-07", "GOV-08"],
    "workflow": ["PLAN-01", "PLAN-02", "PLAN-03", "PLAN-04", "PLAN-05", "PLAN-06", "PLAN-07", "PLAN-08", "GOV-01", "GOV-02", "GOV-03", "GOV-04", "GOV-05", "GOV-06", "GOV-07", "GOV-08"],
    "relay": ["AUTH-01", "AUTH-02", "AUTH-03", "AUTH-04", "AUTH-05", "AUTH-06", "AUTH-07", "AUTH-08"],
    "qemu": ["RUN-01", "RUN-02", "RUN-03", "RUN-04", "RUN-05", "RUN-06", "RUN-07", "RUN-08", "RUN-09", "RUN-10", "REACH-01", "REACH-02", "REACH-03", "REACH-04", "REACH-05", "REACH-06", "REACH-07", "REACH-08", "RELEASE-01", "RELEASE-02", "RELEASE-03", "RELEASE-04", "RELEASE-05", "RELEASE-06", "RELEASE-07", "RELEASE-08", "RELEASE-09", "RELEASE-10", "DATA-01", "DATA-02", "DATA-03", "DATA-04", "DATA-05", "DATA-06", "DATA-07", "DATA-08", "DATA-09", "DATA-10", "EDGE-01", "EDGE-02", "EDGE-03", "EDGE-04", "EDGE-05", "EDGE-06", "EDGE-07", "EDGE-08", "OPS-01", "OPS-02", "OPS-03", "OPS-04", "OPS-05", "OPS-06", "OPS-07", "OPS-08"],
    "real_vps": ["RUN-01", "RUN-02", "RUN-03", "RUN-04", "RUN-05", "RUN-06", "RUN-07", "RUN-08", "RUN-09", "RUN-10", "REACH-01", "REACH-02", "REACH-03", "REACH-04", "REACH-05", "REACH-06", "REACH-07", "REACH-08", "RELEASE-01", "RELEASE-02", "RELEASE-03", "RELEASE-04", "RELEASE-05", "RELEASE-06", "RELEASE-07", "RELEASE-08", "RELEASE-09", "RELEASE-10", "DATA-01", "DATA-02", "DATA-03", "DATA-04", "DATA-05", "DATA-06", "DATA-07", "DATA-08", "DATA-09", "DATA-10", "SECRET-01", "SECRET-02", "SECRET-03", "SECRET-04", "SECRET-05", "SECRET-06", "SECRET-07", "SECRET-08", "EDGE-01", "EDGE-02", "EDGE-03", "EDGE-04", "EDGE-05", "EDGE-06", "EDGE-07", "EDGE-08", "GOV-01", "GOV-02", "GOV-03", "GOV-04", "GOV-05", "GOV-06", "GOV-07", "GOV-08", "OPS-01", "OPS-02", "OPS-03", "OPS-04", "OPS-05", "OPS-06", "OPS-07", "OPS-08"],
    "bas_ui": ["UX-01", "UX-02", "UX-03", "UX-04", "UX-05", "UX-06", "UX-07", "UX-08"],
    "independent_review": ["UX-01", "UX-02", "UX-03", "UX-04", "UX-05", "UX-06", "UX-07", "UX-08"],
    "soak": ["OPS-01", "OPS-02", "OPS-03", "OPS-04", "OPS-05", "OPS-06", "OPS-07", "OPS-08"],
}
FAULT_CAPABILITIES = ["worker_restart", "lease_expiry", "lost_reply", "transport_disconnect", "target_reboot", "data_interruption"]
CLEANUP_POLICIES = ["retain_failed_evidence_then_remove_owned_ephemeral_resources", "retain_all_owned_resources"]
CASE_ASSERTIONS = {"AUTH-01":{"assertion":"Denied before dispatch with no secret or target effect","trigger":"Unauthenticated request to every sensitive route"},"AUTH-02":{"assertion":"Object lookup and mutation denied without disclosing restricted identity","trigger":"Wrong namespace or environment"},"AUTH-03":{"assertion":"Next protected operation rejected; existing session policy enforced","trigger":"Expired or revoked grant"},"AUTH-04":{"assertion":"Authorization and result correlation bind the intended machine","trigger":"Cross-target substituted identifier"},"AUTH-05":{"assertion":"Denied across UI CLI API and relay","trigger":"Read-only actor requests mutation"},"AUTH-06":{"assertion":"Upgrade and continued session obey origin and grant policy","trigger":"Terminal or WebSocket authorization"},"AUTH-07":{"assertion":"Cross-scope reads fail without value or existence leakage","trigger":"Credential or evidence identifier enumeration"},"AUTH-08":{"assertion":"Production configuration cannot enable an unauthenticated mutation path","trigger":"Test-mode or development bypass in deployment"},"DATA-01":{"assertion":"Consistent restore with measured recovery point","trigger":"Backup under acknowledged writes"},"DATA-02":{"assertion":"Rejected; existing data remains protected","trigger":"Corrupted or incomplete backup"},"DATA-03":{"assertion":"Actionable recovery failure without false completion","trigger":"Backup encryption key unavailable"},"DATA-04":{"assertion":"Data preserved and external functionality verified","trigger":"Compatible code rollback"},"DATA-05":{"assertion":"Service and logical data verified within frozen RTO","trigger":"Fresh-host restore"},"DATA-06":{"assertion":"Blocked with supported recovery path","trigger":"Incompatible schema rollback"},"DATA-07":{"assertion":"Declared transactional or recovery semantics proven","trigger":"Migration fails partway"},"DATA-08":{"assertion":"No adopt delete or overwrite without explicit authority","trigger":"Volume belongs to another deployment"},"DATA-09":{"assertion":"Protected recovery points and active operation leases preserved","trigger":"Retention pruning"},"DATA-10":{"assertion":"Recovery identity works without original controller session","trigger":"Restore credential prerequisites"},"EDGE-01":{"assertion":"Independent observation reports mismatch","trigger":"DNS wrong or stale"},"EDGE-02":{"assertion":"Correct hostname chain and renewal outcome verified","trigger":"Certificate issue and renewal"},"EDGE-03":{"assertion":"Only declared endpoints accessible","trigger":"Public ingress scan"},"EDGE-04":{"assertion":"Unauthenticated public reach is denied","trigger":"Private backend direct access"},"EDGE-05":{"assertion":"Typed least-privilege effects; unrelated rules preserved","trigger":"Firewall remediation"},"EDGE-06":{"assertion":"Declared services and endpoint recover under owner policy","trigger":"Target reboot"},"EDGE-07":{"assertion":"Consumer rejects healthy verdict","trigger":"Health HTTP 200 with unhealthy body"},"EDGE-08":{"assertion":"Cannot satisfy candidate gate","trigger":"Stale or wrong-release health receipt"},"GOV-01":{"assertion":"Decision cannot approve","trigger":"Missing mandatory evidence cell"},"GOV-02":{"assertion":"Rejected as incompatible","trigger":"Evidence belongs to other release or target"},"GOV-03":{"assertion":"Publish rejects identity mismatch","trigger":"Approved bytes replaced before publish"},"GOV-04":{"assertion":"Reconcile destination identity without duplicate unsafe effects","trigger":"Publication reply lost"},"GOV-05":{"assertion":"External fetch or inspection matches approved identity","trigger":"Successful staging publish"},"GOV-06":{"assertion":"Policy rejects or follows documented in-flight rule","trigger":"Revoked approval before mutation"},"GOV-07":{"assertion":"Explicit unavailable; no synthetic pass","trigger":"Evidence fetch lacks bytes"},"GOV-08":{"assertion":"Governed recovery targets exact supported release","trigger":"Recovery after publication"},"OPS-01":{"assertion":"Availability and latency remain within frozen budgets","trigger":"Sustained healthy traffic"},"OPS-02":{"assertion":"Actionable alert arrives within detection budget","trigger":"Scheduled failure and alert"},"OPS-03":{"assertion":"Bounded degradation and owner remediation; no broad destructive cleanup","trigger":"Disk and resource pressure"},"OPS-04":{"assertion":"No accumulating processes leases files or operations","trigger":"Repeated update and recovery cycles"},"OPS-05":{"assertion":"Data disposition explicit; access revoked; public route removed","trigger":"Workload retirement"},"OPS-06":{"assertion":"Zero unexplained owned residual resources or spend","trigger":"Qualification cleanup"},"OPS-07":{"assertion":"Both identities qualify or required support remains incomplete","trigger":"Minimal fixture on amd64 and arm64"},"OPS-08":{"assertion":"Generic deployment and update boundary succeeds without product-specific code","trigger":"Representative hosted consumer"},"PLAN-01":{"assertion":"Same normalized configuration and executable-plan digest","trigger":"Equivalent UI CLI API inputs"},"PLAN-02":{"assertion":"Apply rejects stale prerequisites before mutation","trigger":"Changed target facts after preview"},"PLAN-03":{"assertion":"Apply rejects digest mismatch","trigger":"Changed closure or artifact after preview"},"PLAN-04":{"assertion":"Actionable unsupported result rather than incomplete bundle","trigger":"Unknown resource capability"},"PLAN-05":{"assertion":"Stable plan identity and ordered effects","trigger":"Repeated preview with same immutable inputs"},"PLAN-06":{"assertion":"No raw secret in serialization logs or review","trigger":"Plan contains secret reference"},"PLAN-07":{"assertion":"No later step silently performs equivalent action","trigger":"Human denies privileged action"},"PLAN-08":{"assertion":"Explicit conflict or deterministic revision handling","trigger":"Conflicting desired-state edits"},"REACH-01":{"assertion":"Trust and scoped onboarding receipt before privileged use","trigger":"Enroll previously unknown target"},"REACH-02":{"assertion":"Re-enrollment policy; old identity not silently accepted","trigger":"Machine identity changes"},"REACH-03":{"assertion":"Equivalent policy authorization results and correlation","trigger":"Direct adapter and Bridge path"},"REACH-04":{"assertion":"Bounded resume or restart with final digest verification","trigger":"Transport interruption during artifact transfer"},"REACH-05":{"assertion":"Existing durable target operation reattached","trigger":"Controller reconnect"},"REACH-06":{"assertion":"Typed unavailable standing; no hidden private SSH fallback","trigger":"Shared reach unavailable"},"REACH-07":{"assertion":"Analyzer closure matches onboarding installed/running selection","trigger":"Target configuration subset"},"REACH-08":{"assertion":"Bootstrap and repair preserve its processes files and network rules","trigger":"Coexisting unrelated workload"},"RELEASE-01":{"assertion":"Rejected before extraction or activation","trigger":"Corrupt bundle byte"},"RELEASE-02":{"assertion":"Trust policy rejects artifact","trigger":"Valid digest from untrusted producer"},"RELEASE-03":{"assertion":"Rejected without writes outside owned staging","trigger":"Archive traversal or unsafe link"},"RELEASE-04":{"assertion":"Preflight rejects before active release changes","trigger":"Wrong architecture or ABI"},"RELEASE-05":{"assertion":"Rejected rather than rebuilt from mutable source","trigger":"Target CLI differs from approved manifest"},"RELEASE-06":{"assertion":"Active release remains serviceable; staged residue owned","trigger":"Interrupted staging"},"RELEASE-07":{"assertion":"External endpoint serves exact new release","trigger":"Healthy compatible update"},"RELEASE-08":{"assertion":"Activation withheld or recovery restores verified healthy standing","trigger":"Unhealthy candidate"},"RELEASE-09":{"assertion":"Active and recovery leases prevent deletion","trigger":"Artifact garbage collection race"},"RELEASE-10":{"assertion":"Explicit rollback eligibility tied to schema and config","trigger":"Old release compatibility boundary"},"RUN-01":{"assertion":"One durable operation and one effective mutation","trigger":"Duplicate admission same key"},"RUN-02":{"assertion":"Conflict; original intent remains unchanged","trigger":"Same key with different request"},"RUN-03":{"assertion":"Recorded intent resumes or reconciles without lost work","trigger":"Worker restart after intent persistence"},"RUN-04":{"assertion":"Stale fence cannot commit; target result reconciled","trigger":"Worker lease expires during remote effect"},"RUN-05":{"assertion":"No blind replay; owner obtains authoritative standing","trigger":"Remote success reply lost"},"RUN-06":{"assertion":"Unknown or waiting state with one durable next action","trigger":"Target disconnect mid-step"},"RUN-07":{"assertion":"Server operation remains correctly owned","trigger":"Client timeout or cancellation"},"RUN-08":{"assertion":"Serialization or explicit conflict protects invariants","trigger":"Competing operation on same deployment"},"RUN-09":{"assertion":"Reconciles actual active release and durable data","trigger":"Target restart during activation"},"RUN-10":{"assertion":"Explicit incompatibility; no guessed success","trigger":"Stored operation has unknown version"},"SECRET-01":{"assertion":"Only authorized consumer receives intended version","trigger":"Initial materialization"},"SECRET-02":{"assertion":"New version works; rollout standing recorded","trigger":"Rotation accepted by provider"},"SECRET-03":{"assertion":"Prior usable state retained or explicit recovery standing","trigger":"Provider rejects rotation"},"SECRET-04":{"assertion":"Provider and consumer cease authorized use as specified","trigger":"Revoked credential"},"SECRET-05":{"assertion":"Zero leakage in logs artifacts command arguments and UI","trigger":"Synthetic secret canaries"},"SECRET-06":{"assertion":"Version binding prevents mixed unexplained state","trigger":"Concurrent rotation and deployment"},"SECRET-07":{"assertion":"No plaintext fallback; actionable pending or failure","trigger":"Credential service unavailable"},"SECRET-08":{"assertion":"Scoped time-bounded audited path and subsequent revocation","trigger":"Recovery break-glass operation"},"UX-01":{"assertion":"Operator completes without undocumented shell steps","trigger":"First deploy through UI"},"UX-02":{"assertion":"Equivalent effects and inspectable standing","trigger":"Same operation through CLI and API"},"UX-03":{"assertion":"Logical focus and accessible controls throughout","trigger":"Keyboard-only critical journey"},"UX-04":{"assertion":"No lost primary actions or hidden error context","trigger":"Narrow viewport and long content"},"UX-05":{"assertion":"Progress reattaches by durable identity","trigger":"Reconnect during long operation"},"UX-06":{"assertion":"Explains cause remediation and safe next action","trigger":"Denied permission or failed prerequisite"},"UX-07":{"assertion":"Scoped intentional disclosure with safe default rendering","trigger":"Secret display and copy affordances"},"UX-08":{"assertion":"Runbook enables diagnosis and restore without author assistance","trigger":"Independent operator recovery"}}
LANE_EVIDENCE_OWNERS = {"package":"scenario-to-cloud","api":"scenario-to-cloud","workflow":"scenario-to-cloud","relay":"Vrooli Bridge","qemu":"scenario-to-cloud QEMU lane","real_vps":"scenario-to-cloud VPS lane","bas_ui":"Browser Automation Studio","independent_review":"independent reviewer","soak":"scenario-to-cloud soak owner"}

def assertion_driver(case_id, lane):
    spec = CASE_ASSERTIONS.get(case_id)
    owner = LANE_EVIDENCE_OWNERS.get(lane)
    if spec is None or owner is None:
        return None
    return {"driver_ref":"certification-assertion:v1:" + case_id + ":" + lane + ":" + TARGET_GROUND, "owner":owner, "trigger":spec["trigger"], "expected_assertion":spec["assertion"], "status":"owner_assertion_required"}

OWNED_TARGET_PREFIX = "disposable-vps:"
SPEND_CEILING_MAX = 0
OWNED_TARGET_MAX = 1
EXECUTION_SECONDS_MAX = 7200
# One kernel invoke is bounded well under the lane budget; a non-terminal
# operation after this wait is reported with its id, never as a failure.
WAIT_SECONDS = 90
# Post-apply operation completion and health are still not case-level assertions.

def text(name, default=""):
    return str(inputs.get(name, default) or "").strip()

request_id = text("request_id"); release_ref = text("release_ref"); target_ref = text("target_ref")
environment_ref = text("environment_ref"); workload_ref = text("workload_ref"); authorization_ref = text("authorization_ref")
deployment_id = text("deployment_id")
expected_target_generation = inputs.get("target_generation")
matrix_revision = text("matrix_revision", MATRIX_REVISION); lane = text("lane", "")
target_ground = text("target_ground", TARGET_GROUND)
cases = inputs.get("cases") or []
limits = inputs.get("limits") or {}
faults = inputs.get("faults") or {}
cleanup = inputs.get("cleanup") or {}
allowed_faults = faults.get("allowed") if isinstance(faults, dict) else None
cleanup_policy = str(cleanup.get("policy", "")).strip() if isinstance(cleanup, dict) else ""

envelope = program.envelope("scenario-to-cloud.cloud-real-vps-qualification", "1")
envelope["signals"] = {"qualification_id": "qual:" + request_id if request_id else "", "standing": "refused", "cells": [], "release": {"release_ref": release_ref}, "target": {"target_ref": target_ref, "target_ground": target_ground, "target_generation": expected_target_generation, "environment_ref": environment_ref, "workload_ref": workload_ref, "deployment_id": deployment_id}, "next_action": ""}
fail = program.fail
observed = {"deployment": None, "plan_digest": "", "plan_release": "", "closure_status": "", "validation_run": None, "operation": None, "target_receipts": [], "target_receipt_error": "", "health": None, "edge": None}

def refuse(klass, detail, where="validate"):
    envelope["signals"]["standing"] = "refused"
    envelope["signals"]["next_action"] = detail
    return fail("refused", klass, detail, where)

def invalid(detail):
    envelope["signals"]["standing"] = "refused"
    envelope["signals"]["next_action"] = detail
    return fail("failed", "invalid_input", detail, "validate")

def is_immutable_digest(value):
    if not value.startswith("sha256:"):
        return False
    digest = value[len("sha256:"):]
    return len(digest) == 64 and all(c in "0123456789abcdef" for c in digest)

def first_row(handle):
    rows = handle.head(1)
    return rows[0] if rows and isinstance(rows[0], dict) else {}

def validate_bridge_target(target, expected_machine_id, expected_generation):
    """Return a refusal reason unless the binding is the expected Bridge enrollment."""
    if str(target.get("transport") or "") != "bridge":
        return "real_vps requires Bridge transport; un-enrolled SSH targets are not proving grounds"
    machine_id = str(target.get("machineId") or "").strip()
    if not machine_id or machine_id != expected_machine_id:
        return "deployment machine identity does not match the operator-supplied disposable VPS identity"
    if not str(target.get("nodeId") or "").strip():
        return "Bridge target node id is missing"
    observed_generation = target.get("enrollmentGeneration")
    if type(observed_generation) is not int or observed_generation <= 0:
        return "Bridge enrollment generation must be a positive integer"
    if type(expected_generation) is not int or expected_generation <= 0:
        return "operator-supplied target_generation must be a positive integer"
    if observed_generation != expected_generation:
        return "Bridge enrollment generation differs from the operator-supplied target_generation"
    return None

def validate_target_receipt(row, deployment_id, operation_id, plan_digest, step, machine_id, node_id, generation):
    """Fail closed unless a fresh response is bound to this operation/target/step."""
    if str(row.get("deployment_id") or "") != deployment_id:
        return "receipt deployment identity mismatch"
    if str(row.get("operation_id") or "") != operation_id:
        return "receipt operation identity mismatch"
    if str(row.get("plan_digest") or "") != plan_digest:
        return "receipt plan digest mismatch"
    if str(row.get("step") or "") != step:
        return "receipt step mismatch"
    target = row.get("target") if isinstance(row.get("target"), dict) else {}
    if target.get("transport") != "bridge" or target.get("machine_id") != machine_id or target.get("node_id") != node_id or target.get("enrollment_generation") != generation:
        return "receipt target identity mismatch"
    receipt = row.get("receipt") if isinstance(row.get("receipt"), dict) else {}
    if receipt.get("found") is not True:
        return "target owner has no receipt for this operation step"
    if receipt.get("outcome") not in ("succeeded", "unchanged"):
        return "target owner receipt outcome is " + str(receipt.get("outcome") or "unknown")
    if not isinstance(receipt.get("fence"), int) or isinstance(receipt.get("fence"), bool) or receipt.get("fence") <= 0:
        return "target owner receipt fence is missing or invalid"
    return None

# --- validate: admission. Refuses before any read or mutation. ---
def step_validate():
    for name, value in (("request_id", request_id), ("release_ref", release_ref), ("target_ref", target_ref), ("environment_ref", environment_ref), ("workload_ref", workload_ref)):
        if not value:
            return invalid(name + " is required")
    if not authorization_ref:
        return refuse("no_authorization", "authorization grant absent: supply authorization_ref naming the operator grant that designates " + target_ref + " as a disposable owned fixture")
    if lane not in REAL_VPS_LANE_CASES or target_ground != TARGET_GROUND:
        return invalid("this program qualifies only matrix lanes applicable to target_ground=" + TARGET_GROUND)
    if matrix_revision != MATRIX_REVISION:
        return refuse("stale_artifact", "matrix_revision " + matrix_revision + " is not the frozen revision " + MATRIX_REVISION)
    if not is_immutable_digest(release_ref):
        return refuse("stale_artifact", "release_ref must be an immutable sha256:<64 hex> release manifest digest; mutable references are stale by definition")
    if not environment_ref.startswith("cl2-staging:"):
        return refuse("incompatible_target", "environment_ref must be an isolated cl2-staging:<id> environment; production and general environments are never eligible")
    if not isinstance(expected_target_generation, int) or isinstance(expected_target_generation, bool) or expected_target_generation <= 0:
        return refuse("incompatible_target", "target_generation must be the positive enrollment generation observed or supplied for this disposable VPS")
    if not target_ref.startswith(OWNED_TARGET_PREFIX):
        return refuse("incompatible_target", "target_ref must name an owned disposable real VPS (" + OWNED_TARGET_PREFIX + "<id>); operator machines and enrolled cloud targets are not fixtures")
    if not isinstance(cases, list) or not cases:
        return invalid("cases must be a non-empty list of matrix case ids")
    if len(cases) != 1:
        return invalid("one matrix case is admitted per target operation; batch qualification requires a case-specific fixture and assertion driver")
    unknown = [c for c in cases if c not in REAL_VPS_LANE_CASES.get(lane, [])]
    if unknown:
        return invalid("cases not required on the selected certification lane: " + ", ".join(sorted(str(c) for c in unknown)[:8]))
    if not isinstance(limits, dict):
        return invalid("limits must be an object")
    if int(limits.get("spend_ceiling_minor_units", 0) or 0) > SPEND_CEILING_MAX:
        return refuse("no_authorization", "spend_ceiling_minor_units must be " + str(SPEND_CEILING_MAX) + ": no spend is authorised for additional qualification spend (EXT-10)")
    if int(limits.get("owned_target_count", 1) or 1) > OWNED_TARGET_MAX:
        return refuse("incompatible_target", "owned_target_count exceeds the lane budget of " + str(OWNED_TARGET_MAX))
    if int(limits.get("execution_seconds", EXECUTION_SECONDS_MAX) or EXECUTION_SECONDS_MAX) > EXECUTION_SECONDS_MAX:
        return invalid("execution_seconds exceeds the real-VPS lane timeout of " + str(EXECUTION_SECONDS_MAX))
    if allowed_faults is None or not isinstance(allowed_faults, list):
        return invalid("faults.allowed must be a list (empty means no fault injection)")
    unknown_faults = [f for f in allowed_faults if f not in FAULT_CAPABILITIES]
    if unknown_faults:
        return refuse("unknown_fault_capability", "unknown fault capability: " + ", ".join(sorted(str(f) for f in unknown_faults)) + "; declared capabilities: " + ", ".join(FAULT_CAPABILITIES))
    if cleanup_policy not in CLEANUP_POLICIES:
        return invalid("cleanup.policy must be one of " + ", ".join(CLEANUP_POLICIES))
    envelope["signals"]["standing"] = "admitted"
    envelope["signals"]["cells"] = [{"case_id": c, "lane": lane, "target_ground": TARGET_GROUND, "verdict": "pending", "assertion_driver": assertion_driver(c, lane)} for c in cases]
    envelope["evidence"].append("authorization:" + authorization_ref)
    return "collect"

# --- collect: governed reads only. Binds the request to one deployment on the
# owned guest and compiles the plan it would apply. ---
def step_collect():
    global deployment_id
    envelope["phase"] = "collect"
    try:
        if deployment_id:
            ref = first_row(scenario_to_cloud.deployment.get(id=deployment_id)).get("ref") or {}
        else:
            ref = first_row(scenario_to_cloud.deployment.resolve(selector={"scenario_id": workload_ref, "environment": environment_ref})).get("ref") or {}
    except program.RemoteError as exc:
        if exc.http_status == 404:
            return refuse("incompatible_target", "no deployment is bound to workload " + workload_ref + " in environment " + environment_ref + " (or deployment_id " + deployment_id + "); create the fixture deployment on the owned guest first", "collect")
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    deployment_id = str(ref.get("id") or "")
    target = ref.get("target") or {}
    machine_id = str(target.get("machineId") or "")
    observed["deployment"] = {"id": deployment_id, "scenario_id": ref.get("scenarioId"), "environment": ref.get("environment"), "machine_id": machine_id, "node_id": target.get("nodeId"), "enrollment_generation": target.get("enrollmentGeneration"), "transport": target.get("transport")}
    envelope["signals"]["target"]["deployment_id"] = deployment_id
    if not deployment_id:
        return refuse("incompatible_target", "deployment reference carries no id", "collect")
    target_ref_data = ref.get("target") or target
    target_identity_error = validate_bridge_target(target_ref_data, target_ref, expected_target_generation)
    if target_identity_error:
        return refuse("incompatible_target", target_identity_error, "collect")
    if str(ref.get("environment") or "") != environment_ref:
        return refuse("incompatible_target", "deployment " + deployment_id + " lives in environment " + str(ref.get("environment")) + ", not " + environment_ref, "collect")
    try:
        plan = first_row(scenario_to_cloud.deployment.plan(deployment_id=deployment_id))
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    observed["plan_digest"] = str(plan.get("planDigest") or "")
    observed["closure_status"] = str(plan.get("closureStatus") or "")
    observed["plan_release"] = str((plan.get("plan") or {}).get("releaseDigest") or "")
    # The latest recorded validation run for this scenario is evidence, not a
    # gate: an unreachable test-genie leaves the row unavailable.
    try:
        runs = validate("scenario-to-cloud").head(1)
        observed["validation_run"] = {"row": runs[0]} if runs else {"row": None}
    except Exception as exc:
        observed["validation_run"] = {"unavailable": str(exc)[:160]}
    return "decide"

# --- decide: pure function of inputs and collected rows. ---
def step_decide():
    envelope["phase"] = "decide"
    envelope["signals"]["deployment"] = observed["deployment"]
    envelope["signals"]["plan"] = {"plan_digest": observed["plan_digest"], "release_digest": observed["plan_release"], "closure_status": observed["closure_status"]}
    envelope["signals"]["faults_allowed"] = list(allowed_faults)
    envelope["signals"]["cleanup_policy"] = cleanup_policy
    envelope["evidence"].append("lane:certification/matrix.json")
    envelope["evidence"].append("images:A2-approved VPS image provenance")
    run = observed["validation_run"] or {}
    if isinstance(run.get("row"), dict):
        run_id = run["row"].get("id") or run["row"].get("runId") or ""
        if run_id:
            envelope["evidence"].append("test-genie-run:" + str(run_id))
    elif run.get("unavailable"):
        envelope["signals"]["validation_run"] = "unavailable"
    if not observed["plan_digest"]:
        return fail("failed", "ambiguous_response", "plan compile returned no plan_digest", "decide")
    if observed["closure_status"] != "derived":
        return refuse("stale_artifact", "closure is " + (observed["closure_status"] or "unknown") + ", not derived; the deployment's declaration closure must be derivable before qualification", "decide")
    if observed["plan_release"] != release_ref:
        return refuse("stale_artifact", "compiled plan carries release " + (observed["plan_release"] or "<none>") + " but the request names " + release_ref + "; refresh the deployment's release before qualifying", "decide")
    # Fault cells need faultinject arming on the owned guest, which has no
    # governed binding yet; they are named, not silently dropped.
    if allowed_faults:
        envelope["signals"]["fault_injection"] = "no_governed_binding: faultinject arming is not a scenario-to-cloud binding; fault cells stay pending after the deploy journey"
    envelope["signals"]["standing"] = "ready_to_act"
    return "act"

# --- act: one admitted plan digest, one request key, then observation. ---
def step_act():
    envelope["phase"] = "act"
    try:
        applied = first_row(scenario_to_cloud.deployment.apply(deployment_id=deployment_id, plan_digest=observed["plan_digest"], request_key=request_id, _confirm=True))
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        envelope["signals"]["next_action"] = "grant the session effect:destructive (or the binding's permission) and resubmit with the same request_id; the apply is idempotent per request key" if klass == "no_grant" else str(exc)[:200]
        return fail(status, klass, exc, "act")
    operation_id = str(applied.get("operationId") or "")
    observed["operation"] = {"operation_id": operation_id, "state": applied.get("state"), "plan_digest": applied.get("planDigest")}
    envelope["signals"]["operation"] = observed["operation"]
    if operation_id:
        envelope["evidence"].append("operation:" + operation_id)
    if not operation_id:
        envelope["signals"]["standing"] = "no_op" if applied.get("state") == "no_op" else "applied_without_operation"
        return "classify"
    try:
        standing = first_row(scenario_to_cloud.operation.wait(operation_id=operation_id, timeout_seconds=WAIT_SECONDS))
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        envelope["signals"]["next_action"] = "operation " + operation_id + " is server-owned; reattach with scenario-to-cloud operation wait " + operation_id
        return fail(status, klass, exc, "act")
    observed["operation"].update({"state": standing.get("state"), "terminal": bool(standing.get("terminal")), "fence": standing.get("fence")})
    observed["operation"]["step_receipts"] = standing.get("stepReceipts") or standing.get("step_receipts") or []
    envelope["signals"]["operation"] = observed["operation"]
    admitted_target = ref.get("target") or {}
    candidate_steps = [r for r in observed["operation"]["step_receipts"] if isinstance(r, dict) and r.get("source") == "target_receipt" and r.get("outcome") in ("succeeded", "unchanged")]
    for step_receipt in candidate_steps:
        step = str(step_receipt.get("step") or "")
        if not step:
            observed["target_receipt_error"] = "committed target-receipt step is empty"
            break
        try:
            row = first_row(scenario_to_cloud.operation.target_receipt(deployment_id=deployment_id, operation_id=operation_id, step=step))
        except program.BindingError as exc:
            observed["target_receipt_error"] = str(exc)[:200]
            break
        error = validate_target_receipt(row, deployment_id, operation_id, observed["plan_digest"], step, target_ref, str(admitted_target.get("nodeId") or ""), expected_target_generation)
        if error:
            observed["target_receipt_error"] = error
            break
        target_receipt = row.get("receipt")
        observed["target_receipts"].append({"step": step, "fence": target_receipt.get("fence"), "outcome": target_receipt.get("outcome")})
        envelope["evidence"].append("target-owner-receipt:" + operation_id + ":" + step + ":fence=" + str(target_receipt.get("fence")))
    envelope["signals"]["target_receipts"] = observed["target_receipts"]
    envelope["signals"]["target_receipt_error"] = observed["target_receipt_error"] or None
    for name, call in (("health", lambda: scenario_to_cloud.deployment.health(deployment_id=deployment_id)), ("edge", lambda: scenario_to_cloud.edge.status(deployment_id=deployment_id))):
        try:
            observed[name] = first_row(call()).get("observation") or {}
        except program.BindingError as exc:
            status, klass = program.classify(exc)
            observed[name] = {"unavailable": klass}
    return "classify"

# --- classify: a deterministic table over the operation and observations. ---
def step_classify():
    envelope["phase"] = "classify"
    op = observed["operation"] or {}
    health = observed["health"] or {}
    edge = observed["edge"] or {}
    status = str(health.get("status") or "")
    freshness = str(health.get("freshness") or "")
    healthy = status.endswith("HEALTHY") and not status.endswith("UNHEALTHY") and freshness.endswith("CURRENT") and str(health.get("observedReleaseDigest") or "") == release_ref
    envelope["signals"]["health"] = {"status": status or "unknown", "freshness": freshness or "unknown", "observed_release_digest": health.get("observedReleaseDigest"), "unavailable": health.get("unavailable")}
    envelope["signals"]["edge"] = {"routes": len(edge.get("routes") or []), "private_listeners": len(edge.get("privateListeners") or []), "unavailable": edge.get("unavailable")}
    if not op.get("terminal"):
        envelope["signals"]["standing"] = "operation_pending"
        envelope["signals"]["next_action"] = "operation " + str(op.get("operation_id")) + " is still " + str(op.get("state")) + "; wait once with scenario-to-cloud operation wait, then re-observe health. A lost observer never fails a server-owned operation"
        return "report"
    if str(op.get("state")) != "succeeded":
        envelope["signals"]["standing"] = "journey_failed"
        envelope["signals"]["next_action"] = "operation " + str(op.get("operation_id")) + " ended " + str(op.get("state")) + "; read its receipts before any retry"
        return fail("failed", "journey_failed", "deploy journey operation ended " + str(op.get("state")), "classify")
    if not observed["target_receipts"] or observed["target_receipt_error"]:
        envelope["signals"]["standing"] = "journey_unverified"
        envelope["signals"]["next_action"] = "target owner receipt is missing or unbound; reconcile target reach and read the exact operation step receipt before any assertion"
        return fail("failed", "target_receipt_unverified", observed["target_receipt_error"] or "no successful target-owner step receipt was committed", "classify")
    if not healthy:
        envelope["signals"]["standing"] = "journey_unverified"
        envelope["signals"]["next_action"] = "operation succeeded but health is " + (status or "unknown") + "/" + (freshness or "unknown") + " for release " + str(health.get("observedReleaseDigest")) + "; do not write passed receipts without a current healthy observation"
        return fail("failed", "health_unverified", "health observation does not verify the deployed release", "classify")
    envelope["signals"]["standing"] = "deploy_health_verified_pending_assertions"
    driver = envelope["signals"]["cells"][0]["assertion_driver"]
    envelope["signals"]["next_action"] = "Bridge VPS deploy and health are verified, but the case assertion remains pending: invoke " + str(driver["owner"]) + " for " + driver["driver_ref"] + " and attach the owner receipt and assertion artifact before certification.WriteReceipt. Bind that result to operation " + str(op.get("operation_id"))
    for cell in envelope["signals"]["cells"]:
        cell["operation_ref"] = op.get("operation_id")
        cell["target_receipt_refs"] = [
            "target-receipt://" + str(op.get("operation_id")) + "/" + str(receipt.get("step")) + "?fence=" + str(receipt.get("fence"))
            for receipt in observed["target_receipts"]
        ]
        cell["verdict"] = "pending_assertions"
        cell["assertion_driver"] = assertion_driver(cell["case_id"], cell["lane"])
    envelope["status"] = "ok"
    return "report"

def step_report():
    envelope["phase"] = "report"
    envelope["status"] = "ok" if not envelope["errors"] else envelope["status"]
    if len(json.dumps(envelope, allow_nan=False, separators=(",", ":")).encode()) > 60000:
        envelope["signals"]["cells"] = envelope["signals"]["cells"][:8]
        envelope["signals"]["cells_truncated"] = True
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None

STATES = {"validate": step_validate, "collect": step_collect, "decide": step_decide, "act": step_act, "classify": step_classify, "report": step_report}
program.run(STATES, "validate")

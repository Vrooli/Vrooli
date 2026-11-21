"""One recoverable native inference stage; retained jobs are the checkpoints."""
import json
import hashlib
import uuid

inputs = program.inputs()
envelope = program.envelope("image-tools.scene-preparation-stage", "1")
action = inputs.get("action", "")
operation = inputs.get("operation", "")
operation_id = inputs.get("operation_id", "")
job_id = inputs.get("job_id", "")
params = inputs.get("params", {})
request = inputs.get("request", {})
preparation_id = inputs.get("preparation_id", "")
# Keep the envelope bounded and avoid copying image prompts into reports.
envelope["inputs"] = {"action": action, "operation": operation,
                      "operation_id": operation_id, "job_id": job_id}



def step_validate():
    if action not in ("submit", "observe", "cancel", "assemble", "inspect-preparation", "review", "assess-repair", "import-scene"):
        return program.fail("failed", "invalid_input", "unsupported preparation action", "validate")
    if action == "submit":
        if operation not in ("segment", "scene_geometry", "object_removal", "inpaint", "image_to_object"):
            return program.fail("failed", "invalid_input", "operation is not a scene-preparation stage", "validate")
        try:
            if str(uuid.UUID(operation_id)) != operation_id:
                raise ValueError("noncanonical UUID")
        except (ValueError, TypeError, AttributeError):
            return program.fail("failed", "invalid_input", "submit requires a stable canonical UUID operation_id", "validate")
        if not isinstance(params, dict) or len(json.dumps(params)) > 16384:
            return program.fail("failed", "invalid_input", "params must be one typed object bounded to 16 KiB", "validate")
        if params.get("allowByok") or params.get("allow_byok"):
            return program.fail("failed", "invalid_input", "this local preparation workflow does not dispatch paid providers", "validate")
        if any(key in params for key in ("allow_byok", "fallback_policy", "allow_reclaim")):
            return program.fail("failed", "invalid_input", "params use the documented protobuf JSON camelCase field names", "validate")
        if params.get("fallbackPolicy", "local_only") != "local_only" or params.get("allowReclaim", False) is not False:
            return program.fail("failed", "invalid_input", "preparation stages require local_only and allowReclaim=false", "validate")
        if not isinstance(inputs.get("input_ref"), str) or not 0 < len(inputs["input_ref"]) <= 512:
            return program.fail("failed", "invalid_input", "submit requires an owned input_ref", "validate")
        if len(str(inputs.get("mask_ref", ""))) > 512 or (operation in ("object_removal", "inpaint") and not inputs.get("mask_ref")):
            return program.fail("failed", "invalid_input", "repair requires an explicit owned mask_ref", "validate")
        if not isinstance(params.get("modelOverride"), str) or not 0 < len(params["modelOverride"]) <= 128:
            return program.fail("failed", "invalid_input", "pin the explicitly selected installed modelOverride", "validate")
    elif action == "assess-repair":
        if not isinstance(inputs.get("mask_ref"), str) or not 0 < len(inputs["mask_ref"]) <= 256 or not isinstance(params, dict):
            return program.fail("failed", "invalid_input", "assess-repair requires an owned mask_ref and optional typed crop params", "validate")
    elif action in ("assemble", "review", "import-scene"):
        if not isinstance(request, dict) or len(json.dumps(request)) > 131072:
            return program.fail("failed", "invalid_input", "request must be a typed preparation object bounded to 128 KiB", "validate")
        if action == "assemble" and (not request.get("manifest") or not request.get("artifacts") or not request.get("label")):
            return program.fail("failed", "invalid_input", "assemble requires a manifest, exact owned artifact bindings and a label", "validate")
        if action == "import-scene":
            identity = request.get("preparationId", "")
            if not isinstance(identity, str) or len(identity) != 64 or any(c not in "0123456789abcdef" for c in identity) or request.get("expectedBundleSha256") != identity:
                return program.fail("failed", "invalid_input", "import requires the exact matching immutable preparation ID and bundle SHA256", "validate")
            for name, bound in (("licenseId", 128), ("attribution", 1024), ("sourceLabel", 512)):
                if not isinstance(request.get(name), str) or not request[name].strip() or len(request[name]) > bound:
                    return program.fail("failed", "invalid_input", "import requires explicit bounded source rights and provenance claims", "validate")
            if not isinstance(request.get("allowLocalRender", False), bool):
                return program.fail("failed", "invalid_input", "allowLocalRender must be an explicit boolean", "validate")
            rationale = request.get("sourceUseRationale", "")
            if not isinstance(rationale, str) or len(rationale) > 1024 or any(c in rationale for c in ("\x00", "\r")) or bool(rationale.strip()) != request.get("allowLocalRender", False):
                return program.fail("failed", "invalid_input", "local render requires a bounded source-use rationale separate from visual acceptance", "validate")
        if action == "review":
            if not request.get("id") or not request.get("review"):
                return program.fail("failed", "invalid_input", "review requires its exact retained id and completed visual review", "validate")
            try:
                operation_key = request.get("operationId", "")
                if str(uuid.UUID(operation_key)) != operation_key:
                    raise ValueError("noncanonical UUID")
            except (ValueError, TypeError, AttributeError):
                return program.fail("failed", "invalid_input", "review requires a stable canonical operationId UUID reused on retry", "validate")
    elif action == "inspect-preparation":
        if not isinstance(preparation_id, str) or len(preparation_id) != 64 or any(c not in "0123456789abcdef" for c in preparation_id):
            return program.fail("failed", "invalid_input", "inspect-preparation requires an immutable SHA256 identity", "validate")
    elif not isinstance(job_id, str) or not job_id or len(job_id) > 128:
        return program.fail("failed", "invalid_input", "observe and cancel require the retained job_id", "validate")
    return "act"


def step_act():
    envelope["phase"] = "act"
    try:
        if action == "import-scene":
            envelope["inputs"]["request_sha256"] = hashlib.sha256(json.dumps(request, sort_keys=True, separators=(",", ":")).encode("utf-8")).hexdigest()
            response = device_studio.scenes.from_preparation(preparation_id=request["preparationId"], expected_bundle_sha256=request["expectedBundleSha256"], license_id=request["licenseId"], attribution=request["attribution"], source_label=request["sourceLabel"], allow_local_render=request.get("allowLocalRender", False), source_use_rationale=request.get("sourceUseRationale", "")).raw()
            revision = response.get("revision", {})
            photo = revision.get("candidate", {}).get("photographicBackground", {})
            if not revision.get("sceneId") or not revision.get("revision") or len(revision.get("sha256", "")) != 64 or photo.get("sourceBundleSha256") != request["preparationId"] or not response.get("libraryBundleId") or not response.get("libraryRevision") or len(response.get("libraryManifestSha256", "")) != 64 or response.get("scaleStatus") != "estimated_monocular_scale_not_measured":
                return program.fail("failed", "owner_identity_mismatch", "scene import omitted exact source/asset/revision or estimated-scale lineage", "act")
            envelope["signals"] = {"stage_state": "scene_candidate", "preparation_id": request["preparationId"], "scene_revision": {"id": revision["sceneId"], "revision": revision["revision"], "sha256": revision["sha256"]}, "library_revision": {"id": response["libraryBundleId"], "revision": response["libraryRevision"], "manifest_sha256": response["libraryManifestSha256"]}, "scale_status": response["scaleStatus"], "review_required": True}
            envelope["evidence"] = [request["preparationId"], revision["sha256"], response["libraryManifestSha256"]]
        elif action == "assess-repair":
            response = image_tools.preparation.assess_repair(mask_ref=inputs["mask_ref"], inpaint=params.get("inpaint", {})).raw()
            assessment = response.get("assessment", {})
            if not assessment.get("riskLevel"):
                return program.fail("failed", "owner_identity_mismatch", "owner returned no input/context assessment", "act")
            envelope["signals"] = {"stage_state": "assessed", "repair_assessment": assessment, "review_required": True}
            envelope["evidence"] = [inputs["mask_ref"]]
        elif action in ("assemble", "inspect-preparation", "review"):
            envelope["inputs"]["request_sha256"] = hashlib.sha256(json.dumps(request, sort_keys=True, separators=(",", ":")).encode("utf-8")).hexdigest()
            if action == "assemble":
                response = image_tools.preparation.create(manifest=request["manifest"], artifacts=request["artifacts"], label=request["label"]).raw()
            elif action == "review":
                response = image_tools.preparation.review(id=request["id"], review=request["review"], overscan_reviewed=request.get("overscanReviewed", False), operation_id=request["operationId"]).raw()
            else:
                response = image_tools.preparation.get(id=preparation_id).raw()
            p = response.get("photograph", {})
            identity = p.get("id", "")
            if len(identity) != 64 or not p.get("bundleRef") or (action == "inspect-preparation" and identity != preparation_id):
                return program.fail("failed", "owner_identity_mismatch", "owner returned no matching preparation identity/bundle", "act")
            decision = p.get("manifest", {}).get("review", {}).get("decision", "pending")
            if action == "assemble" and decision != "pending":
                return program.fail("failed", "owner_identity_mismatch", "new preparation did not start pending review", "act")
            if action == "review" and (p.get("parentId") != request["id"] or decision != request["review"].get("decision")):
                return program.fail("failed", "owner_identity_mismatch", "review does not retain the requested parent/decision", "act")
            envelope["signals"] = {"preparation_id": identity, "parent_id": p.get("parentId", ""), "bundle_ref": p["bundleRef"], "stage_state": decision, "review_required": decision != "accepted"}
            envelope["evidence"] = [identity, p["bundleRef"]]
        elif action == "submit":
            effective = dict(params)
            effective.update({"allowByok": False, "fallbackPolicy": "local_only", "allowReclaim": False})
            envelope["inputs"]["params_sha256"] = hashlib.sha256(json.dumps(effective, sort_keys=True, separators=(",", ":")).encode("utf-8")).hexdigest()
            result = image_tools.ai.submit_stored(
                operation=operation, input_ref=inputs["input_ref"],
                mask_ref=inputs.get("mask_ref", ""), operation_id=operation_id, params=effective).raw()
            identity = result.get("jobId", "")
            if not identity or result.get("modelId") != params["modelOverride"]:
                return program.fail("failed", "owner_identity_mismatch", "owner returned no matching selected model/job identity", "act")
            envelope["signals"] = {"job_id": identity, "operation_id": operation_id,
                                   "model_id": result["modelId"], "tier": result.get("tier", ""),
                                   "stage_state": "admitted", "review_required": True,
                                   "next_command": "image-tools jobs wait " + identity}
            envelope["evidence"] = [identity, operation_id]
        else:
            if action == "cancel":
                response = image_tools.jobs.cancel(id=job_id).raw()
            else:
                response = image_tools.jobs.get(id=job_id).raw()
            job = response.get("job", {})
            if job.get("id") != job_id:
                return program.fail("failed", "owner_identity_mismatch", "owner returned a different job identity", "act")
            state = job.get("state", "JOB_STATE_UNSPECIFIED")
            envelope["signals"] = {
                "job_id": job_id, "stage_state": state,
                "result_ref": job.get("resultRef", ""),
                "actual_model": job.get("resultMeta", {}).get("model", ""),
                "review_required": True,
                "cancel_terminal": action == "cancel" and state in ("JOB_STATE_CANCELED", "JOB_STATE_FAILED", "JOB_STATE_SUCCEEDED")}
            envelope["evidence"] = [job_id]
            if state == "JOB_STATE_FAILED":
                return program.fail("failed", "stage_failed", str(job.get("error", "owner inference failed"))[:180], "act")
            if state == "JOB_STATE_SUCCEEDED" and not job.get("resultRef"):
                return program.fail("failed", "owner_identity_mismatch", "succeeded stage has no retained result artifact", "act")
    except Exception as exc:
        status, klass = program.classify(exc)
        envelope["signals"]["recovery"] = {
            "action": action if action in ("submit", "assemble", "review", "inspect-preparation", "import-scene") else "observe",
            "preparation_id": preparation_id,
            "operation_id": operation_id, "job_id": job_id,
            "instruction": "Preserve the exact original inputs. Retry admission with the same identity; never mint a replacement for an unknown outcome."}
        return program.fail(status, klass, str(exc)[:180], "act")
    envelope["status"] = "ok"
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()


program.run({"validate": step_validate, "act": step_act, "report": step_report}, "validate")

"""One bounded public-owner step for Launch Video's Device Studio handoff."""
import json

NAME = "device-studio.launch-video-shot"
VERSION = "1"
OUTPUT_PROFILES = {
    "RENDER_MODE_STILL": "RENDER_OUTPUT_PROFILE_PNG_V1",
    "RENDER_MODE_VIDEO_SAMPLE_PNG": "RENDER_OUTPUT_PROFILE_PNG_V1",
    "RENDER_MODE_VIDEO_FINAL_MP4": "RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1",
    "RENDER_MODE_PHYSICAL_STILL_PNG": "RENDER_OUTPUT_PROFILE_PNG_V1",
    "RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4": "RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1",
    "RENDER_MODE_PHYSICAL_PREVIEW_PNG": "RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1",
}
envelope = {"program": NAME, "version": VERSION, "status": "failed", "phase": "validate",
            "signals": {}, "errors": [], "evidence": []}


def fail(kind, detail, status="partial"):
    envelope["status"] = status
    envelope["phase"] = "report"
    envelope["errors"].append({"class": kind, "where": "owner", "detail": str(detail)[:240]})


def first(handle):
    rows = handle.head(1)
    return rows[0] if rows else {}


def owner_revision(value):
    if not isinstance(value, str) or not value.isascii() or not value.isdecimal():
        return None
    parsed = int(value)
    return parsed if parsed > 0 and str(parsed) == value else None


def job_fields(job):
    result = {key: job.get(key) for key in (
        "id", "projectId", "shotId", "shotRevision", "status", "version",
        "summary", "renderMode", "outputProfile") if key in job}
    output = job.get("output")
    if isinstance(output, dict):
        result["output"] = {key: output.get(key) for key in (
            "mediaType", "sizeBytes", "widthPixels", "heightPixels", "sha256", "contentUrl")
            if key in output}
        result["export_command"] = (
            "device-studio renders export " + str(job.get("id", "<render-id>")) +
            " --project-id " + str(job.get("projectId", "<project-id>")) + " --out <new-file>")
    return result


def main():
    try:
        inputs = program.inputs()
        action = inputs.get("action", "")
        project_id = inputs.get("project_id", "")
        if not isinstance(project_id, str) or not project_id or len(project_id) > 128:
            raise ValueError("project_id must identify one existing Device Studio project")

        if action == "save":
            shot_id, name = inputs.get("shot_id", ""), inputs.get("name", "")
            expected = inputs.get("expected_revision")
            snapshot = inputs.get("snapshot")
            if not isinstance(shot_id, str) or not shot_id or not isinstance(name, str) or not name:
                raise ValueError("save requires shot_id and name")
            if not isinstance(expected, int) or expected < 1:
                raise ValueError("save requires the positive head revision observed by the editor")
            if not isinstance(snapshot, dict) or len(json.dumps(snapshot, separators=(",", ":"))) > 65536:
                raise ValueError("snapshot must be a typed object no larger than 64 KiB")
            envelope["phase"] = "act"
            try:
                saved = first(device_studio.shots.save(
                    shot_id=shot_id, expected_revision=expected, name=name, snapshot=snapshot))
            except Exception as exc:
                fail("save_outcome_uncertain", "Inspect the same shot and head revision before retrying: " + str(exc))
                envelope["signals"]["recovery"] = {"project_id": project_id, "shot_id": shot_id,
                                                    "expected_revision": expected}
                return envelope
            revision = saved.get("revision", {}) if isinstance(saved, dict) else {}
            shot = saved.get("shot", {}) if isinstance(saved, dict) else {}
            revnum = owner_revision(revision.get("revision")) if isinstance(revision, dict) else None
            if revnum is None or not isinstance(revision.get("recipeSha256"), str):
                fail("owner_rejected", "SaveShot returned no immutable revision or recipe digest", "failed")
                return envelope
            if shot.get("projectId") != project_id or shot.get("id") != shot_id:
                fail("owner_rejected", "SaveShot response identity does not match the requested project and shot", "failed")
                return envelope
            try:
                reopened = first(device_studio.shots.revision_get(shot_id=shot_id, revision=revnum))
            except Exception as exc:
                fail("owner_read_failed", "Save completed; reopen the exact revision before continuing: " + str(exc))
                envelope["signals"]["recovery"] = {"project_id": project_id, "shot_id": shot_id,
                                                    "revision": revnum, "recipe_sha256": revision.get("recipeSha256")}
                return envelope
            exact = reopened.get("revision", {}) if isinstance(reopened, dict) else {}
            if exact.get("shotId") != shot_id or owner_revision(exact.get("revision")) != revnum or exact.get("recipeSha256") != revision.get("recipeSha256"):
                fail("revision_mismatch", "The exact saved revision did not reopen with the same owner digest", "failed")
                return envelope
            envelope["status"] = "ok"
            envelope["phase"] = "report"
            envelope["signals"]["shot"] = {
                "project_id": project_id, "shot_id": shot_id, "revision": revnum,
                "recipe_sha256": exact.get("recipeSha256"), "scene_sha256": exact.get("sceneSha256"),
                "presentation_sha256": exact.get("presentationSha256"),
                "plate": exact.get("plate"), "screen": exact.get("screen"),
                "reopened": True}
            envelope["evidence"] = [project_id, shot_id, str(revnum), str(exact.get("recipeSha256"))]

        elif action == "submit":
            shot_id, revision = inputs.get("shot_id", ""), inputs.get("revision")
            key, mode = inputs.get("idempotency_key", ""), inputs.get("render_mode", "")
            if not isinstance(shot_id, str) or not shot_id or not isinstance(revision, int) or revision < 1:
                raise ValueError("submit requires the exact saved shot_id and positive revision")
            if not isinstance(key, str) or not key or len(key) > 200:
                raise ValueError("submit requires a stable owner idempotency_key (maximum 200 characters)")
            if mode not in ("RENDER_MODE_STILL", "RENDER_MODE_VIDEO_SAMPLE_PNG", "RENDER_MODE_VIDEO_FINAL_MP4",
                            "RENDER_MODE_PHYSICAL_STILL_PNG", "RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4",
                            "RENDER_MODE_PHYSICAL_PREVIEW_PNG"):
                raise ValueError("render_mode must be one of the declared Device Studio modes")
            envelope["phase"] = "act"
            try:
                pinned = first(device_studio.shots.revision_get(shot_id=shot_id, revision=revision))
            except Exception as exc:
                fail("owner_read_failed", "No render was submitted; reopen this exact revision before retrying: " + str(exc))
                return envelope
            exact = pinned.get("revision", {}) if isinstance(pinned, dict) else {}
            if exact.get("shotId") != shot_id or owner_revision(exact.get("revision")) != revision:
                fail("revision_mismatch", "The requested immutable shot revision is unavailable", "refused")
                return envelope
            try:
                submitted = first(device_studio.renders.submit(
                    project_id=project_id, shot_id=shot_id, revision=revision,
                    idempotency_key=key, render_mode=mode, output_profile=OUTPUT_PROFILES[mode]))
            except program.RemoteError as exc:
                if exc.http_status == 400:
                    fail("owner_rejected", "SubmitRender rejected the request before accepting a job: " + str(exc), "refused")
                else:
                    fail("submit_outcome_uncertain", "Observe or retry the same revision with the same idempotency key: " + str(exc))
                    envelope["signals"]["recovery"] = {"project_id": project_id, "shot_id": shot_id,
                                                        "revision": revision, "idempotency_key": key}
                return envelope
            except Exception as exc:
                fail("submit_outcome_uncertain", "Observe or retry the same revision with the same idempotency key: " + str(exc))
                envelope["signals"]["recovery"] = {"project_id": project_id, "shot_id": shot_id,
                                                    "revision": revision, "idempotency_key": key}
                return envelope
            job = submitted.get("job", {}) if isinstance(submitted, dict) else {}
            render_id = job.get("id") if isinstance(job, dict) else ""
            if not render_id or job.get("projectId") != project_id or owner_revision(job.get("shotRevision")) != revision:
                fail("owner_rejected", "SubmitRender returned a mismatched project, job or immutable revision", "failed")
                return envelope
            try:
                reread = first(device_studio.renders.get(project_id=project_id, render_id=render_id))
            except Exception as exc:
                fail("owner_read_failed", "Submission returned a job; reconcile this exact job: " + str(exc))
                envelope["signals"]["recovery"] = {"action": "observe", "project_id": project_id,
                                                    "render_id": render_id, "idempotency_key": key}
                return envelope
            current = reread.get("job", {}) if isinstance(reread, dict) else {}
            if current.get("id") != render_id or current.get("projectId") != project_id:
                fail("owner_read_failed", "The submitted job could not be reread under its exact project identity")
                return envelope
            envelope["status"] = "ok"
            envelope["phase"] = "report"
            envelope["signals"]["shot"] = {"project_id": project_id, "shot_id": shot_id,
                                             "revision": revision, "recipe_sha256": exact.get("recipeSha256")}
            envelope["signals"]["render"] = job_fields(current)
            envelope["signals"]["recovery"] = {"action": "observe", "project_id": project_id,
                                                 "render_id": render_id, "idempotency_key": key}
            envelope["evidence"] = [project_id, shot_id, str(revision), str(exact.get("recipeSha256")), render_id]

        elif action in ("observe", "cancel"):
            render_id = inputs.get("render_id", "")
            if not isinstance(render_id, str) or not render_id:
                raise ValueError(action + " requires render_id")
            envelope["phase"] = "act"
            try:
                if action == "cancel":
                    version = inputs.get("expected_version")
                    if not isinstance(version, int) or version < 1:
                        raise ValueError("cancel requires the current positive expected_version")
                    response = first(device_studio.renders.cancel(
                        render_id=render_id, project_id=project_id, expected_version=version))
                else:
                    response = first(device_studio.renders.get(project_id=project_id, render_id=render_id))
            except Exception as exc:
                fail("owner_cancel_uncertain" if action == "cancel" else "owner_read_failed", str(exc))
                envelope["signals"]["recovery"] = {"action": "observe", "project_id": project_id,
                                                    "render_id": render_id}
                return envelope
            job = response.get("job", {}) if isinstance(response, dict) else {}
            if job.get("id") != render_id or job.get("projectId") != project_id:
                fail("owner_read_failed", "Render status does not match the requested project and job")
                return envelope
            envelope["status"] = "ok"
            envelope["phase"] = "report"
            envelope["signals"]["render"] = job_fields(job)
            envelope["evidence"] = [project_id, render_id, str(job.get("version", "")), str(job.get("status", ""))]
        else:
            raise ValueError("action must be save, submit, observe or cancel")
    except Exception as exc:
        fail("invalid_input", exc, "refused")
    return envelope

print(json.dumps(main(), ensure_ascii=False, separators=(",", ":")))

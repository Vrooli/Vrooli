import contextlib
import io
import json
from pathlib import Path
import re
import unittest


ROOT = Path(__file__).resolve().parents[5]
PROGRAM = ROOT / "scenarios/device-studio/.vrooli/program-runtime/launch-video-shot.py"
CONTRACT = ROOT / "scenarios/device-studio/.vrooli/program-runtime/launch-video-shot.json"


class Handle:
    def __init__(self, payload):
        self.payload = payload

    def head(self, count):
        return [self.payload][:count]


class RemoteError(Exception):
    def __init__(self, message, http_status):
        super().__init__(message)
        self.http_status = http_status


class Binding:
    def __init__(self, calls, responses, path=""):
        self.calls, self.responses, self.path = calls, responses, path

    def __getattr__(self, part):
        path = part if not self.path else self.path + "." + part
        return Binding(self.calls, self.responses, path)

    def __call__(self, **kwargs):
        self.calls.append((self.path, kwargs))
        response = self.responses[self.path]
        if isinstance(response, Exception):
            raise response
        return Handle(response)


def execute(inputs, responses):
    calls = []
    namespace = {"program": type("Inputs", (), {"inputs": staticmethod(lambda: inputs), "RemoteError": RemoteError})(),
                 "device_studio": Binding(calls, responses, "device_studio")}
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(PROGRAM.read_text(), str(PROGRAM), "exec"), namespace)
    return json.loads(output.getvalue()), calls


class LaunchVideoShotTests(unittest.TestCase):
    def test_sandbox_compatible_early_return_does_not_use_system_exit(self):
        source = PROGRAM.read_text()
        self.assertIn("def main():", source)
        self.assertNotIn("SystemExit", source)

    def test_submit_revision_is_declared_and_owner_int64_fixture_is_canonical(self):
        contract = json.loads(CONTRACT.read_text())
        self.assertEqual("integer", contract["inputs"]["revision"]["type"])
        self.assertIn('inputs.get("revision")', PROGRAM.read_text())
        source = PROGRAM.read_text()
        self.assertNotIn("lib.device_studio", source)
        declared = {item["id"] for item in contract["bindings"]}
        for group, method in re.findall(r"device_studio\.(shots|renders)\.([a-z_]+)", source):
            self.assertIn(f"device-studio/{group}/{method.replace('_', '-')}", declared)

    def test_save_reopens_exact_owner_revision_before_reporting(self):
        revision = {"shotId": "s", "revision": "4", "recipeSha256": "a" * 64,
                    "sceneSha256": "b" * 64, "presentationSha256": "c" * 64}
        result, calls = execute(
            {"action": "save", "project_id": "p", "shot_id": "s", "expected_revision": 3,
             "name": "Launch phone", "snapshot": {"scene": {"plateMediaId": "m"}}},
            {"device_studio.shots.save": {"shot": {"id": "s", "projectId": "p"}, "revision": revision},
             "device_studio.shots.revision_get": {"revision": revision}})
        self.assertEqual("ok", result["status"])
        self.assertTrue(result["signals"]["shot"]["reopened"])
        self.assertEqual("a" * 64, result["signals"]["shot"]["recipe_sha256"])
        self.assertEqual(["device_studio.shots.save", "device_studio.shots.revision_get"],
                         [call[0] for call in calls])
        self.assertEqual("s", calls[0][1]["shot_id"])
        self.assertEqual(3, calls[0][1]["expected_revision"])

    def test_submit_pins_exact_revision_and_reconciles_same_job_once(self):
        revision = {"shotId": "s", "revision": "4", "recipeSha256": "a" * 64}
        job = {"id": "r", "projectId": "p", "shotId": "s", "shotRevision": "4",
               "status": "RENDER_JOB_STATUS_QUEUED", "version": 1}
        result, calls = execute(
            {"action": "submit", "project_id": "p", "shot_id": "s", "revision": 4,
             "idempotency_key": "stable-key", "render_mode": "RENDER_MODE_VIDEO_FINAL_MP4"},
            {"device_studio.shots.revision_get": {"revision": revision},
             "device_studio.renders.submit": {"job": job},
             "device_studio.renders.get": {"job": job}})
        self.assertEqual("ok", result["status"])
        self.assertEqual("r", result["signals"]["render"]["id"])
        self.assertEqual("stable-key", result["signals"]["recovery"]["idempotency_key"])
        submit = next(kwargs for path, kwargs in calls if path.endswith("renders.submit"))
        self.assertEqual(4, submit["revision"])
        self.assertEqual("RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1", submit["output_profile"])
        self.assertEqual("stable-key", submit["idempotency_key"])
        reread = next(kwargs for path, kwargs in calls if path.endswith("renders.get"))
        self.assertEqual("r", reread["render_id"])

    def test_submit_rejects_noncanonical_owner_revision(self):
        result, calls = execute(
            {"action": "submit", "project_id": "p", "shot_id": "s", "revision": 4,
             "idempotency_key": "stable-key", "render_mode": "RENDER_MODE_VIDEO_FINAL_MP4"},
            {"device_studio.shots.revision_get": {"revision": {"shotId": "s", "revision": "04"}}})
        self.assertEqual("refused", result["status"])
        self.assertEqual("revision_mismatch", result["errors"][0]["class"])
        self.assertEqual(1, len(calls))

    def test_each_declared_mode_uses_its_canonical_owner_output_profile(self):
        profiles = {
            "RENDER_MODE_STILL": "RENDER_OUTPUT_PROFILE_PNG_V1",
            "RENDER_MODE_VIDEO_SAMPLE_PNG": "RENDER_OUTPUT_PROFILE_PNG_V1",
            "RENDER_MODE_VIDEO_FINAL_MP4": "RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1",
            "RENDER_MODE_PHYSICAL_STILL_PNG": "RENDER_OUTPUT_PROFILE_PNG_V1",
            "RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4": "RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1",
            "RENDER_MODE_PHYSICAL_PREVIEW_PNG": "RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1",
        }
        for mode, profile in profiles.items():
            with self.subTest(mode=mode):
                revision = {"shotId": "s", "revision": "4", "recipeSha256": "a" * 64}
                job = {"id": "r", "projectId": "p", "shotRevision": "4", "version": 1}
                _, calls = execute(
                    {"action": "submit", "project_id": "p", "shot_id": "s", "revision": 4,
                     "idempotency_key": "stable-key", "render_mode": mode},
                    {"device_studio.shots.revision_get": {"revision": revision},
                     "device_studio.renders.submit": {"job": job},
                     "device_studio.renders.get": {"job": job}})
                submit = next(kwargs for path, kwargs in calls if path == "device_studio.renders.submit")
                self.assertEqual(profile, submit["output_profile"])

    def test_typed_http_400_is_definitive_owner_rejection(self):
        revision = {"shotId": "s", "revision": "4", "recipeSha256": "a" * 64}
        result, calls = execute(
            {"action": "submit", "project_id": "p", "shot_id": "s", "revision": 4,
             "idempotency_key": "stable-key", "render_mode": "RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4"},
            {"device_studio.shots.revision_get": {"revision": revision},
             "device_studio.renders.submit": RemoteError("invalid render mode", 400)})
        self.assertEqual("refused", result["status"])
        self.assertEqual("owner_rejected", result["errors"][0]["class"])
        self.assertNotIn("recovery", result["signals"])
        self.assertEqual(2, len(calls))

    def test_only_succeeded_job_offers_owner_verified_export_command(self):
        job = {"id": "r", "projectId": "p", "status": "RENDER_JOB_STATUS_SUCCEEDED",
               "version": 3, "output": {"mediaType": "video/mp4", "sizeBytes": 9216,
                                         "sha256": "d" * 64, "contentUrl": "/opaque"}}
        result, _ = execute({"action": "observe", "project_id": "p", "render_id": "r"},
                            {"device_studio.renders.get": {"job": job}})
        self.assertEqual("ok", result["status"])
        self.assertEqual("d" * 64, result["signals"]["render"]["output"]["sha256"])
        self.assertIn("--out <new-file>", result["signals"]["render"]["export_command"])

    def test_submit_transport_error_preserves_same_key_as_uncertain(self):
        revision = {"shotId": "s", "revision": "4", "recipeSha256": "a" * 64}
        result, calls = execute(
            {"action": "submit", "project_id": "p", "shot_id": "s", "revision": 4,
             "idempotency_key": "stable-key", "render_mode": "RENDER_MODE_VIDEO_FINAL_MP4"},
            {"device_studio.shots.revision_get": {"revision": revision},
             "device_studio.renders.submit": RuntimeError("connection lost")})
        self.assertEqual("partial", result["status"])
        self.assertEqual("submit_outcome_uncertain", result["errors"][0]["class"])
        self.assertEqual("stable-key", result["signals"]["recovery"]["idempotency_key"])
        self.assertEqual(2, len(calls))

    def test_cancel_uses_owner_expected_version_without_loop(self):
        job = {"id": "r", "projectId": "p", "status": "RENDER_JOB_STATUS_CANCEL_REQUESTED", "version": 8}
        result, calls = execute({"action": "cancel", "project_id": "p", "render_id": "r", "expected_version": 7},
                                {"device_studio.renders.cancel": {"job": job}})
        self.assertEqual("ok", result["status"])
        self.assertEqual(7, calls[0][1]["expected_version"])
        self.assertEqual("r", calls[0][1]["render_id"])
        self.assertEqual(1, len(calls))


if __name__ == "__main__":
    unittest.main()

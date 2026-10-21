import contextlib
import io
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import curate


class CurationTests(unittest.TestCase):
    def test_family_routes_match_current_public_starter_registry(self):
        family = next(x for x in curate.MANIFEST["examples"] if x["id"] == "family-showcase")
        expected = ["ipad-screen-uv-oriented-180", "generic-book-fold", "generic-round-watch-v2",
                    "monitor-screen-mapped-v2", "generic-television-concept-v3",
                    "generic-smart-display-concept-v4", "generic-digital-frame-concept-v4-glassless",
                    "generic-digital-frame-wall-v4-glassless", "generic-vehicle-dashboard-carplay-sim",
                    "generic-vehicle-dashboard-android-auto-sim"]
        self.assertEqual(family["starter_ids"], expected)
        self.assertEqual(set(family["source_assignments"]), set(expected))
        self.assertIn("candidate-only", family["qualification"])

    def test_private_archive_absence_does_not_block_source_verification(self):
        with tempfile.TemporaryDirectory() as empty_repo, patch.object(curate, "REPO_ROOT", Path(empty_repo)):
            output = io.StringIO()
            with contextlib.redirect_stdout(output):
                curate.verify_files()
        self.assertIn("archived references not present", output.getvalue())
        self.assertIn("no output claim", output.getvalue())

    def test_existing_shot_pin_reopens_without_duplicate_create(self):
        snapshot = {"presentation": {"sourceMediaId": "media-1"}}
        state = {"shots": {"photo:hero": {"id": "shot-1", "revision": 1}}}
        calls = []

        def rpc(_service, _owner, method, _body):
            calls.append(method)
            if method == "GetShot":
                return {"head": {"revision": "1"}}
            if method == "GetShotRevision":
                return {"revision": {"snapshot": snapshot}}
            if method == "CompileShot":
                return {"compiled": {"renderEligible": True, "recipeSha256": "r" * 64,
                                      "screen": {"id": "media-1", "sha256": "s" * 64}}}
            raise AssertionError(f"unexpected mutation or RPC: {method}")

        with patch.object(curate, "pages", return_value=[{"id": "shot-1", "name": "hero"}]), \
             patch.object(curate, "rpc", side_effect=rpc), patch.object(curate, "write_state"):
            result = curate.create_shot("project-1", "photo", "hero", snapshot, state)

        self.assertEqual(result, "shot-1")
        self.assertNotIn("CreateShot", calls)
        self.assertTrue(state["shots"]["photo:hero"]["compiled"])
        self.assertEqual(state["shots"]["photo:hero"]["sourceMediaSha256"], ["s" * 64])

    def test_existing_physical_preset_resumes_with_app_only_phone_presentation(self):
        example = next(x for x in curate.MANIFEST["examples"] if x["id"] == "phone-rotation")
        motion = {"id": example["motion_id"], "revision": example["motion_revision"], "sha256": example["motion_sha256"]}
        state = {"presets": {"ds-demo-phone-rotation": {"revision": 1, "sha256": "p" * 64}}, "shots": {}}
        row = {"revision": 1, "sha256": "p" * 64, "setup": {"motion": motion}}
        captured = {}

        def create(_project, example_id, name, snapshot, current_state):
            captured["snapshot"] = snapshot
            current_state["shots"][f"{example_id}:{name}"] = {"id": "shot-1"}
            return "shot-1"

        with patch.object(curate, "save_motion", return_value=motion), \
             patch.object(curate, "preset", return_value=row), \
             patch.object(curate, "initializer", side_effect=AssertionError("starter must not rerun")), \
             patch.object(curate, "save_preset", side_effect=AssertionError("preset must not be duplicated")), \
             patch.object(curate, "create_shot", side_effect=create), patch.object(curate, "write_state"):
            result = curate.store_physical_demo(example, "project-1", "app-frame", state)

        self.assertEqual(result, "shot-1")
        presentation = captured["snapshot"]["presentation"]
        self.assertEqual(presentation["sourceMediaId"], "app-frame")
        self.assertEqual(presentation["captureArea"], "CAPTURE_AREA_APP_ONLY")
        self.assertEqual(presentation["kind"], "PRESENTATION_KIND_PHONE_SYSTEM")
        self.assertEqual(captured["snapshot"]["physicalBinding"]["motion"], motion)
        self.assertEqual(state["shots"]["phone-rotation:Ten second phone yaw"]["preset"]["id"], "ds-demo-phone-rotation")


if __name__ == "__main__":
    unittest.main()

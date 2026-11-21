import contextlib
import io
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import curate


class CurationTests(unittest.TestCase):
    def test_watch_synthetic_revision_only_changes_source_and_scene_binding(self):
        source = {"sourceMediaId": "old", "kind": "PRESENTATION_KIND_PLAIN"}
        physical = {"device": {"id": "watch", "revision": 2},
                    "scene": {"id": "old-scene", "revision": 1},
                    "backend": "PHYSICAL_RENDER_BACKEND_OPTIX"}
        old = {"presentation": source, "physicalBinding": physical,
               "motionMode": "MOTION_MODE_FIXED"}
        before = json.loads(json.dumps(old))
        result = curate.watch_synthetic_snapshot(old, "fresh-watch-media")

        self.assertEqual(old, before)
        self.assertEqual(result["presentation"]["sourceMediaId"], "fresh-watch-media")
        self.assertEqual(result["presentation"]["kind"], "PRESENTATION_KIND_PLAIN")
        self.assertEqual(result["physicalBinding"]["scene"], {
            "id": curate.WATCH_SYNTHETIC_SCENE["id"],
            "revision": 2,
            "sha256": curate.WATCH_SYNTHETIC_SCENE["sha256"],
        })
        self.assertEqual(result["physicalBinding"]["device"], physical["device"])
        self.assertEqual(result["physicalBinding"]["backend"], "PHYSICAL_RENDER_BACKEND_OPTIX")
        self.assertEqual(result["motionMode"], "MOTION_MODE_FIXED")

    def test_watch_revision4_checkpoint_reopens_without_duplicate_candidate_writes(self):
        key = "family-showcase:Candidate 03 — generic-round-watch-v2"
        media = {"id": "watch-media", "sha256": curate.WATCH_SYNTHETIC_SHOT["source_sha256"]}
        snapshot = {
            "presentation": {"sourceMediaId": media["id"], "kind": "PRESENTATION_KIND_PLAIN"},
            "physicalBinding": {"scene": {"id": curate.WATCH_SYNTHETIC_SCENE["id"],
                "revision": curate.WATCH_SYNTHETIC_SCENE["revision"],
                "sha256": curate.WATCH_SYNTHETIC_SCENE["sha256"]}},
        }
        state = {"shots": {key: {"id": curate.WATCH_SYNTHETIC_SHOT["id"], "revision": 4,
                    "recipeSha256": curate.WATCH_SYNTHETIC_SHOT["recipe_sha256"]}},
                 "familyRevision3": {key: {"accepted": {"targetSnapshotSha256": "prior-r3-pin"}}}}
        calls = []

        def rpc(_service, _owner, method, _body):
            calls.append(method)
            if method == "GetSceneHead":
                return {"revision": {"revision": 2, "sha256": curate.WATCH_SYNTHETIC_SCENE["sha256"]}}
            if method == "GetPhysicalShotPresetHead":
                return {"revision": {"revision": 3, "sha256": curate.WATCH_SYNTHETIC_PRESET["sha256"]}}
            if method == "GetShot":
                return {"head": {"revision": 4, "recipeSha256": curate.WATCH_SYNTHETIC_SHOT["recipe_sha256"]}}
            if method == "GetShotRevision":
                return {"revision": {"snapshot": snapshot}}
            if method == "CompileShot":
                return {"compiled": {"renderEligible": True,
                    "recipeSha256": curate.WATCH_SYNTHETIC_SHOT["recipe_sha256"]}}
            raise AssertionError(f"accepted watch-r4 checkpoint must not write through {method}")

        with patch.object(curate, "rpc", side_effect=rpc), \
             patch.object(curate, "ensure_watch_synthetic_media", return_value=media), \
             patch.object(curate, "write_state"):
            result = curate.append_watch_synthetic_revision4("project-1", state)

        self.assertEqual(result["revision"], 4)
        self.assertEqual(result["recipeSha256"], curate.WATCH_SYNTHETIC_SHOT["recipe_sha256"])
        self.assertIn("familyWatchRevision4", state)
        self.assertFalse({"SaveSceneCandidate", "SavePhysicalShotPreset", "SaveShot"} & set(calls))

    def test_optix_revision3_is_only_a_backend_change_and_preserves_input(self):
        saved = {"presentation": {"sourceMediaId": "screen"}, "physicalBinding": {"device": {"id": "d"}, "backend": "PHYSICAL_RENDER_BACKEND_CPU"}, "motionMode": "MOTION_MODE_FIXED"}
        original = json.loads(json.dumps(saved))
        result = curate.optix_revision3_snapshot(saved)
        self.assertEqual(saved, original)
        self.assertEqual(result["physicalBinding"]["device"], saved["physicalBinding"]["device"])
        self.assertEqual(result["presentation"], saved["presentation"])
        self.assertEqual(result["motionMode"], saved["motionMode"])
        self.assertEqual(result["physicalBinding"]["backend"], "PHYSICAL_RENDER_BACKEND_OPTIX")

    def test_revision2_append_reopens_compiles_and_preserves_revision1(self):
        snapshot = {"presentation": {"sourceMediaId": "old-media"}, "physicalBinding": {"device": {"id": "old"}}}
        state = {"schema": "device-studio-demo-run-state/v1", "shots": {
            "family-showcase:Candidate 02 — generic-book-fold": {"id": "shot-2", "revision": 1, "compiled": True, "recipeSha256": "old-recipe"}}}
        setup = {"device": {"id": "device-v2", "revision": 2, "sha256": "d" * 64},
                 "scene": {"id": "scene-v1", "revision": 1, "sha256": "s" * 64},
                 "motion": {"id": "motion-v1", "revision": 1, "sha256": "m" * 64}, "displayId": "main"}
        route = {"setup": setup, "preset": {"id": "preset-v2", "revision": 2, "sha256": "p" * 64}}
        source = {"id": "media-new", "sha256": "a" * 64}
        calls = []
        with patch.object(curate, "display_dimensions", return_value=(1200, 1200)):
            target = curate.family_revision2_snapshot(snapshot, setup, source["id"])
        target_hash = curate.digest_json(target)

        def rpc(_service, _owner, method, body):
            calls.append(method)
            if method == "GetShot": return {"head": {"revision": "1", "recipeSha256": "old-recipe"}}
            if method == "GetShotRevision" and int(body["revision"]) == 1: return {"revision": {"snapshot": snapshot}}
            if method == "SaveShot": return {"revision": {"revision": "2"}}
            if method == "GetShotRevision" and int(body["revision"]) == 2: return {"revision": {"snapshot": target}}
            if method == "CompileShot": return {"compiled": {"renderEligible": True, "recipeSha256": "new-recipe"}}
            raise AssertionError(f"unexpected API call {method}")

        with patch.object(curate, "rpc", side_effect=rpc), patch.object(curate, "display_dimensions", return_value=(1200, 1200)), patch.object(curate, "write_state"):
            result = curate.append_family_revision2("project-1", state, "generic-book-fold", route, source)

        self.assertEqual(result["revision"], 2)
        self.assertEqual(state["schema"], "device-studio-demo-run-state/v2")
        pin = state["shots"]["family-showcase:Candidate 02 — generic-book-fold"]
        self.assertEqual(pin["history"][0]["revision"], 1)
        self.assertEqual(pin["recipeSha256"], "new-recipe")
        self.assertEqual(state["familyRevision2"]["family-showcase:Candidate 02 — generic-book-fold"]["accepted"]["targetSnapshotSha256"], target_hash)
        self.assertEqual(calls.count("SaveShot"), 1)

    def test_revision2_exact_pending_retry_does_not_write_twice(self):
        setup = {"device": {"id": "d", "revision": 2, "sha256": "d" * 64}, "scene": {"id": "s", "revision": 1, "sha256": "s" * 64}, "motion": {"id": "m", "revision": 1, "sha256": "m" * 64}}
        source = {"id": "media", "sha256": "a" * 64}
        old = {"presentation": {"sourceMediaId": "old"}}
        with patch.object(curate, "display_dimensions", return_value=(1200, 1200)):
            target = curate.family_revision2_snapshot(old, setup, source["id"])
        key = "family-showcase:Candidate 02 — generic-book-fold"
        state = {"shots": {key: {"id": "shot", "revision": 1, "recipeSha256": "old"}},
                 "familyRevision2": {key: {"expectedRevision": 1, "targetRevision": 2,
                    "targetSnapshotSha256": curate.digest_json(target), "preset": {"id": "preset"},
                    "sourceMediaId": "media", "sourceMediaSha256": source["sha256"]}}}
        calls = []

        def rpc(_service, _owner, method, body):
            calls.append(method)
            if method == "GetShot": return {"head": {"revision": "2"}}
            if method == "GetShotRevision": return {"revision": {"snapshot": target}}
            if method == "CompileShot": return {"compiled": {"renderEligible": True, "recipeSha256": "new"}}
            raise AssertionError(f"retry must not mutate via {method}")

        with patch.object(curate, "rpc", side_effect=rpc), patch.object(curate, "display_dimensions", return_value=(1200, 1200)), patch.object(curate, "write_state"):
            result = curate.append_family_revision2("project-1", state, "generic-book-fold", {"setup": setup, "preset": {"id": "preset"}}, source)
        self.assertEqual(result["revision"], 2)
        self.assertNotIn("SaveShot", calls)

    def test_accepted_revision2_rerun_reopens_without_mutation(self):
        setup = {"device": {"id": "d", "revision": 2, "sha256": "d" * 64}, "scene": {"id": "s", "revision": 1, "sha256": "s" * 64}, "motion": {"id": "m", "revision": 1, "sha256": "m" * 64}}
        snapshot = {"presentation": {"sourceMediaId": "media"}, "physicalBinding": setup}
        source = {"id": "media", "sha256": "a" * 64}
        key = "family-showcase:Candidate 02 — generic-book-fold"
        state = {"shots": {key: {"id": "shot", "revision": 2, "compiled": True, "recipeSha256": "recipe"}},
                 "familyRevision2": {key: {"accepted": {"revision": 2, "recipeSha256": "recipe",
                    "targetSnapshotSha256": curate.digest_json(snapshot)}}}}
        calls = []

        def rpc(_service, _owner, method, body):
            calls.append(method)
            if method == "GetShot": return {"head": {"revision": "2", "recipeSha256": "recipe"}}
            if method == "GetShotRevision": return {"revision": {"snapshot": snapshot}}
            if method == "CompileShot": return {"compiled": {"renderEligible": True, "recipeSha256": "recipe"}}
            raise AssertionError(f"accepted rerun must not mutate via {method}")

        with patch.object(curate, "rpc", side_effect=rpc), patch.object(curate, "write_state"):
            result = curate.append_family_revision2("project-1", state, "generic-book-fold", {"setup": setup, "preset": {"id": "preset"}}, source)
        self.assertTrue(result["resumed"])
        self.assertNotIn("SaveShot", calls)

    def test_uncheckpointed_revision2_is_not_adopted_as_user_content(self):
        key = "family-showcase:Candidate 02 — generic-book-fold"
        state = {"shots": {key: {"id": "shot", "revision": 1}}}
        with patch.object(curate, "rpc", return_value={"head": {"revision": "2"}}):
            with self.assertRaisesRegex(curate.Stop, "without this curator's durable pending checkpoint"):
                curate.append_family_revision2("project-1", state, "generic-book-fold", {}, {})

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
        self.assertEqual(curate.FAMILY_REVISION2["ipad-screen-uv-oriented-180"],
                         ("ipad-portrait-cupfree-fit-v2", "tablet_app"))
        self.assertEqual(curate.FAMILY_REVISION2["monitor-screen-mapped-v2"],
                         ("innerscene-monitor-study-hero-screenmap-v2", "monitor_app"))
        self.assertEqual(family["source_assignments"]["ipad-screen-uv-oriented-180"], "tablet_app")
        self.assertEqual(family["source_assignments"]["monitor-screen-mapped-v2"], "monitor_app")

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

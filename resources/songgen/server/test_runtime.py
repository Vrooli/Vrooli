import copy
import fcntl
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

from runtime import Refused, Runtime, canonical, write_json


class RuntimeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.data = self.root / "data"
        self.artifact = self.root / "artifact"
        (self.artifact / "source").mkdir(parents=True)
        self.bundle = self.data / "models/example"
        self.bundle.mkdir(parents=True)
        self.weight = self.bundle / "model.bin"
        self.weight.write_bytes(b"test-model")
        self.profile = {"model_root": "models/example", "model_dir": "models/example", "model_id": "test/model", "revision": "pinned-revision",
                        "source_revision": "pinned-source", "license": "test-license", "sung_lyrics": True, "voice_reference": False,
                        "max_duration": 150, "min_steps": 0, "max_steps": 0, "min_guidance": 1, "max_guidance": 1,
                        "max_execution_seconds": 10, "retention_days": 30,
                        "files": [{"path": "models/example/model.bin", "bytes": 10, "sha256": hashlib.sha256(b"test-model").hexdigest()}]}
        write_json(self.artifact / "source/profiles.json", {"schema_version": 1, "profiles": {"example": self.profile}})
        self.runtime = Runtime(self.data, self.root / "state", self.artifact, self.root / "data.lock", now=lambda: 40 * 86400)
        self.request = {"profile": "example", "prompt": "Authored groove", "lyrics": "[Chorus]\nA short hook",
                        "duration": 1.0, "seed": 0, "steps": 0, "guidance": 1.0,
                        "expected_revision": "pinned-revision", "timeout_seconds": 5}

    def usage(self, **changes):
        entry = {"last_used": 0, "installed_observed_at": 0,
                 "installation_fingerprint": self.runtime.installation_fingerprint(self.profile),
                 "pinned": False, "revision": "pinned-revision", **changes}
        write_json(self.runtime.usage_path, {"example": entry})

    def test_readiness_tracks_model_availability_without_claiming_residency(self):
        self.assertEqual(self.runtime.health()["status"], "ready")
        self.assertFalse(self.runtime.health()["model_loaded"])
        self.weight.unlink()
        self.assertEqual(self.runtime.health()["status"], "unavailable")
        self.assertFalse(self.runtime.health()["workload_active"])

    def test_request_rejects_unsupported_or_nonfinite_controls(self):
        for field, value in [("guidance", float("nan")), ("duration", float("inf")), ("seed", True),
                             ("steps", 1), ("seed", -1), ("expected_revision", "other"), ("lyrics", ""), ("prompt", " ")]:
            with self.subTest(field=field, value=value), self.assertRaises(Refused):
                self.runtime.validate({**self.request, field: value})
        self.assertEqual(self.runtime.validate(self.request), self.profile)

    def test_external_score_is_bounded_inline_abc_and_native_only(self):
        score = 'X:1\nM:4/4\nK:C\nC4 D4 E4 F4|\n'
        request = {**self.request, 'score': score}
        with self.assertRaisesRegex(Refused, 'only by YuE2'):
            self.runtime.validate(request)
        self.runtime.profiles['example']['backend'] = 'yue2-native'
        self.assertEqual(self.runtime.validate(request), self.runtime.profiles['example'])
        for value in ['', '/etc/passwd', 'https://example.invalid/score.abc', [],
                      score + '\U0001f3b5'*4096]:
            with self.subTest(value_type=type(value).__name__), self.assertRaisesRegex(Refused, 'inline ABC'):
                self.runtime.validate({**request, 'score': value})
        with self.assertRaises(Refused):
            self.runtime.validate({**request, 'score_file': '/etc/passwd'})

    def test_last_use_age_whole_bundle_and_pin_control_cleanup(self):
        self.usage()
        preview = self.runtime.models({})
        self.assertTrue(preview["items"][0]["eligible"])
        # A pin overrides age and invalidates an earlier preview.
        self.runtime.models({"pin": {"example": True}})
        with self.assertRaisesRegex(Refused, "preview changed"):
            self.runtime.models({"dry_run": False, "preview": preview})
        self.assertTrue(self.bundle.exists())
        self.runtime.models({"pin": {"example": False}})
        preview = self.runtime.models({})
        exported = self.data / "operator.wav"
        exported.write_bytes(b"keep")
        result = self.runtime.models({"dry_run": False, "preview": preview})
        self.assertEqual(result["removed_profiles"], ["example"])
        self.assertFalse(self.bundle.exists())
        self.assertEqual(exported.read_bytes(), b"keep")

    def test_partial_deletion_is_recoverable_with_fresh_preview(self):
        self.usage()
        preview = self.runtime.models({})
        def fail_partial(root):
            self.weight.unlink()
            raise OSError("simulated disk failure")
        with patch("runtime.shutil.rmtree", side_effect=fail_partial), self.assertRaises(OSError):
            self.runtime.models({"dry_run": False, "preview": preview})
        # No hidden retirement directory. A fresh preview can finish this exact
        # declared bundle even though it is no longer a complete installation.
        self.assertEqual(list(self.bundle.parent.iterdir()), [self.bundle])
        preview = self.runtime.models({})
        self.assertTrue(preview["items"][0]["eligible"])
        self.runtime.models({"dry_run": False, "preview": preview})
        self.assertFalse(self.bundle.exists())

    def test_reacquired_bundle_cannot_inherit_partial_retirement_intent(self):
        self.usage()
        preview = self.runtime.models({})
        def fail_partial(root):
            self.weight.unlink()
            raise OSError("simulated disk failure")
        with patch("runtime.shutil.rmtree", side_effect=fail_partial), self.assertRaises(OSError):
            self.runtime.models({"dry_run": False, "preview": preview})
        # An acquisition can complete while this supervisor stays alive. Its
        # newly complete bundle must not inherit the interrupted deletion age.
        self.weight.write_bytes(b"test-model")
        self.assertFalse(self.runtime.models({})["items"][0]["eligible"])
        self.runtime.observe_installations()
        fresh = self.runtime.models({})["items"][0]
        self.assertFalse(fresh["eligible"])
        self.assertEqual(fresh["installed_observed_at"], 40 * 86400)

    def test_generation_sweeps_only_expired_peer_profiles(self):
        peer = copy.deepcopy(self.profile)
        peer["model_root"] = peer["model_dir"] = "models/peer"
        peer["files"][0]["path"] = "models/peer/model.bin"
        (self.data / "models/peer").mkdir()
        (self.data / peer["files"][0]["path"]).write_bytes(b"test-model")
        self.runtime.profiles["peer"] = peer
        self.usage()
        usage = self.runtime.usage()
        usage["peer"] = {**usage["example"], "installation_fingerprint": self.runtime.installation_fingerprint(peer)}
        write_json(self.runtime.usage_path, usage)
        self.fake_worker()
        self.runtime.generate(self.request)
        self.assertTrue(self.bundle.exists())
        self.assertFalse((self.data / peer["model_root"]).exists())

    def test_unknown_or_recent_use_never_becomes_eligible(self):
        for entry in [None, {"last_used": 39 * 86400}, {"revision": "different"}]:
            with self.subTest(entry=entry):
                if entry is None:
                    self.runtime.usage_path.unlink(missing_ok=True)
                else:
                    self.usage(**entry)
                preview = self.runtime.models({})
                self.assertFalse(preview["items"][0]["eligible"])

    def test_corrupt_usage_and_incomplete_bundles_refuse_cleanup(self):
        self.runtime.usage_path.write_text('{"example":{"last_used":NaN}}')
        with self.assertRaisesRegex(Refused, "usage state is corrupt"):
            self.runtime.models({})
        self.usage()
        self.weight.unlink()
        self.assertFalse(self.runtime.models({})["items"][0]["eligible"])

    def test_changed_bundle_and_unexpected_files_protect_data(self):
        self.usage()
        preview = self.runtime.models({})
        self.weight.write_bytes(b"new-model!")
        with self.assertRaisesRegex(Refused, "preview changed"):
            self.runtime.models({"dry_run": False, "preview": preview})
        (self.bundle / "operator-notes.txt").write_text("keep")
        self.assertFalse(self.runtime.models({})["items"][0]["eligible"])

    def test_acquisition_and_generation_fence_cleanup(self):
        self.usage()
        with self.runtime.guard:
            with self.assertRaisesRegex(Refused, "generation is active"):
                self.runtime.models({})
        with self.runtime.lock_path.open("a+b") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            with self.assertRaisesRegex(Refused, "being acquired"):
                self.runtime.models({})
            with self.assertRaisesRegex(Refused, "being acquired"):
                self.runtime.generate(self.request)
        self.assertIsNone(self.runtime.active)
        self.assertFalse(self.runtime.guard.locked())

    def test_symlink_bundle_is_never_followed_or_removed(self):
        outside = self.root / "outside"
        self.bundle.rename(outside)
        self.bundle.symlink_to(outside, target_is_directory=True)
        with self.assertRaisesRegex(Refused, "bundle root"):
            self.runtime.models({})
        self.assertTrue((outside / "model.bin").exists())

    def test_first_install_and_reacquisition_get_fresh_retention_window(self):
        self.assertFalse(self.runtime.models({})["items"][0]["eligible"])
        self.usage()
        self.assertTrue(self.runtime.models({})["items"][0]["eligible"])
        self.weight.unlink()
        self.weight.write_bytes(b"test-model")
        self.assertFalse(self.runtime.models({})["items"][0]["eligible"])
        self.runtime.observe_installations()
        preview = self.runtime.models({})
        self.assertFalse(preview["items"][0]["eligible"])
        self.assertEqual(preview["items"][0]["installed_observed_at"], 40 * 86400)

    def fake_worker(self, *, sleep=False, wrong_seed=False):
        source = """
import json, pathlib, sys, time, wave
stage=pathlib.Path(sys.argv[1]); job=json.loads((stage/'job.json').read_text())
if SLEEP: time.sleep(30)
with wave.open(str(stage/'audio.wav'),'wb') as w:
 w.setnchannels(1);w.setsampwidth(2);w.setframerate(16000);w.writeframes(b'\\0\\0'*1600)
result={'seed':job['request']['seed']+WRONG,'model_id':job['profile']['model_id'],'revision':job['profile']['revision']}
(stage/'result.json').write_text(json.dumps(result))
""".replace("SLEEP", repr(sleep)).replace("WRONG", repr(int(wrong_seed)))
        def popen(argv, **kwargs):
            return subprocess.Popen([sys.executable, "-c", source, argv[-1]], **kwargs)
        self.runtime.popen = popen

    def test_completed_generation_binds_exact_request_and_audio(self):
        self.fake_worker()
        raw = json.dumps(self.request, ensure_ascii=False).encode()
        result = self.runtime.generate(self.request, raw=raw)
        self.assertEqual(result["request_sha256"], hashlib.sha256(raw).hexdigest())
        final = self.runtime.outputs / result["operation_id"]
        self.assertEqual(hashlib.sha256((final / "audio.wav").read_bytes()).hexdigest(), result["output"]["sha256"])
        self.assertEqual(result["output"]["frames"], 1600)
        self.assertFalse((final / ".active").exists())
        self.assertEqual(self.runtime.usage()["example"]["last_used"], 40 * 86400)
        self.assertIsNone(self.runtime.process)

    def test_worker_identity_failure_cannot_publish(self):
        self.fake_worker(wrong_seed=True)
        with self.assertRaisesRegex(Refused, "mismatched"):
            self.runtime.generate(self.request)
        self.assertEqual(list(self.runtime.outputs.iterdir()), [])

    def test_disconnect_kills_and_reaps_worker_before_next_job(self):
        self.fake_worker(sleep=True)
        with self.assertRaisesRegex(Refused, "cancelled"):
            self.runtime.generate(self.request, disconnected=lambda: True)
        self.assertIsNone(self.runtime.process)
        self.assertFalse(self.runtime.guard.locked())
        self.assertEqual(list(self.runtime.outputs.iterdir()), [])
        self.fake_worker()
        self.assertEqual(self.runtime.generate(self.request)["status"], "completed")


if __name__ == "__main__":
    unittest.main()

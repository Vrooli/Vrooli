"""No-model tests of real HTTP handlers, filesystem publication and offline loader.

Torch/upstream/audio encoding are lightweight fakes. No model weights, GPU or
inference runtime are loaded; these checks do not qualify the native closure.
"""
from __future__ import annotations

import asyncio
import copy
import hashlib
import json
import os
from pathlib import Path
import tempfile
import threading
import time
import types
import unittest
from unittest.mock import patch
import wave

import httpx
import numpy as np

import loader
import server


def config():
    return {"sample_rate": 44100, "sample_size": 5292032, "model": {
        "conditioning": {"configs": [{"type": "t5gemma", "id": "text", "config": {
            "model_name": "google/t5gemma-b-b-ul2", "repo_id": "remote", "subfolder": "nested", "max_length": 128}}]},
        "diffusion": {"diffusion_objective": "rf_denoiser", "sampling_distribution_shift_options": {"type": "test"}}}}


def closure(root: Path):
    model = root / "model"
    model.mkdir()
    files = []
    for name in loader.REQUIRED_FILES:
        data = json.dumps(config()).encode() if name == "model_config.json" else b"reviewed-test-closure"
        path = model / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        files.append({"path": name, "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()})
    lock = root / "model-lock.json"
    lock.write_text(json.dumps({"model_id": loader.MODEL_ID, "model_revision": loader.MODEL_REVISION,
                                "source_revision": loader.SOURCE_REVISION, "license": "test-only", "files": files}))
    return model, lock


class LoaderTests(unittest.TestCase):
    def test_complete_closure_and_path_only_normalization(self):
        with tempfile.TemporaryDirectory() as temp:
            model, lock = closure(Path(temp))
            witness = loader.verify_inputs(model, lock)
            self.assertEqual(len(witness["files"]), 12)
            original = config()
            before = copy.deepcopy(original)
            effective = loader.local_config(original, model)
            text = effective["model"]["conditioning"]["configs"][0]["config"]
            self.assertEqual(text["model_path"], str(model / "t5gemma-b-b-ul2"))
            self.assertNotIn("repo_id", text)
            self.assertNotIn("subfolder", text)
            self.assertEqual(original, before)
            (model / "NOTICE").write_bytes(b"corrupt")
            with self.assertRaisesRegex(ValueError, "checksum/size"):
                loader.verify_inputs(model, lock)

    def test_missing_unqualified_symlink_and_fifo_replacement_refused(self):
        with tempfile.TemporaryDirectory() as temp:
            model, lock = closure(Path(temp))
            entry = json.loads(lock.read_text())
            entry["files"][0]["sha256"] = ""
            lock.write_text(json.dumps(entry))
            with self.assertRaisesRegex(ValueError, "unqualified"):
                loader.verify_inputs(model, lock)
            path = model / "NOTICE"
            path.unlink()
            path.symlink_to(model / "LICENSE.md")
            with self.assertRaisesRegex(ValueError, "regular"):
                loader.digest_file(path)
            path.unlink()
            path.write_bytes(b"regular-before-open")
            real_open = os.open

            def replacement(name, flags, *args):
                path.unlink()
                os.mkfifo(path)
                return real_open(name, flags, *args)

            with patch.object(loader.os, "open", replacement):
                with self.assertRaisesRegex(ValueError, "changed"):
                    loader.digest_file(path)

    def test_cpu_half_precedes_cuda_and_allocator_ceiling_precedes_load(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            model, lock = closure(root)
            for name in ["server.py", "loader.py", "requirements.lock"]:
                (root / name).write_text("test-only source identity")
            order = []
            inner = types.SimpleNamespace(to=lambda device: order.append(("move", device)),
                model=types.SimpleNamespace(parameters=lambda: iter([types.SimpleNamespace(dtype="torch.float16")])),
                conditioner=types.SimpleNamespace(conditioners={"text": types.SimpleNamespace(model=types.SimpleNamespace(
                    parameters=lambda: iter([types.SimpleNamespace(dtype="torch.float32")])))}))
            cuda = types.SimpleNamespace(is_available=lambda: True, reset_peak_memory_stats=lambda: None,
                synchronize=lambda: None, max_memory_allocated=lambda: 5, max_memory_reserved=lambda: 7,
                get_device_name=lambda: "fake GPU", get_device_properties=lambda i: types.SimpleNamespace(total_memory=16*1024**3),
                set_per_process_memory_fraction=lambda amount, device: order.append(("ceiling", amount, device)))
            torch = types.SimpleNamespace(cuda=cuda, version=types.SimpleNamespace(cuda="12.6"),
                                          __version__="2.7.1+cu126", _C=types.SimpleNamespace(_GLIBCXX_USE_CXX11_ABI=True))
            fa = types.SimpleNamespace(__version__="2.8.3.post2", flash_attn_func=lambda: None, flash_attn_varlen_func=lambda: None)

            def load(effective, checkpoint, **kwargs):
                order.append(("load", kwargs))
                settings = effective["model"]["conditioning"]["configs"][0]["config"]
                self.assertEqual(settings["model_path"], str(model / "t5gemma-b-b-ul2"))
                self.assertNotIn("repo_id", settings)
                self.assertEqual(os.environ["HF_HUB_OFFLINE"], "1")
                return inner

            modules = {"torch": torch, "flash_attn": fa,
                "stable_audio_3.loading_utils": types.SimpleNamespace(load_diffusion_cond=load),
                "stable_audio_3.model": types.SimpleNamespace(StableAudioModel=lambda *args: args),
                "stable_audio_3.models": types.SimpleNamespace(transformer=types.SimpleNamespace(
                    flash_attn_func=fa.flash_attn_func, flash_attn_varlen_func=fa.flash_attn_varlen_func, index_first_axis=lambda: None))}
            with patch.dict("sys.modules", modules), patch.object(loader, "HERE", root), patch.dict(os.environ, {"SA3_COMPOSED_SHA256": "a"*64}), patch.object(loader.importlib.metadata, "version", lambda name: "5.10.4" if name == "transformers" else "2.8.3.post2"):
                result = loader.load_local(model, lock)
            self.assertEqual(order, [("ceiling", 0.5, 0), ("load", {"device": "cpu", "model_half": True}), ("move", "cuda")])
            self.assertFalse(inner.use_lora)
            self.assertEqual(inner.lora_names, [])
            self.assertEqual(result["text_conditioner_id"], "text")
            self.assertEqual(result["runtime"]["torch_allocator_ceiling_bytes"], 8*1024**3)
            self.assertIsNone(result["runtime"]["non_torch_peak_bytes"])
            self.assertEqual(result["runtime"]["main_parameter_dtype"], "torch.float16")
            self.assertEqual(result["runtime"]["conditioner_parameter_dtype"], "torch.float32")


class FakeTensor:
    def __init__(self, frames):
        self.audio = np.zeros((1, 2, frames), dtype=np.float32)
    def detach(self): return self
    def cpu(self): return self
    def numpy(self): return self.audio


def fake_audio_module():
    def write(path, audio, rate, **kwargs):
        with wave.open(str(path), "wb") as wav:
            wav.setnchannels(2)
            wav.setsampwidth(3)
            wav.setframerate(rate)
            wav.writeframes(b"\0" * len(audio) * 6)
    def info(path):
        with wave.open(str(path), "rb") as wav:
            return types.SimpleNamespace(frames=wav.getnframes(), channels=wav.getnchannels(), samplerate=wav.getframerate(), subtype="PCM_24")
    return types.SimpleNamespace(write=write, info=info)


class ServerTests(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.calls = []
        self.seed = None
        self.entered, self.release = threading.Event(), threading.Event()
        self.block, self.failure, self.limit = False, False, 128
        self.cuda = types.SimpleNamespace(synchronize=lambda: None, reset_peak_memory_stats=lambda: None,
                                         max_memory_allocated=lambda: 123, max_memory_reserved=lambda: 456)
        torch = types.SimpleNamespace(cuda=self.cuda, initial_seed=lambda: self.seed)

        def generate(**kwargs):
            self.calls.append(kwargs)
            self.seed = kwargs["seed"]
            self.entered.set()
            if self.block:
                if not self.release.wait(3):
                    raise RuntimeError("fake generation release timed out")
            kwargs["callback"]()
            if self.failure:
                raise RuntimeError("fake inference failure")
            return FakeTensor(int(kwargs["duration"]*44100))

        conditioner = types.SimpleNamespace(tokenizer=lambda text, **kwargs: {"input_ids": list(range(len(text.split())))}, max_length=self.limit)
        self.conditioner = conditioner
        shift_type = type("LogSNRShift", (), {"__module__": "stable_audio_3.inference.distribution_shift"})
        shift = shift_type()
        shift.rate, shift.anchor_logsnr, shift.logsnr_end = 0, -6.2, 2.0
        self.state = {"model": types.SimpleNamespace(generate=generate, model=types.SimpleNamespace(
            conditioner=types.SimpleNamespace(conditioners={"text": conditioner}), diffusion_objective="rf_denoiser", sampling_dist_shift=shift)),
            "config": config(), "identity": {"model_id": loader.MODEL_ID, "model_revision": loader.MODEL_REVISION},
            "runtime": {"test_fake": True}, "text_conditioner_id": "text", "effective_config_sha256": "b"*64,
            "initialization": {"seconds": 0}}
        self.patches = [patch.object(server, "OUTPUT_DIR", self.root), patch.object(server, "_loaded", self.state),
                        patch.object(server, "_init_error", ""), patch.dict("sys.modules", {"torch": torch, "soundfile": fake_audio_module()})]
        for item in self.patches: item.start()
        self.client = httpx.AsyncClient(transport=httpx.ASGITransport(app=server.app), base_url="http://resource")

    async def asyncTearDown(self):
        self.release.set()
        for _ in range(100):
            if not server._generation.locked(): break
            await asyncio.sleep(.01)
        await self.client.aclose()
        for item in reversed(self.patches): item.stop()
        self.temp.cleanup()

    async def test_real_http_success_exact_seed_and_atomic_receipt_audio(self):
        raw = b'{"prompt":"original instrumental","duration":1,"seed":0}'
        response = await self.client.post("/v1/compose", content=raw)
        self.assertEqual(response.status_code, 200, response.text)
        receipt = response.json()
        self.assertEqual(receipt["request_sha256"], hashlib.sha256(raw).hexdigest())
        self.assertEqual(receipt["sampling"]["seed"], 0)
        self.assertFalse(receipt["sampling"]["prompt_truncated"])
        self.assertEqual(receipt["sampling"]["loaded_sampling_shift"]["parameters"]["rate"], 0)
        self.assertIs(self.calls[0]["dist_shift"], self.state["model"].model.sampling_dist_shift)
        self.assertIsNone(receipt["measurements"]["whole_scope_peak_ram_bytes"])
        directory = self.root / receipt["operation_id"]
        self.assertEqual(json.loads((directory / "generation-receipt.json").read_bytes()), receipt)
        downloaded = await self.client.get(f"/v1/outputs/{receipt['operation_id']}/audio")
        self.assertEqual(hashlib.sha256(downloaded.content).hexdigest(), receipt["output"]["sha256"])
        self.assertEqual(receipt["output"]["frames"], 44100)
        self.assertEqual(list(self.root.glob(".pending-*")), [])
        self.assertFalse(server._generation.locked())

    async def test_invalid_and_truncated_prompts_never_infer(self):
        cases = [{"prompt": "cue", "seed": -1}, {"prompt": "cue", "seed": True}, {"prompt": "cue", "seed": 2**32},
                 {"prompt": "cue"}, {"prompt": "cue", "seed": 0, "duration": True},
                 {"prompt": "cue", "seed": 0, "cfg_scale": float("inf")}, {"prompt": "cue", "seed": 0, "lyrics": "unexpected"}]
        for data in cases:
            response = await self.client.post("/v1/compose", content=json.dumps(data))
            self.assertEqual(response.status_code, 400, response.text)
        self.conditioner.max_length = 1
        response = await self.client.post("/v1/compose", json={"prompt": "two words", "seed": 0})
        self.assertEqual(response.status_code, 400, response.text)
        self.assertEqual(self.calls, [])
        self.assertFalse(server._generation.locked())
        response = await self.client.post("/v1/compose", content=b"x" * 65537)
        self.assertEqual(response.status_code, 413)

    async def test_busy_exclusion_cancellation_drains_without_publication(self):
        self.block = True
        pending = asyncio.create_task(self.client.post("/v1/compose", json={"prompt": "cue", "seed": 7, "duration": 1}))
        for _ in range(100):
            if self.entered.is_set(): break
            await asyncio.sleep(.01)
        self.assertTrue(self.entered.is_set())
        busy = await self.client.post("/v1/compose", json={"prompt": "cue", "seed": 99})
        self.assertEqual(busy.status_code, 503)
        self.assertEqual(len(self.calls), 1)
        pending.cancel()
        with self.assertRaises(asyncio.CancelledError): await pending
        self.assertTrue(server._generation.locked())
        self.assertEqual((await self.client.get("/health")).json()["active_operation"]["state"], "draining")
        self.release.set()
        for _ in range(100):
            if not server._generation.locked(): break
            await asyncio.sleep(.01)
        self.assertFalse(server._generation.locked())
        self.assertEqual(list(self.root.iterdir()), [])
        self.assertIsNone((await self.client.get("/health")).json()["active_operation"])

    async def test_error_deadline_and_collision_release_without_overwrite(self):
        self.failure = True
        failed = await self.client.post("/v1/compose", json={"prompt": "cue", "seed": 0, "duration": 1})
        self.assertEqual(failed.status_code, 500)
        self.assertFalse(server._generation.locked())
        self.failure = False
        identity = "a" * 32
        existing = self.root / identity
        existing.mkdir()
        (existing / "prior").write_bytes(b"preserved")
        with patch.object(server.uuid, "uuid4", lambda: types.SimpleNamespace(hex=identity)):
            failed = await self.client.post("/v1/compose", json={"prompt": "cue", "seed": 0, "duration": 1})
        self.assertEqual(failed.status_code, 500)
        self.assertEqual((existing / "prior").read_bytes(), b"preserved")
        self.assertEqual(list(self.root.glob(".pending-*")), [])
        self.assertFalse(server._generation.locked())
        server._generation.acquire()
        with self.assertRaises(server.GenerationCancelled):
            server.run_generation(server.ComposeRequest(prompt="cue", seed=0), b"{}", "b"*32, threading.Event(), time.monotonic()-1)
        self.assertFalse(server._generation.locked())

    async def test_readiness_and_atomic_empty_destination_race(self):
        with patch.object(server, "_loaded", None), patch.object(server, "_init_error", "loader failed"):
            health = await self.client.get("/health")
            self.assertEqual(health.status_code, 503)
            self.assertFalse(health.json()["model_loaded"])
            response = await self.client.post("/v1/compose", json={"prompt": "cue", "seed": 0})
            self.assertEqual(response.status_code, 503)
        stage, final = self.root / "stage", self.root / "final"
        stage.mkdir(); final.mkdir()
        with self.assertRaises(FileExistsError): server.publish_directory(stage, final)
        self.assertTrue(stage.is_dir())
        self.assertEqual(list(final.iterdir()), [])

    async def test_actual_handler_deadline_stops_at_callback_and_releases(self):
        self.block = True
        pending = asyncio.create_task(self.client.post("/v1/compose", json={
            "prompt": "cue", "seed": 0, "duration": 1, "timeout_seconds": 1}))
        for _ in range(100):
            if self.entered.is_set(): break
            await asyncio.sleep(.01)
        self.assertTrue(self.entered.is_set())
        await asyncio.sleep(1.02)
        self.release.set()
        response = await pending
        self.assertEqual(response.status_code, 408, response.text)
        self.assertIn("deadline", response.text)
        self.assertFalse(server._generation.locked())
        self.assertEqual(list(self.root.iterdir()), [])

    async def test_wrong_upstream_shape_and_effective_seed_refuse_publication(self):
        original = self.state["model"].generate
        for condition in ["shape", "seed"]:
            def wrong(**kwargs):
                output = original(**kwargs)
                if condition == "shape":
                    output.audio = output.audio[:, :, :-1]
                else:
                    self.seed = kwargs["seed"] + 1
                return output
            self.state["model"].generate = wrong
            response = await self.client.post("/v1/compose", json={"prompt": "cue", "duration": 1, "seed": 0})
            self.assertEqual(response.status_code, 500, response.text)
            self.assertFalse(server._generation.locked())
            self.assertEqual(list(self.root.iterdir()), [])


if __name__ == "__main__":
    unittest.main(verbosity=2)

"""Resource contracts use actual loader/compose code with fake model classes.

Sampling regressions use CPU Torch without model inference. No real server
startup, model construction, weight acquisition or GPU work is performed.
"""
import ast
import hashlib
import math
import re
import secrets
import json
from pathlib import Path
import sys
import tempfile
import threading
import time
import types
import unittest
from unittest.mock import patch
from typing import Literal

from pydantic import BaseModel, Field, ValidationError

# Load CPU Torch outside temporary sys.modules inference fixtures: importing it
# inside patch.dict would remove its Python modules while retaining C state.
import torch


def request_model():
    tree = ast.parse(Path(__file__).with_name("server.py").read_text())
    model = next(node for node in tree.body if isinstance(node, ast.ClassDef) and node.name == "ComposeRequest")
    namespace = {"BaseModel": BaseModel, "Field": Field, "Literal": Literal, "secrets": secrets}
    exec(compile(ast.Module(body=[model], type_ignores=[]), "server.py", "exec"), namespace)
    return namespace["ComposeRequest"]


def compose_fixture(identity):
    tree = ast.parse(Path(__file__).with_name("server.py").read_text())
    function = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "compose")
    function.decorator_list = []

    class Refusal(Exception):
        def __init__(self, status_code, detail):
            self.status_code, self.detail = status_code, detail

    class Response:
        def __init__(self, path, **kwargs):
            self.path, self.headers = path, {}

    namespace = {"ComposeRequest": object, "FileResponse": Response, "_ensure_loaded": lambda: None,
                 "_loaded_identity": identity, "_generation_lock": threading.Lock(), "HTTPException": Refusal, "Path": Path, "time": time,
                 "VARIANT": "acestep-v15-turbo", "PLANNER": "acestep-5Hz-lm-0.6B", "_loaded_planner": "acestep-5Hz-lm-0.6B", "_dit": object(), "_llm": object(), "hashlib": hashlib, "math": math, "re": re}
    helpers = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name in {"_sampling_headers", "_profile_name", "_language_headers"}]
    exec(compile(ast.Module(body=helpers+[function], type_ignores=[]), "server.py", "exec"), namespace)
    return namespace, Refusal


class OutputIdentityTest(unittest.TestCase):
    def test_health_readiness_requires_both_models_and_no_initialization_error(self):
        from fastapi import FastAPI, Response
        from fastapi.testclient import TestClient
        tree = ast.parse(Path(__file__).with_name("server.py").read_text())
        function = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "health")
        for dit, llm, error, expected_status, expected_state in [
            (None, object(), "", 503, "initializing"),
            (object(), None, "", 503, "initializing"),
            (object(), object(), "model initialization failed", 503, "degraded"),
            (object(), object(), "", 200, "ok"),
        ]:
            with self.subTest(status=expected_status, state=expected_state, dit=dit is not None, llm=llm is not None):
                app = FastAPI()
                namespace = {"app": app, "Response": Response, "_dit": dit, "_llm": llm,
                             "_init_error": error, "VARIANT": "fake-turbo", "PLANNER": "fake-planner",
                             "DEVICE": "cuda", "OFFLOAD_DIT": True, "_init_seconds": 12.5, "_loaded_identity": None, "_loaded_planner": None}
                helpers = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "_profile_name"]
                exec(compile(ast.Module(body=helpers+[function], type_ignores=[]), "server.py", "exec"), namespace)
                with TestClient(app) as client:
                    response = client.get("/health")
                self.assertEqual(response.status_code, expected_status)
                body = response.json()
                self.assertEqual(body["status"], expected_state)
                self.assertEqual(body["error"], error)
                self.assertEqual(body["model_loaded"], dit is not None and llm is not None)
                self.assertEqual(body["initialization_seconds"], 12.5)
                self.assertEqual(body["device"], "cuda")
                self.assertEqual(body["explicit_language"], {"version": "explicit-v1", "languages": ["en", "ko"], "variant": "acestep-v15-sft", "planner": "acestep-5Hz-lm-1.7B", "rewrite_allowed": False})

    def test_identity_records_loaded_variant_and_rung_despite_later_desired_change(self):
        source = Path(__file__).with_name("server.py").read_text()
        tree = ast.parse(source)
        function = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "_ensure_loaded")
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            namespace = {"ROOT": root, "CHECKPOINT_DIR": root / "checkpoints", "VARIANT": "acestep-v15-sft", "PLANNER": "planner", "DEVICE": "cpu", "OFFLOAD_DIT": False, "_dit": None, "_llm": None, "_device_keepalive": None, "_loaded_identity": None, "_lock": threading.Lock(), "json": json, "time": time}

            class FakeDiT:
                def initialize_service(self, **kwargs):
                    self.loaded_offload = kwargs["offload_dit_to_cpu"]
                    self.loaded_variant = kwargs["config_path"]
                    namespace["OFFLOAD_DIT"] = True
                    namespace["VARIANT"] = "acestep-v15-turbo"
                    namespace["PLANNER"] = "changed-after-snapshot"
                    return "ok", True

            class FakeLLM:
                def initialize(self, **kwargs):
                    self.loaded_planner = kwargs["lm_model_path"]
                    return "ok", True

            handler = types.ModuleType("acestep.handler")
            handler.AceStepHandler = FakeDiT
            planner = types.ModuleType("acestep.llm_inference")
            planner.LLMHandler = FakeLLM
            with patch.dict(sys.modules, {"acestep.handler": handler, "acestep.llm_inference": planner}):
                exec(compile(ast.Module(body=[function], type_ignores=[]), "server.py", "exec"), namespace)
                namespace["_ensure_loaded"]()
            self.assertFalse(namespace["_dit"].loaded_offload)
            self.assertEqual(namespace["_dit"].loaded_variant, "acestep-v15-sft")
            self.assertEqual(namespace["_llm"].loaded_planner, "planner")
            self.assertEqual(namespace["_loaded_planner"], "planner")
            self.assertEqual(namespace["_loaded_identity"], ("ACE-Step/Ace-Step1.5:acestep-v15-sft", "full"))
            namespace["_ensure_loaded"]()
            self.assertEqual(namespace["_loaded_identity"][1], "full")

    def test_failed_final_initialization_does_not_publish_loaded_identity(self):
        tree = ast.parse(Path(__file__).with_name("server.py").read_text())
        function = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "_ensure_loaded")
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            namespace = {"ROOT": root, "CHECKPOINT_DIR": root / "checkpoints", "VARIANT": "acestep-v15-turbo", "PLANNER": "acestep-5Hz-lm-0.6B", "DEVICE": "cuda", "OFFLOAD_DIT": False, "_dit": None, "_llm": None, "_device_keepalive": None, "_loaded_identity": None, "_loaded_planner": None, "_lock": threading.Lock(), "json": json, "time": time}
            class FakeModel:
                def initialize_service(self, **kwargs):
                    return "ok", True
                def initialize(self, **kwargs):
                    return "ok", True
            handler, planner, torch = types.ModuleType("acestep.handler"), types.ModuleType("acestep.llm_inference"), types.ModuleType("torch")
            handler.AceStepHandler, planner.LLMHandler = FakeModel, FakeModel
            def failed_keepalive(*args, **kwargs):
                raise RuntimeError("fake CUDA initialization failure")
            torch.empty = failed_keepalive
            with patch.dict(sys.modules, {"acestep.handler": handler, "acestep.llm_inference": planner, "torch": torch}):
                exec(compile(ast.Module(body=[function], type_ignores=[]), "server.py", "exec"), namespace)
                with self.assertRaisesRegex(RuntimeError, "fake CUDA"):
                    namespace["_ensure_loaded"]()
            self.assertIsNone(namespace["_dit"])
            self.assertIsNone(namespace["_llm"])
            self.assertIsNone(namespace["_loaded_identity"])
            self.assertIsNone(namespace["_loaded_planner"])
            self.assertIn("fake CUDA", namespace["_init_error"])

    def test_degrade_refuses_acknowledgement_during_inflight_full_load(self):
        tree = ast.parse(Path(__file__).with_name("server.py").read_text())
        functions = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name in {"_ensure_loaded", "degrade"}]
        for function in functions:
            function.decorator_list = []
        entered, proceed = threading.Event(), threading.Event()

        class Refusal(Exception):
            def __init__(self, status_code, detail):
                self.status_code, self.detail = status_code, detail

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            namespace = {"ROOT": root, "RUNG_FILE": root / "rung", "CHECKPOINT_DIR": root / "checkpoints", "VARIANT": "acestep-v15-turbo", "PLANNER": "planner", "DEVICE": "cpu", "OFFLOAD_DIT": False, "_dit": None, "_llm": None, "_device_keepalive": None, "_loaded_identity": None, "_lock": threading.Lock(), "json": json, "time": time, "HTTPException": Refusal}

            class FakeDiT:
                def initialize_service(self, **kwargs):
                    entered.set()
                    if not proceed.wait(2):
                        raise RuntimeError("test barrier deadline")
                    return "ok", True

            class FakeLLM:
                def initialize(self, **kwargs):
                    return "ok", True

            handler, planner = types.ModuleType("acestep.handler"), types.ModuleType("acestep.llm_inference")
            handler.AceStepHandler, planner.LLMHandler = FakeDiT, FakeLLM
            exec(compile(ast.Module(body=functions, type_ignores=[]), "server.py", "exec"), namespace)
            failures = []
            def load():
                try:
                    namespace["_ensure_loaded"]()
                except Exception as failure:
                    failures.append(failure)
            with patch.dict(sys.modules, {"acestep.handler": handler, "acestep.llm_inference": planner}):
                worker = threading.Thread(target=load)
                worker.start()
                try:
                    self.assertTrue(entered.wait(2))
                    with self.assertRaises(Refusal) as refusal:
                        namespace["degrade"]("offload-dit")
                    self.assertEqual(refusal.exception.status_code, 503)
                finally:
                    proceed.set()
                    worker.join(2)
                self.assertFalse(worker.is_alive())
            self.assertFalse(failures)
            self.assertEqual(namespace["_loaded_identity"][1], "full")
            acknowledgement = namespace["degrade"]("offload-dit")
            self.assertTrue(acknowledgement["restart_required"])
            self.assertEqual(acknowledgement["applied_rung"], "full")

    def test_incompatible_admission_refuses_before_importing_inference(self):
        tree = ast.parse(Path(__file__).with_name("server.py").read_text())
        function = next(node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name == "compose")
        function.decorator_list = []
        class Refusal(Exception):
            def __init__(self, status_code, detail):
                self.status_code, self.detail = status_code, detail
        namespace = {"ComposeRequest": object, "FileResponse": object, "_ensure_loaded": lambda: None, "_loaded_identity": ("model", "full"), "HTTPException": Refusal}
        exec(compile(ast.Module(body=[function], type_ignores=[]), "server.py", "exec"), namespace)
        # No acestep inference module is installed/provided by this fixture.
        # The mismatch must refuse before that import or any generation call.
        with self.assertRaises(Refusal) as refusal:
            namespace["compose"](types.SimpleNamespace(expected_rung="offload-dit"))
        self.assertIn("inference refused", refusal.exception.detail)

    def test_admitted_rungs_validate_actual_request_model(self):
        model = request_model()
        for value in [[], ["unknown"], ["full", "offload-dit", "full"], "full", [1], {}]:
            with self.subTest(value=value), self.assertRaises(ValidationError):
                model(caption="authored", admitted_rungs=value)
        for value in [None, ["full"], ["offload-dit"], ["full", "offload-dit"]]:
            with self.subTest(value=value):
                self.assertEqual(model(caption="authored", admitted_rungs=value).admitted_rungs, value)

    def test_restrictions_intersect_before_any_inference_import(self):
        for loaded, strict, allowed in [
            ("full", None, ["offload-dit"]),
            ("offload-dit", "full", ["full", "offload-dit"]),
            ("full", "full", ["offload-dit"]),
            ("offload-dit", "full", None),
        ]:
            with self.subTest(loaded=loaded, strict=strict, allowed=allowed):
                namespace, refusal_type = compose_fixture(("actual-model", loaded))
                # Catch attempted inference imports explicitly, even on hosts
                # where a model package happens to be installed.
                real_import = __import__
                def guarded_import(name, *args, **kwargs):
                    if name == "acestep.inference":
                        self.fail("incompatible admission imported inference")
                    return real_import(name, *args, **kwargs)
                with patch("builtins.__import__", side_effect=guarded_import), self.assertRaises(refusal_type) as refusal:
                    namespace["compose"](types.SimpleNamespace(expected_rung=strict, admitted_rungs=allowed))
                self.assertEqual(refusal.exception.status_code, 503)
                self.assertIn("inference refused", refusal.exception.detail)

    def test_compatible_generation_returns_actual_offload_identity(self):
        for allowed in [["full", "offload-dit"], None]:
            with self.subTest(allowed=allowed), tempfile.TemporaryDirectory() as directory:
                namespace, _ = compose_fixture(("actual-offload-model", "offload-dit"))
                namespace["OUTPUT_DIR"] = Path(directory) / "outputs"
                audio = Path(directory) / "fake.wav"
                audio.write_bytes(b"fixture output, no model or media job")
                calls = []
                def generate(dit, llm, **kwargs):
                    calls.append(kwargs["params"])
                    return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": vars(kwargs["params"])}])
                inference = types.ModuleType("acestep.inference")
                inference.GenerationConfig = types.SimpleNamespace
                inference.GenerationParams = types.SimpleNamespace
                inference.generate_music = generate
                request = request_model()(caption="authored", seed=7, admitted_rungs=allowed)
                with patch.dict(sys.modules, {"acestep.inference": inference}):
                    response = namespace["compose"](request)
                self.assertEqual(len(calls), 1)
                self.assertEqual(calls[0].caption, "authored")
                self.assertFalse(calls[0].use_cot_caption)
                self.assertEqual(response.path, str(audio))
                self.assertEqual(response.headers["X-ACE-Step-Profile-Rung"], "offload-dit")
                self.assertEqual(response.headers["X-ACE-Step-Model-ID"], "actual-offload-model")

    def test_uint32_seed_and_expected_variant_request_contract(self):
        model = request_model()
        for seed in [-1, 2**32, "7", 1.5, True]:
            with self.subTest(seed=seed), self.assertRaises(ValidationError):
                model(caption="authored", seed=seed)
        for seed in [0, 2**32-1]:
            self.assertEqual(model(caption="authored", seed=seed).seed, seed)
        for shift in [0, 6, float("nan"), float("inf")]:
            with self.subTest(shift=shift), self.assertRaises(ValidationError):
                model(caption="authored", shift=shift)
        with self.assertRaises(ValidationError):
            model(caption="authored", expected_variant="unknown")

    def test_explicit_language_request_and_legacy_presence(self):
        model = request_model()
        for language in ["", "KO", " ko", "en,ko", "ja", 7, True]:
            with self.subTest(language=language), self.assertRaises(ValidationError):
                model(caption="authored", vocal_language=language)
        for language in [None, "en", "ko"]:
            self.assertEqual(model(caption="authored", vocal_language=language).vocal_language, language)
        self.assertIsNone(model(caption="authored").vocal_language)
        with self.assertRaises(ValidationError):
            model(caption="authored", expected_planner="unknown")

    def test_explicit_language_early_guards_never_import_inference(self):
        cases = [
            ("acestep-v15-turbo", "acestep-5Hz-lm-1.7B", {}, "SFT"),
            ("acestep-v15-sft", "acestep-5Hz-lm-0.6B", {}, "1.7B"),
            ("acestep-v15-sft", "acestep-5Hz-lm-1.7B", {"rewrite_caption": True}, "rewriting"),
            ("acestep-v15-sft", "acestep-5Hz-lm-1.7B", {"expected_planner": "acestep-5Hz-lm-0.6B"}, "expected planner"),
        ]
        real_import = __import__
        for variant, planner, fields, message in cases:
            with self.subTest(variant=variant, planner=planner, fields=fields):
                namespace, refusal = compose_fixture(("ACE-Step/Ace-Step1.5:"+variant, "full"))
                namespace["_loaded_planner"] = planner
                def guarded(name, *args, **kwargs):
                    if name == "acestep.inference": self.fail("early refusal imported inference")
                    return real_import(name, *args, **kwargs)
                with patch("builtins.__import__", side_effect=guarded), self.assertRaises(refusal) as error:
                    namespace["compose"](request_model()(caption="authored", vocal_language="ko", **fields))
                self.assertIn(message, error.exception.detail)
                self.assertFalse(namespace["_generation_lock"].locked())

    def test_explicit_language_parameters_and_actual_text_witnesses(self):
        for language in ["en", "ko", None]:
            with self.subTest(language=language), tempfile.TemporaryDirectory() as directory:
                namespace, refusal = compose_fixture(("ACE-Step/Ace-Step1.5:acestep-v15-sft", "full"))
                namespace.update(_loaded_planner="acestep-5Hz-lm-1.7B", OUTPUT_DIR=Path(directory)/"outputs")
                audio = Path(directory)/"fake.wav"; audio.write_bytes(b"no inference")
                request = request_model()(caption="original scene", lyrics="불이 꺼져도 길은 남아\nKeep one light", seed=7, vocal_language=language)
                observed = []
                def generate(dit, llm, **kw):
                    actual = vars(kw["params"]).copy(); observed.append(actual)
                    return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": actual}])
                inference = types.ModuleType("acestep.inference")
                inference.GenerationConfig = types.SimpleNamespace; inference.GenerationParams = types.SimpleNamespace; inference.generate_music = generate
                with patch.dict(sys.modules, {"acestep.inference": inference}):
                    response = namespace["compose"](request)
                self.assertEqual(observed[0]["vocal_language"], language or "en")
                if language is None:
                    self.assertNotIn("use_cot_language", observed[0])
                    self.assertNotIn("X-ACE-Step-Language-Mode", response.headers)
                else:
                    for field in ["use_cot_language", "use_cot_caption", "use_cot_metas"]:
                        self.assertIs(observed[0][field], False)
                    self.assertEqual(response.headers["X-ACE-Step-Vocal-Language"], language)
                    self.assertEqual(response.headers["X-ACE-Step-Language-Mode"], "explicit-v1")
                    self.assertEqual(response.headers["X-ACE-Step-CoT-Language"], "false")
                    for field in ["caption", "lyrics"]:
                        self.assertEqual(response.headers[f"X-ACE-Step-{field.title()}-SHA256"], hashlib.sha256(getattr(request,field).encode()).hexdigest())
                    for field, wrong in [("vocal_language", "unknown"), ("use_cot_language", True), ("use_cot_caption", 0), ("use_cot_metas", None), ("caption", "rewritten"), ("lyrics", "translated")]:
                        with self.subTest(dishonest_field=field), self.assertRaises(RuntimeError):
                            namespace["_language_headers"]({"params": {**observed[0], field: wrong}}, request)
                    with self.assertRaises(RuntimeError): namespace["_language_headers"]({}, request)
                    for field in ["vocal_language", "caption", "lyrics", "use_cot_language"]:
                        missing = dict(observed[0]); missing.pop(field)
                        with self.subTest(missing=field), self.assertRaises(RuntimeError):
                            namespace["_language_headers"]({"params": missing}, request)

    def test_legacy_null_language_keeps_rewrite_defaults(self):
        with tempfile.TemporaryDirectory() as directory:
            namespace, _ = compose_fixture(("ACE-Step/Ace-Step1.5:acestep-v15-turbo", "full"))
            namespace["OUTPUT_DIR"] = Path(directory)/"outputs"
            audio = Path(directory)/"fake.wav"; audio.write_bytes(b"no inference")
            def generate(dit, llm, **kw):
                params = vars(kw["params"])
                self.assertEqual(params["vocal_language"], "en")
                self.assertTrue(params["use_cot_caption"])
                self.assertTrue(params["use_cot_metas"])
                self.assertNotIn("use_cot_language", params)
                return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": params}])
            inference = types.ModuleType("acestep.inference")
            inference.GenerationConfig = types.SimpleNamespace; inference.GenerationParams = types.SimpleNamespace; inference.generate_music = generate
            with patch.dict(sys.modules, {"acestep.inference": inference}):
                response = namespace["compose"](request_model()(caption="authored", rewrite_caption=True, vocal_language=None))
            self.assertNotIn("X-ACE-Step-Language-Mode", response.headers)

    def test_fixed_seed_profile_and_code_witness_are_actual(self):
        for variant, planner, steps, guidance, shift in [
            ("acestep-v15-turbo", "acestep-5Hz-lm-0.6B", 8, 1.0, 3.0),
            ("acestep-v15-turbo", "acestep-5Hz-lm-1.7B", 8, 1.0, 3.0),
            ("acestep-v15-sft", "acestep-5Hz-lm-1.7B", 50, 7.0, 1.0),
        ]:
            with self.subTest(variant=variant, planner=planner), tempfile.TemporaryDirectory() as directory:
                namespace, refusal = compose_fixture(("ACE-Step/Ace-Step1.5:"+variant, "full"))
                namespace.update(VARIANT=variant, PLANNER=planner, _loaded_planner=planner, OUTPUT_DIR=Path(directory)/"outputs")
                audio = Path(directory)/"fake.wav"; audio.write_bytes(b"fake media, no generation")
                code = "<|audio_code_1|><|audio_code_5|>"
                def generate(dit, llm, **kw):
                    self.assertFalse(kw["config"].use_random_seed)
                    self.assertEqual(kw["config"].seeds, [4294967295])
                    self.assertTrue(kw["params"].thinking)
                    self.assertEqual(kw["params"].shift, shift)
                    return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": {**vars(kw["params"]), "audio_codes": code}}])
                inference = types.ModuleType("acestep.inference")
                inference.GenerationConfig = types.SimpleNamespace; inference.GenerationParams = types.SimpleNamespace; inference.generate_music = generate
                req = request_model()(caption="authored", seed=4294967295, expected_variant=variant)
                with patch.dict(sys.modules, {"acestep.inference": inference}):
                    response = namespace["compose"](req)
                self.assertEqual(response.headers["X-ACE-Step-Seed"], "4294967295")
                self.assertEqual(response.headers["X-ACE-Step-Planner"], planner)
                self.assertEqual(response.headers["X-ACE-Step-Inference-Steps"], str(steps))
                self.assertEqual(float(response.headers["X-ACE-Step-Guidance-Scale"]), guidance)
                self.assertEqual(response.headers["X-ACE-Step-LM-Code-SHA256"], hashlib.sha256(code.encode()).hexdigest())
                self.assertEqual(response.headers["X-ACE-Step-LM-Code-Count"], "2")
                # Upstream output must acknowledge the exact seed/profile before publication.
                def wrong(dit, llm, **kw):
                    return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": {**vars(kw["params"]), "seed": 12}}])
                inference.generate_music = wrong
                with patch.dict(sys.modules, {"acestep.inference": inference}), self.assertRaises(refusal) as failed:
                    namespace["compose"](req)
                self.assertIn("seed", failed.exception.detail)

    def test_wrong_loaded_variant_refuses_before_inference_import(self):
        namespace, refusal = compose_fixture(("ACE-Step/Ace-Step1.5:acestep-v15-turbo", "full"))
        real_import = __import__
        def guarded(name, *args, **kwargs):
            if name == "acestep.inference": self.fail("variant mismatch imported inference")
            return real_import(name, *args, **kwargs)
        with patch("builtins.__import__", side_effect=guarded), self.assertRaises(refusal) as error:
            namespace["compose"](request_model()(caption="authored", expected_variant="acestep-v15-sft"))
        self.assertIn("variant", error.exception.detail)

    def test_real_http_compose_validates_seed_and_publishes_sampling_headers(self):
        from fastapi import FastAPI, HTTPException
        from fastapi.responses import FileResponse
        from fastapi.testclient import TestClient
        with tempfile.TemporaryDirectory() as directory:
            namespace, _ = compose_fixture(("ACE-Step/Ace-Step1.5:acestep-v15-turbo", "full"))
            namespace.update(HTTPException=HTTPException, FileResponse=FileResponse, OUTPUT_DIR=Path(directory)/"outputs")
            namespace["compose"].__annotations__.update({"request":request_model(), "return":FileResponse})
            app = FastAPI(); app.post("/v1/compose")(namespace["compose"])
            audio = Path(directory)/"fake.wav"; audio.write_bytes(b"test-only output bytes")
            inference = types.ModuleType("acestep.inference")
            inference.GenerationConfig = types.SimpleNamespace; inference.GenerationParams = types.SimpleNamespace
            inference.generate_music = lambda dit, llm, **kw: types.SimpleNamespace(success=True, audios=[{"path":str(audio),"params":vars(kw["params"])}])
            with patch.dict(sys.modules, {"acestep.inference": inference}), TestClient(app) as client:
                self.assertEqual(client.post("/v1/compose", json={"caption":"authored", "seed":-1}).status_code,422)
                mismatch = client.post("/v1/compose", json={"caption":"authored", "expected_variant":"acestep-v15-sft"})
                self.assertEqual(mismatch.status_code,503)
                response = client.post("/v1/compose", json={"caption":"authored", "seed":9, "shift":2})
                self.assertEqual(response.status_code,200,response.text)
                self.assertEqual(response.headers["X-ACE-Step-Seed"],"9")
                self.assertEqual(response.headers["X-ACE-Step-Shift"],"2")
                self.assertNotIn("X-ACE-Step-LM-Code-Count",response.headers)

    def test_real_http_sampling_headers_round_trip_valid_float_precision(self):
        from fastapi import FastAPI, HTTPException
        from fastapi.responses import FileResponse
        from fastapi.testclient import TestClient
        for variant in ["acestep-v15-turbo", "acestep-v15-sft"]:
            with self.subTest(variant=variant), tempfile.TemporaryDirectory() as directory:
                namespace, _ = compose_fixture(("ACE-Step/Ace-Step1.5:"+variant, "full"))
                namespace.update(HTTPException=HTTPException, FileResponse=FileResponse, OUTPUT_DIR=Path(directory)/"outputs")
                namespace["compose"].__annotations__.update({"request": request_model(), "return": FileResponse})
                app = FastAPI()
                app.post("/v1/compose")(namespace["compose"])
                audio = Path(directory)/"fake.wav"
                audio.write_bytes(b"test-only output bytes")
                inference = types.ModuleType("acestep.inference")
                inference.GenerationConfig = types.SimpleNamespace
                inference.GenerationParams = types.SimpleNamespace
                inference.generate_music = lambda dit, llm, **kw: types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": vars(kw["params"])}])
                shift, guidance = 1.12345678901234, 7.12345678901234
                with patch.dict(sys.modules, {"acestep.inference": inference}), TestClient(app) as client:
                    response = client.post("/v1/compose", json={"caption": "authored", "seed": 9, "shift": shift, "guidance_scale": guidance})
                self.assertEqual(response.status_code, 200, response.text)
                for header, expected in [
                    ("X-ACE-Step-Shift", shift),
                    ("X-ACE-Step-Guidance-Scale", 1.0 if "turbo" in variant else guidance),
                ]:
                    with self.subTest(header=header):
                        self.assertEqual(float(response.headers[header]), expected)

    def test_compose_cpu_sampling_repeats_seed_after_intervening_request(self):
        import torch
        with tempfile.TemporaryDirectory() as directory:
            namespace, _ = compose_fixture(("ACE-Step/Ace-Step1.5:acestep-v15-turbo", "full"))
            namespace["OUTPUT_DIR"] = Path(directory)/"outputs"
            audio = Path(directory)/"fake.wav"
            audio.write_bytes(b"test-only output bytes")
            samples = []
            def generate(dit, llm, **kw):
                tokens = torch.multinomial(torch.tensor([0.1, 0.2, 0.3, 0.4], device="cpu"), 64, replacement=True)
                sample = tokens.tolist()
                samples.append(sample)
                codes = "".join(f"<|audio_code_{token}|>" for token in sample)
                return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": {**vars(kw["params"]), "audio_codes": codes}}])
            inference = types.ModuleType("acestep.inference")
            inference.GenerationConfig = types.SimpleNamespace
            inference.GenerationParams = types.SimpleNamespace
            inference.generate_music = generate
            torch.manual_seed(12345)
            with patch.dict(sys.modules, {"acestep.inference": inference}):
                responses = [namespace["compose"](request_model()(caption="authored", seed=seed)) for seed in [7, 99, 7]]
            self.assertEqual(samples[0], samples[2])
            self.assertNotEqual(samples[0], samples[1])
            self.assertEqual(responses[0].headers["X-ACE-Step-LM-Code-SHA256"], responses[2].headers["X-ACE-Step-LM-Code-SHA256"])

    def test_generation_busy_refuses_without_reseeding_and_errors_release_lock(self):
        import torch
        from concurrent.futures import ThreadPoolExecutor
        with tempfile.TemporaryDirectory() as directory:
            namespace, refusal = compose_fixture(("ACE-Step/Ace-Step1.5:acestep-v15-turbo", "full"))
            namespace["OUTPUT_DIR"] = Path(directory)/"outputs"
            audio = Path(directory)/"fake.wav"
            audio.write_bytes(b"test-only output bytes")
            entered, release = threading.Event(), threading.Event()
            calls = []
            def generate(dit, llm, **kw):
                calls.append(kw["params"].seed)
                if len(calls) == 1:
                    entered.set()
                    if not release.wait(3):
                        raise RuntimeError("test generation barrier timed out")
                return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": vars(kw["params"])}])
            inference = types.ModuleType("acestep.inference")
            inference.GenerationConfig = types.SimpleNamespace
            inference.GenerationParams = types.SimpleNamespace
            inference.generate_music = generate
            model = request_model()
            with patch.dict(sys.modules, {"acestep.inference": inference}), patch("torch.manual_seed", wraps=torch.manual_seed) as reset, ThreadPoolExecutor(max_workers=1) as pool:
                first = pool.submit(namespace["compose"], model(caption="authored", seed=7))
                try:
                    self.assertTrue(entered.wait(2), "first request did not enter generation")
                    with self.assertRaises(refusal) as busy:
                        namespace["compose"](model(caption="authored", seed=99))
                    self.assertEqual(busy.exception.status_code, 503)
                    self.assertIn("busy", busy.exception.detail)
                    self.assertEqual(calls, [7])
                    self.assertEqual(reset.call_count, 1)
                finally:
                    release.set()
                first.result(timeout=2)
                def fail(dit, llm, **kw):
                    raise RuntimeError("test inference failure")
                inference.generate_music = fail
                with self.assertRaises(refusal) as failed:
                    namespace["compose"](model(caption="authored", seed=7))
                self.assertIn("test inference failure", failed.exception.detail)
                inference.generate_music = generate
                self.assertEqual(namespace["compose"](model(caption="authored", seed=7)).headers["X-ACE-Step-Seed"], "7")

    def test_generation_lock_releases_after_seed_reset_and_witness_failures(self):
        with tempfile.TemporaryDirectory() as directory:
            namespace, refusal = compose_fixture(("ACE-Step/Ace-Step1.5:acestep-v15-turbo", "full"))
            namespace["OUTPUT_DIR"] = Path(directory)/"outputs"
            audio = Path(directory)/"fake.wav"
            audio.write_bytes(b"test-only output bytes")
            inference = types.ModuleType("acestep.inference")
            inference.GenerationConfig = types.SimpleNamespace
            inference.GenerationParams = types.SimpleNamespace
            def success(dit, llm, **kw):
                return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": vars(kw["params"])}])
            inference.generate_music = success
            request = request_model()(caption="authored", seed=7)
            with patch.dict(sys.modules, {"acestep.inference": inference}):
                with patch("torch.manual_seed", side_effect=RuntimeError("test reset failure")):
                    with self.assertRaises(refusal) as failed:
                        namespace["compose"](request)
                self.assertIn("test reset failure", failed.exception.detail)
                namespace["compose"](request)
                def wrong_witness(dit, llm, **kw):
                    return types.SimpleNamespace(success=True, audios=[{"path": str(audio), "params": {**vars(kw["params"]), "seed": 99}}])
                inference.generate_music = wrong_witness
                with self.assertRaises(refusal) as failed:
                    namespace["compose"](request)
                self.assertIn("seed", failed.exception.detail)
                inference.generate_music = success
                namespace["compose"](request)

    def test_sampling_witness_refuses_missing_or_dishonest_result_fields(self):
        namespace, _ = compose_fixture(("model", "full"))
        witness = namespace["_sampling_headers"]
        actual = {"seed":9,"inference_steps":8,"guidance_scale":1.0,"shift":3.0}
        for field, value in [("seed",None),("seed",True),("seed","9"),("inference_steps",8.0),("inference_steps",20),("guidance_scale",float("nan")),("shift",1.0),("audio_codes","x"*(2*1024*1024+1))]:
            with self.subTest(field=field,value_type=type(value).__name__), self.assertRaises(RuntimeError):
                witness({"params":{**actual,field:value}},9,8,1.0,3.0,"acestep-5Hz-lm-0.6B")
        with self.assertRaises(RuntimeError):
            witness({},9,8,1.0,3.0,"acestep-5Hz-lm-0.6B")

    def test_health_distinguishes_configured_and_immutable_loaded_profile(self):
        from fastapi import FastAPI, Response
        from fastapi.testclient import TestClient
        tree = ast.parse(Path(__file__).with_name("server.py").read_text())
        functions = [node for node in tree.body if isinstance(node,ast.FunctionDef) and node.name in {"health","_profile_name"}]
        app=FastAPI()
        namespace={"app":app,"Response":Response,"_dit":object(),"_llm":object(),"_init_error":"","VARIANT":"acestep-v15-sft","PLANNER":"acestep-5Hz-lm-1.7B","DEVICE":"cuda","OFFLOAD_DIT":False,"_init_seconds":1.0,"_loaded_identity":("ACE-Step/Ace-Step1.5:acestep-v15-turbo","full"),"_loaded_planner":"acestep-5Hz-lm-0.6B"}
        exec(compile(ast.Module(body=functions,type_ignores=[]),"server.py","exec"),namespace)
        with TestClient(app) as client:
            response=client.get("/health")
        self.assertEqual(response.status_code,200)
        body=response.json()
        self.assertEqual(body["variant"],"acestep-v15-turbo")
        self.assertEqual(body["planner"],"acestep-5Hz-lm-0.6B")
        self.assertEqual(body["loaded_profile"],"fast")
        self.assertEqual(body["configured_profile"],"detail")

    def test_missing_direct_seed_draws_once_and_explicit_zero_is_preserved(self):
        model = request_model()
        with patch("secrets.randbits", side_effect=[17,18]) as draw:
            self.assertEqual(model(caption="authored").seed,17)
            self.assertEqual(model(caption="authored").seed,18)
            self.assertEqual(model(caption="authored",seed=0).seed,0)
            self.assertEqual(draw.call_args_list,[unittest.mock.call(32),unittest.mock.call(32)])


if __name__ == "__main__":
    unittest.main()

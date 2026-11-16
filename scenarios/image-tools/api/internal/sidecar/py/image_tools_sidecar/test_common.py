from types import SimpleNamespace

from image_tools_sidecar import _common


def test_onnx_session_options_bound_cpu_fanout(monkeypatch):
    captured = {}

    class FakeOptions:
        pass

    class FakeRuntime:
        ExecutionMode = SimpleNamespace(ORT_SEQUENTIAL="sequential")
        SessionOptions = FakeOptions

        @staticmethod
        def InferenceSession(path, sess_options, providers):
            captured.update(path=path, options=sess_options, providers=providers)
            return SimpleNamespace(get_providers=lambda: providers)

        @staticmethod
        def get_available_providers():
            return ["CUDAExecutionProvider", "CPUExecutionProvider"]

    monkeypatch.setattr(_common.os, "cpu_count", lambda: 32)
    monkeypatch.setattr(_common, "require_deps", lambda: (None, FakeRuntime, None))
    _common._SESSION_CACHE.clear()
    try:
        _common.make_session("/models/fixture.onnx")
    finally:
        _common._SESSION_CACHE.clear()

    options = captured["options"]
    assert options.intra_op_num_threads == 8
    assert options.inter_op_num_threads == 1
    assert options.execution_mode == "sequential"
    assert captured["providers"] == ["CPUExecutionProvider"]
    assert _common.onnx_thread_settings() == {
        "intra_op_num_threads": 8,
        "inter_op_num_threads": 1,
        "execution_mode": "sequential",
    }


def test_cuda_session_uses_only_admitted_provider_and_separate_cache(monkeypatch):
    captured = []

    class FakeOptions:
        pass

    class FakeRuntime:
        ExecutionMode = SimpleNamespace(ORT_SEQUENTIAL="sequential")
        SessionOptions = FakeOptions

        @staticmethod
        def get_available_providers():
            return ["CUDAExecutionProvider", "CPUExecutionProvider"]

        @staticmethod
        def InferenceSession(path, sess_options, providers):
            captured.append((path, providers))
            return SimpleNamespace(get_providers=lambda: providers)

    monkeypatch.setattr(_common, "require_deps", lambda: (None, FakeRuntime, None))
    _common._SESSION_CACHE.clear()
    try:
        monkeypatch.setenv("IMAGE_TOOLS_DEVICE", "cuda")
        cuda = _common.make_session("/models/fixture.onnx")
        monkeypatch.setenv("IMAGE_TOOLS_DEVICE", "cpu")
        cpu = _common.make_session("/models/fixture.onnx")
    finally:
        _common._SESSION_CACHE.clear()

    assert cuda is not cpu
    assert captured == [
        ("/models/fixture.onnx", ["CUDAExecutionProvider"]),
        ("/models/fixture.onnx", ["CPUExecutionProvider"]),
    ]


def test_cuda_session_fails_closed_when_provider_is_missing(monkeypatch):
    class FakeOptions:
        pass

    class FakeRuntime:
        ExecutionMode = SimpleNamespace(ORT_SEQUENTIAL="sequential")
        SessionOptions = FakeOptions

        @staticmethod
        def get_available_providers():
            return ["CPUExecutionProvider"]

    monkeypatch.setattr(_common, "require_deps", lambda: (None, FakeRuntime, None))
    _common._SESSION_CACHE.clear()
    monkeypatch.setenv("IMAGE_TOOLS_DEVICE", "cuda")
    try:
        try:
            _common.make_session("/models/fixture.onnx")
        except SystemExit as exc:
            assert exc.code == 5
        else:
            raise AssertionError("CUDA request unexpectedly fell back to CPU")
    finally:
        _common._SESSION_CACHE.clear()


def test_onnx_thread_policy_handles_unknown_cpu_count(monkeypatch):
    monkeypatch.setattr(_common.os, "cpu_count", lambda: None)
    assert _common.onnx_thread_settings()["intra_op_num_threads"] == 1

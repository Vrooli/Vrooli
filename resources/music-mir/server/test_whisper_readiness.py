import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch
from urllib.error import URLError


SERVER_PATH = Path(__file__).with_name("server.py")
spec = importlib.util.spec_from_file_location("music_mir_server", SERVER_PATH)
server = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(server)


class Response:
    def __init__(self, status):
        self.status = status

    def __enter__(self):
        return self

    def __exit__(self, *_):
        return False


class WhisperReadinessTest(unittest.TestCase):
    def test_reports_unconfigured_without_probing_non_loopback(self):
        with patch.object(server, "WHISPER_URL", "https://example.org"), patch.object(server, "urlopen") as probe:
            self.assertEqual(server.whisper_readiness(), {
                "configured": False, "ready": False, "status": "not_configured"})
            probe.assert_not_called()

    def test_reports_local_service_readiness_without_sending_audio(self):
        with patch.object(server, "WHISPER_URL", "http://127.0.0.1:8090"), patch.object(
                server, "urlopen", return_value=Response(200)) as probe:
            self.assertEqual(server.whisper_readiness(), {
                "configured": True, "ready": True, "status": "ready"})
            probe.assert_called_once_with("http://127.0.0.1:8090/", timeout=1)

    def test_reports_unreachable_local_service_as_partial_dependency(self):
        with patch.object(server, "WHISPER_URL", "http://localhost:8090"), patch.object(
                server, "urlopen", side_effect=URLError("refused")):
            self.assertEqual(server.whisper_readiness(), {
                "configured": True, "ready": False, "status": "unavailable"})


if __name__ == "__main__":
    unittest.main()

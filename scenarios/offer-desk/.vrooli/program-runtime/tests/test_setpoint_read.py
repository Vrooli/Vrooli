"""Exercise setpoint-read's typed error and owner status boundary in the real kernel."""
import hashlib
import io
import json
import sys
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[5]
sys.dont_write_bytecode = True
sys.path.insert(0, str(ROOT / "scenarios/program-runtime/kernel"))
from host.engine import SessionKernel  # noqa: E402


PROGRAM_ROOT = ROOT / "scenarios/offer-desk/.vrooli/program-runtime"
SOURCE = (PROGRAM_ROOT / "setpoint-read.py").read_text()
CONTRACT = json.loads((PROGRAM_ROOT / "setpoint-read.json").read_text())
LIBRARY = {
    "name": "setpoint-read",
    "scenario": "offer-desk",
    "contract": True,
    "current": True,
    "version": int(CONTRACT["version"]),
    "source": SOURCE,
    "declaration": CONTRACT,
    "digest": hashlib.sha256(SOURCE.encode()).hexdigest(),
}
BINDINGS = [
    {
        "id": "offer-desk/offers/board-show",
        "scenario": "offer-desk",
        "group": "offers",
        "command": "board_show",
        "effect": "read",
        "row_field_candidates": ["entries"],
        "meta_fields": ["defaultAliveGap"],
    },
    {
        "id": "offer-desk/offers/meters",
        "scenario": "offer-desk",
        "group": "offers",
        "command": "meters",
        "effect": "read",
        "row_field_candidates": ["meters"],
        "meta_fields": ["deliverableMeterGaps", "undeclaredStreams"],
    },
    {
        "id": "offer-desk/offers/space",
        "scenario": "offer-desk",
        "group": "offers",
        "command": "space",
        "effect": "read",
        "row_field_candidates": ["cells"],
        "meta_fields": ["denominatorConfidence"],
    },
    {
        "id": "vrooli-memory/learning/measure",
        "scenario": "vrooli-memory",
        "group": "learning",
        "command": "measure",
        "effect": "read",
        "row_field_candidates": ["cohorts"],
        "meta_fields": ["reliable", "reason", "truncated", "eligibleAttempts"],
    },
]
RESPONSES = {
    "offer-desk/offers/board-show": {"entries": [], "defaultAliveGap": "available"},
    "offer-desk/offers/meters": {"meters": [], "deliverableMeterGaps": [], "undeclaredStreams": []},
    "offer-desk/offers/space": {"cells": [], "denominatorConfidence": "high"},
    "vrooli-memory/learning/measure": {
        "cohorts": [],
        "reliable": True,
        "reason": "ok",
        "truncated": False,
        "eligibleAttempts": 0,
    },
}


class Response:
    def __init__(self, value):
        self.data = json.dumps(value).encode()

    def read(self):
        return self.data

    def __enter__(self):
        return self

    def __exit__(self, *_):
        return None


class SetpointReadPrograms(unittest.TestCase):
    def run_program(self, failure=None):
        calls = []

        def fake_open(request, **_):
            if request.full_url.endswith("/reachability"):
                return Response(
                    {
                        "offer-desk": {"reachable": True},
                        "vrooli-memory": {"reachable": True},
                    }
                )
            if request.full_url.endswith("/unresolved"):
                return Response({})

            body = json.loads(request.data)
            binding_id = body["binding_id"]
            calls.append(binding_id)
            if failure and binding_id == "offer-desk/offers/board-show":
                klass, code, status, detail = failure
                payload = {
                    "error": detail,
                    "class": klass,
                    "status": status,
                    "http_status": code,
                }
                raise urllib.error.HTTPError(
                    request.full_url,
                    code,
                    "fixture failure",
                    {},
                    io.BytesIO(json.dumps(payload).encode()),
                )
            return Response(RESPONSES[binding_id])

        kernel = SessionKernel(
            bindings=BINDINGS,
            session_id="offer-desk-setpoint-fake",
            bridge_url="http://fixture/execute",
            agent_bridge_url="http://fixture/delegate",
            libraries=[LIBRARY],
        )
        with patch("host.engine.urllib.request.urlopen", side_effect=fake_open):
            execution = kernel.execute(
                SOURCE,
                include_materialized=True,
                program_id="offer-desk-setpoint-fake",
                provenance="test",
            )
        self.assertTrue(execution["ok"], execution.get("error"))
        self.assertCountEqual(RESPONSES, calls)
        return json.loads(execution["stdout"])

    def test_typed_remote_error_stays_visible_without_lowering_board_status(self):
        envelope = self.run_program(
            ("remote_error", 400, "failed", "schema mismatch")
        )

        self.assertEqual("ok", envelope["status"])
        self.assertEqual("remote_error", envelope["errors"][0]["class"])
        self.assertEqual("unreliable:remote_error", envelope["signals"]["rows"][0]["reason"])

    def test_typed_unreachable_read_marks_board_partial(self):
        envelope = self.run_program(
            ("scenario_unreachable", 503, "unavailable", "scenario_not_running")
        )

        self.assertEqual("partial", envelope["status"])
        self.assertEqual("scenario_unreachable", envelope["errors"][0]["class"])
        self.assertEqual("scenario_unreachable", envelope["signals"]["rows"][0]["reason"])


if __name__ == "__main__":
    unittest.main()

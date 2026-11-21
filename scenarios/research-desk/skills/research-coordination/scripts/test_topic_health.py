"""Tests for topic_health.py: python3 -I -m unittest discover -s . -p 'test_topic_health.py'."""
import io
import json
import sys
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))
import topic_health as th  # noqa: E402

TOOLS = """# Tools

## Browser capture

- **Used for:** real-browser proof.
- **Fallback:** retained captures and
  direct implementation work.
- **Monitoring scenario:** `bas`

## Validation

- **Fallback:** owner-level checks.
- **Monitoring scenarios:** `test-genie`, `gone`

## Notes without a monitor

- **Fallback:** nothing to watch.
"""

# A stand-in control plane: one answer per slug, chosen by the slug.
FAKE_VROOLI = """#!/usr/bin/env python3
import json, sys
slug = sys.argv[3]
answers = {
    "bas": {"success": True, "scenario": {"status": "running", "health_status": "healthy"}},
    "test-genie": {"success": True, "scenario": {"status": "stopped", "health_status": ""}},
    "gone": {"success": False, "code": "scenario_not_found", "error": "not found"},
    "search-hub": {"success": True, "scenario": {"status": "running", "health_status": "unhealthy", "health_error": "db"}},
}
if slug in answers:
    print(json.dumps(answers[slug]))
else:
    print("dial tcp 127.0.0.1:8090: connection refused", file=sys.stderr)
    sys.exit(1)
"""


FAKE_DESK = """#!/bin/sh
echo 'Error: request failed: Get "http://localhost:1/health": invalid agent credential authority response' >&2
exit 1
"""
OK_DESK = "#!/bin/sh\necho '{\"commissions\": []}'\n"


class TopicHealthTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        root = Path(self.tmp.name)
        self.home = root / "home"
        for name, tools in (("own", TOOLS), ("plain", None)):
            t = self.home / "topics" / name
            t.mkdir(parents=True)
            (t / "MISSION.md").write_text("# m\n")
            if tools:
                (t / "TOOLS.md").write_text(tools)
        (self.home / "topics" / "no-mission").mkdir()
        self.vrooli = root / "vrooli"
        self.vrooli.write_text(FAKE_VROOLI)
        self.vrooli.chmod(0o755)
        self.desk = root / "desk"
        self.desk.write_text(OK_DESK)
        self.desk.chmod(0o755)

    def tearDown(self):
        self.tmp.cleanup()

    def run_main(self, *args):
        out = io.StringIO()
        with redirect_stdout(out):
            th.main(["--home", str(self.home), "--vrooli", str(self.vrooli), "--research-desk", str(self.desk), *args])
        return out.getvalue()

    def test_parse_keeps_monitored_capabilities_and_joins_wrapped_fallbacks(self):
        caps = th.parse_tools(TOOLS)
        self.assertEqual([c["capability"] for c in caps], ["Browser capture", "Validation"])
        self.assertEqual(caps[0]["fallback"], "retained captures and direct implementation work.")
        self.assertEqual(caps[1]["scenarios"], ["test-genie", "gone"])

    def test_states_follow_the_control_plane_and_a_failed_question_is_unknown(self):
        r = json.loads(self.run_main("--json"))
        state = {t["scenario"]: t["state"] for t in r["tools"]}
        self.assertEqual(state["bas"], "up")
        self.assertEqual(state["test-genie"], "down")       # stopped
        self.assertEqual(state["gone"], "down")             # no such scenario
        self.assertEqual(state["search-hub"], "down")       # running but unhealthy
        self.assertEqual(state["prompt-manager"], "unknown")  # control plane did not answer
        self.assertEqual(r["topics"], 2)                    # a folder without MISSION.md is not a topic
        self.assertEqual(r["own_inventory"], ["own"])
        plain = {t["scenario"] for t in r["tools"] if "plain" in t["topics"]}
        self.assertEqual(plain, {"agent-credentials"} | {s for c in th.COMMON for s in c["scenarios"]})
        self.assertEqual(state["agent-credentials"], "up")

    def test_report_names_each_tool_that_is_not_up_with_its_fallback(self):
        text = self.run_main("--topic", "own")
        self.assertIn("## test-genie — down (status stopped)", text)
        self.assertIn("Validation: owner-level checks.", text)
        self.assertNotIn("## bas", text)

    def test_missing_control_plane_makes_every_tool_unknown_never_up(self):
        out = io.StringIO()
        with redirect_stdout(out):
            th.main(["--home", str(self.home), "--vrooli", str(self.home / "absent"), "--research-desk", str(self.home / "absent"), "--json"])
        r = json.loads(out.getvalue())
        self.assertEqual(r["counts"]["up"], 0)
        self.assertEqual(r["counts"]["unknown"], len(r["tools"]))

    def test_a_refusing_credential_broker_is_down_even_when_every_service_is_healthy(self):
        self.desk.write_text(FAKE_DESK)
        text = self.run_main("--topic", "own")
        self.assertIn("## agent-credentials — down (Error: request failed", text)
        self.assertIn("run binding-free programs with Program Runtime's standalone runner", text)
        self.assertIn("1 topic(s): own", text)

    def test_friction_that_cannot_be_read_is_unknown(self):
        r = json.loads(self.run_main("--topic", "own", "--friction", "--json",
                                     "--program-runtime", str(self.home / "absent")))
        self.assertEqual(r["friction"]["status"], "unknown")
        self.assertIn("not installed", r["friction"]["reason"])


if __name__ == "__main__":
    unittest.main()

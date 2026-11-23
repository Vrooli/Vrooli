"""The warm worker answers one JSON line per request and keeps stdout clean for the protocol."""
import json
import os
import subprocess
import sys
import tempfile
import textwrap
import unittest

SIDECAR_PARENT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


class WorkerTest(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        self.pkg = os.path.join(self.dir.name, "probe_pkg")
        os.makedirs(self.pkg)
        open(os.path.join(self.pkg, "__init__.py"), "w").close()
        with open(os.path.join(self.pkg, "noisy.py"), "w") as handle:
            handle.write(textwrap.dedent('''
                import sys

                def main():
                    print("stdout chatter that must not corrupt the protocol")
                    mode = sys.argv[1]
                    if mode == "ok":
                        print('{"metrics": 1}', file=sys.stderr)
                        return {"labels": "hair"}
                    if mode == "exit":
                        print("semantic segmentation failed: this parser has no label for 'book'")
                        raise SystemExit(1)
                    if mode == "input_exit":
                        from image_tools_sidecar import _common
                        print("no label for 'book'")
                        raise SystemExit(_common.exit_status_for(_common.InputError("no label")))
                    if mode == "input_raise":
                        from image_tools_sidecar import _common
                        raise _common.InputError("mask does not fit the image")
                    raise ValueError("")
            '''))

    def ask(self, *modes):
        stdin = "".join(json.dumps({"argv": ["-m", "probe_pkg.noisy", m]}) + "\n" for m in modes)
        done = subprocess.run(
            [sys.executable, "-m", "image_tools_sidecar.worker"], input=stdin, capture_output=True, text=True,
            env={"PYTHONPATH": os.pathsep.join([self.dir.name, SIDECAR_PARENT]), "PATH": os.environ.get("PATH", "")}, timeout=60,
        )
        return [json.loads(line) for line in done.stdout.splitlines()]

    def test_success_returns_meta_and_module_chatter_stays_off_the_protocol(self):
        (ok,) = self.ask("ok")
        self.assertEqual(ok, {"ok": True, "meta": {"labels": "hair"}})

    def test_a_module_that_prints_its_reason_and_exits_reports_that_reason(self):
        ok, failed, after = self.ask("ok", "exit", "ok")
        self.assertTrue(ok["ok"] and after["ok"])
        self.assertFalse(failed["ok"])
        self.assertIn("this parser has no label for 'book'", failed["error"])
        self.assertNotIn("{", failed["error"])  # structured metrics lines are not reasons

    def test_an_exception_without_a_message_falls_back_to_the_captured_output(self):
        (failed,) = self.ask("raise")
        self.assertFalse(failed["ok"])
        self.assertIn("stdout chatter", failed["error"])

    def test_a_failure_the_caller_caused_is_marked_as_invalid_input_and_a_fault_is_not(self):
        exited, raised, fault, plain_exit = self.ask("input_exit", "input_raise", "raise", "exit")
        self.assertEqual(exited["kind"], "invalid_input")
        self.assertEqual(raised["kind"], "invalid_input")
        self.assertNotIn("kind", fault)       # an unexplained exception is a fault of the run
        self.assertNotIn("kind", plain_exit)  # so is a module that exits 1


if __name__ == "__main__":
    unittest.main()

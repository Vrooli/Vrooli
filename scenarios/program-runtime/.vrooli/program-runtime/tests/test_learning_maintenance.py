"""Owner queue consumer contracts: receipt privacy and evidence-preserving transitions."""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).parents[1]


class LearningMaintenanceTests(unittest.TestCase):
    def run_program(self, inputs, queue=None, error=None):
        calls = []
        def read(**kwargs):
            calls.append(("read", kwargs))
            if error:
                raise RuntimeError(error)
            return NS(head=lambda n: queue or {"findings": []})
        def advance(**kwargs):
            calls.append(("advance", kwargs))
            return NS(head=lambda n: [{"finding": {"state": kwargs["next_state"], "claim_ref": "private-worker-receipt"}}])
        def list_read(**kwargs):
            handle = read(**kwargs)
            return NS(head=lambda n: [handle.head(n)])
        program = NS(inputs=lambda: inputs, envelope=lambda name,version: {
            "program":name,"version":version,"signals":{},"errors":[],"evidence":[]},
            classify=lambda exc: ("unavailable", "scenario_unreachable") if isinstance(exc,RuntimeError) else ("failed","invalid_input"))
        output=io.StringIO()
        with contextlib.redirect_stdout(output):
            exec(compile((ROOT/"learning-maintain.py").read_text(),"learning-maintain","exec"),
                 {"program":program,"tasks":NS(learning_findings=list_read,learning_finding_transition=advance)})
        return json.loads(output.getvalue()),calls

    def test_queue_reader_keeps_identity_but_not_feedback_authority(self):
        result,calls=self.run_program({"owner":"browser-automation-studio"},queue={"findings":[{
            "finding_id":"f1","owner":"browser-automation-studio","state":"routed",
            "feedback_ref":"do-not-export","claim_ref":"also-private","evidence":["run:1"]}]})
        self.assertEqual(result["status"],"ok")
        self.assertEqual(result["signals"]["findings"][0]["evidence"],["run:1"])
        self.assertNotIn("do-not-export",json.dumps(result))
        self.assertNotIn("also-private",json.dumps(result))
        self.assertEqual(calls,[("read",{"owner":"browser-automation-studio"})])

    def test_unavailable_and_truncated_are_not_an_empty_healthy_queue(self):
        result,_=self.run_program({},error="offline")
        self.assertEqual(result["status"],"unavailable")
        result,_=self.run_program({},queue={"findings":[],"truncated":True})
        self.assertEqual(result["status"],"partial")

    def test_worker_advance_preserves_claim_and_independent_evidence(self):
        result,calls=self.run_program({"action":"advance","owner":"fixture","finding_id":"f1",
            "expected_state":"repaired","next_state":"validated","claim_ref":"worker-receipt",
            "evidence":["test-genie:verified-repair"]})
        self.assertEqual(result["status"],"ok")
        self.assertEqual(calls[0][1]["claim_ref"],"worker-receipt")
        self.assertEqual(calls[0][1]["evidence"],["test-genie:verified-repair"])

    def test_contracts_validate_against_authoring_schema(self):
        import jsonschema
        schema=json.loads((ROOT.parents[1]/"schemas/program-contract.schema.json").read_text())
        for name in ("learning-maintain","learning-feedback","setpoint-read","improve-cycle"):
            with self.subTest(name=name):
                jsonschema.validate(json.loads((ROOT/(name+".json")).read_text()),schema)


if __name__ == "__main__":
    unittest.main()

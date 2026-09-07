import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).parents[1]

class PrepareTests(unittest.TestCase):
    def invoke(self, content="owning instructions", validation_error=""):
        calls = []
        def get(**kw):
            calls.append(("program", kw))
            return NS(head=lambda n: [{"name": "fixture.run", "ownerSkill": "fixture", "declaredInputs": ["target"], "contentDigest": "digest", "validationError": validation_error}])
        def read(**kw):
            calls.append(("skill", kw))
            self.assertEqual(kw["output"], "combined")
            return NS(meta=lambda: {"combined": content, "combinedHash": "skill-digest"})
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "prepare-operation.py").read_text(), "prepare-operation", "exec"), {
                "inputs": {"name": "fixture.run"}, "program_runtime": NS(library=NS(get=get)), "prompt_manager": NS(skill=NS(read=read))})
        return json.loads(stream.getvalue()), calls

    def test_one_program_one_owner_skill_with_input_names_and_identity(self):
        result, calls = self.invoke()
        self.assertEqual(result["status"], "ok")
        self.assertEqual(len(calls), 2)
        self.assertEqual(result["signals"]["program"]["declaredInputs"], ["target"])
        self.assertEqual(result["signals"]["skill"]["content"], "owning instructions")

    def test_missing_or_truncated_skill_cannot_claim_complete_preparation(self):
        for content in ("", "x" * 24001):
            result, _ = self.invoke(content)
            self.assertEqual(result["status"], "partial")
            self.assertEqual(result["errors"][0]["class"], "skill_incomplete")

    def test_invalid_contract_is_not_ready(self):
        result, _ = self.invoke(validation_error="unknown binding")
        self.assertEqual(result["status"], "partial")

    def test_contract(self):
        import jsonschema
        schema = json.loads((ROOT.parents[1] / "schemas/program-contract.schema.json").read_text())
        jsonschema.validate(json.loads((ROOT / "prepare-operation.json").read_text()), schema)

if __name__ == "__main__":
    unittest.main()

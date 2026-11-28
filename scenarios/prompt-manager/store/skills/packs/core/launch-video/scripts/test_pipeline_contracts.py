"""Cross-owner identity contracts; all data is temporary fixture content."""
import json
from pathlib import Path
import tempfile
import unittest

import pipeline
import production
import product_brief
import workflow


class PipelineContractTests(unittest.TestCase):
    def test_file_identity_is_shared_across_coordinator_and_workflow_owners(self):
        workflow_digest = pipeline.verify_render_bundle.__globals__["digest"]
        self.assertIs(pipeline.sha256, workflow_digest)
        self.assertIs(pipeline.sha256, production.digest)
        self.assertIs(pipeline.sha256, product_brief.digest)
        with tempfile.TemporaryDirectory() as temporary:
            fixture = Path(temporary) / "bound-input.json"
            fixture.write_bytes(b'{"fixture_only":true}\n')
            expected = product_brief.digest(fixture)
            self.assertEqual(expected, pipeline.sha256(fixture))
            self.assertEqual(expected, workflow_digest(fixture))

    def test_handoff_keeps_channel_and_pre_render_scope(self):
        handoff = pipeline.build_channel_handoff(
            product="vega", channel="landing", manifest_sha256="a" * 64,
            source_map_path="packets/vega.json", source_map_sha256="b" * 64,
            packet_intake={"sha256": "c" * 64}, route_id="route-1",
            source_verification={"source_records": 2, "source_record_manifest_sha256": "d" * 64},
            channel_binding={"script_sha256": "e" * 64, "ordered_claims": ["claim-1"]},
            plan_sha256="f" * 64, gate_cues_sha256="1" * 64,
            sequence_manifest_sha256="2" * 64,
            tools={"node": {"path": "/fixture/node", "sha256": "3" * 64}})
        self.assertEqual("inputs-bound-pre-render", handoff["status"])
        self.assertEqual(["claim-1"], handoff["channel_binding"]["ordered_claims"])
        self.assertIn("No render or cleanup is established", handoff["limits"])

    def test_failure_receipt_contains_one_identity_row_per_channel(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            def write(name, content):
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content)
                return path
            references = {}
            for channel in ("landing", "vertical"):
                refs = {}
                for key in ("plan", "gate_cues", "sequence_manifest"):
                    path = write(f"{channel}/{key}.json", json.dumps({"fixture": key}))
                    refs[key] = path.relative_to(root).as_posix()
                    refs[key + "_sha256"] = pipeline.sha256(path)
                refs["channel"] = channel
                references[channel] = refs
            write("source.json", '{"fixture":true}')
            write("offer.json", '{"fixture":true}')
            manifest = {"product": "vega", "source_map": "source.json",
                        "source_map_sha256": pipeline.sha256(root / "source.json"),
                        "offer_signoff": "offer.json",
                        "offer_signoff_sha256": pipeline.sha256(root / "offer.json"),
                        "cuts": list(references.values())}
            manifest_path = write("manifest.json", json.dumps(manifest))
            output = root / "failed-output"
            output.mkdir()
            pipeline.write_failure_receipt(output, manifest_path,
                RuntimeError("fixture validation failure"), False, False)
            report = json.loads((output / "pipeline.json").read_text())
            cuts = report["manifest_context"]["cuts"]
            self.assertEqual(["landing", "vertical"], [cut["channel"] for cut in cuts])
            self.assertEqual("unknown; failure envelope does not establish owner cleanup",
                             report["cleanup"])


if __name__ == "__main__":
    unittest.main()

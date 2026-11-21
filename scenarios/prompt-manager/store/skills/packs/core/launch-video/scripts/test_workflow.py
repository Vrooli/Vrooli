import json
from pathlib import Path
import tempfile
import unittest
import workflow


class WorkflowTests(unittest.TestCase):
    def test_snapshot_detects_mutation_and_preserves_old_receipts(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root/'composition.html').write_text('original composition')
            plan = root/'production.json'
            plan.write_text(json.dumps({'render_inputs': ['composition.html']}))
            output = root/'snapshot'
            receipt = workflow.freeze(plan, output)
            self.assertEqual([], workflow.verify_snapshot(output, receipt['plan_sha256']))
            self.assertIn('snapshot belongs to a different production plan', workflow.verify_snapshot(output, 'other-plan'))
            (root/'composition.html').write_text('new authoring state')
            self.assertEqual('original composition', (output/'composition.html').read_text())
            self.assertEqual([], workflow.verify_snapshot(output))
            (output/'composition.html').write_text('mutated render input')
            self.assertTrue(workflow.verify_snapshot(output))
            with self.assertRaises(FileExistsError):
                workflow.freeze(plan, output)

    def test_snapshot_rejects_escape_and_symlink(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root/'file').write_text('content')
            (root/'link').symlink_to(root/'file')
            plan = root/'production.json'
            for name in ['../outside', '/etc/passwd', 'link']:
                plan.write_text(json.dumps({'render_inputs': [name]}))
                with self.assertRaises(ValueError):
                    workflow.freeze(plan, root/'out')

    def test_managed_render_must_bind_complete_snapshot_and_actual_video(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root/'index.html').write_text('composition')
            plan = root/'production.json'
            plan.write_text(json.dumps({'render_inputs': ['index.html']}))
            snapshot = root/'snapshot'
            workflow.freeze(plan, snapshot)
            bundle = root/'bundle'; bundle.mkdir()
            (bundle/'render.mp4').write_bytes(b'rendered test bytes')
            receipt = {'status': 'completed', 'output_sha256': workflow.digest(bundle/'render.mp4'),
                       'request': {'input_dir': str(snapshot), 'inputs': [
                           {'path': str(p.relative_to(snapshot)), 'sha256': workflow.digest(p)}
                           for p in snapshot.rglob('*') if p.is_file()]}}
            def check():
                (bundle/'receipt.json').write_text(json.dumps(receipt))
                return workflow.verify_render_bundle(snapshot, bundle)
            self.assertEqual([], check())
            receipt['request']['input_dir'] = str(root/'other')
            self.assertIn('managed render used a different snapshot directory', check())
            receipt['request']['input_dir'] = str(snapshot)
            removed = receipt['request']['inputs'].pop()
            self.assertIn('managed render inputs differ from the complete snapshot', check())
            receipt['request']['inputs'].append(removed)
            (bundle/'render.mp4').write_bytes(b'changed output')
            self.assertIn('managed render output hash mismatch', check())
            receipt['status'] = 'failed'
            self.assertIn('managed render did not complete', check())


if __name__ == '__main__':
    unittest.main()

import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import export_gates
import production
from contextlib import redirect_stdout
from io import StringIO


class ComposedExportGateTests(unittest.TestCase):
    def setup_inputs(self, root, *, qr=False, text_cues=True):
        video = root / 'export.mp4'
        video.write_bytes(b'fixture exact export bytes')
        plan = root / 'production.json'
        plan.write_text(json.dumps({
            'audio': {'target_lufs': -16, 'join_times': [1.0]},
            'scenes': [{'id': 'hook', 'start': 0, 'duration': 3,
                        'text': [{'value': 'Great agents', 'hold': 2}]}],
            'cta': {'qr': qr, 'url': 'https://example.com/product'},
            'video': 'export.mp4',
        }))
        cues = root / 'gates.json'
        cues.write_text(json.dumps({'text_cues': [{'id': 'hook', 'text': 'Great agents',
                                                    'start_seconds': 0, 'end_seconds': 2,
                                                    'region': [0, 0, 10, 10]}]
                                    if text_cues else {}}))
        return video, plan, cues


    def test_production_gate_command_dispatches_to_composed_runner(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root)
            output = root / 'out'
            fake_receipt = {'status': 'pass', 'results': {'audio': 'pass'}}
            with patch('sys.argv', ['production.py', 'gate', str(video), '--plan', str(plan),
                                    '--cues', str(cues), '--out', str(output)]), \
                 patch.object(export_gates, 'run', return_value=fake_receipt) as runner, \
                 redirect_stdout(StringIO()):
                result = production.main()
            self.assertEqual(result, 0)
            runner.assert_called_once_with(Path(video), Path(plan), Path(cues), Path(output))

    def test_composed_export_gates_pass_with_frozen_cues_and_clean_signals(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root)
            source_hash = export_gates.digest(video)
            with patch.object(export_gates.frame_flicker, 'scan_video', return_value=(30, [])), \
                 patch.object(export_gates.frame_flicker, 'scan_repeated_video', return_value=(30, [])), \
                 patch.object(export_gates.text_entrance, 'scan', return_value={'sha256': source_hash, 'status': 'pass'}), \
                 patch.object(export_gates.quality_gates, 'measure_video_audio', return_value={
                     'source_sha256': source_hash, 'lufs': -16, 'true_peak_dbtp': -2,
                     'loop_similarity': .2, 'seam_jump_db': 0, 'channel_rms': [[.2, .3]]}):
                report = export_gates.run(video, plan, cues, root / 'out')
            self.assertEqual(report['status'], 'pass')
            self.assertEqual(report['results']['source-stability'], 'pass')
            self.assertTrue((root / 'out/export-gates.json').is_file())
            self.assertTrue((root / 'out/transient-frame.json').is_file())
            self.assertTrue((root / 'out/temporal-flicker.json').is_file())


    def test_selected_export_must_match_plan_path(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root)
            other = root / 'other.mp4'
            other.write_bytes(b'different export')
            with self.assertRaisesRegex(ValueError, 'does not match'):
                export_gates.run(other, plan, cues, root / 'out')

    def test_missing_text_cues_fails_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root, text_cues=False)
            source_hash = export_gates.digest(video)
            with patch.object(export_gates.frame_flicker, 'scan_video', return_value=(30, [])), \
                 patch.object(export_gates.frame_flicker, 'scan_repeated_video', return_value=(30, [])), \
                 patch.object(export_gates.quality_gates, 'measure_video_audio', return_value={
                     'source_sha256': source_hash, 'lufs': -16, 'true_peak_dbtp': -2,
                     'loop_similarity': .2, 'seam_jump_db': 0, 'channel_rms': [[.2]]}):
                report = export_gates.run(video, plan, cues, root / 'out')
            self.assertEqual(report['status'], 'fail')
            self.assertEqual(report['results']['text-entrance'], 'fail')

    def test_planted_frame_event_fails_aggregate(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root)
            source_hash = export_gates.digest(video)
            with patch.object(export_gates.frame_flicker, 'scan_video', return_value=(30, [{'frame': 9, 'timestamp_seconds': .3,
                                                                  'transient_pixels': 32,
                                                                  'peak_component_pixels': 32}])), \
                 patch.object(export_gates.frame_flicker, 'scan_repeated_video', return_value=(30, [])), \
                 patch.object(export_gates.text_entrance, 'scan', return_value={'sha256': source_hash, 'status': 'pass'}), \
                 patch.object(export_gates.quality_gates, 'measure_video_audio', return_value={
                     'source_sha256': source_hash, 'lufs': -16, 'true_peak_dbtp': -2,
                     'loop_similarity': .2, 'seam_jump_db': 0, 'channel_rms': [[.2]]}):
                report = export_gates.run(video, plan, cues, root / 'out')
            self.assertEqual(report['status'], 'fail')
            self.assertEqual(report['results']['transient-frame'], 'fail')


    def test_malformed_scene_plan_fails_as_a_gate_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root)
            plan_data = json.loads(plan.read_text())
            plan_data['scenes'] = [None]
            plan.write_text(json.dumps(plan_data))
            source_hash = export_gates.digest(video)
            with patch.object(export_gates.frame_flicker, 'scan_video', return_value=(30, [])), \
                 patch.object(export_gates.frame_flicker, 'scan_repeated_video', return_value=(30, [])), \
                 patch.object(export_gates.text_entrance, 'scan', return_value={'sha256': source_hash, 'status': 'pass'}), \
                 patch.object(export_gates.quality_gates, 'measure_video_audio', return_value={
                     'source_sha256': source_hash, 'lufs': -16, 'true_peak_dbtp': -2,
                     'loop_similarity': .2, 'seam_jump_db': 0, 'channel_rms': [[.2]]}):
                report = export_gates.run(video, plan, cues, root / 'out')
            self.assertEqual(report['status'], 'fail')
            self.assertEqual(report['results']['audio-and-plan-quality'], 'fail')



    def test_qr_cue_contract_requires_simulated_phone_capture(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root, qr=True)
            cues_data = json.loads(cues.read_text())
            cues_data['qr'] = {'times': [9.0, 10.0], 'widths': [390, 960],
                               'phone_simulation': False}
            cues.write_text(json.dumps(cues_data))
            source_hash = export_gates.digest(video)
            with patch.object(export_gates.frame_flicker, 'scan_video', return_value=(30, [])), \
                 patch.object(export_gates.frame_flicker, 'scan_repeated_video', return_value=(30, [])), \
                 patch.object(export_gates.text_entrance, 'scan', return_value={'sha256': source_hash, 'status': 'pass'}), \
                 patch.object(export_gates.quality_gates, 'measure_video_audio', return_value={
                     'source_sha256': source_hash, 'lufs': -16, 'true_peak_dbtp': -2,
                     'loop_similarity': .2, 'seam_jump_db': 0, 'channel_rms': [[.2]]}):
                report = export_gates.run(video, plan, cues, root / 'out')
            self.assertEqual(report['status'], 'fail')
            self.assertEqual(report['results']['qr'], 'fail')

    def test_source_change_during_scan_fails_stability_and_hash_binding(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root)
            source_hash = export_gates.digest(video)
            def mutate_during_scan(*_args):
                video.write_bytes(b'changed during gate run')
                return {'sha256': source_hash, 'status': 'pass'}
            with patch.object(export_gates.frame_flicker, 'scan_video', return_value=(30, [])), \
                 patch.object(export_gates.frame_flicker, 'scan_repeated_video', return_value=(30, [])), \
                 patch.object(export_gates.text_entrance, 'scan', side_effect=mutate_during_scan), \
                 patch.object(export_gates.quality_gates, 'measure_video_audio', return_value={
                     'source_sha256': source_hash, 'lufs': -16, 'true_peak_dbtp': -2,
                     'loop_similarity': .2, 'seam_jump_db': 0, 'channel_rms': [[.2]]}):
                report = export_gates.run(video, plan, cues, root / 'out')
            self.assertEqual(report['status'], 'fail')
            self.assertEqual(report['results']['source-stability'], 'fail')
            quality = json.loads((root / 'out/audio-and-plan-quality.json').read_text())
            self.assertIn('source-binding', {item['gate'] for item in quality['findings']})

    def test_qr_without_frozen_times_fails_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            video, plan, cues = self.setup_inputs(root, qr=True)
            source_hash = export_gates.digest(video)
            with patch.object(export_gates.frame_flicker, 'scan_video', return_value=(30, [])), \
                 patch.object(export_gates.frame_flicker, 'scan_repeated_video', return_value=(30, [])), \
                 patch.object(export_gates.text_entrance, 'scan', return_value={'sha256': source_hash, 'status': 'pass'}), \
                 patch.object(export_gates.quality_gates, 'measure_video_audio', return_value={
                     'source_sha256': source_hash, 'lufs': -16, 'true_peak_dbtp': -2,
                     'loop_similarity': .2, 'seam_jump_db': 0, 'channel_rms': [[.2]]}):
                report = export_gates.run(video, plan, cues, root / 'out')
            self.assertEqual(report['status'], 'fail')
            self.assertEqual(report['results']['qr'], 'fail')


if __name__ == '__main__':
    unittest.main()

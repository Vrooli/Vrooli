import unittest
import hashlib
import tempfile
from pathlib import Path
import json
import subprocess
import sys
import quality_gates
import numpy as np


class QualityGateTests(unittest.TestCase):
    def setUp(self):
        self.plan = {
            'audio': {'target_lufs': -16},
            'scenes': [
                {'id': 'a', 'start': 0, 'duration': 4, 'text': [{'value': 'Clear proof', 'hold': 1}]},
                {'id': 'b', 'start': 4, 'duration': 4, 'text': []},
            ],
        }
        self.signals = {
            'lufs': -16, 'true_peak_dbtp': -1.2, 'loop_similarity': .2,
            'seam_jump_db': 2, 'channel_rms': [[.02] * 80, [.018] * 80],
        }
        self.channels = [
            {'channel': 'landing', 'width': 1920, 'height': 1080, 'burned_captions': True,
             'sound_on_reviewed': True, 'export_sha256': '1' * 64},
            {'channel': 'X', 'width': 1920, 'height': 1080, 'burned_captions': True,
             'hook_seconds': 1.0, 'product_proof_seconds': 4.0, 'export_sha256': '2' * 64},
            {'channel': 'Reddit', 'width': 1080, 'height': 1080, 'burned_captions': True,
             'recomposed': True, 'qr': False, 'unsupported_route_claim': False,
             'export_sha256': '3' * 64},
            {'channel': 'vertical', 'width': 1080, 'height': 1920, 'burned_captions': True,
             'recomposed': True, 'center_crop_only': False, 'safe_area_conflicts': 0,
             'phone_legibility_pass': True, 'implies_native_mobile': False,
             'export_sha256': '4' * 64},
        ]

    def channel_failures(self):
        self.plan['channel_cuts'] = self.channels
        return {finding['channel']: finding for finding in quality_gates.evaluate(self.plan, self.signals)
                if finding['gate'] == 'channel-cut' and finding.get('channel')}

    def test_channel_controls_pass_for_all_required_variants(self):
        self.assertEqual({}, self.channel_failures())

    def test_vertical_cut_catches_center_crop_and_safe_area_defect(self):
        self.channels[-1]['center_crop_only'] = True
        self.channels[-1]['safe_area_conflicts'] = 1
        self.assertIn('vertical', self.channel_failures())

    def test_x_cut_catches_late_hook_and_proof(self):
        self.channels[1]['hook_seconds'] = 3
        self.channels[1]['product_proof_seconds'] = 8
        self.assertIn('X', self.channel_failures())

    def test_reddit_cut_catches_unsupported_qr_or_claim(self):
        self.channels[2]['qr'] = True
        self.channels[2]['unsupported_route_claim'] = True
        self.assertIn('Reddit', self.channel_failures())

    def test_landing_requires_muted_and_sound_on_cues(self):
        self.channels[0]['burned_captions'] = False
        self.channels[0]['sound_on_reviewed'] = False
        self.assertIn('landing', self.channel_failures())

    def failures(self):
        return {finding['gate'] for finding in quality_gates.evaluate(self.plan, self.signals)}

    def test_clean_control_passes(self):
        self.assertEqual(set(), self.failures())

    def test_missing_scenes_fail_closed(self):
        self.plan['scenes'] = []
        self.assertIn('plan-coverage', self.failures())

    def test_reading_time_defect_is_planted_and_caught(self):
        self.plan['scenes'][0]['text'][0]['hold'] = .2
        self.assertIn('reading-time', self.failures())

    def test_cut_rate_defect_is_planted_and_caught(self):
        self.plan['scenes'] = [{'id': str(i), 'start': i * 2, 'duration': 2, 'text': []} for i in range(6)]
        self.assertIn('cut-rate', self.failures())

    def test_loudness_defects_are_planted_and_caught(self):
        self.signals['lufs'] = -20
        self.signals['true_peak_dbtp'] = -0.2
        self.assertTrue({'lufs', 'true-peak-dbtp'} <= self.failures())

    def test_loop_and_seam_defects_are_planted_and_caught(self):
        self.signals['loop_similarity'] = .999
        self.signals['seam_jump_db'] = 20
        self.assertTrue({'loop-similarity', 'seam-jump-db'} <= self.failures())

    def test_single_channel_dropout_is_planted_and_caught(self):
        self.signals['channel_rms'][1][30] = 0
        self.assertIn('channel-cut', self.failures())

    def test_receipt_binds_measurements_to_exact_export_hash(self):
        with tempfile.TemporaryDirectory() as temp:
            video = Path(temp) / 'candidate.mp4'
            video.write_bytes(b'planted-export-bytes')
            sha = hashlib.sha256(video.read_bytes()).hexdigest()
            self.signals['source_sha256'] = sha
            receipt = quality_gates.evaluate_export(video, self.plan, self.signals)
            self.assertEqual(receipt['status'], 'pass')
            self.assertEqual(receipt['sha256'], sha)
            self.signals['source_sha256'] = '0' * 64
            receipt = quality_gates.evaluate_export(video, self.plan, self.signals)
            self.assertEqual(receipt['status'], 'fail')
            self.assertIn('source-binding', {x['gate'] for x in receipt['findings']})

    def test_receipt_cli_runs_and_writes_source_bound_result(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            video = root / 'candidate.mp4'
            video.write_bytes(b'planted-export-bytes')
            sha = hashlib.sha256(video.read_bytes()).hexdigest()
            self.signals['source_sha256'] = sha
            plan = root / 'plan.json'
            signals = root / 'signals.json'
            plan.write_text(json.dumps(self.plan))
            signals.write_text(json.dumps(self.signals))
            output = root / 'quality-receipt.json'
            script = Path(__file__).with_name('quality_gates.py')
            result = subprocess.run([sys.executable, str(script), str(video), '--plan', str(plan),
                                     '--signals', str(signals), '--out', str(output)],
                                    capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            receipt = json.loads(output.read_text())
            self.assertEqual(receipt['status'], 'pass')
            self.assertEqual(receipt['sha256'], sha)

    def test_pcm_measurement_catches_exact_planted_loop(self):
        rng = np.random.default_rng(57)
        source = rng.normal(0, .1, 8_000).astype(np.float32)
        pcm = rng.normal(0, .1, (32_000, 2)).astype(np.float32)
        pcm[:8_000, :] = source[:, None]
        pcm[16_000:24_000, :] = source[:, None]
        measured = quality_gates.measure_audio_pcm(
            pcm, 1000, loop_window_seconds=8, loop_hop_seconds=2,
            minimum_loop_gap_seconds=16)
        self.assertGreater(measured['loop_similarity'], .999)

    def test_pcm_measurement_isolates_channel_dropout_and_explicit_seam(self):
        rate = 12_000
        t = np.arange(32 * rate, dtype=np.float32) / rate
        mono = .2 * np.sin(2 * np.pi * 440 * t)
        pcm = np.column_stack((mono, mono)).astype(np.float32)
        pcm[16 * rate:, :] += 1.0
        pcm[20 * rate:21 * rate, 1] = 0
        measured = quality_gates.measure_audio_pcm(pcm, rate, join_times=[16.0])
        self.assertGreater(measured['seam_jump_db'], 12)
        self.assertLess(min(measured['channel_rms'][1][200:210]), .001)
        self.assertGreater(min(measured['channel_rms'][0][200:210]), .1)

    def test_pcm_measurement_has_clean_controls_without_loop_or_seam(self):
        rate = 1000
        rng = np.random.default_rng(91)
        pcm = rng.normal(0, .1, (32_000, 2)).astype(np.float32)
        measured = quality_gates.measure_audio_pcm(
            pcm, rate, loop_window_seconds=8, loop_hop_seconds=2,
            minimum_loop_gap_seconds=16, join_times=[16])
        self.assertLess(measured['loop_similarity'], .98)
        self.assertLess(measured['seam_jump_db'], 12)
        self.assertTrue(all(min(channel) > .001 for channel in measured['channel_rms']))


if __name__ == '__main__':
    unittest.main()

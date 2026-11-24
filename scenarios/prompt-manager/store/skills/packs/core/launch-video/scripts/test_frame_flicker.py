import unittest

import numpy as np

from frame_flicker import detect_frames, detect_repeated_flicker, group_repeated_windows


class FrameFlickerTests(unittest.TestCase):
    def base(self, n=6):
        return [np.zeros((96, 128), dtype=np.uint8) for _ in range(n)]

    def test_clean_static_control_passes(self):
        self.assertEqual(detect_frames(self.base(), 30), [])

    def test_single_frame_local_flash_is_caught_with_timestamp(self):
        frames = self.base()
        frames[2][20:48, 30:70] = 240
        hits = detect_frames(frames, 30)
        self.assertEqual([h['frame'] for h in hits], [2])
        self.assertAlmostEqual(hits[0]['timestamp_seconds'], 2 / 30, places=5)

    def test_sustained_change_is_not_mislabeled_as_flash(self):
        frames = self.base()
        for frame in frames[2:]:
            frame[20:48, 30:70] = 240
        self.assertEqual(detect_frames(frames, 30), [])

    def test_smooth_motion_is_not_mislabeled_as_flash(self):
        frames = []
        for x in range(6):
            frame = np.zeros((96, 128), dtype=np.uint8)
            frame[20:48, 20 + x:60 + x] = 240
            frames.append(frame)
        self.assertEqual(detect_frames(frames, 30), [])

    def test_cut_is_not_mislabeled_as_flash(self):
        frames = self.base()
        for frame in frames[2:]:
            frame[:] = 240
        self.assertEqual(detect_frames(frames, 30), [])


    def test_repeated_static_clean_control_passes(self):
        self.assertEqual(detect_repeated_flicker(self.base(36), 30), [])

    def test_repeated_stationary_flash_is_caught(self):
        frames = []
        for index in range(36):
            frame = np.zeros((96, 128), dtype=np.uint8)
            if index % 6 in (2, 3):
                frame[20:52, 30:70] = 240
            frames.append(frame)
        hits = detect_repeated_flicker(frames, 30)
        self.assertTrue(hits)
        self.assertGreaterEqual(hits[-1]['max_transitions_in_window'], 4)


    def test_overlapping_flicker_windows_merge_to_one_timed_event(self):
        windows = [
            {'start_seconds': 0.0, 'end_seconds': 1.0, 'max_transitions_in_window': 8, 'flicker_pixels': 500},
            {'start_seconds': .5, 'end_seconds': 1.5, 'max_transitions_in_window': 10, 'flicker_pixels': 700},
            {'start_seconds': 1.5, 'end_seconds': 2.5, 'max_transitions_in_window': 8, 'flicker_pixels': 600},
        ]
        grouped = group_repeated_windows(windows, 30)
        self.assertEqual(len(grouped), 1)
        self.assertEqual(grouped[0]['window_count'], 3)
        self.assertEqual(grouped[0]['max_transitions_in_window'], 10)

    def test_single_frame_flash_does_not_pass_repeated_flicker_gate(self):
        frames = self.base(36)
        frames[15][20:48, 30:70] = 240
        self.assertEqual(detect_repeated_flicker(frames, 30), [])

    def test_smooth_motion_does_not_pass_repeated_flicker_gate(self):
        frames = []
        for x in range(36):
            frame = np.zeros((96, 128), dtype=np.uint8)
            frame[20:48, 20 + x % 40:60 + x % 40] = 240
            frames.append(frame)
        self.assertEqual(detect_repeated_flicker(frames, 30), [])

    def test_dimension_change_fails_closed(self):
        frames = self.base()
        frames[3] = np.zeros((90, 128), dtype=np.uint8)
        with self.assertRaisesRegex(ValueError, 'dimensions changed'):
            detect_frames(frames, 30)


if __name__ == '__main__':
    unittest.main()

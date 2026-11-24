import unittest
from pathlib import Path

import cv2

from verify_qr import simulate_phone_capture


class SimulatedPhoneQRTests(unittest.TestCase):
    def test_score_a_qr_survives_mild_phone_capture_at_playback_sizes(self):
        home = Path(__file__).resolve().parents[8].parent
        fixture = home / '.vrooli' / 'research' / 'topics' / 'launch-video' / 'runs' / '20261029-ending-concepts' / 'score-a' / 'aquila-prismatic-final.mp4'
        # The retained score-A output and baseline are immutable; use its known
        # static QR hold to exercise playback-size capture transforms.
        capture = cv2.VideoCapture(str(fixture))
        try:
            capture.set(cv2.CAP_PROP_POS_MSEC, 100200)
            ok, source = capture.read()
        finally:
            capture.release()
        self.assertTrue(ok)
        expected = 'https://vrooli.com/apps/aquila'
        for width in (480, 960):
            resized = cv2.resize(source, (width, round(source.shape[0] * width / source.shape[1])), interpolation=cv2.INTER_AREA)
            simulated = simulate_phone_capture(resized)
            decoded, _, _ = cv2.QRCodeDetector().detectAndDecode(simulated)
            self.assertEqual(decoded, expected, f'phone simulation at {width}px')

    def test_blank_capture_is_a_planted_decode_defect(self):
        blank = __import__('numpy').zeros((270, 480, 3), dtype='uint8')
        decoded, _, _ = cv2.QRCodeDetector().detectAndDecode(simulate_phone_capture(blank))
        self.assertEqual(decoded, '')


if __name__ == '__main__':
    unittest.main()

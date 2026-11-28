"""The consent gate's face counter: what it reports for each image, and how it fails.

The detector is a stub; what YuNet finds on real faces is covered by face_restore's
decoder tests and measured live (docs/internal/SECURITY.md, "Who is a person").
"""

import base64
import io
import json
import os
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

from PIL import Image

from image_tools_sidecar import face_count as fc
from image_tools_sidecar import face_restore


def item(size=(64, 48)):
    buf = io.BytesIO()
    Image.new("RGB", size, (90, 120, 60)).save(buf, format="PNG")
    return {"media_type": "image/png", "base64": base64.b64encode(buf.getvalue()).decode()}


class CountTests(unittest.TestCase):
    def count(self, images, found):
        with mock.patch.object(face_restore, "detect_faces", side_effect=found):
            return fc.count_faces({"images": images}, "/unused", detector=object())

    def test_reports_one_count_per_image_in_order(self):
        self.assertEqual(self.count([item(), item(), item()], [[], [{}, {}], [{}]]), {"counts": [0, 2, 1]})

    def test_a_request_without_images_is_refused(self):
        with self.assertRaises(SystemExit):
            fc.count_faces({"images": []}, "/unused", detector=object())

    def test_an_undecodable_image_fails_instead_of_counting_zero(self):
        # Zero faces clears the consent rule, so a decode failure must never look like it.
        with self.assertRaises(SystemExit) as raised:
            fc.count_faces({"images": [{"media_type": "image/png", "base64": base64.b64encode(b"not an image").decode()}]}, "/unused", detector=object())
        self.assertEqual(raised.exception.code, 6)


class MissingWeightsTests(unittest.TestCase):
    def test_missing_detector_weights_exit_four_naming_the_model(self):
        with tempfile.TemporaryDirectory() as root, self.assertRaises(SystemExit) as raised:
            fc.count_faces({"images": [item()]}, root)
        self.assertEqual(raised.exception.code, 4)

    def test_the_cli_names_the_missing_weights_on_stderr(self):
        with tempfile.TemporaryDirectory() as root:
            proc = subprocess.run(
                [sys.executable, "-m", "image_tools_sidecar.face_count", "--models-root", root],
                input=json.dumps({"images": [item()]}),
                capture_output=True,
                text=True,
                env={**os.environ, "PYTHONPATH": os.pathsep.join(filter(None, [os.path.dirname(os.path.dirname(fc.__file__)), os.environ.get("PYTHONPATH", "")]))},
            )
        self.assertEqual(proc.returncode, 4)
        self.assertIn("missing-weights=yunet", proc.stderr)


if __name__ == "__main__":
    unittest.main()

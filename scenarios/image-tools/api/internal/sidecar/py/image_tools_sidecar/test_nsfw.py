import unittest
from types import SimpleNamespace

import numpy as np
from PIL import Image

from image_tools_sidecar import _common
from image_tools_sidecar.nsfw import _preprocess


class NSFWPreprocessTests(unittest.TestCase):
    def test_dynamic_onnx_axes_use_the_model_specific_fallback(self):
        session = SimpleNamespace(get_inputs=lambda: [SimpleNamespace(shape=[1, 3, "height", "width"])])
        self.assertEqual(_common.model_input_hw(session, default_hw=(384, 384)), (384, 384))

    def test_preprocess_matches_pinned_vit_processor_contract(self):
        image = Image.new("RGB", (1, 1), (0, 128, 255))
        tensor = _preprocess(np, image, (1, 1), resample=Image.Resampling.BILINEAR)

        expected = np.array([-1.0, 1.0 / 255.0, 1.0], dtype=np.float32)
        self.assertEqual(tensor.shape, (1, 3, 1, 1))
        np.testing.assert_allclose(tensor[0, :, 0, 0], expected, atol=1e-6)

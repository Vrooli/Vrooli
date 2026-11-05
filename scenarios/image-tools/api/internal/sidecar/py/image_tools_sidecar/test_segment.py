"""SAM export contracts; no model download or GPU. Run with unittest."""
import unittest
import tempfile
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import numpy as np
from PIL import Image

from . import segment


class SAMContracts(unittest.TestCase):
    def test_embedded_preprocessing_receives_raw_rgb_longest_side(self):
        encoder = SimpleNamespace(get_inputs=lambda: [SimpleNamespace(shape=["h", "w", 3])])
        pixels, size = segment._sam_input(np, Image, Image.new("RGB", (800, 400), (128, 64, 32)), encoder)
        self.assertEqual(size, (1024, 512))
        self.assertEqual(pixels.shape, (512, 1024, 3))
        np.testing.assert_array_equal(pixels[0, 0], [128, 64, 32])

    def test_external_preprocessing_pads_after_normalization(self):
        encoder = SimpleNamespace(get_inputs=lambda: [SimpleNamespace(shape=[1, 3, 1024, 1024])])
        pixels, _ = segment._sam_input(np, Image, Image.new("RGB", (400, 800), (128, 64, 32)), encoder)
        self.assertEqual(pixels.shape, (1, 3, 1024, 1024))
        np.testing.assert_allclose(pixels[0, :, 0, 0], [(128-123.675)/58.395, (64-116.28)/57.12, (32-103.53)/57.375], atol=1e-6)
        self.assertEqual(float(pixels[0, :, :, 512:].max()), 0)

    def test_noncentral_positive_negative_points_and_no_box_token(self):
        coords, labels = segment._prompt_arrays(np, {"points":[{"x":0.75,"y":0.25},{"x":0.1,"y":0.9,"negative":True}]}, (1024, 512))
        np.testing.assert_allclose(coords, [[[768,128],[102.4,460.8],[0,0]]], atol=1e-4)
        np.testing.assert_array_equal(labels, [[1,0,-1]])

    def test_box_corners_use_sam_labels_two_and_three(self):
        coords, labels = segment._prompt_arrays(np, {"box":{"x":0.25,"y":0.125,"width":0.5,"height":0.75}}, (1024,512))
        np.testing.assert_array_equal(coords, [[[256,64],[768,448]]])
        np.testing.assert_array_equal(labels, [[2,3]])

    def test_invalid_or_absent_prompts_fail(self):
        for prompt in ({}, {"points":[{"x":0.5,"y":0.5,"negative":True}]}, {"points":[{"x":float("nan"),"y":0.5}]}, {"box":{"x":0.5,"y":0,"width":0.6,"height":1}}):
            with self.subTest(prompt=prompt), self.assertRaises(ValueError):
                segment._prompt_arrays(np, prompt, (1024,512))

    def test_has_mask_flag_is_not_a_mask_tensor(self):
        names = ["image_embeddings","point_coords","point_labels","mask_input","has_mask_input","orig_im_size"]
        decoder = SimpleNamespace(get_inputs=lambda: [SimpleNamespace(name=n,shape=[2]) for n in names])
        coords, labels = segment._prompt_arrays(np, {"points":[{"x":0.7,"y":0.3}]}, (1024,512))
        feed = segment._decoder_feed(np, decoder, np.zeros((1,256,64,64)), Image.new("RGB",(800,400)), coords, labels)
        self.assertEqual(feed["has_mask_input"].shape, (1,))
        self.assertEqual(feed["mask_input"].shape, (1,1,256,256))
        np.testing.assert_array_equal(feed["orig_im_size"], [400,800])

    def test_best_scored_source_size_logits_become_binary_mask(self):
        masks = np.full((1,2,4,8), -2.0, dtype=np.float32)
        masks[0,1,1:3,2:6] = 0.2  # logits > 0; not a probability threshold
        encoder = SimpleNamespace(get_inputs=lambda:[SimpleNamespace(name="image",shape=[1,3,8,8])],run=lambda *_:[np.zeros((1,256,64,64))])
        decoder = SimpleNamespace(get_inputs=lambda:[],get_outputs=lambda:[SimpleNamespace(name="masks"),SimpleNamespace(name="iou_predictions")],run=lambda *_:[masks,np.array([[0.1,0.9]])])
        with tempfile.TemporaryDirectory() as tmp, patch.object(segment._common,"make_session",side_effect=[encoder,decoder]):
            out = str(Path(tmp)/"mask.png")
            segment._run_sam(np, Image, SimpleNamespace(prompts='{"points":[{"x":0.5,"y":0.5}]}',out=out), Image.new("RGB",(8,4)),"encoder","decoder")
            with Image.open(out) as image:
                self.assertEqual(image.getbbox(), (2,1,6,3))
                self.assertEqual(set(np.asarray(image).reshape(-1)), {0,255})


if __name__ == "__main__":
    unittest.main()

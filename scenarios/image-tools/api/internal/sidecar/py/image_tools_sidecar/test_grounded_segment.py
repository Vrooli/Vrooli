"""Grounded segmentation contracts that need no weights: prompt parsing, query form, box choice, layout."""
import os
import tempfile
import unittest

from image_tools_sidecar import grounded_segment, sam2


class PhraseTest(unittest.TestCase):
    def test_phrase_is_normalised(self):
        self.assertEqual(grounded_segment.parse_phrase('{"text": "  the   tan  jacket "}'), "the tan jacket")

    def test_rejects_missing_empty_or_extra_prompt_keys(self):
        for raw in ('{}', '{"text": ""}', '{"text": "  "}', '{"text": 3}', '{"text": "a", "box": {}}', '[]'):
            with self.subTest(raw=raw), self.assertRaises(ValueError):
                grounded_segment.parse_phrase(raw)

    def test_rejects_overlong_phrase(self):
        with self.assertRaises(ValueError):
            grounded_segment.parse_phrase('{"text": "%s"}' % ("a" * (grounded_segment.MAX_PHRASE_CHARS + 1)))

    def test_detector_query_is_lowercase_and_full_stopped_once(self):
        self.assertEqual(grounded_segment.detector_query("The Jacket."), "the jacket.")
        self.assertEqual(grounded_segment.detector_query("hair"), "hair.")


class BoxChoiceTest(unittest.TestCase):
    def test_highest_score_wins(self):
        box, score = grounded_segment.best_box([[0, 0, 5, 5], [1, 1, 9, 9]], [0.3, 0.61])
        self.assertEqual(box, [1.0, 1.0, 9.0, 9.0])
        self.assertAlmostEqual(score, 0.61)

    def test_no_detections_is_none(self):
        self.assertIsNone(grounded_segment.best_box([], []))


class LayoutTest(unittest.TestCase):
    def test_grounded_install_needs_both_directories(self):
        with tempfile.TemporaryDirectory() as root:
            with self.assertRaises(ValueError):
                grounded_segment.layout(root)
            os.makedirs(os.path.join(root, "detector"))
            os.makedirs(os.path.join(root, "segmenter"))
            self.assertEqual(
                grounded_segment.layout(root),
                (os.path.join(root, "detector"), os.path.join(root, "segmenter")),
            )

    def test_sam2_reads_the_nested_checkpoint_of_a_grounded_install(self):
        with tempfile.TemporaryDirectory() as root:
            self.assertEqual(sam2.segmenter_dir(root), root)
            os.makedirs(os.path.join(root, "segmenter"))
            self.assertEqual(sam2.segmenter_dir(root), os.path.join(root, "segmenter"))


if __name__ == "__main__":
    unittest.main()

"""Semantic parser contracts that need no weights: phrase -> label resolution, prompt parsing, mask cleanup."""
import unittest

import numpy as np

from image_tools_sidecar import _common, semantic_segment as seg

# The real label sets of the two installed parsers (their config.json id2label).
CLOTHES = {
    0: "Background", 1: "Hat", 2: "Hair", 3: "Sunglasses", 4: "Upper-clothes", 5: "Skirt", 6: "Pants",
    7: "Dress", 8: "Belt", 9: "Left-shoe", 10: "Right-shoe", 11: "Face", 12: "Left-leg", 13: "Right-leg",
    14: "Left-arm", 15: "Right-arm", 16: "Bag", 17: "Scarf",
}
FASHION = {
    0: "unlabelled", 1: "shirt, blouse", 2: "top, t-shirt, sweatshirt", 3: "sweater", 4: "cardigan", 5: "jacket",
    6: "vest", 7: "pants", 8: "shorts", 9: "skirt", 10: "coat", 11: "dress", 12: "jumpsuit", 13: "cape",
    14: "glasses", 15: "hat", 16: "headband, head covering, hair accessory", 17: "tie", 18: "glove", 19: "watch",
    20: "belt", 21: "leg warmer", 22: "tights, stockings", 23: "sock", 24: "shoe", 25: "bag, wallet", 26: "scarf",
}


class ResolveLabelsTest(unittest.TestCase):
    def test_hair_is_the_hair_label_and_never_a_hair_accessory(self):
        self.assertEqual(seg.resolve_labels("her hair", CLOTHES), [2])
        with self.assertRaises(_common.InputError):  # the fashion parser has no hair class, only an accessory
            seg.resolve_labels("hair", FASHION)

    def test_the_same_word_picks_the_closest_label_each_parser_has(self):
        self.assertEqual(seg.resolve_labels("the jacket", FASHION), [5])
        self.assertEqual(seg.resolve_labels("the jacket", CLOTHES), [4])  # no jacket class: upper-clothes
        self.assertEqual(seg.resolve_labels("top", FASHION), [2])
        self.assertEqual(seg.resolve_labels("a t-shirt", FASHION), [2])

    def test_outfit_selects_every_garment_the_parser_has(self):
        self.assertEqual(seg.resolve_labels("outfit", CLOTHES), [4, 5, 6, 7, 8, 17])
        self.assertEqual(seg.resolve_labels("her clothing", FASHION), [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 17, 20, 22, 26])

    def test_side_labels_answer_to_the_plain_word(self):
        self.assertEqual(seg.resolve_labels("shoes", CLOTHES), [9, 10])
        self.assertEqual(seg.resolve_labels("arms", CLOTHES), [14, 15])

    def test_unknown_things_say_what_the_parser_can_select(self):
        with self.assertRaises(ValueError) as caught:
            seg.resolve_labels("the coffee cup", CLOTHES)
        self.assertIn("hair", str(caught.exception))
        self.assertIn("grounded", str(caught.exception))

    def test_jeans_are_pants(self):
        self.assertEqual(seg.resolve_labels("dark jeans", CLOTHES), [6])


class PromptTest(unittest.TestCase):
    def test_text_and_points(self):
        self.assertEqual(seg.parse_prompt('{"text": "  the   hair "}'), ("text", "the hair"))
        self.assertEqual(seg.parse_prompt('{"points":[{"x":0.1,"y":0.2},{"x":0.3,"y":0.4,"negative":true}]}'), ("points", [(0.1, 0.2)]))

    def test_rejects_unusable_prompts(self):
        for raw in ('{"text": ""}', '{"text": 3}', '{"text": "a", "points": []}', '{"box": {"x":0,"y":0,"width":1,"height":1}}',
                    '{"points":[{"x":0.1,"y":0.2,"negative":true}]}', '[]'):
            with self.subTest(raw=raw), self.assertRaises(ValueError):
                seg.parse_prompt(raw)

    def test_rejects_overlong_phrase(self):
        with self.assertRaises(ValueError):
            seg.parse_prompt('{"text": "%s"}' % ("a" * (seg.MAX_PHRASE_CHARS + 1)))


class PointsAndCleanupTest(unittest.TestCase):
    def test_points_select_the_labels_under_them_and_skip_background(self):
        label_map = np.zeros((10, 10), dtype=int)
        label_map[:, 5:] = 2
        self.assertEqual(seg.labels_under_points(label_map, [(0.9, 0.5), (0.1, 0.1)]), [2])
        with self.assertRaises(ValueError):
            seg.labels_under_points(label_map, [(0.1, 0.1)])

    def test_cleanup_drops_specks_and_fills_pinholes(self):
        mask = np.zeros((200, 200), dtype=bool)
        mask[20:120, 20:120] = True
        mask[60, 60] = False  # a pinhole
        mask[180, 180] = True  # a speck
        cleaned = seg.clean_mask(mask)
        self.assertTrue(cleaned[60, 60])
        self.assertFalse(cleaned[180, 180])
        self.assertTrue(cleaned[50, 50])


if __name__ == "__main__":
    unittest.main()

import base64
import importlib.util
from pathlib import Path
import unittest

SCRIPT = Path(__file__).with_name("decision_card.py")
SPEC = importlib.util.spec_from_file_location("decision_card", SCRIPT)
assert SPEC and SPEC.loader
decision_card = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(decision_card)


TINY_PNG = "data:image/png;base64," + base64.b64encode(b"\x89PNG\r\n\x1a\nfixture").decode()


def card():
    return {
        "schema": "research-feature-decision/v1",
        "id": "EX8-NAV-1",
        "title": "Keep the mobile navigation",
        "location": {"screen": "History page", "taps": ["Open Notification Hub", "Tap History"]},
        "summary": "The bar keeps five main destinations visible on a phone.",
        "evidence": [{"alt": "Before: the phone page has no bottom navigation", "image": TINY_PNG}],
        "options": [
            {"key": "keep", "label": "Keep", "effect": "Restore and retain the five-destination bar."},
            {"key": "drop", "label": "Drop", "effect": "Delete the bottom-navigation code."},
        ],
        "recommendation": "keep",
        "status": "resolved",
        "answer": "keep",
        "terms": [],
    }


class DecisionCardTests(unittest.TestCase):
    def test_accepts_a_resolved_card_with_embedded_evidence_and_effects(self):
        self.assertEqual([], decision_card.validate(card()))
        rendered = decision_card.render(card())
        self.assertIn(TINY_PNG, rendered)
        self.assertIn("Recorded answer: Keep", rendered)
        self.assertIn("Tap History", rendered)

    def test_rejects_a_link_instead_of_embedded_image(self):
        item = card()
        item["evidence"][0]["image"] = "/private/history-phone.png"
        self.assertTrue(any("embed" in error for error in decision_card.validate(item)))

    def test_rejects_bytes_that_do_not_match_the_declared_image_type(self):
        item = card()
        item["evidence"][0]["image"] = "data:image/png;base64," + base64.b64encode(b"not an image").decode()
        self.assertTrue(any("do not match png" in error for error in decision_card.validate(item)))

    def test_requires_a_real_drop_effect_and_screen_route(self):
        item = card()
        item["options"][1]["effect"] = "Decline this choice."
        item["location"]["taps"] = []
        errors = decision_card.validate(item)
        self.assertTrue(any("deleted or removed" in error for error in errors))
        self.assertTrue(any("taps" in error for error in errors))

    def test_does_not_allow_an_answer_on_a_pending_card(self):
        item = card()
        item["status"] = "pending"
        self.assertTrue(any("must not claim an answer" in error for error in decision_card.validate(item)))

    def test_requires_one_plain_sentence_and_word_definitions_field(self):
        item = card()
        item["summary"] = "This bar keeps screens close. It stays visible."
        item.pop("terms")
        errors = decision_card.validate(item)
        self.assertTrue(any("one complete sentence" in error for error in errors))
        self.assertTrue(any("missing fields: terms" in error for error in errors))


if __name__ == "__main__":
    unittest.main()

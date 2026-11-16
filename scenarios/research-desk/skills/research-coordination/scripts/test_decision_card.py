import base64
import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + ".py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


decision_card, needs_you = load("decision_card"), load("needs_you")
TINY_PNG = "data:image/png;base64," + base64.b64encode(b"\x89PNG\r\n\x1a\nfixture").decode()


def card(**changes):
    value = {
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
    value.update(changes)
    return value


def bas_card():
    """A card as BAS's feature-inventory diff writes it (playwright-driver/scripts/feature-inventory.mjs)."""
    return card(id="FEATURE_LOSS_ROUTE__SETTINGS", feature_id="route:/settings", title="Settings",
                location={"screen": "/settings", "taps": ["Open /settings", "Review route: Settings"]},
                summary="This route was visible before the change and is missing from the served screen now.",
                evidence=[{"alt": "Before the change: Settings on /settings.", "image": TINY_PNG}],
                options=[{"key": "keep", "label": "Keep this feature", "effect": "Restore or retain this feature in the changed build."},
                         {"key": "drop", "label": "Drop this feature", "effect": "Delete or remove the code that provided this feature."}],
                status="pending", answer=None)


class Validate(unittest.TestCase):
    def test_accepts_resolved_and_bas_cards(self):
        self.assertEqual([], decision_card.validate(card()))
        self.assertEqual([], decision_card.validate(bas_card()))

    def test_rejects_a_link_instead_of_an_embedded_image(self):
        item = card(evidence=[{"alt": "Before", "image": "/private/history-phone.png"}])
        self.assertTrue(any("embed" in error for error in decision_card.validate(item)))

    def test_rejects_bytes_that_do_not_match_the_declared_image_type(self):
        item = card(evidence=[{"alt": "Before", "image": "data:image/png;base64," + base64.b64encode(b"not an image").decode()}])
        self.assertTrue(any("do not match png" in error for error in decision_card.validate(item)))

    def test_requires_a_real_drop_effect_and_screen_route(self):
        item = card(location={"screen": "History page", "taps": []})
        item["options"][1]["effect"] = "Decline this choice."
        errors = decision_card.validate(item)
        self.assertTrue(any("deleted or removed" in error for error in errors))
        self.assertTrue(any("taps" in error for error in errors))

    def test_does_not_allow_an_answer_on_a_pending_card(self):
        self.assertTrue(any("must not claim an answer" in error for error in decision_card.validate(card(status="pending"))))

    def test_requires_one_plain_sentence_and_the_terms_field(self):
        item = card(summary="This bar keeps screens close. It stays visible.")
        item.pop("terms")
        errors = decision_card.validate(item)
        self.assertTrue(any("one complete sentence" in error for error in errors))
        self.assertTrue(any("missing fields: terms" in error for error in errors))


class Render(unittest.TestCase):
    def test_a_card_renders_as_a_hub_decision_page(self):
        page = decision_card.render(card(terms=[{"word": "Bar", "meaning": "The row of buttons."}]))
        self.assertTrue(page.startswith("<!doctype html>"))
        for want in (TINY_PNG, 'class="hub-figure"', 'data-od-answer="keep"', 'data-od-answer="drop"', "Recommended", "Tap History",
                     "Answered:</strong> keep", "The row of buttons.", "window.hub.prefill"):
            self.assertIn(want, page)

    def test_render_prints_the_needs_you_item_that_sends_the_page(self):
        with tempfile.TemporaryDirectory() as directory:
            source, page = Path(directory, "card.json"), Path(directory, "page.html")
            source.write_text(json.dumps(bas_card()))
            result = subprocess.run([sys.executable, "-I", str(Path(__file__).with_name("decision_card.py")), "render", "--file", str(source), "--output", str(page)],
                                    capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("Settings", page.read_text())
            item = result.stdout.split("needs_you.py write):\n", 1)[1]
            items, problems = needs_you.parse("## Needs you\n" + item, "Needs you")
            self.assertEqual(problems, [])
            packet = page.with_name("BRIEF.md")
            self.assertEqual(items, [{"id": "FEATURE_LOSS_ROUTE__SETTINGS", "title": "Keep or drop “Settings”?", "options": ["keep", "drop"],
                                      "recommended": "keep", "explainer": str(page.resolve()), "packet": str(packet.resolve())}])
            self.assertIn("- **`keep` (recommended).** Restore or retain this feature", packet.read_text())


if __name__ == "__main__":
    unittest.main()

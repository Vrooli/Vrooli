import importlib.util
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("needs_you.py")
SPEC = importlib.util.spec_from_file_location("needs_you", SCRIPT)
assert SPEC and SPEC.loader
needs_you = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(needs_you)


class NeedsYouGrammarTests(unittest.TestCase):
    def test_rejects_recommendation_missing_from_options_with_fix_hint(self):
        text = "## Needs you\n\n### AV4 — Pick one\nRecommended: keep — Keep it.\nOptions: propose — Propose.\n"
        errors = needs_you.validate(text, "Needs you")
        self.assertEqual(len(errors), 1)
        self.assertIn("Recommended key 'keep' is not offered", errors[0])
        self.assertIn("Fix the Recommended key or add it to Options:", errors[0])

    def test_accepts_matching_explicit_and_default_options(self):
        explicit = "## Needs you\n### AV4 — Pick\nRecommended: keep — Keep it.\nOptions: keep — Keep; wait — Wait.\n"
        default = "## Needs you\n### AV5 — Pick\nRecommended: accept — Accept it.\n"
        self.assertEqual(needs_you.validate(explicit + default, "Needs you"), [])

    def test_ignores_other_sections(self):
        text = "## Needs you\n### AV4 — Pick\nRecommended: keep — Keep it.\nOptions: keep — Keep.\n## Closeout\nRecommended: nope — ignored.\n"
        self.assertEqual(needs_you.validate(text, "Needs you"), [])


if __name__ == "__main__":
    unittest.main()

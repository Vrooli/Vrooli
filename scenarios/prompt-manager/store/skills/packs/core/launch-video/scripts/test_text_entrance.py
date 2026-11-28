import unittest

from text_entrance import classify_samples


def sample(time, *words):
    return {'time_seconds': time, 'sample_step_seconds': .1, 'words': list(words)}


class TextEntranceTests(unittest.TestCase):
    def test_clean_atomic_full_phrase_and_hold_pass(self):
        result = classify_samples('Great agents', [sample(t, 'Great', 'agents') for t in (0, .1, .2, .3)],
                                  require_atomic_entrance=True, minimum_hold_seconds=.3)
        self.assertEqual(result['status'], 'pass')
        self.assertAlmostEqual(result['longest_complete_hold_seconds'], .4)

    def test_ocr_typo_is_not_counted_as_a_required_word(self):
        result = classify_samples('Great agents', [sample(.3, 'Great', 'agenis.'),
                                   sample(.4, 'Great', 'agents.')],
                                  require_atomic_entrance=True, minimum_hold_seconds=.2)
        self.assertEqual(result['status'], 'fail')
        self.assertEqual(result['first_partial_seconds'], .3)
        self.assertEqual(result['first_complete_seconds'], .4)



    def test_confidence_threshold_cannot_be_lowered_below_50(self):
        with self.assertRaisesRegex(ValueError, 'between 50 and 100'):
            classify_samples('Great agents', [sample(.1, 'Great', 'agents')],
                             minimum_word_confidence=49.9)

    def test_low_confidence_exact_word_does_not_complete_cue(self):
        low = {'time_seconds': .3, 'sample_step_seconds': .1,
               'ocr_tokens': [{'text': 'Great', 'confidence': 96.0},
                              {'text': 'agents.', 'confidence': 35.3}]}
        high = {'time_seconds': .4, 'sample_step_seconds': .1,
                'ocr_tokens': [{'text': 'Great', 'confidence': 96.0},
                               {'text': 'agents.', 'confidence': 94.9}]}
        result = classify_samples('Great agents', [low, high],
                                  require_atomic_entrance=True, minimum_hold_seconds=.2)
        self.assertEqual(result['status'], 'fail')
        self.assertEqual(result['first_partial_seconds'], .3)
        self.assertEqual(result['first_complete_seconds'], .4)

    def test_planted_partial_entrance_is_caught_at_exact_time(self):
        result = classify_samples('Great agents', [sample(0), sample(.1, 'Great'),
                                   sample(.2, 'Great'), sample(.3, 'Great', 'agents')],
                                  require_atomic_entrance=True)
        self.assertEqual(result['status'], 'fail')
        finding = next(f for f in result['findings'] if f['gate'] == 'partial-text-entrance')
        self.assertEqual(finding['first_partial_seconds'], .1)

    def test_planted_short_readable_hold_is_caught(self):
        result = classify_samples('Great agents', [sample(0), sample(.1, 'Great', 'agents'),
                                   sample(.2, 'Great', 'agents'), sample(.3)],
                                  minimum_hold_seconds=.5)
        self.assertIn('visible-reading-time', {f['gate'] for f in result['findings']})


if __name__ == '__main__':
    unittest.main()

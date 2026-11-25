"""Contracts of the labelled-corpus replay (pure JSON input, no models)."""

import importlib.util
import os
import unittest

PATH = os.path.join(os.path.dirname(__file__), "safety-corpus-replay.py")
spec = importlib.util.spec_from_file_location("safety_corpus_replay", PATH)
replay = importlib.util.module_from_spec(spec)
spec.loader.exec_module(replay)


def case(label, *, scores=None, faces=None, thresholds=None, reason="sexual_minor_face", action="refuse"):
    finding = {
        "nsfw_score": (scores or {}).get("falconsai-nsfw", 0.0),
        "sexual": action in ("refuse", "flag"),
        "refuse": action == "refuse",
        "faces": faces or [],
    }
    if scores is not None:
        finding["nsfw_scores"] = scores
    return {
        "schema_version": 1,
        "job_id": "job-1",
        "label": label,
        "safety": {"action": action, "reason_code": reason},
        "evidence": {
            "thresholds": thresholds or {"falconsai-nsfw": 0.30, "adamcodd-vit-nsfw": 0.80, "age": 0.20, "face.min_px": 40},
            "models": ["falconsai-nsfw", "adamcodd-vit-nsfw"],
        },
        "finding": finding,
    }


class ReplayTests(unittest.TestCase):
    def test_current_thresholds_reproduce_the_labelled_decisions(self):
        cases = [
            case("confirmed", scores={"falconsai-nsfw": 0.10, "adamcodd-vit-nsfw": 0.90}),
            case("false_positive", scores={"falconsai-nsfw": 0.40, "adamcodd-vit-nsfw": 0.10},
                 faces=[{"size_px": 120, "p_under_20": 0.50}]),
        ]
        report = replay.replay(cases, {}, None, None)
        self.assertEqual(report["counts"]["true_positive"], 1)
        self.assertEqual(report["counts"]["false_positive"], 1)
        self.assertEqual(report["false_positive_rate"], 0.5)
        self.assertEqual(report["error_rate"], 0.5)

    def test_a_threshold_override_can_turn_a_false_positive_into_a_true_negative(self):
        cases = [case("false_positive", scores={"falconsai-nsfw": 0.40}, thresholds={"falconsai-nsfw": 0.30})]
        self.assertEqual(replay.replay(cases, {}, None, None)["counts"]["false_positive"], 1)
        looser = replay.replay(cases, {"falconsai-nsfw": 0.50}, None, None)
        self.assertEqual(looser["counts"]["false_positive"], 0)
        self.assertEqual(looser["counts"]["true_negative"], 1)

    def test_raising_the_age_threshold_can_clear_a_minor_face_refusal(self):
        cases = [
            case("false_positive", scores={"falconsai-nsfw": 0.90},
                 faces=[{"size_px": 120, "p_under_20": 0.25}]),
        ]
        self.assertEqual(replay.replay(cases, {}, None, None)["counts"]["false_positive"], 1)
        stricter = replay.replay(cases, {}, 0.30, None)
        self.assertEqual(stricter["counts"]["false_positive"], 0)
        self.assertEqual(stricter["counts"]["true_negative"], 1)

    def test_an_unassessable_sexual_case_is_counted_and_flagged_as_second_opinion_dependent(self):
        cases = [case("confirmed", scores={"falconsai-nsfw": 0.90}, faces=[], reason="sexual_unassessable")]
        report = replay.replay(cases, {}, None, None)
        self.assertEqual(report["counts"]["true_positive"], 1)
        self.assertEqual(report["second_opinion_dependent"], 1)

    def test_uncertain_and_non_replayable_cases_stay_out_of_the_confusion_matrix(self):
        uncertain = case("uncertain", scores={"falconsai-nsfw": 0.9})
        no_finding = {"schema_version": 1, "label": "false_positive", "safety": {"action": "refuse", "reason_code": "sexual_minor_face"}}
        report = replay.replay([uncertain, no_finding], {}, None, None)
        self.assertEqual(report["counts"]["uncertain"], 1)
        self.assertEqual(report["counts"]["not_replayable"], 1)
        self.assertEqual(report["counts"]["false_positive"], 0)

    def test_a_model_capability_flag_is_not_replayed_from_scores(self):
        c = case("false_positive", scores={"falconsai-nsfw": 0.05}, action="flag", reason="model")
        report = replay.replay([c], {}, None, None)
        self.assertEqual(report["counts"]["not_replayable"], 1)
        self.assertEqual(report["counts"]["model_flagged"], 1)

    def test_a_missing_scores_map_is_not_replayable(self):
        c = case("confirmed", scores={"falconsai-nsfw": 0.9})
        del c["finding"]["nsfw_scores"]
        self.assertEqual(replay.replay([c], {}, None, None)["counts"]["not_replayable"], 1)


if __name__ == "__main__":
    unittest.main()

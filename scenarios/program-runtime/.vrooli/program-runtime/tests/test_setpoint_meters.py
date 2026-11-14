"""Setpoint meters must not read `unavailable` when the sensor answered.

An `unavailable` row and a healthy row are indistinguishable on a board: both are simply not
red. The learning rows are exactly where that mattered — advice application, fragment reuse and
attribution all declined themselves while the loop sat idle, so a completely unused learning
mechanism produced a board with nothing wrong on it. These tests hold the distinction:
a sensor that could not be read is unavailable, and a sensor that answered zero is a red zero.
"""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).parents[1]


class Handle:
    def __init__(self, rows, meta=None):
        self.rows = rows
        self._meta = meta or {}

    def head(self, n):
        return self.rows[:n]

    def count(self):
        return len(self.rows)

    def meta(self):
        return self._meta

    def map(self, fn):
        return Handle([fn(r) for r in self.rows])

    def sort(self, key, reverse=False):
        return Handle(sorted(self.rows, key=lambda r: r.get(key, 0), reverse=reverse))

    def group_by(self, key):
        return {}

    def filter(self, fn):
        return Handle([r for r in self.rows if fn(r)])


def comparison(applied, rejected, reliable=True, reason=""):
    """Shape of the vrooli-memory compare-outcomes envelope the advice row reads."""
    return Handle([{ "signals": {
        "rows": [{"row": "advice-outcomes", "reading": {"cohorts": []}}],
        "aggregate": {"advice": {"applied": applied, "rejected": rejected}},
        "reliability": {"reliable": reliable, "reason": reason},
    }}])


class SetpointMeters(unittest.TestCase):
    def read_board(self, *, advice, fragments, library=(), agent_runs=()):
        surface = NS()

        def unavailable(*_a, **_k):
            raise RuntimeError("scenario_not_running")

        namespace = NS(
            programs=NS(
                governance_share=unavailable, list=lambda **k: Handle(list(agent_runs)),
                portfolio=lambda **k: Handle([{}]), mine=unavailable,
            ),
            bindings=NS(act=unavailable, condition=unavailable),
            sessions=NS(delegations=unavailable),
            library=NS(list=lambda **k: Handle(list(library))),
            shapes=NS(list=unavailable),
            tasks=NS(),
        )
        scope = {
            "program": NS(
                inputs=lambda: {},
                classify=lambda exc: ("unavailable", "scenario_unreachable"),
                run=lambda states, state: self._drive(states, state),
            ),
            "program_runtime": namespace,
            "tasks": NS(learning_findings=unavailable, delivery_metrics=lambda **k: unavailable(),
                        fragment_list=lambda **k: fragments),
            "lib": NS(program_runtime=NS(portfolio_audit=unavailable)),
            "vrooli_memory": NS(),
            "gather": lambda *calls: [call() for call in calls],
            "advice_source": advice,
        }
        source = (ROOT / "setpoint-read.py").read_text()
        # Bind the two optional reads this board needs without standing up the scenarios.
        source = source.replace('"advice": lambda: advice_read(),', '"advice": lambda: advice_source,')
        source = source.replace('"fragments": lambda: optional_read("fragment_list"),',
                                '"fragments": lambda: fragments_source,')
        scope["fragments_source"] = fragments
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            exec(compile(source, "setpoint-read", "exec"), scope)
        return {r["row"]: r for r in json.loads(output.getvalue())["signals"]["rows"]}

    @staticmethod
    def _drive(states, state):
        while state:
            state = states[state]()

    def test_zero_advice_decisions_reads_as_an_out_of_band_zero(self):
        rows = self.read_board(advice=comparison(0, 0), fragments=Handle([{"fragments": []}]))
        advice = rows["advice-application-ratio"]
        self.assertFalse(advice["unavailable"], "an answered sensor is readable, not an outage")
        self.assertIs(advice["in_band"], False)
        self.assertEqual(0.0, advice["reading"]["ratio"])
        self.assertEqual("no_advice_decisions", advice["reason"])

    def test_applied_advice_above_the_band_reads_green(self):
        rows = self.read_board(advice=comparison(8, 2), fragments=Handle([{"fragments": []}]))
        advice = rows["advice-application-ratio"]
        self.assertIs(advice["in_band"], True)
        self.assertEqual(0.8, advice["reading"]["ratio"])
        self.assertIsNone(advice["reason"])

    def test_an_unreliable_source_is_still_declined_rather_than_scored(self):
        """A source that says its own scan was incomplete must not be read as a zero."""
        rows = self.read_board(advice=comparison(0, 0, reliable=False, reason="partial outcome scan"),
                               fragments=Handle([{"fragments": []}]))
        advice = rows["advice-application-ratio"]
        self.assertTrue(advice["unavailable"])
        self.assertIn("partial outcome scan", advice["reason"])

    def test_no_verified_fragment_names_the_empty_state_not_a_broken_sensor(self):
        rows = self.read_board(advice=comparison(0, 0), fragments=Handle([{"fragments": []}]))
        self.assertEqual("no_verified_fragments", rows["fragment-cache-hit-rate"]["reason"])

    # The library projection as the server actually emits it in-kernel: the analyzer's
    # learningVerbs carries bare names, while the contract's declared verbs carry the prefix.
    # These tests used to fake "learningVerbs": ["learn.act"], a shape the server never produces,
    # which is how the row passed here while reading 0 in production.
    ACT_PROGRAM = {"name": "program-runtime.learn-verbs-example",
                   "verbs": ["learn.task", "learn.act", "learn.outcome"],
                   "learningVerbs": ["act", "outcome", "task"]}
    TASK_ONLY = {"name": "other.program", "verbs": ["learn.task"], "learningVerbs": ["task"]}

    def test_act_adoption_is_red_while_nothing_has_learned_a_fragment(self):
        """The row that catches a complete mechanism no program calls."""
        rows = self.read_board(advice=comparison(0, 0), fragments=Handle([{"fragments": []}]),
                               library=[self.ACT_PROGRAM, self.TASK_ONLY])
        adoption = rows["act-adoption"]
        self.assertFalse(adoption["unavailable"])
        self.assertIs(adoption["in_band"], False)
        self.assertEqual(1, adoption["reading"]["programs_declaring_act"])
        self.assertEqual(["program-runtime.learn-verbs-example"], adoption["reading"]["examples"])
        self.assertEqual(0, adoption["reading"]["verified_fragments"])

    def test_act_adoption_counts_the_bare_verb_names_the_analyzer_emits(self):
        """Regression: the row read 0 in production while three programs declared learn.act.

        An `or` chain over learningVerbs and verbs stopped at the non-empty bare-name list, where
        "learn.act" never appears. Each projection must be able to count a program on its own.
        """
        analyzer_only = {"name": "a.analyzed", "learningVerbs": ["act", "task"]}
        declared_only = {"name": "b.declared", "verbs": ["learn.act"]}
        snake_case = {"name": "c.cli", "learning_verbs": ["act"]}
        rows = self.read_board(advice=comparison(0, 0), fragments=Handle([{"fragments": []}]),
                               library=[analyzer_only, declared_only, snake_case, self.TASK_ONLY])
        self.assertEqual(3, rows["act-adoption"]["reading"]["programs_declaring_act"])
        self.assertEqual(["a.analyzed", "b.declared", "c.cli"], rows["act-adoption"]["reading"]["examples"])

    def test_act_adoption_turns_green_on_the_first_verified_fragment(self):
        fragments = Handle([{"fragments": [{"step_key": "fragment-v2:a", "verified": 3, "contexts": 2,
                                            "contradicted_since_edit": 0, "cached_runs": 2}]}])
        rows = self.read_board(advice=comparison(0, 0), fragments=fragments, library=[self.ACT_PROGRAM])
        self.assertIs(rows["act-adoption"]["in_band"], True)
        self.assertEqual(1, rows["act-adoption"]["reading"]["verified_fragments"])

    def test_a_published_baseline_is_no_longer_a_promotion_candidate(self):
        """Regression: promoting the right fragment left this row red, so it could never read in band.

        The bridge marks a fragment `published` when its contract already declares it as a reviewed
        baseline. Only unpublished qualified fragments are candidates; published ones stay visible.
        """
        qualified = {"verified": 5, "contexts": 2, "contradicted_since_edit": 0, "cached_runs": 4}
        fragments = Handle([{"fragments": [
            dict(qualified, step_key="fragment-v2:promoted", published=True),
            dict(qualified, step_key="fragment-v2:pending"),
            {"step_key": "fragment-v2:young", "verified": 2, "contexts": 1, "contradicted_since_edit": 0},
        ]}])
        rows = self.read_board(advice=comparison(0, 0), fragments=fragments, library=[self.ACT_PROGRAM])
        self.assertEqual(1, rows["promotable-fragments"]["reading"])
        self.assertIs(rows["promotable-fragments"]["in_band"], False)
        reading = rows["fragment-cache-hit-rate"]["reading"]
        self.assertEqual(1, reading["published"])
        self.assertEqual(["fragment-v2:pending"], [c["step_key"] for c in reading["candidates"]])

    def test_every_qualified_fragment_published_reads_in_band(self):
        fragments = Handle([{"fragments": [{"step_key": "fragment-v2:promoted", "verified": 5, "contexts": 2,
                                            "contradicted_since_edit": 0, "cached_runs": 4, "published": True}]}])
        rows = self.read_board(advice=comparison(0, 0), fragments=fragments, library=[self.ACT_PROGRAM])
        self.assertEqual(0, rows["promotable-fragments"]["reading"])
        self.assertIs(rows["promotable-fragments"]["in_band"], True)

    def test_attribution_is_measured_from_the_agent_corpus_not_left_dark(self):
        runs = [{"callerRunId": ""} for _ in range(5)]
        rows = self.read_board(advice=comparison(0, 0), fragments=Handle([{"fragments": []}]), agent_runs=runs)
        attribution = rows["attribution"]
        self.assertFalse(attribution["unavailable"], "the corpus that program-adoption reads also answers this")
        self.assertIs(attribution["in_band"], False)
        self.assertEqual(5, attribution["reading"]["unattributed_agent_runs"])
        self.assertEqual(0.0, attribution["reading"]["attributed_share"])

    def test_fully_attributed_runs_read_green(self):
        runs = [{"callerRunId": "run-" + str(i)} for i in range(4)]
        rows = self.read_board(advice=comparison(0, 0), fragments=Handle([{"fragments": []}]), agent_runs=runs)
        self.assertIs(rows["attribution"]["in_band"], True)
        self.assertEqual(1.0, rows["attribution"]["reading"]["attributed_share"])

    def test_an_empty_agent_corpus_is_still_declined(self):
        """No runs at all is genuinely unmeasurable; that distinction must survive."""
        rows = self.read_board(advice=comparison(0, 0), fragments=Handle([{"fragments": []}]), agent_runs=())
        self.assertTrue(rows["attribution"]["unavailable"])


if __name__ == "__main__":
    unittest.main()


class RegenerateSetpointReadings(unittest.TestCase):
    """The documented refresh path must not depend on a hand-bumped row count.

    The generator refused every run with "expected 19 setpoint rows, got 21" while the table and
    the envelope agreed name for name, so the Today column could not be refreshed at all.
    """

    def regenerate(self, table_rows, envelope_rows):
        import importlib.util
        import tempfile
        script = ROOT.parents[1] / "scripts" / "regenerate-setpoint-readings.py"
        spec = importlib.util.spec_from_file_location("regenerate_setpoint_readings", script)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        header = "| Row | Sensor | Band | Today (generated) |\n|---|---|---|---|\n"
        body = "".join(f"| {name} | sensor | band | old |\n" for name in table_rows)
        skill = Path(tempfile.mkdtemp()) / "SKILL.md"
        skill.write_text("# Skill\n\n## 2. Setpoint\n\n" + header + body + "\n### 3. Sensors\n\ntail\n")
        envelope = {"signals": {"rows": [{"row": name, "reading": 1, "in_band": True} for name in envelope_rows]}}
        module.rewrite(skill, envelope, "2026-09-10")
        return skill.read_text()

    def test_any_row_count_regenerates_when_table_and_envelope_agree(self):
        names = [f"row-{i}" for i in range(21)]
        self.assertEqual(21, self.regenerate(names, names).count("2026-09-10: 1 (in band)"))

    def test_a_row_the_table_lacks_is_refused(self):
        with self.assertRaisesRegex(ValueError, "skill is missing setpoint rows: row-new"):
            self.regenerate(["row-a"], ["row-a", "row-new"])

    def test_a_row_the_envelope_no_longer_reports_is_refused(self):
        with self.assertRaisesRegex(ValueError, "envelope does not report skill setpoint rows: row-gone"):
            self.regenerate(["row-a", "row-gone"], ["row-a"])

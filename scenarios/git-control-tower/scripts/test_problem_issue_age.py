from datetime import date
import importlib.util
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).with_name("problem_issue_age.py")
SPEC = importlib.util.spec_from_file_location("gct_problem_issue_age", SCRIPT)
age = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = age
SPEC.loader.exec_module(age)


class ProblemIssueAgeTests(unittest.TestCase):
    def test_current_and_stale_anchors_map_by_stable_id(self):
        source = """# Problems: GCT

## Open — recent issue <!-- pt5-id: GCT-RECENT; opened: 2026-11-18 -->
- Still unresolved.

## Open — old issue <!-- pt5-id: GCT-OLD; opened: 2026-10-01 -->
- Still unresolved.
"""
        found = age.parse_open_issues(source, as_of=date(2026, 11, 21))
        self.assertEqual([item.item_id for item in found], ["GCT-RECENT", "GCT-OLD"])
        self.assertEqual([item.age_days for item in found], [3, 51])
        self.assertEqual(len(found), 2)
        evidence = age.item_age_evidence(source, "PROBLEMS.md", as_of=date(2026, 11, 21))
        self.assertEqual(evidence["status"], "available")
        self.assertEqual(evidence["eligible_count"], 2)
        self.assertEqual(evidence["aged_count"], 2)
        self.assertEqual(evidence["items"][0], {
            "id": "GCT-RECENT", "opened_at": "2026-11-18", "age_days": 3,
            "source_ref": "PROBLEMS.md#pt5-id=GCT-RECENT",
            "basis": "explicit stable item ID and opened date on canonical Open heading",
        })
    def test_open_heading_without_anchor_is_malformed(self):
        with self.assertRaisesRegex(ValueError, "lacks a valid PT5 ID/opened date anchor"):
            age.parse_open_issues("## Open — unanchored issue\n- unresolved\n")

    def test_resolved_history_and_deferred_ideas_do_not_count(self):
        source = """## Corrected — old issue (2026-11-01)
- Historical text.

## Open — current issue <!-- pt5-id: GCT-CURRENT; opened: 2026-11-20 -->
- Still unresolved.

## Deferred Ideas
- Not an active issue.
"""
        found = age.parse_open_issues(source, as_of=date(2026, 11, 21))
        self.assertEqual([(item.item_id, item.age_days) for item in found], [("GCT-CURRENT", 1)])

    def test_dated_work_ladder_history_is_not_mistaken_for_open_issue_age(self):
        source = """## Resolved — old repair (2026-11-17)
- Evidence and closure disposition.

## Work ladder — native voice implementation (2026-10-21)
- Historical checkpoints and follow-ups.

## 2026-10-19 — Markdown preview repair (W3)
- Implementation evidence.
"""
        with self.assertRaisesRegex(ValueError, "cannot infer an empty backlog"):
            age.parse_open_issues(source, as_of=date(2026, 11, 21))
        evidence = age.item_age_evidence(source, "owner/PROBLEMS.md", as_of=date(2026, 11, 21))
        self.assertEqual(evidence["status"], "unavailable")
        self.assertIsNone(evidence["eligible_count"])
        self.assertEqual(evidence["aged_count"], 0)
        self.assertEqual(evidence["items"], [])

    def test_unanchored_open_section_has_unknown_denominator_not_zero(self):
        evidence = age.item_age_evidence("## Open — current work\n- unresolved\n", "owner/PROBLEMS.md")
        self.assertEqual(evidence["status"], "unavailable")
        self.assertIsNone(evidence["eligible_count"])
        self.assertEqual(evidence["aged_count"], 0)
        self.assertEqual(evidence["items"], [])

    def test_dated_entry_archive_without_explicit_open_state_is_unknown(self):
        source = """# Problems — Owner
## Entries
### 2026-11-10 — current incident follow-up
- Decision and recovery work remain in progress.
### 2026-06-03 — prior repair (resolved)
- Historical resolution evidence.
"""
        evidence = age.item_age_evidence(source, "owner/PROBLEMS.md", as_of=date(2026, 11, 21))
        self.assertEqual(evidence["status"], "unavailable")
        self.assertIsNone(evidence["eligible_count"])
        self.assertEqual(evidence["items"], [])
        self.assertIn("cannot infer an empty backlog", evidence["reason"])

    def test_annotated_open_issue_bullets_count_and_map_dates(self):
        source = """## Open Issues
### P0 Blockers
- Resolved in owner repair; do not count this historical bullet.
### P1 Issues
- <!-- pt5-id: AUTOHEAL-RETENTION-SOAK; opened: 2026-08-13 --> The retention soak remains outstanding.
- <!-- pt5-id: AUTOHEAL-TESTGENIE-STORAGE-WORKFLOW; opened: 2026-08-13 --> Test evidence remains unavailable.
### P2 Minor Issues
_None_
## Deferred Ideas
- Not an issue.
"""
        evidence = age.item_age_evidence(source, "scenarios/vrooli-autoheal/docs/internal/PROBLEMS.md",
                                         as_of=date(2026, 11, 21))
        self.assertEqual(evidence["status"], "available")
        self.assertEqual(evidence["eligible_count"], 2)
        self.assertEqual(evidence["aged_count"], 2)
        self.assertEqual([item["id"] for item in evidence["items"]], [
            "AUTOHEAL-RETENTION-SOAK", "AUTOHEAL-TESTGENIE-STORAGE-WORKFLOW"])
        self.assertEqual([item["age_days"] for item in evidence["items"]], [100, 100])
        self.assertTrue(all(item["opened_at"] == "2026-08-13" for item in evidence["items"]))

    def test_any_unannotated_open_issue_bullet_keeps_whole_owner_row_unavailable(self):
        source = """## Open Issues
### P1 Issues
- <!-- pt5-id: AUTOHEAL-KNOWN; opened: 2026-08-13 --> Known date.
- Another open item with unknown date.
"""
        evidence = age.item_age_evidence(source, "owner/PROBLEMS.md", as_of=date(2026, 11, 21))
        self.assertEqual(evidence["status"], "unavailable")
        self.assertIsNone(evidence["eligible_count"])
        self.assertEqual(evidence["aged_count"], 0)
        self.assertEqual(evidence["items"], [])
        self.assertIn("lacks a valid PT5 ID/opened date anchor", evidence["reason"])

    def test_four_explicit_owner_open_items_have_complete_age_coverage(self):
        source = """## Open Issues
- <!-- pt5-id: SYSTEM-MONITOR-PROCESS-KILL; opened: 2026-02-16 --> Process kill endpoint is missing.
- <!-- pt5-id: SYSTEM-MONITOR-MEMORY-LIFETIME; opened: 2026-02-16 --> Development memory loses history.
- <!-- pt5-id: SYSTEM-MONITOR-AUTH; opened: 2026-02-16 --> Authentication is absent.
- <!-- pt5-id: SYSTEM-MONITOR-STREAMING; opened: 2026-02-16 --> UI updates use polling.
## Resolved Issues
- Not part of the current open denominator.
"""
        evidence = age.item_age_evidence(source, "scenarios/system-monitor/docs/internal/PROBLEMS.md",
                                         as_of=date(2026, 11, 21))
        self.assertEqual(evidence["status"], "available")
        self.assertEqual(evidence["eligible_count"], 4)
        self.assertEqual(evidence["aged_count"], 4)
        self.assertEqual([item["age_days"] for item in evidence["items"]], [278] * 4)
        self.assertEqual([item["opened_at"] for item in evidence["items"]], ["2026-02-16"] * 4)

    def test_unanchored_open_section_has_unknown_denominator_not_zero(self):
        evidence = age.item_age_evidence("## Open — current work\n- unresolved\n", "owner/PROBLEMS.md")
        self.assertEqual(evidence["status"], "unavailable")
        self.assertIsNone(evidence["eligible_count"])
        self.assertEqual(evidence["aged_count"], 0)
        self.assertEqual(evidence["items"], [])

    def test_concurrent_change_during_read_fails_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "PROBLEMS.md"
            path.write_text("## Open — issue <!-- pt5-id: GCT-A; opened: 2026-11-20 -->\n")
            real_fstat = age.os.fstat
            # Keep the descriptor valid across both fstat calls.
            with path.open("rb") as stream:
                stat = real_fstat(stream.fileno())
                changed = SimpleNamespace(st_dev=stat.st_dev, st_ino=stat.st_ino,
                                          st_size=stat.st_size, st_mtime_ns=stat.st_mtime_ns + 1)
            with patch.object(age.os, "fstat", side_effect=[stat, changed]):
                with self.assertRaisesRegex(ValueError, "changed while issue ages"):
                    age.measure(path, as_of=date(2026, 11, 21))


if __name__ == "__main__":
    unittest.main()

"""Behavioral fixtures for the owned program, invoked by Test Genie's Go unit phase."""
import io
import json
import unittest
from contextlib import redirect_stdout
from datetime import datetime, timezone, timedelta
from pathlib import Path
from types import SimpleNamespace

SOURCE = (Path(__file__).resolve().parents[2] / '.vrooli/program-runtime/vision-walk-prep.py').read_text()

class Handle:
    def __init__(self, rows=None, meta=None): self.rows, self.metadata = rows or [], meta or {}
    def head(self, n): return self.rows[:n]
    def count(self): return len(self.rows)
    def meta(self): return self.metadata
    def filter(self, fn): return Handle([r for r in self.rows if fn(r)], self.metadata)
    def sort(self, key, reverse=False): return Handle(sorted(self.rows, key=lambda r:r.get(key,''), reverse=reverse), self.metadata)

class WalkTests(unittest.TestCase):
    def run_program(self, values=None, journal=None, failure=None, handoff=None, board=None, previous=None, work=None):
        now=datetime.now(timezone.utc).isoformat()
        calls=[]
        def invoke(name, fn):
            def call(**kwargs):
                calls.append((name,kwargs))
                if failure == 'all' or failure == name: raise RuntimeError('scenario is unreachable')
                return fn(kwargs)
            return call
        def journals(k):
            if k.get('kind','').startswith('vision-walk-briefing'): return Handle(previous or [])
            if k.get('kind','').startswith('walk-checkpoint'): return Handle(journal or [])
            return Handle([{'id':k['scope'],'body':'Verified record','createdAt':now,'kind':'observation'}])
        handoff = handoff or {'content':'Portfolio evidence','updatedAt':now}
        ns={'inputs': values or {}, 'gather':lambda *fs:[f() for f in fs],
            'command_center':SimpleNamespace(walk=SimpleNamespace(read=invoke('board',lambda _:Handle(board or [{'id':'zero','value':0,'trust':'VALID','coverage':'NOW','empirical':'MISS','observedAt':now}],{'generatedAt':now,'total':1})))),
            'swarm_manager':SimpleNamespace(goals=SimpleNamespace(list=invoke('goals',lambda _:Handle([{'goal':{'name':'g','title':'Goal','status':'active'}}]))),backlog=SimpleNamespace(list=invoke('work',lambda _:Handle(work or [])))),
            'prompt_manager':SimpleNamespace(team=SimpleNamespace(handoff_latest=invoke('handoff',lambda _:Handle(meta={'data':handoff})))),
            'meta_optimization_manager':SimpleNamespace(focus=SimpleNamespace(next=invoke('meta',lambda _:Handle([])))),
            'infrastructure_manager':SimpleNamespace(focus=SimpleNamespace(next=invoke('infra',lambda _:Handle([])))),
            'source_ledger':SimpleNamespace(journal=SimpleNamespace(list=invoke('journal',journals)))}
        out=io.StringIO()
        with redirect_stdout(out): exec(compile(SOURCE,'vision-walk-prep.py','exec'),ns)
        self.assertEqual(len(out.getvalue().splitlines()),1)
        self.assertLessEqual(len(out.getvalue().encode()),60000)
        return json.loads(out.getvalue()),calls
    def test_all_phases_and_real_zero(self):
        e,calls=self.run_program()
        self.assertEqual(e['status'],'ok')
        self.assertEqual([p['phase'] for p in e['signals']['phases']],['1','2','3','4','5','5.3','5.5','5.7','6','7','8','9'])
        self.assertEqual(e['signals']['sources']['outcomes']['rows'][0]['value'],0)
        self.assertEqual(e['signals']['fleet_health']['status'],'read_elsewhere')
        self.assertTrue(all(n in ['board','goals','work','handoff','meta','infra','journal'] for n,_ in calls))
    def test_outage_preserves_healthy_sources(self):
        e,_=self.run_program(failure='meta')
        self.assertEqual(e['status'],'partial')
        self.assertEqual(e['signals']['sources']['meta_focus']['status'],'unavailable')
        self.assertEqual(e['signals']['sources']['outcomes']['status'],'available')
    def test_all_unavailable_never_empty_success(self):
        e,_=self.run_program(failure='all'); self.assertEqual(e['status'],'unavailable')
        self.assertEqual(e['signals']['checkpoint']['status'],'unavailable')
    def test_invalid_inputs_do_not_call_owners(self):
        for value in [0,9,True,'5']:
            e,calls=self.run_program({'limit':value});self.assertEqual(e['status'],'failed');self.assertEqual(calls,[])
    def test_active_checkpoint_is_exact_even_when_old(self):
        cp={'walk_id':'walk-1','state':'active','resume_phase':'5.5','content':'## Walk Checkpoint\nUnicode ✓\nAlready filed: goal/x\n'}
        e,_=self.run_program(journal=[{'id':'checkpoint-id','body':json.dumps(cp),'createdAt':'2020-01-01T00:00:00Z'}])
        self.assertEqual(e['signals']['checkpoint']['checkpoint'],cp)
        self.assertEqual(e['signals']['checkpoint']['entry_id'],'checkpoint-id')
    def test_completion_prevents_legacy_resurrection(self):
        cp={'walk_id':'walk-1','state':'completed'}
        e,_=self.run_program(journal=[{'id':'done','body':json.dumps(cp)}],handoff={'content':'## Walk Checkpoint\nold','updatedAt':datetime.now(timezone.utc).isoformat()})
        self.assertEqual(e['signals']['checkpoint']['status'],'completed')
    def test_invalid_checkpoint_never_becomes_fresh_walk(self):
        e,_=self.run_program(journal=[{'id':'broken','body':'not-json'}])
        self.assertEqual(e['status'],'partial');self.assertEqual(e['signals']['checkpoint']['status'],'invalid')
    def test_stale_and_undated_handoffs_visible(self):
        for at in ['',(datetime.now(timezone.utc)-timedelta(days=3)).isoformat()]:
            e,_=self.run_program(handoff={'content':'old evidence','updatedAt':at})
            self.assertEqual(e['status'],'partial');self.assertGreater(e['signals']['quality']['stale_or_undated_sources'],0)
    def test_legacy_checkpoint_requires_a_real_heading_and_stops_at_next_section(self):
        e,_=self.run_program(handoff={'content':'No `## Walk Checkpoint` section exists.'})
        self.assertEqual(e['signals']['checkpoint']['status'],'none')
        e,_=self.run_program(handoff={'content':'Intro\n## Walk Checkpoint\nResume 5.5\n### Detail\nKeep this\n## Other\nExclude this'})
        self.assertEqual(e['signals']['checkpoint']['content'],'## Walk Checkpoint\nResume 5.5\n### Detail\nKeep this\n')
    def test_old_pending_items_are_selected_and_absence_never_means_resolved(self):
        rows=[{'kind':'idea','name':'new','status':'blocked','updated':'2026-09-04T00:00:00Z'},
              {'kind':'idea','name':'old','status':'review','updated':'2020-01-01T00:00:00Z'}]
        before,_=self.run_program({'limit':1},work=rows)
        self.assertEqual(before['signals']['sources']['pending_work']['rows'][0]['ref'],'idea/old')
        after,_=self.run_program({'limit':1},work=[rows[0]],previous=[{'id':'baseline','body':json.dumps({'envelope':before})}])
        delta=after['signals']['changes']
        self.assertEqual(delta['baseline_entry_id'],'baseline')
        self.assertEqual(delta['sources']['pending_work']['not_observed'],['idea/old'])
        self.assertNotIn('resolved',delta['sources']['pending_work'])
    def test_observation_refresh_is_not_a_changed_outcome(self):
        before,_=self.run_program(board=[{'id':'zero','value':0,'trust':'VALID','observedAt':'2026-09-01T00:00:00Z'}])
        after,_=self.run_program(board=[{'id':'zero','value':0,'trust':'VALID','observedAt':'2026-09-02T00:00:00Z'}],previous=[{'id':'baseline','body':json.dumps({'envelope':before})}])
        change=after['signals']['changes']['sources']['outcomes']
        self.assertEqual(change['changed'],[])
        self.assertEqual(change['refreshed'],['zero'])
    def test_test_channel_and_invalid_baseline_are_explicit(self):
        e,calls=self.run_program({'channel':'test'},previous=[{'id':'legacy','body':'old prose'}])
        self.assertEqual(e['signals']['changes']['status'],'invalid_baseline')
        kinds=[kw.get('kind') for name,kw in calls if name=='journal']
        self.assertIn('walk-checkpoint-test',kinds)
        self.assertIn('vision-walk-briefing-test',kinds)
    def test_handoff_changes_compare_full_content_not_timestamp_or_excerpt(self):
        original = 'Context ✓ ' * 200 + 'Deployment blocked.'
        before,_ = self.run_program(handoff={'content':original,'updatedAt':'2026-09-04T00:00:00Z'})
        previous = [{'id':'handoff-baseline','body':json.dumps({'envelope':before})}]
        for content, expected in [(original, 'unchanged'), (original + ' Decision needed.', 'changed')]:
            after,_ = self.run_program(handoff={'content':content,'updatedAt':'2026-09-05T00:00:00Z'}, previous=previous)
            self.assertEqual(after['signals']['changes']['baseline_entry_id'], 'handoff-baseline')
            for name in ('portfolio_handoff','strategist_handoff'):
                self.assertEqual(before['signals']['sources'][name]['text'], after['signals']['sources'][name]['text'])
                self.assertEqual(after['signals']['changes']['sources'][name]['content_change']['status'], expected)
    def test_handoff_comparison_preserves_missing_and_legacy_evidence(self):
        before,_ = self.run_program(handoff={'content':'Deployment blocked.'})
        for name in ('portfolio_handoff','strategist_handoff'):
            before['signals']['sources'][name].pop('content_sha256')
        for truncated, expected in [(False,'changed'), (True,'unknown')]:
            for name in ('portfolio_handoff','strategist_handoff'):
                before['signals']['sources'][name]['text_truncated'] = truncated
            previous = [{'id':'legacy-baseline','body':json.dumps({'envelope':before})}]
            after,_ = self.run_program(handoff={'content':'Deployment ready.'}, previous=previous)
            self.assertEqual(after['signals']['changes']['sources']['portfolio_handoff']['content_change']['status'], expected)
        after,_ = self.run_program(failure='handoff', previous=previous)
        self.assertEqual(after['signals']['changes']['sources']['portfolio_handoff']['content_change']['status'], 'unavailable')
        missing,_ = self.run_program(failure='handoff')
        after,_ = self.run_program(previous=[{'id':'outage-baseline','body':json.dumps({'envelope':missing})}])
        self.assertEqual(after['signals']['changes']['sources']['portfolio_handoff']['content_change']['status'], 'unknown')
    def test_oversized_checkpoint_fails_without_cut_json(self):
        cp={'walk_id':'w','state':'active','resume_phase':'2','content':'x'*70000}
        e,_=self.run_program(journal=[{'id':'huge','body':json.dumps(cp)}])
        self.assertEqual(e['status'],'failed');self.assertEqual(e['errors'][-1]['class'],'output_bound_exceeded')

if __name__=='__main__': unittest.main()

"""Behavioral fixtures for the owned program, invoked by Test Genie's Go unit phase."""
import io
import json
import unittest
from contextlib import redirect_stdout
from datetime import datetime, timezone, timedelta
from itertools import combinations
from pathlib import Path
from types import SimpleNamespace

SOURCE = (Path(__file__).resolve().parents[2] / '.vrooli/program-runtime/vision-walk-prep.py').read_text()

# Peer members publish continuity as declared ledger topics, one kind each.
RECORD_KINDS = {'goal-portfolio-record': 'portfolio_record', 'outcome-target-record': 'strategist_record',
                'contrarian-scan': 'contrarian_record'}
# Which scenario each faked binding belongs to; a demand lease is taken per scenario.
SCENARIO = {'board': 'command-center', 'goals': 'swarm-manager', 'work': 'swarm-manager',
            'meta': 'meta-optimization-manager', 'infra': 'infrastructure-manager',
            'journal': 'source-ledger', 'teams': 'prompt-manager'}

class Handle:
    def __init__(self, rows=None, meta=None): self.rows, self.metadata = rows or [], meta or {}
    def head(self, n): return self.rows[:n]
    def count(self): return len(self.rows)
    def meta(self): return self.metadata
    def filter(self, fn): return Handle([r for r in self.rows if fn(r)], self.metadata)
    def sort(self, key, reverse=False): return Handle(sorted(self.rows, key=lambda r:r.get(key,''), reverse=reverse), self.metadata)

class WalkTests(unittest.TestCase):
    ROSTER = ['director-swarm','monetization','marketing-crew','meta-optimization','infra-health','scenario-qa']

    def run_program(self, values=None, journal=None, failure=None, records=None, board=None, previous=None, work=None, roster=None, failure_message='scenario is unreachable'):
        now=datetime.now(timezone.utc).isoformat()
        calls=[]
        slots=[]
        current={'slot':None}
        records = records or {}
        def invoke(name, fn):
            def call(**kwargs):
                calls.append((name,kwargs))
                if current['slot'] is not None: slots[current['slot']].add(SCENARIO[name])
                if failure == 'all' or failure == name: raise RuntimeError(failure_message)
                return fn(kwargs)
            return call
        def fake_gather(*fs):
            out=[]
            for f in fs:
                slots.append(set()); current['slot']=len(slots)-1
                out.append(f())
            current['slot']=None
            return out
        def journals(k):
            kind=k.get('kind','')
            if kind.startswith('vision-walk-briefing'): return Handle(previous or [])
            if kind.startswith('walk-checkpoint'): return Handle(journal or [])
            if kind in RECORD_KINDS:
                return Handle(records.get(kind, [{'id':kind+'-1','body':'Peer evidence','createdAt':now,'kind':kind}]))
            return Handle([{'id':k['scope'],'body':'Verified record','createdAt':now,'kind':'observation'}])
        def backlog(k):
            return Handle([r for r in (work or []) if r.get('status') in k.get('statuses', [])
                           and (not r.get('archivedAt') or k.get('archived') == 'ARCHIVED_FILTER_ALL')])
        ns={'inputs': values or {}, 'gather':fake_gather,
            'command_center':SimpleNamespace(walk=SimpleNamespace(read=invoke('board',lambda _:Handle(board or [{'id':'zero','value':0,'trust':'VALID','coverage':'NOW','empirical':'MISS','observedAt':now}],{'generatedAt':now,'total':1})))),
            'swarm_manager':SimpleNamespace(goals=SimpleNamespace(list=invoke('goals',lambda _:Handle([{'goal':{'name':'g','title':'Goal','status':'active'}}]))),backlog=SimpleNamespace(list=invoke('work',backlog))),
            'prompt_manager':SimpleNamespace(team=SimpleNamespace(
                list=invoke('teams',lambda _:Handle([{'id':t} for t in (self.ROSTER if roster is None else roster)])))),
            'meta_optimization_manager':SimpleNamespace(focus=SimpleNamespace(next=invoke('meta',lambda _:Handle([])))),
            'infrastructure_manager':SimpleNamespace(focus=SimpleNamespace(next=invoke('infra',lambda _:Handle([])))),
            'source_ledger':SimpleNamespace(journal=SimpleNamespace(list=invoke('journal',journals)))}
        out=io.StringIO()
        with redirect_stdout(out): exec(compile(SOURCE,'vision-walk-prep.py','exec'),ns)
        self.assertEqual(len(out.getvalue().splitlines()),1)
        self.assertLessEqual(len(out.getvalue().encode()),60000)
        return json.loads(out.getvalue()),calls,slots
    def test_all_phases_and_real_zero(self):
        e,calls,_=self.run_program()
        self.assertEqual(e['status'],'ok')
        self.assertEqual([p['phase'] for p in e['signals']['phases']],['1','2','3','4','5','5.3','5.5','5.7','5.9','6','7','8','9'])
        self.assertEqual(e['signals']['sources']['outcomes']['rows'][0]['value'],0)
        self.assertEqual(e['signals']['fleet_health']['status'],'read_elsewhere')
        self.assertTrue(all(n in SCENARIO for n,_ in calls))

    def test_contention_is_not_reported_as_a_scenario_outage(self):
        """The control-plane registry being busy is retryable and self-inflicted."""
        e,_,_=self.run_program(failure='journal')
        classes={x['class'] for x in e['errors'] if x['where'].startswith('team_notes')}
        self.assertEqual(classes,{'scenario_unreachable'})
        contended,_,_=self.run_program(failure='journal',failure_message='demand lease contention for source-ledger: database is locked (517)')
        classes={x['class'] for x in contended['errors'] if x['where'].startswith('team_notes')}
        self.assertEqual(classes,{'lease_contention'})
        self.assertNotIn('scenario_unreachable',[x['class'] for x in contended['errors']])

    def test_no_handoff_surface_is_read(self):
        """Peer continuity is a declared ledger topic, never a runtime response snapshot."""
        e,calls,_=self.run_program()
        self.assertNotIn('handoff', json.dumps(e).lower())
        self.assertEqual([n for n,_ in calls if n=='handoff'],[])
        for name,kind in [('portfolio_record','goal-portfolio-record'),('strategist_record','outcome-target-record'),
                          ('contrarian_record','contrarian-scan')]:
            source=e['signals']['sources'][name]
            self.assertEqual(source['binding'],'source-ledger/journal/list')
            self.assertEqual(source['record_kind'],kind)
            self.assertEqual(source['status'],'available')

    def test_peer_phases_carry_their_record(self):
        e,_,_=self.run_program()
        phases={p['phase']:p['sources'] for p in e['signals']['phases']}
        self.assertIn('portfolio_record',phases['3'])
        self.assertIn('contrarian_record',phases['3'])
        self.assertEqual(phases['4'],['strategist_record'])

    def test_silent_peer_record_is_empty_and_degrades_status(self):
        e,_,_=self.run_program(records={'contrarian-scan':[]})
        self.assertEqual(e['signals']['sources']['contrarian_record']['status'],'empty')
        self.assertEqual(e['signals']['quality']['silent_peer_records'],['contrarian_record'])
        self.assertEqual(e['status'],'partial')

    def test_stale_peer_record_stays_visible(self):
        old=(datetime.now(timezone.utc)-timedelta(days=3)).isoformat()
        e,_,_=self.run_program(records={'goal-portfolio-record':[{'id':'p1','body':'Old portfolio','createdAt':old,'kind':'goal-portfolio-record'}]})
        self.assertEqual(e['signals']['sources']['portfolio_record']['freshness'],'stale')
        self.assertEqual(e['status'],'partial')
        self.assertGreater(e['signals']['quality']['stale_or_undated_sources'],0)

    def test_peer_record_change_is_detected_by_entry_identity(self):
        before,_,_=self.run_program()
        previous=[{'id':'baseline','body':json.dumps({'envelope':before})}]
        now=datetime.now(timezone.utc).isoformat()
        after,_,_=self.run_program(records={'outcome-target-record':[{'id':'target-2','body':'New targets','createdAt':now,'kind':'outcome-target-record'}]},previous=previous)
        change=after['signals']['changes']['sources']['strategist_record']
        self.assertEqual(change['newly_observed'],['target-2'])
        self.assertEqual(change['not_observed'],['outcome-target-record-1'])
        self.assertNotIn('content_change',change)

    def test_undeclared_team_is_reported_not_read(self):
        e,calls,_=self.run_program(roster=self.ROSTER+['new-team'])
        self.assertEqual(e['signals']['teams']['undeclared'],['new-team'])
        self.assertEqual(e['signals']['quality']['teams_without_phase'],['new-team'])
        self.assertNotIn('team_notes:new-team', e['signals']['sources'])
        self.assertNotIn('team:new-team', [k.get('scope') for _,k in calls])

    def test_absent_declared_team_is_reported(self):
        e,_,_=self.run_program(roster=[t for t in self.ROSTER if t != 'scenario-qa'])
        self.assertEqual(e['signals']['teams']['declared_absent'],['scenario-qa'])
        self.assertNotIn('team_notes:scenario-qa', e['signals']['sources'])

    def test_roster_outage_never_reads_complete(self):
        e,_,_=self.run_program(failure='teams')
        self.assertEqual(e['signals']['teams']['status'],'unavailable')
        self.assertEqual(e['status'],'partial')
        self.assertIn('team_notes:scenario-qa', e['signals']['sources'])

    def test_outage_preserves_healthy_sources(self):
        e,_,_=self.run_program(failure='meta')
        self.assertEqual(e['status'],'partial')
        self.assertEqual(e['signals']['sources']['meta_focus']['status'],'unavailable')
        self.assertEqual(e['signals']['sources']['outcomes']['status'],'available')
    def test_all_unavailable_never_empty_success(self):
        e,_,_=self.run_program(failure='all'); self.assertEqual(e['status'],'unavailable')
        self.assertEqual(e['signals']['checkpoint']['status'],'unavailable')
    def test_invalid_inputs_do_not_call_owners(self):
        for value in [0,9,True,'5']:
            e,calls,_=self.run_program({'limit':value});self.assertEqual(e['status'],'failed');self.assertEqual(calls,[])
    def test_active_checkpoint_is_exact_even_when_old(self):
        cp={'walk_id':'walk-1','state':'active','resume_phase':'5.5','content':'## Walk Checkpoint\nUnicode ✓\nAlready filed: goal/x\n'}
        e,_,_=self.run_program(journal=[{'id':'checkpoint-id','body':json.dumps(cp),'createdAt':'2020-01-01T00:00:00Z'}])
        self.assertEqual(e['signals']['checkpoint']['checkpoint'],cp)
        self.assertEqual(e['signals']['checkpoint']['entry_id'],'checkpoint-id')
    def test_completed_checkpoint_is_terminal(self):
        e,_,_=self.run_program(journal=[{'id':'done','body':json.dumps({'walk_id':'walk-1','state':'completed'})}])
        self.assertEqual(e['signals']['checkpoint']['status'],'completed')
    def test_absent_checkpoint_is_none_without_a_fallback_surface(self):
        e,_,_=self.run_program()
        self.assertEqual(e['signals']['checkpoint'],{'status':'none'})
    def test_invalid_checkpoint_never_becomes_fresh_walk(self):
        e,_,_=self.run_program(journal=[{'id':'broken','body':'not-json'}])
        self.assertEqual(e['status'],'partial');self.assertEqual(e['signals']['checkpoint']['status'],'invalid')
    def test_old_pending_items_are_selected_and_absence_never_means_resolved(self):
        rows=[{'kind':'idea','name':'new','status':'in_review','updated':'2026-09-04T00:00:00Z'},
              {'kind':'idea','name':'old','status':'in_review','updated':'2020-01-01T00:00:00Z'}]
        before,_,_=self.run_program({'limit':1},work=rows)
        self.assertEqual(before['signals']['sources']['pending_work']['rows'][0]['ref'],'idea/old')
        after,_,_=self.run_program({'limit':1},work=[rows[0]],previous=[{'id':'baseline','body':json.dumps({'envelope':before})}])
        delta=after['signals']['changes']
        self.assertEqual(delta['baseline_entry_id'],'baseline')
        self.assertEqual(delta['sources']['pending_work']['not_observed'],['idea/old'])
        self.assertNotIn('resolved',delta['sources']['pending_work'])
    def test_recent_completed_work_includes_archived_records(self):
        rows=[{'kind':'fix','name':'recent','status':'completed','updated':'2026-09-06T00:00:00Z','archivedAt':'2026-09-06T01:00:00Z'},
              {'kind':'fix','name':'older','status':'completed','updated':'2026-08-01T00:00:00Z'}]
        e,_,_=self.run_program({'limit':1},work=rows)
        self.assertEqual(e['signals']['sources']['recent_work']['rows'][0]['ref'],'fix/recent')
        self.assertEqual(e['signals']['sources']['recent_work']['total'],2)
    def test_observation_refresh_is_not_a_changed_outcome(self):
        before,_,_=self.run_program(board=[{'id':'zero','value':0,'trust':'VALID','observedAt':'2026-09-01T00:00:00Z'}])
        after,_,_=self.run_program(board=[{'id':'zero','value':0,'trust':'VALID','observedAt':'2026-09-02T00:00:00Z'}],previous=[{'id':'baseline','body':json.dumps({'envelope':before})}])
        change=after['signals']['changes']['sources']['outcomes']
        self.assertEqual(change['changed'],[])
        self.assertEqual(change['refreshed'],['zero'])
    def test_test_channel_isolates_owned_records_only(self):
        e,calls,_=self.run_program({'channel':'test'},previous=[{'id':'stale','body':'old prose'}])
        self.assertEqual(e['signals']['changes']['status'],'invalid_baseline')
        kinds=[kw.get('kind') for name,kw in calls if name=='journal']
        self.assertIn('walk-checkpoint-test',kinds)
        self.assertIn('vision-walk-briefing-test',kinds)
        # Peer topics belong to their producers; a rehearsal never forks their continuity.
        for kind in RECORD_KINDS:
            self.assertIn(kind,kinds)
            self.assertNotIn(kind+'-test',kinds)
    def test_oversized_checkpoint_fails_without_cut_json(self):
        cp={'walk_id':'w','state':'active','resume_phase':'2','content':'x'*70000}
        e,_,_=self.run_program(journal=[{'id':'huge','body':json.dumps(cp)}])
        self.assertEqual(e['status'],'failed');self.assertEqual(e['errors'][-1]['class'],'output_bound_exceeded')

if __name__=='__main__': unittest.main()

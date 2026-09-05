"""Behavioral contract probes; adapters are isolated, no physical device is used."""
import contextlib
import io
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).parents[1]
class Handle:
    def __init__(self, rows): self.rows = rows
    def head(self, n): return self.rows[:n]
    def count(self): return len(self.rows)
    def filter(self, fn): return Handle([r for r in self.rows if fn(r)])
    def map(self, fn): return Handle([fn(r) for r in self.rows])

class Programs(unittest.TestCase):
    def run_program(self, name, inputs, api):
        scope = {'inputs': inputs, 'device_control': api}
        with contextlib.redirect_stdout(io.StringIO()):
            exec(compile((ROOT / (name + '.py')).read_text(), name, 'exec'), scope)
        return scope['envelope']

    def test_ambiguous_device_never_reads_or_runs_flows(self):
        api=NS(device=NS(list=lambda:Handle([{'id':'one','name':'TV'}, {'id':'two','name':'TV'}])))
        r=self.run_program('prepare-task', {'device':'TV','context_key':'app:v1'}, api)
        self.assertEqual('device_selection_required',r['errors'][0]['class'])

    def test_exact_id_wins_over_name(self):
        api=NS(device=NS(list=lambda:Handle([{'id':'tv','name':'TV'}, {'id':'other','name':'tv'}])), flow=NS(list=lambda **kw:Handle([])))
        r=self.run_program('prepare-task', {'device':'tv','context_key':'app:v1'}, api)
        self.assertEqual('tv',r['signals']['device_id'])

    def test_failed_candidate_never_saved(self):
        calls=[]
        api=NS(flow=NS(validate=lambda **kw:Handle([{'runnable':True}]), run=lambda **kw:Handle([{'runId':'failed','disposition':'failed'}]), save=lambda **kw:calls.append(kw)))
        r=self.run_program('author-flow',self.candidate(),api)
        self.assertEqual('failed',r['status']);self.assertEqual([],calls)

    def test_preflight_refusal_never_acts(self):
        def validate(**kw):
            self.assertTrue(kw['require_assertion'])
            raise RuntimeError('acceptance assertion missing')
        r=self.run_program('author-flow',self.candidate(),NS(flow=NS(validate=validate)))
        self.assertEqual('failed',r['status'])
        self.assertEqual('collect',r['errors'][0]['where'])

    def test_passing_candidate_saves_exact_source_once(self):
        saved=[]
        def save(**kw): saved.append(kw);return Handle([{'id':'durable','version':1}])
        api=NS(flow=NS(validate=lambda **kw:Handle([{'runnable':True}]),run=lambda **kw:Handle([{'runId':'verified','disposition':'passed'}]),save=save))
        r=self.run_program('author-flow',self.candidate(),api)
        self.assertEqual('ok',r['status']);self.assertEqual(1,len(saved))
        self.assertEqual('verified',saved[0]['run_id']);self.assertEqual('tv',saved[0]['device_id'])

    def test_replay_wrong_context_never_actuates(self):
        api=NS(flow=NS(get=lambda **kw:Handle([{'id':'saved','version':1,'deviceId':'different','contextKey':'app:v1'}])))
        r=self.run_program('replay-flow',{'flow_id':'saved','version':1,'device_id':'tv','context_key':'app:v1','actor':'test'},api)
        self.assertEqual('identity_mismatch',r['errors'][0]['class'])

    def candidate(self):
        return {'device_id':'tv','context_key':'app:v1','strategy_id':'tv','actor':'test','flow':{'steps':[{'id':'assert','kind':'property-assert'}]}}

if __name__=='__main__': unittest.main()

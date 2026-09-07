"""Behavioral contract probes; adapters are isolated, no physical device is used."""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).parents[1]
class Handle:
    def __init__(self, rows, raw=None): self.rows = rows; self._raw = rows if raw is None else raw
    def head(self, n): return self.rows[:n]
    def count(self): return len(self.rows)
    def filter(self, fn): return Handle([r for r in self.rows if fn(r)])
    def map(self, fn): return Handle([fn(r) for r in self.rows])
    def raw(self): return self._raw

class Programs(unittest.TestCase):
    def run_program(self, name, inputs, api, library=None, runtime=None):
        scope = {'inputs': inputs, 'device_control': api, 'lib': library, 'program_runtime': runtime}
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

    def test_task_discovery_does_not_execute_a_suggested_flow(self):
        library = NS(device_control=NS(prepare_task=lambda **kw:Handle([
            {'status':'ok','signals':{'device_id':'tv','flows':[{'id':'suggested','version':1}]}}])))
        result = self.run_program('do-task',{'device':'TV','context_key':'room:v1','actor':'test'},None,library)
        self.assertEqual('selection_required',result['errors'][0]['class'])
        self.assertEqual('unknown',result['signals']['outcome'])

    def test_task_preserves_verified_child_and_identity(self):
        calls=[]
        def replay(**kwargs):
            calls.append(kwargs)
            return Handle([{'status':'ok','signals':{'outcome':'verified_success','run_id':'run-1'},
                            'evidence':['run:run-1']}])
        library=NS(device_control=NS(prepare_task=lambda **kw:Handle([
            {'status':'ok','signals':{'device_id':'durable-tv'}}]),replay_flow=replay))
        result=self.run_program('do-task',{'device':'TV','context_key':'room:v1','actor':'test',
            'flow_id':'saved','version':3},None,library)
        self.assertEqual('verified_success',result['signals']['outcome'])
        self.assertEqual('durable-tv',calls[0]['device_id'])
        self.assertEqual(3,calls[0]['version'])
        self.assertEqual(['run:run-1'],result['evidence'])

    def test_task_does_not_upgrade_child_acknowledgement(self):
        library=NS(device_control=NS(prepare_task=lambda **kw:Handle([
            {'status':'ok','signals':{'device_id':'tv'}}]),
            author_flow=lambda **kw:Handle([{'status':'ok','signals':{'run_id':'pending'}}])))
        result=self.run_program('do-task',{'device':'TV','context_key':'room:v1','actor':'test',
            'flow':{'steps':[{'kind':'property-assert'}]}},None,library)
        self.assertEqual('unknown',result['signals']['outcome'])

    def test_task_refuses_ambiguous_intent_before_discovery(self):
        result=self.run_program('do-task',{'device':'TV','context_key':'room:v1','actor':'test',
            'flow':{'steps':[{}]},'flow_id':'saved','version':1},None,None)
        self.assertEqual('invalid_input',result['errors'][0]['class'])

    def test_volume_program_uses_one_semantic_binding(self):
        calls=[]
        def volume(**kwargs):
            calls.append(kwargs)
            return Handle([], {'status':'ok','operationId':'op-1','verificationClass':'conflicted','evidence':['state-before:op-1']})
        api=NS(device=NS(volume=volume))
        result=self.run_program('volume', {'device':'fixture-device','actor':'operator','goal':'turn down by 50 percent'}, api)
        self.assertEqual('ok', result['status'])
        self.assertEqual('conflicted', result['signals']['verification_class'])
        self.assertEqual('op-1', result['signals']['operation_id'])
        self.assertEqual({'device':'fixture-device','actor':'operator','goal':'turn down by 50 percent','_confirm':False}, calls[0])

    def test_volume_program_propagates_explicit_confirmation_control_flag(self):
        calls=[]
        def volume(**kwargs):
            calls.append(kwargs)
            return Handle([], {'status':'ok','operationId':'op-2','verificationClass':'verified','evidence':['state-after:op-2']})
        api=NS(device=NS(volume=volume))
        result=self.run_program('volume', {'device':'fixture-device','actor':'operator','goal':'set volume to 4%', 'confirm':True}, api)
        self.assertEqual('ok', result['status'])
        self.assertTrue(calls[0]['_confirm'])
        self.assertNotIn('confirm', calls[0])

    def test_volume_program_compacts_large_result_before_emitting_envelope(self):
        large_state = {'properties': {'volume': {'value': 0.04, 'status': 'available', 'reason': 'x' * 10000}}}
        def volume(**kwargs):
            return Handle([], {'status':'ok','operationId':'op-3','verificationClass':'verified',
                'before': large_state, 'after': large_state, 'plan': {'action_transport':'cast'},
                'evidence':['state-after:op-3']})
        api=NS(device=NS(volume=volume))
        result=self.run_program('volume', {'device':'fixture-device','actor':'operator','goal':'set volume to 4%', 'confirm':True}, api)
        self.assertEqual('ok', result['status'])
        self.assertLess(len(json.dumps(result)), 2000)
        self.assertEqual(0.04, result['signals']['result']['after']['properties']['volume']['value'])

    def test_volume_program_rejects_missing_semantic_request_without_binding(self):
        called=[]
        api=NS(device=NS(volume=lambda **kwargs: called.append(kwargs)))
        result=self.run_program('volume', {'device':'fixture-device','actor':'operator'}, api)
        self.assertEqual('invalid_input', result['errors'][0]['class'])
        self.assertEqual([], called)

    def test_setpoint_preserves_shared_unknowns_and_adds_owner_condition(self):
        calls=[]
        measured={'row':'completion-effort','reading':None,'unavailable':True,'reason':'empty_sample','in_band':None}
        def compare(**kwargs):
            calls.append(kwargs)
            return Handle([{'status':'ok','signals':{'rows':[measured]},'evidence':['memory:window']}])
        library=NS(vrooli_memory=NS(compare_outcomes=compare))
        runtime=NS(bindings=NS(condition=lambda **kw:Handle([{'status':'CONDITION_STATUS_HEALTHY'}])))
        result=self.run_program('setpoint-read',{'context_key':'tv:v1'},None,library,runtime)
        self.assertEqual({'scope':'device-control-usage','context_key':'tv:v1'},calls[0])
        self.assertEqual(measured,result['signals']['rows'][0])
        self.assertTrue(result['signals']['rows'][1]['in_band'])

    def test_setpoint_retains_measurement_if_condition_unavailable(self):
        library=NS(vrooli_memory=NS(compare_outcomes=lambda **kw:Handle([
            {'status':'ok','signals':{'rows':[{'row':'completion-effort','reading':None}]}}])))
        result=self.run_program('setpoint-read',{},None,library,NS())
        self.assertEqual('partial',result['status'])
        self.assertEqual('completion-effort',result['signals']['rows'][0]['row'])
        self.assertIsNone(result['signals']['rows'][1]['reading'])

    def candidate(self):
        return {'device_id':'tv','context_key':'app:v1','strategy_id':'tv','actor':'test','flow':{'steps':[{'id':'assert','kind':'property-assert'}]}}

if __name__=='__main__': unittest.main()

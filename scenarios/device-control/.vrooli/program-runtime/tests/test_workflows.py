"""Behavioral contract probes; adapters are isolated, no physical device is used."""
import contextlib
import io
import json
from pathlib import Path
import sys
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).resolve().parents[1]
KERNEL_HOST = ROOT.parents[2] / "program-runtime" / "kernel" / "host"
if str(KERNEL_HOST) not in sys.path:
    sys.path.insert(0, str(KERNEL_HOST))
from program_helper import ProgramHelper
class Handle:
    def __init__(self, rows, raw=None): self.rows = rows; self._raw = rows if raw is None else raw
    def head(self, n): return self.rows[:n]
    def count(self): return len(self.rows)
    def filter(self, fn): return Handle([r for r in self.rows if fn(r)])
    def map(self, fn): return Handle([fn(r) for r in self.rows])
    def raw(self): return self._raw

class Step:
    def __init__(self): self.record = {'outcome': 'unknown'}
    def __enter__(self): return self
    def __exit__(self, *_): return False
    def outcome(self, status, evidence=None): self.record.update(outcome=status, evidence=evidence or [])
    def note(self, *_): pass

class Learn:
    def __init__(self): self.started = False; self.feedbacks = []; self.results = []; self.notes = []; self.outcomes = []
    def task(self, **_): self.started = True; return {}
    def result(self, name, artifact=None):
        self.started = True; self.results.append((name, artifact)); return "prt_feedback_v1_" + str(name)
    def step(self, *_args, **_kwargs): self.started = True; return Step()
    def note(self, *args, **kwargs): self.started = True; self.notes.append((args, kwargs))
    def outcome(self, *args, **kwargs): self.started = True; self.outcomes.append((args, kwargs))
    def feedback(self, attempt_id, disposition, evidence):
        self.feedbacks.append((attempt_id, disposition, evidence)); return {"delivery": "pending"}
    def choose(self, options, default): self.started = True; return {'selected_id': default, 'learning': {'advice': []}}

class Program:
    def __init__(self, inputs, outputs=None): self._inputs = inputs; self._child_helper = ProgramHelper("2", outputs or {})
    def inputs(self): return self._inputs
    def child_envelope(self, handle, **kwargs):
        rows = handle.head(1)
        if rows and isinstance(rows[0], dict):
            row = dict(rows[0])
            row.setdefault("phase", "report")
            row.setdefault("signals", {})
            row.setdefault("errors", [])
            row.setdefault("evidence", [])
            child = "prepare-task" if kwargs.get("where") == "collect" else ("replay-flow" if self._inputs.get("flow_id") else "author-flow")
            contract = json.loads((ROOT / (child + ".json")).read_text())["outputs"]
            handle = NS(head=lambda n: [row][:n], count=lambda: 1, meta=lambda: {"output_contract": contract})
        return self._child_helper.child_envelope(handle, **kwargs)
    def fail(self, status, klass, detail, where):
        import inspect
        envelope = inspect.currentframe().f_back.f_globals.get('envelope')
        if isinstance(envelope, dict):
            envelope['status'] = status
            envelope['errors'].append({'class': klass, 'detail': str(detail), 'where': where})
        return 'report'
    def classify(self, exc): return ('partial', 'binding_error') if 'scenario_not_running' in str(exc) else ('failed', 'binding_error')
    def run(self, states, state):
        while state:
            state = states[state]()

class Programs(unittest.TestCase):
    def run_program(self, name, inputs, api, library=None, runtime=None):
        outputs = json.loads((ROOT / (name + '.json')).read_text()).get('outputs', {})
        scope = {'inputs': inputs, 'device_control': api, 'lib': library, 'program_runtime': runtime, 'program': Program(inputs, outputs), 'learn': Learn()}
        with contextlib.redirect_stdout(io.StringIO()) as output:
            exec(compile((ROOT / (name + '.py')).read_text(), name, 'exec'), scope)
        if name in ('author-flow', 'replay-flow'):
            self.assertEqual(scope['envelope'], json.loads(output.getvalue()), 'governed output must be a JSON envelope')
        self.learning = scope['learn']
        return scope['envelope']

    def test_verified_replay_grades_earlier_recommendation(self):
        prepared = {'status': 'ok', 'signals': {'device_id': 'tv', 'outcome': 'unknown'}, 'evidence': []}
        replayed = {'status': 'ok', 'signals': {'outcome': 'verified_success', 'flow_id': 'flow', 'version': 1}, 'evidence': ['assertion:volume'], 'errors': []}
        library = NS(device_control=NS(prepare_task=lambda **_: Handle([prepared]), replay_flow=lambda **_: Handle([replayed])))
        result = self.run_program('do-task', {'device': 'tv', 'context_key': 'v1', 'actor': 'operator', 'flow_id': 'flow', 'version': 1, 'advice_attempt_id': 'earlier'}, None, library)
        self.assertEqual('ok', result['status'])
        self.assertEqual([('earlier', 'supported', ['assertion:volume'])], self.learning.feedbacks)

    def test_malformed_child_status_cannot_record_verified_success(self):
        prepared = {'status': 'ok', 'signals': {'device_id': 'tv', 'outcome': 'unknown'}, 'evidence': []}
        replayed = {'status': 'confused', 'signals': {'outcome': 'verified_success', 'flow_id': 'flow', 'version': 1},
                    'errors': [], 'evidence': ['assertion:volume']}
        library = NS(device_control=NS(prepare_task=lambda **_: Handle([prepared]), replay_flow=lambda **_: Handle([replayed])))
        test = Programs()
        result = test.run_program('do-task', {'device': 'tv', 'context_key': 'v1', 'actor': 'operator', 'flow_id': 'flow', 'version': 1}, None, library)
        self.assertEqual('failed', result['status'])
        self.assertEqual('invalid_response', result['errors'][-1]['class'])
        self.assertEqual('unknown', result['signals']['outcome'])
        self.assertEqual('flow', result['signals']['flow_id'])
        self.assertEqual([], test.learning.notes)
        self.assertEqual('unknown', test.learning.outcomes[-1][0][0])

    def test_verified_replay_hands_back_a_reference_for_later_correction(self):
        """A correction usually arrives after the run; the caller must still be able to name it."""
        api = NS(flow=NS(get=lambda **kw: Handle([{'id': 'saved', 'version': 1, 'deviceId': 'tv', 'contextKey': 'app:v1'}]),
                         replay=lambda **kw: Handle([{'runId': 'run-9', 'disposition': 'passed'}])))
        result = self.run_program('replay-flow', {'flow_id': 'saved', 'version': 1, 'device_id': 'tv',
                                                  'context_key': 'app:v1', 'actor': 'test'}, api)
        reference = result['signals']['learning']['feedback_ref']
        self.assertTrue(reference.startswith('prt_feedback_v1_'))
        # The reference names the exact revision, so a later correction cannot hit a sibling.
        self.assertEqual(('replay', {'kind': 'flow', 'owner': 'device-control', 'id': 'saved', 'revision': '1'}),
                         self.learning.results[-1])

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

    # screenshot-only check-app: one capture call that wakes the screen and returns evidence ids or the gap's next actions.
    def test_given_a_named_tv_when_screenshotting_then_one_capture_wakes_and_returns_evidence_ids(self):
        calls=[]
        def screenshot(**kw):
            calls.append(kw); return Handle([], {'disposition':'passed','evidenceIds':['ev-1']})
        result=self.run_program('check-app', {'device':'SmartTV 4K','actor':'o'}, NS(device=NS(screenshot=screenshot)))
        self.assertEqual('ok', result['status'])
        self.assertEqual(['ev-1'], result['evidence'])
        self.assertTrue(calls[0]['wake'])
        self.assertFalse(calls[0]['home'], 'Home can change the foreground app and stays opt-in')

    def test_given_a_tv_without_a_frame_when_screenshotting_then_the_gap_carries_next_actions(self):
        gap={'disposition':'capability_gap','message':'no transport','nextAction':'device-control device relation-confirm c1'}
        result=self.run_program('check-app', {'device':'tv','actor':'o'}, NS(device=NS(screenshot=lambda **kw: Handle([], gap))))
        self.assertEqual('failed', result['status'])
        self.assertEqual(['device-control device relation-confirm c1'], result['signals']['next_actions'])

    # check-app: an apk is installed, opened and screenshotted by one owner call; an installed app is opened then captured.
    def test_given_an_apk_when_checking_then_one_install_call_does_everything_and_no_open_is_sent(self):
        calls=[]
        def install(**kw):
            calls.append(kw)
            return Handle([], {'status':'installed_verified','package':'com.acme.demo','launchStatus':'foreground_verified','foregroundPackage':'com.acme.demo','evidenceIds':['ev-9']})
        api=NS(device=NS(install=install, open=lambda **kw: self.fail('open must not be called'), screenshot=lambda **kw: self.fail('screenshot must not be called')))
        result=self.run_program('check-app', {'device':'moto','actor':'o','apk':'/tmp/a.apk'}, api)
        self.assertEqual('ok', result['status'])
        self.assertEqual(['ev-9'], result['evidence'])
        self.assertTrue(calls[0]['open'] and calls[0]['screenshot'])

    def test_given_an_installed_app_when_checking_then_no_install_is_sent_and_it_is_captured_after_opening(self):
        order=[]
        def open_app(**kw): order.append('open'); return Handle([], {'status':'foreground_verified','package':'com.netflix.ninja','foregroundPackage':'com.netflix.ninja'})
        def shot(**kw): order.append('shot'); return Handle([], {'disposition':'passed','evidenceIds':['ev-1']})
        api=NS(device=NS(install=lambda **kw: self.fail('install must not be sent'), open=open_app, screenshot=shot))
        result=self.run_program('check-app', {'device':'tv','actor':'o','app':'netflix'}, api)
        self.assertEqual('ok', result['status'])
        self.assertEqual(['open','shot'], order)

    def test_given_an_app_that_never_reaches_the_foreground_when_checking_then_no_screenshot_is_taken(self):
        api=NS(device=NS(open=lambda **kw: Handle([], {'status':'launched_unverified'}), screenshot=lambda **kw: self.fail('no screenshot of a wrong screen')))
        result=self.run_program('check-app', {'device':'tv','actor':'o','app':'netflix'}, api)
        self.assertEqual('failed', result['status'])
        self.assertNotEqual('verified_success', result['signals']['outcome'])

    def test_given_no_device_when_checking_then_it_is_invalid_input(self):
        result=self.run_program('check-app', {'actor':'o'}, NS(device=NS()))
        self.assertEqual('invalid_input', result['errors'][0]['class'])

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

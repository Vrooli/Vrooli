"""Promotion and orchestration invariants with bounded owner responses."""
import contextlib
import io
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT=Path(__file__).parents[1]
# The real kernel sandbox and schema checker. Act sections are validated against these, not a
# stub: a fragment the kernel would reject, or output the caller's verifier rejects, fails here.
KERNEL_HOST=ROOT.parents[2]/'program-runtime'/'kernel'/'host'


def kernel():
    import sys
    if str(KERNEL_HOST) not in sys.path:
        sys.path.insert(0, str(KERNEL_HOST))
    from fragments import wrap
    from learn import _matches_schema
    return wrap, _matches_schema


class ActRefused(RuntimeError):
    """Stands in for the kernel's StepFailed: schema mismatch or a failed postcondition."""


def declared_baseline(program, step_name):
    """The reviewed baseline published into the contract by program-runtime.fragment-promote."""
    contract = ROOT / (program + '.json')
    if not contract.exists():
        return None
    import json
    baselines = (json.loads(contract.read_text()).get('learning') or {}).get('baselines') or {}
    return (baselines.get(step_name) or {}).get('fragment')


class Handle:
    def __init__(self,rows):self.rows=rows
    def head(self,n):return self.rows[:n]
    def count(self):return len(self.rows)

class Step:
    def __init__(self): self.record = {'outcome': 'unknown'}
    def __enter__(self): return self
    def __exit__(self, *_): return False
    def outcome(self, status, evidence=None): self.record.update(outcome=status, evidence=evidence or [])
    def note(self, *_): pass

class Learn:
    """Records every verb call; `preferences` and `avoids` drive choose the way the kernel does."""
    def __init__(self, preferences=(), avoids=(), fragments=None, source='fallback'):
        self.started = False; self.preferences = list(preferences); self.avoids = list(avoids)
        self.notes = []; self.outcomes = []; self.graded = []; self.chosen = []; self.keys = []
        # `fragments` substitutes a candidate for a named Act section, standing in for a cached or
        # model-written one; `source` is the provenance the section reports back.
        self.fragments = dict(fragments or {}); self.source = source; self.acted = []
        self.program = ''

    def act(self, name, intent, inputs, schema, bindings=None, attempts=3, fallback_fragment=None, *,
            verify=None, verifier_revision='', **kwargs):
        """Run the section the way the kernel does: real sandbox, real schema check, real verifier.

        Resolution mirrors the kernel's own order minus the durable cache: a substituted candidate,
        then the section's fallback, then the reviewed baseline declared in the contract. Once a
        section is promoted and its fallback removed, every test here exercises the published
        baseline artifact rather than a copy of it.
        """
        self.started = True
        wrap, matches_schema = kernel()
        fragment = self.fragments.get(name, fallback_fragment)
        if fragment is None:
            fragment = declared_baseline(self.program, name)
            if fragment is None:
                raise ActRefused('no candidate, fallback or declared baseline for ' + name)
            self.source = 'baseline'
        output = wrap(fragment)(inputs, bindings)
        if not matches_schema(output, schema):
            raise ActRefused('schema_mismatch')
        status, evidence = verify(output)
        self.acted.append((name, verifier_revision, status, list(evidence)))
        if status != 'verified_success':
            raise ActRefused('verification_failed: ' + ','.join(evidence))
        return {'output': output, 'fragment_source': self.fragments.get(name) and 'model' or self.source,
                'feedback_ref': 'feedback:' + name, 'verified': True, 'model_calls': 0}
    def task(self, **kwargs): self.started = True; self.keys.append(kwargs.get('key')); return {'attempt_id': 'attempt-current', 'task_id': 'task-current', 'key': kwargs.get('key')}
    def result_status(self, **kwargs): return {"available": True, "eligible": True}
    def result(self, name, artifact=None): return "feedback:" + name
    def recall(self, **kwargs): return {"notes": {"target-note": [{"fact": "Known site fact"}]}}
    def step(self, *_args, **_kwargs): self.started = True; return Step()
    def note(self, kind, body, evidence=None): self.started = True; self.notes.append((kind, body))
    def outcome(self, status, evidence=None, measurements=None): self.started = True; self.outcomes.append((status, list(evidence or []), measurements))
    def feedback(self, attempt_id, disposition, evidence, correction=''): self.started = True; self.graded.append((attempt_id, disposition, list(evidence))); return {'attempt_id': attempt_id, 'disposition': disposition, 'delivery': 'delivered', 'observation_id': 'observation-x'}
    def choose(self, options, default):
        self.started = True; self.chosen.append(list(options))
        candidates = [o for o in options if o not in self.avoids]
        hits = [o for o in candidates if o in self.preferences]
        selected = hits[0] if hits else default
        return {'selected_id': selected, 'source': 'advice' if hits else 'default', 'why': [], 'learning': {'advice': []}}

class Program:
    def __init__(self, inputs): self._inputs = inputs
    def inputs(self): return self._inputs
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
    def run_program(self,name,inputs,api=None,lib=None,learn=None,ai=None,search_hub_rows=None,workflow_health_rows=None):
        self.learn = learn or Learn()
        self.learn.program = name
        scope={'inputs':inputs,'browser_automation_studio':api,'lib':lib,'program':Program(inputs),'learn':self.learn,'ai':ai,
               'search_hub':NS(query=NS(query=lambda **kw:Handle(list(search_hub_rows or [])))),
               'workflow_health':NS(workflows=NS(search=lambda **kw:Handle(list(workflow_health_rows or []))))}
        with contextlib.redirect_stdout(io.StringIO()):exec(compile((ROOT/(name+'.py')).read_text(),name,'exec'),scope)
        return scope['envelope']

    def author(self,status):
        calls=[]
        def create(**kw):calls.append(kw);return Handle([{'workflow':{'id':'saved','version':1}}])
        api=NS(workflows=NS(validate=lambda **kw:Handle([{'result':{'valid':True}}]),execute_adhoc=lambda **kw:Handle([{'executionId':'evidence','status':status}]),create=create),executions=NS(timeline=lambda **kw:Handle([])))
        r=self.run_program('author-flow',{'flow':{'nodes':[{'id':'verify'}]},'project_id':'test-project'},api)
        return r,calls

    def test_failed_and_pending_candidates_do_not_persist(self):
        for status in ['EXECUTION_STATUS_FAILED','EXECUTION_STATUS_RUNNING','EXECUTION_STATUS_PENDING']:
            r,calls=self.author(status);self.assertNotEqual('ok',r['status']);self.assertEqual([],calls)

    def test_completed_candidate_persists_once(self):
        r,calls=self.author('EXECUTION_STATUS_COMPLETED');self.assertEqual('partial',r['status']);self.assertEqual([],calls);self.assertEqual('candidate',r['signals']['qualification'])
        self.assertEqual('unknown',r['signals']['outcome'])

    def test_assertion_evidence_controls_author_and_smoke_outcome(self):
        good={'action':{'type':'ACTION_TYPE_ASSERT'},'aggregates':{'status':'STEP_STATUS_COMPLETED'},
              'context':{'assertion':{'success':True}}}
        for entries,expected in [([], 'unknown'),([good],'verified_success'),
            ([dict(good,context={'assertion':{'success':False}})],'unknown'),
            ([dict(good,context={'assertion':{}})],'unknown'),
            ([dict(good,aggregates={'status':'STEP_STATUS_SKIPPED'})],'unknown'),
            ([good,{'action':{'type':'ACTION_TYPE_NAVIGATE'},'aggregates':{'status':'STEP_STATUS_RUNNING'}}],'unknown'),
            ([good,{'context':{'error':'failed elsewhere'}}],'unknown')]:
            api=NS(workflows=NS(
                validate=lambda **kw:Handle([{'result':{'valid':True}}]),
                get=lambda **kw:Handle([{'id':'w','version':1}]),
                execute=lambda **kw:Handle([{'executionId':'e','status':'EXECUTION_STATUS_COMPLETED'}]),
                execute_adhoc=lambda **kw:Handle([{'executionId':'e','status':'EXECUTION_STATUS_COMPLETED'}]),
                create=lambda **kw:Handle([{'workflow':{'id':'w','version':1}}])),
                executions=NS(get=lambda **kw:Handle([{'status':'EXECUTION_STATUS_COMPLETED'}]),
                    screenshots=lambda **kw:Handle([]),timeline=lambda **kw:Handle(entries)))
            for name,inputs in [('author-flow',{'flow':{'nodes':[{'id':'assert'}]},'project_id':'p'}),
                                ('smoke-flow',{'workflow_id':'w','version':1})]:
                with self.subTest(program=name,entries=entries):
                    result=self.run_program(name,inputs,api)
                    self.assertEqual(expected,result['signals']['outcome'])
            class Truncated(Handle):
                def count(self):return len(self.rows)+1
            api.executions.timeline=lambda **kw:Truncated([good])
            for name,inputs in [('author-flow',{'flow':{'nodes':[{'id':'assert'}]},'project_id':'p'}),
                                ('smoke-flow',{'workflow_id':'w','version':1})]:
                self.assertEqual('unknown',self.run_program(name,inputs,api)['signals']['outcome'])
            def unavailable(**kw):raise RuntimeError('scenario_not_running')
            api.executions.timeline=unavailable
            for name,inputs in [('author-flow',{'flow':{'nodes':[{'id':'assert'}]},'project_id':'p'}),
                                ('smoke-flow',{'workflow_id':'w','version':1})]:
                result=self.run_program(name,inputs,api)
                self.assertEqual('unknown',result['signals']['outcome'])
                self.assertEqual('partial',result['status'])

    def test_runtime_metadata_uses_explicit_domain_outcomes(self):
        import json
        for name in ['do-task','author-flow','smoke-flow','navigate-intent']:
            contract=json.loads((ROOT/(name+'.json')).read_text())
            self.assertNotIn('learning_task', contract)
            self.assertIn('learn.task', contract['verbs'])
            self.assertIn('learn.outcome', contract['verbs'])
        self.assertIn('learn.feedback', json.loads((ROOT/'do-task.json').read_text())['verbs'])
        for name in ['do-task','author-flow','smoke-flow','navigate-intent','find-flows']:
            declared=json.loads((ROOT/(name+'.json')).read_text())['version']
            source=(ROOT/(name+'.py')).read_text()
            self.assertIn('"version": "'+declared+'"',source,name+' source version must match its contract')

    def test_repair_requires_version_before_any_binding(self):
        r=self.run_program('author-flow',{'flow':{'nodes':[{}]},'workflow_id':'existing'})
        self.assertEqual('invalid_input',r['errors'][0]['class'])

    def test_search_does_not_speculatively_execute(self):
        lib=NS(browser_automation_studio=NS(find_flows=lambda **kw:Handle([{'status':'ok','signals':{'candidates':[{'id':'plausible','version':2,'runnable_by_id':True}]}}])))
        r=self.run_program('do-task',{'task':'open settings'},lib=lib)
        self.assertEqual('selection_required',r['errors'][0]['class'])

    def test_excluded_workflow_returns_selection_required(self):
        learn = Learn()
        learn.choose = lambda *_: {'selected_id': None, 'source': 'unavailable', 'learning': {'advice': []}}
        lib = NS(browser_automation_studio=NS(find_flows=lambda **kw: Handle([{'status': 'ok', 'signals': {'candidates': [{'id': 'excluded', 'version': 1, 'runnable_by_id': True}]}}])))
        result = self.run_program('do-task', {'task': 'open settings'}, lib=lib, learn=learn)
        self.assertEqual('partial', result['status'])
        self.assertEqual('selection_required', result['errors'][0]['class'])
        self.assertNotIn('recommended_workflow', result['signals'])

    def test_search_offers_versioned_options_and_recommends_remembered_revision(self):
        candidates=[{'id':'wf-a','version':2,'runnable_by_id':True},{'id':'wf-b','version':1,'runnable_by_id':True},{'id':'asset','runnable_by_id':False}]
        lib=NS(browser_automation_studio=NS(find_flows=lambda **kw:Handle([{'status':'ok','signals':{'candidates':candidates}}])))
        learn=Learn(preferences=['wf-b@1'])
        r=self.run_program('do-task',{'task':'open settings','site':'https://example.test/x'},lib=lib,learn=learn)
        self.assertEqual([['wf-a@2','wf-b@1']],learn.chosen)
        self.assertEqual({'workflow_id':'wf-b','version':1,'option_id':'wf-b@1','source':'advice','why':[]},r['signals']['recommended_workflow'])
        self.assertEqual('attempt-current',r['signals']['learning']['attempt_id'])
        self.assertEqual('example.test',learn.keys[0]['site'])
        self.assertEqual(16,len(learn.keys[0]['task']))

    def test_task_text_is_part_of_the_identity_key(self):
        lib=NS(browser_automation_studio=NS(find_flows=lambda **kw:Handle([{'status':'ok','signals':{'candidates':[]}}])))
        a=Learn();self.run_program('do-task',{'task':'open the settings page','scenario':'s'},lib=lib,learn=a)
        b=Learn();self.run_program('do-task',{'task':'delete a workflow','scenario':'s'},lib=lib,learn=b)
        c=Learn();self.run_program('do-task',{'task':'  Open THE settings   page ','scenario':'s'},lib=lib,learn=c)
        self.assertNotEqual(a.keys[0]['task'],b.keys[0]['task'])
        self.assertEqual(a.keys[0]['task'],c.keys[0]['task'])

    def test_execution_grades_the_recommending_run_and_notes_preference_or_avoid(self):
        verified={'status':'ok','signals':{'outcome':'verified_success'},'evidence':['execution:1']}
        learn=Learn()
        r=self.run_program('do-task',{'task':'check','workflow_id':'wf-a','version':2,'advice_attempt_id':'attempt-earlier'},
                           lib=NS(browser_automation_studio=NS(smoke_flow=lambda **kw:Handle([verified]))),learn=learn)
        self.assertEqual('verified_success',r['signals']['outcome'])
        self.assertEqual([('attempt-earlier','supported',['execution:1'])],learn.graded)
        self.assertIn(('preference',{'option_id':'wf-a@2'}),learn.notes)
        self.assertEqual({'reused_workflow':True},learn.outcomes[-1][2])
        self.assertEqual('delivered',r['signals']['learning']['graded']['delivery'])
        failed={'status':'failed','signals':{'outcome':'failed'},'errors':[{'class':'selector_not_found'}],'evidence':['execution:2']}
        learn=Learn()
        r=self.run_program('do-task',{'task':'check','workflow_id':'wf-a','version':2,'advice_attempt_id':'attempt-earlier'},
                           lib=NS(browser_automation_studio=NS(smoke_flow=lambda **kw:Handle([failed]))),learn=learn)
        self.assertEqual([('attempt-earlier','contradicted',['execution:2'])],learn.graded)
        self.assertIn(('avoid',{'option_id':'wf-a@2','fingerprint':'selector_not_found'}),learn.notes)
        learn=Learn()
        self.run_program('do-task',{'task':'check','workflow_id':'wf-a','version':2},
                         lib=NS(browser_automation_studio=NS(smoke_flow=lambda **kw:Handle([verified]))),learn=learn)
        self.assertEqual([],learn.graded)

    def test_resume_and_session_are_mutually_exclusive(self):
        r=self.run_program('do-task',{'task':'go','session':'s','navigation_id':'n','model':'m'})
        self.assertEqual('invalid_input',r['errors'][0]['class'])

    def test_reached_navigation_never_repeats_effects_to_harden(self):
        flow={'nodes':[{'id':'submit','action':{'type':'ACTION_TYPE_CLICK','click':{'selector':'#send'}}}]}
        child={'status':'ok','signals':{'outcome':'reached','navigation_id':'nav','candidate_flow':flow},'evidence':['navigation:nav']}
        def forbidden(**kw): self.fail('Navigation effect replayed')
        result=self.run_program('do-task',{'task':'send once','navigation_id':'nav','project_id':'p'},
            lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([child]),author_flow=forbidden,smoke_flow=forbidden)))
        self.assertEqual('unknown', result['signals']['outcome'])
        self.assertEqual('candidate', result['signals']['qualification'])
        self.assertEqual(flow, result['signals']['candidate_flow'])

    def test_verified_navigation_returns_output_without_replay(self):
        child={'status':'ok','signals':{'outcome':'verified_success','navigation_id':'nav','output':{'subjects':['hello']}},'evidence':['navigation:nav']}
        result=self.run_program('do-task',{'task':'read email','navigation_id':'nav'},lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([child]))))
        self.assertEqual('verified_success',result['signals']['outcome'])
        self.assertEqual({'subjects':['hello']},result['signals']['output'])
        self.assertEqual('feedback:task',result['signals']['learning']['feedback_ref'])

    def test_auto_select_only_authorized_revision_before_navigation(self):
        calls=[]
        child={'status':'ok','signals':{'outcome':'verified_success'},'evidence':['execution:e']}
        lib=NS(browser_automation_studio=NS(find_flows=lambda **kw:Handle([{'signals':{'candidates':[{'id':'other','version':1,'runnable_by_id':True},{'id':'safe','version':2,'runnable_by_id':True}]}}]),smoke_flow=lambda **kw:calls.append(kw) or Handle([child])))
        result=self.run_program('do-task',{'task':'read','auto_select':True,'authorized_workflows':['safe@2'],'postconditions':[{'selector':'#result','mode':'exists'}],'session':'s','model':'m'},lib=lib)
        self.assertEqual('safe',calls[0]['workflow_id'])
        self.assertEqual('verified_success',result['signals']['outcome'])

    def test_delayed_workflow_contradiction_blocks_selected_revision(self):
        learned=Learn()
        learned.result_status=lambda **kw: {'available':True,'eligible':False,'contradicted':True}
        result=self.run_program('do-task',{'task':'read','workflow_id':'bad','version':2},learn=learned)
        self.assertEqual('learning_ineligible',result['errors'][0]['class'])
        self.assertEqual('refused',result['status'])

    def test_saved_route_returns_requested_task_output(self):
        child={'status':'ok','signals':{'outcome':'verified_success','output':{'subjects':['hello']}},'evidence':['execution:e']}
        calls=[]
        result=self.run_program('do-task',{'task':'read','workflow_id':'w','version':1,'extraction':[{'name':'subjects','selector':'.subject'}]},lib=NS(browser_automation_studio=NS(smoke_flow=lambda **kw:calls.append(kw) or Handle([child]))))
        self.assertEqual([{'name':'subjects','selector':'.subject'}],calls[0]['extraction'])
        self.assertEqual({'subjects':['hello']},result['signals']['output'])

    def test_saved_flow_enforces_task_contract_and_returns_execution_data(self):
        calls=[]
        assertion={'type':'ACTION_TYPE_ASSERT','assert':{'selector':'#ready','mode':'ASSERTION_MODE_EXISTS','timeoutMs':5000}}
        extract={'type':'ACTION_TYPE_EXTRACT','extract':{'selector':'.subject','storeAs':'subjects'}}
        api=NS(workflows=NS(get=lambda **kw:Handle([{'id':'w','version':1,'flowDefinition':{'nodes':[{'action':assertion},{'action':extract}]}}]),execute=lambda **kw:calls.append(kw) or Handle([{'executionId':'e','status':'EXECUTION_STATUS_COMPLETED'}])),
          executions=NS(get=lambda **kw:Handle([{'status':'EXECUTION_STATUS_COMPLETED','result':{'extractedData':{'subjects':['hello']}}}]),screenshots=lambda **kw:Handle([]),timeline=lambda **kw:Handle([{'action':assertion,'aggregates':{'status':'STEP_STATUS_COMPLETED'},'context':{'assertion':{'success':True}}}])))
        inputs={'workflow_id':'w','version':1,'postconditions':[{'selector':'#ready','mode':'exists'}],'extraction':[{'name':'subjects','selector':'.subject'}]}
        result=self.run_program('smoke-flow',inputs,api=api)
        self.assertEqual('verified_success',result['signals']['outcome'])
        self.assertEqual({'subjects':['hello']},result['signals']['output'])
        self.assertEqual(1,len(calls))
        calls.clear();inputs['postconditions']=[{'selector':'#wrong','mode':'exists'}]
        result=self.run_program('smoke-flow',inputs,api=api)
        self.assertEqual('refused',result['status']);self.assertEqual([],calls)

    def test_profile_and_contract_separate_learning_identity(self):
        lib=NS(browser_automation_studio=NS(find_flows=lambda **kw:Handle([{'signals':{'candidates':[]}}])))
        a=Learn();b=Learn()
        self.run_program('do-task',{'task':'read','profile_id':'a'},lib=lib,learn=a)
        self.run_program('do-task',{'task':'read','profile_id':'b'},lib=lib,learn=b)
        self.assertNotEqual(a.keys[0],b.keys[0])

    def test_pending_navigation_returns_navigation_id_for_resume(self):
        child={'status':'partial','signals':{'outcome':'in_progress','navigation_id':'nav-2'},'evidence':['navigation:nav-2']}
        r=self.run_program('do-task',{'task':'go','session':'s','model':'m'},lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([child]))))
        self.assertEqual('nav-2',r['signals']['navigation_id']);self.assertEqual('unknown',r['signals']['outcome'])

    def test_failed_navigation_notes_avoid(self):
        child={'status':'failed','signals':{'outcome':'failed','status':'aborted','navigation_id':'nav-3'},'evidence':['navigation:nav-3']}
        learn=Learn()
        r=self.run_program('do-task',{'task':'go','session':'s','model':'m'},lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([child]))),learn=learn)
        self.assertEqual('failed',r['signals']['outcome']);self.assertIn(('avoid',{'fingerprint':'navigation:aborted'}),learn.notes)

    def test_navigate_intent_rebuilds_a_valid_v2_candidate_flow(self):
        steps=[{'index':0,'action_type':'navigate','url':'http://x/login','success':True},
               {'index':1,'action_type':'click','selector':'#go','success':True},
               {'index':2,'action_type':'type','selector':'#q','value':'hello','success':True},
               {'index':3,'action_type':'scroll','success':True},
               {'index':4,'action_type':'click','selector':'#bad','success':False,'error':'not found'}]
        status_calls=[]
        def status(**kw): status_calls.append(kw); return Handle([{'status':'completed','stepCount':5,'terminal':True,'steps':steps}])
        api=NS(vision_navigation=NS(start=lambda **kw:Handle([{'navigationId':'nav-9','status':'started'}]),status=status))
        r=self.run_program('navigate-intent',{'navigation_id':'nav-9','wait_millis':2500},api)
        self.assertEqual([{'navigation_id':'nav-9','wait_millis':2500}],status_calls)
        self.assertEqual('reached',r['signals']['outcome']);self.assertEqual('ok',r['status'])
        flow=r['signals']['candidate_flow']
        ids=[n['id'] for n in flow['nodes']]
        self.assertEqual(['step-1','step-2','step-3'],ids)
        self.assertEqual(['ACTION_TYPE_NAVIGATE','ACTION_TYPE_CLICK','ACTION_TYPE_INPUT'],[n['action']['type'] for n in flow['nodes']])
        self.assertEqual({'url':'http://x/login'},flow['nodes'][0]['action']['navigate'])
        self.assertEqual({'selector':'#q','value':'hello'},flow['nodes'][2]['action']['input'])
        self.assertFalse(any(n['action']['type']=='ACTION_TYPE_ASSERT' for n in flow['nodes']))
        self.assertEqual([('step-1','step-2'),('step-2','step-3')],[(e['source'],e['target']) for e in flow['edges']])
        self.assertTrue(all(e['type']=='WORKFLOW_EDGE_TYPE_SMOOTHSTEP' for e in flow['edges']))

    NAV_STEPS=[{'index':0,'action_type':'navigate','url':'http://x/login','success':True},
               {'index':1,'action_type':'click','selector':'#go','success':True},
               {'index':2,'action_type':'type','selector':'#q','value':'hello','success':True},
               {'index':3,'action_type':'scroll','success':True},
               {'index':4,'action_type':'click','selector':'#bad','success':False,'error':'not found'}]

    def navigate(self,steps,learn=None,inputs=None):
        api=NS(vision_navigation=NS(start=lambda **kw:Handle([{'navigationId':'nav-9','status':'started'}]),
                                    status=lambda **kw:Handle([{'status':'completed','stepCount':len(steps),'terminal':True,'steps':steps}])))
        return self.run_program('navigate-intent',dict({'navigation_id':'nav-9'},**(inputs or {})),api,learn=learn)

    def test_candidate_flow_accounts_for_every_recorded_step(self):
        """The reviewed fallback runs in the real sandbox and accounts for what it cannot replay."""
        learn=Learn()
        r=self.navigate(self.NAV_STEPS,learn=learn)
        self.assertEqual('fallback',r['signals']['candidate_flow_source'])
        self.assertEqual(['candidate-flow'],[a[0] for a in learn.acted])
        self.assertEqual('verified_success',learn.acted[0][2])
        # step 4 failed and is narrowed out before the section; step 3 (scroll) is unrepresentable
        # and must be declared, not silently lost.
        self.assertEqual([{'index':3,'kind':'scroll','reason':'unknown_action'}],r['signals']['dropped_steps'])
        self.assertEqual(['step-1','step-2','step-3'],[n['id'] for n in r['signals']['candidate_flow']['nodes']])

    def test_unknown_navigator_vocabulary_is_named_not_dropped_silently(self):
        """The drift case: a verb the mapping has never seen leaves a parameter note naming it."""
        learn=Learn()
        steps=[{'action_type':'navigate','url':'http://x/','success':True},
               {'action_type':'hover','selector':'#menu','success':True},
               {'action_type':'drag','selector':'#tile','success':True}]
        r=self.navigate(steps,learn=learn)
        self.assertEqual([{'index':1,'kind':'hover','reason':'unknown_action'},
                          {'index':2,'kind':'drag','reason':'unknown_action'}],r['signals']['dropped_steps'])
        self.assertIn(('parameter',{'name':'unrepresentable_action_kinds','value':'drag,hover'}),learn.notes)

    def test_candidate_flow_verifier_rejects_invented_and_unaccounted_output(self):
        """The verifier judges against the recorded steps, so a plausible-looking candidate that
        invents a selector or quietly drops a step cannot be accepted."""
        invents=("def step(inputs, bindings):\n"
                 "    return {'actions': [{'index': i, 'action': {'type': 'ACTION_TYPE_CLICK',\n"
                 "        'click': {'selector': '#injected'}}} for i in range(len(inputs['steps']))],\n"
                 "        'dropped': []}")
        loses=("def step(inputs, bindings):\n"
               "    return {'actions': [], 'dropped': []}")
        for fragment,expected in [(invents,'flow:selector-not-in-source'),(loses,'flow:accounting-incomplete')]:
            learn=Learn(fragments={'candidate-flow':fragment})
            r=self.navigate(self.NAV_STEPS,learn=learn)
            self.assertEqual('failed',learn.acted[0][2])
            self.assertEqual([expected],learn.acted[0][3])
            # The navigation itself still reports: a learning failure must not destroy the result.
            self.assertEqual('reached',r['signals']['outcome'])
            self.assertEqual('unavailable',r['signals']['candidate_flow_source'])
            self.assertNotIn('candidate_flow',r['signals'])
            self.assertEqual('mapping_unavailable',r['errors'][0]['class'])

    def test_candidate_flow_keeps_postconditions_out_of_generated_code(self):
        """Postconditions are caller authority: they are appended after the section, so a section
        that returns no assert node still yields the caller's assertion tail."""
        learn=Learn()
        r=self.navigate(self.NAV_STEPS,learn=learn,
                        inputs={'postconditions':[{'selector':'#inbox','mode':'exists'}]})
        nodes=r['signals']['candidate_flow']['nodes']
        self.assertEqual('ACTION_TYPE_ASSERT',nodes[-1]['action']['type'])
        self.assertEqual('#inbox',nodes[-1]['action']['assert']['selector'])
        self.assertEqual(['step-1','step-2','step-3','step-4'],[n['id'] for n in nodes])
        self.assertEqual([('step-1','step-2'),('step-2','step-3'),('step-3','step-4')],
                         [(e['source'],e['target']) for e in r['signals']['candidate_flow']['edges']])

    def test_navigate_intent_start_reads_status_once_without_wait(self):
        calls=[]
        api=NS(vision_navigation=NS(start=lambda **kw:calls.append(('start',kw)) or Handle([{'navigationId':'nav-1','status':'started'}]),
                                    status=lambda **kw:calls.append(('status',kw)) or Handle([{'status':'navigating','stepCount':0}])))
        r=self.run_program('navigate-intent',{'session':'s','prompt':'open the home page','model':'m'},api)
        self.assertEqual([c[0] for c in calls],['start','status']);self.assertNotIn('wait_millis',calls[1][1])
        self.assertEqual('in_progress',r['signals']['outcome']);self.assertEqual('navigation_pending',r['errors'][0]['class'])
        r=self.run_program('navigate-intent',{'session':'s','prompt':'x','model':'m','wait_millis':-1},api)
        self.assertEqual('invalid_input',r['errors'][0]['class'])

    def test_smoke_flow_failure_notes_avoid_with_fingerprint(self):
        failed={'action':{'type':'ACTION_TYPE_CLICK'},'aggregates':{'status':'STEP_STATUS_FAILED'},'context':{'errorCode':'ELEMENT_NOT_FOUND','error':'no #x'},'nodeId':'n1'}
        api=NS(workflows=NS(get=lambda **kw:Handle([{'id':'w','version':1}]),execute=lambda **kw:Handle([{'executionId':'e','status':'EXECUTION_STATUS_FAILED'}])),
               executions=NS(get=lambda **kw:Handle([{'status':'EXECUTION_STATUS_FAILED'}]),screenshots=lambda **kw:Handle([]),timeline=lambda **kw:Handle([failed])))
        learn=Learn()
        r=self.run_program('smoke-flow',{'workflow_id':'w','version':1},api,learn=learn)
        self.assertEqual('selector_not_found',r['errors'][0]['class'])
        self.assertIn(('avoid',{'option_id':'w@1','fingerprint':'selector_not_found:ACTION_TYPE_CLICK:ELEMENT_NOT_FOUND'}),learn.notes)
        self.assertEqual({'reused_workflow':True},learn.outcomes[-1][2])

    def test_find_flows_reports_versions_and_falls_back_to_judgement_on_lexical_miss(self):
        wf=[{'id':'w1','version':3,'name':'Studio home smoke','folderPath':'smoke'},{'id':'w2','version':1,'name':'Billing export','folderPath':'ops'}]
        classified=[]
        ai=NS(classify=lambda **kw:classified.append(kw) or Handle([{'label':'fits','text':kw['texts'][0]},{'label':'does_not_fit','text':kw['texts'][1]}]))
        api=NS(workflows=NS(list=lambda **kw:Handle(wf)))
        r=self.run_program('find-flows',{'task':'open the page','k':5},api,ai=ai)
        self.assertEqual(1,len(classified));self.assertTrue(r['signals']['tie_broken_by_ai'])
        self.assertEqual([{'id':'w1','version':3}],[{'id':c['id'],'version':c['version']} for c in r['signals']['candidates']])
        self.assertEqual('judged',r['signals']['candidates'][0]['fit'])
        classified.clear()
        r=self.run_program('find-flows',{'task':'billing export','k':5},api,ai=ai)
        self.assertEqual([],classified);self.assertEqual(1,r['signals']['candidates'][0]['version'])

    def test_find_flows_normalizes_all_three_owner_shapes(self):
        """One mapping over three spellings of identity, name and folder, in the real sandbox.

        The section has no fallback since promotion, so this exercises the reviewed baseline
        published in the contract.
        """
        learn=Learn()
        r=self.run_program('find-flows',{'task':'billing export','scenario':'ops','k':5},
                           NS(workflows=NS(list=lambda **kw:Handle([{'id':'w1','version':3,'name':'Billing export','folderPath':'ops'}]))),
                           learn=learn,
                           workflow_health_rows=[{'id':'h1','title':'Billing export check','path':'bas/ops','snippet':'export','mutating':False,'leafType':'flow'}],
                           search_hub_rows=[{'id':'s1','title':'Billing export fragment','path':'frag/billing','snippet':'export','providerId':'sh'}])
        self.assertEqual('baseline',r['signals']['normalization_source'])
        self.assertEqual('verified_success',learn.acted[0][2])
        self.assertEqual(['find:normalized-3','find:rows-3'],learn.acted[0][3])
        by_source={c['source']:c for c in r['signals']['candidates']}
        self.assertEqual({'workflows','workflow-health','search-hub'},set(by_source))
        self.assertEqual(3,by_source['workflows']['version'])
        self.assertTrue(by_source['workflows']['runnable_by_id'])
        self.assertFalse(by_source['search-hub']['runnable_by_id'])
        self.assertEqual('Billing export check',by_source['workflow-health']['name'])
        self.assertEqual('sh',by_source['search-hub']['provider'])

    def test_published_baseline_satisfies_its_own_declared_fixtures(self):
        """The promoted artifact is the floor when no cache is eligible, including on a machine
        that has neither Memory nor a model. Replay the fixtures the contract itself declares."""
        import json
        contract = json.loads((ROOT / 'find-flows.json').read_text())
        baseline = contract['learning']['baselines']['row-normalization']
        self.assertEqual(1, baseline['version'])
        self.assertTrue(baseline['reviewed_by'])
        self.assertEqual('row-fields/v1', baseline['compatibility']['verifier_revision'])
        wrap, _ = kernel()
        run = wrap(baseline['fragment'])
        self.assertTrue(baseline['fixtures'])
        for fixture in baseline['fixtures']:
            with self.subTest(inputs=fixture['inputs']):
                self.assertEqual([], fixture.get('calls', []), 'a pure section records no binding calls')
                self.assertEqual(fixture['expected'], run(fixture['inputs'], None))

    def test_find_flows_refuses_a_renamed_field_instead_of_ranking_nameless_candidates(self):
        """The drift this section exists for: the owner renames `name` to `label`, so the reviewed
        mapping reads an empty name from a row that plainly carries text. That fails verification
        instead of producing a nameless candidate that sorts last and vanishes below k."""
        learn=Learn()
        api=NS(workflows=NS(list=lambda **kw:Handle([{'id':'w1','version':3,'label':'Billing export','folderPath':'ops'}])))
        r=self.run_program('find-flows',{'task':'billing export','k':5},api,learn=learn)
        self.assertEqual('failed',learn.acted[0][2])
        self.assertEqual(['find:name-empty-while-row-has-text'],learn.acted[0][3])
        self.assertEqual('unavailable',r['signals']['normalization_source'])
        self.assertEqual([],r['signals']['candidates'])
        self.assertEqual('normalization_unavailable',r['errors'][0]['class'])
        self.assertEqual(('unavailable',[],None),learn.outcomes[-1])

    def test_find_flows_accepts_a_repaired_mapping_under_the_same_verifier(self):
        """What adaptation is for: a mapping that reads the moved field passes the identical
        verifier, with no edit to the verifier or the contract."""
        repaired=("def step(inputs, bindings):\n"
                  "    out = []\n"
                  "    for item in inputs['rows']:\n"
                  "        row = item['row']\n"
                  "        name = str(row.get('name') or row.get('label') or '')\n"
                  "        out.append({'index': item['index'], 'id': str(row.get('id') or ''),\n"
                  "            'name': name, 'folder': str(row.get('folderPath') or ''),\n"
                  "            'snippet': '', 'version': int(row.get('version') or 0)})\n"
                  "    return {'normalized': out}")
        learn=Learn(fragments={'row-normalization':repaired})
        api=NS(workflows=NS(list=lambda **kw:Handle([{'id':'w1','version':3,'label':'Billing export','folderPath':'ops'}])))
        r=self.run_program('find-flows',{'task':'billing export','k':5},api,learn=learn)
        self.assertEqual('verified_success',learn.acted[0][2])
        self.assertEqual('model',r['signals']['normalization_source'])
        self.assertEqual([{'id':'w1','version':3,'name':'Billing export'}],
                         [{'id':c['id'],'version':c['version'],'name':c['name']} for c in r['signals']['candidates']])

    def test_find_flows_verifier_rejects_invented_identity(self):
        """A candidate id that appears nowhere in the owner's row is refused, so a fabricated
        identity can never reach learn.choose as a runnable option."""
        invents=("def step(inputs, bindings):\n"
                 "    return {'normalized': [{'index': i['index'], 'id': 'made-up', 'name': 'Billing export',\n"
                 "        'folder': '', 'snippet': '', 'version': 3} for i in inputs['rows']]}")
        learn=Learn(fragments={'row-normalization':invents})
        api=NS(workflows=NS(list=lambda **kw:Handle([{'id':'w1','version':3,'name':'Billing export','folderPath':'ops'}])))
        r=self.run_program('find-flows',{'task':'billing export','k':5},api,learn=learn)
        self.assertEqual(['find:id-not-in-source'],learn.acted[0][3])
        self.assertEqual([],r['signals']['candidates'])

    def test_selected_replay_propagates_failure_and_evidence(self):
        calls=[]
        def smoke(**kw):calls.append(kw);return Handle([{'status':'failed','signals':{'outcome':'failed'},'errors':[{'class':'selector_not_found'}],'evidence':['execution:failed']}])
        r=self.run_program('do-task',{'task':'inspect','workflow_id':'exact','version':3},lib=NS(browser_automation_studio=NS(smoke_flow=smoke)))
        self.assertEqual('failed',r['status']);self.assertEqual(3,calls[0]['version']);self.assertEqual(['execution:failed'],r['evidence'])
        self.assertEqual('failed',r['signals']['outcome'])

    def test_only_explicit_verified_child_with_evidence_verifies_task(self):
        for outcome,status,evidence,expected in [
            ('passed','ok',['execution:1'],'unknown'),
            ('verified_success','ok',[],'unknown'),
            ('verified_success','partial',['execution:1'],'unknown'),
            ('verified_success','ok',['assertion:1'],'verified_success')]:
            child={'status':status,'signals':{'outcome':outcome},'evidence':evidence}
            lib=NS(browser_automation_studio=NS(smoke_flow=lambda **kw:Handle([child])))
            result=self.run_program('do-task',{'task':'check','workflow_id':'exact','version':1},lib=lib)
            self.assertEqual(expected,result['signals']['outcome'])

    def test_completed_navigation_without_steps_remains_unknown(self):
        child={'status':'ok','signals':{'outcome':'reached'},'evidence':['navigation:1']}
        lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([child])))
        result=self.run_program('do-task',{'task':'navigate','session':'s','model':'selected'},lib=lib)
        self.assertEqual('unknown',result['signals']['outcome'])
        self.assertEqual('no_candidates',result['errors'][0]['class'])

    def test_nonterminal_navigation_remains_unknown(self):
        for child_outcome in ['in_progress', 'human_pause']:
            child={'status':'partial','signals':{'outcome':child_outcome},'evidence':['navigation:pending']}
            lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([child])))
            with self.subTest(child_outcome=child_outcome):
                result=self.run_program('do-task',{'task':'navigate','session':'s','model':'selected'},lib=lib)
                self.assertEqual('unknown',result['signals']['outcome'])

    def test_learning_projects_shared_rows_and_selectors(self):
        calls=[]
        rows=[{'row':name,'reading':None,'target':None,'in_band':None,
               'unavailable':True,'reason':'scenario_unreachable'} for name in
              ['failure-recurrence','completion-effort','advice-outcomes','first-action-latency',
               'agent-round-trips','visual-reasoning','workflow-reuse']]
        def compare(**kw):
            calls.append(kw)
            return Handle([{'status':'unavailable','signals':{'rows':rows},
                            'errors':[{'class':'scenario_unreachable'}]}])
        result=self.run_program('learning-read',{'operation':'replay'},lib=NS(vrooli_memory=NS(compare_outcomes=compare)))
        self.assertEqual([{'scope':'bas-usage','cohort_limit':1,'operation':'replay'}],calls)
        self.assertEqual(rows,result['signals']['rows'])
        self.assertEqual('unavailable',result['status'])

    def test_learning_preserves_reliability_and_full_cohort_identity(self):
        cohort={'operation':'operation-'*12,'context':'context-'*20,'attempts':2}
        rows=[{'row':'failure-recurrence','reading':{'cohorts':[cohort]},
               'unavailable':True,'reason':'unreliable:cohort_sample'}]
        child={'status':'partial','inputs':{'from':'resolved-start','to':'resolved-end'},
               'signals':{'rows':rows,'reliability':{'truncated':True}},'errors':[]}
        result=self.run_program('learning-read',{},lib=NS(vrooli_memory=NS(
            compare_outcomes=lambda **kw:Handle([child]))))
        self.assertEqual('partial',result['status'])
        self.assertEqual(child['inputs'],result['inputs'])
        self.assertEqual(child['signals']['reliability'],result['signals']['reliability'])
        actual=result['signals']['rows'][0]['reading']['cohorts'][0]
        self.assertEqual(cohort['operation'],actual['operation'])
        self.assertEqual(cohort['context'],actual['context'])

    def test_incomplete_execution_outcomes_remain_unknown(self):
        for status in [None,'EXECUTION_STATUS_UNSPECIFIED','EXECUTION_STATUS_RUNNING',
                       'EXECUTION_STATUS_PENDING','EXECUTION_STATUS_CANCELLED']:
            api=NS(workflows=NS(get=lambda **kw:Handle([{'id':'w'}]),
                execute=lambda **kw:Handle([{'executionId':'e','status':status}])),
                executions=NS(get=lambda **kw:Handle([{'status':status}]),
                    screenshots=lambda **kw:Handle([])))
            with self.subTest(status=status):
                result=self.run_program('smoke-flow',{'workflow_id':'w','version':1},api)
                self.assertEqual('unknown',result['signals']['outcome'])
                authored,calls=self.author(status)
                self.assertEqual('unknown',authored['signals']['outcome'])
                self.assertEqual([],calls)

    # --- qualification: a reached navigation becomes a durable verified revision ---

    def _reached(self, flow):
        return {'status':'ok','signals':{'outcome':'reached','navigation_id':'nav','candidate_flow':flow},'evidence':['navigation:nav']}

    def _qualify_lib(self, nav_child, authored, verified, calls):
        def author(**kw): calls.append(('author',kw)); return Handle([authored])
        def smoke(**kw): calls.append(('smoke',kw)); return Handle([verified])
        return NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([nav_child]),author_flow=author,smoke_flow=smoke))

    def test_qualify_persists_and_independently_verifies_a_reached_candidate(self):
        """A navigation trace cannot verify the flow rebuilt from it; the persisted revision can."""
        flow={'nodes':[{'id':'open','action':{'type':'ACTION_TYPE_NAVIGATE','navigate':{'url':'http://x'}}}]}
        authored={'status':'ok','signals':{'outcome':'verified_success','workflow_id':'saved','version':3},'evidence':['adhoc:e1']}
        verified={'status':'ok','signals':{'outcome':'verified_success'},'evidence':['execution:e2']}
        calls=[];learn=Learn()
        conditions=[{'selector':'#inbox','mode':'exists'}]
        r=self.run_program('do-task',{'task':'read inbox','navigation_id':'nav','project_id':'p','qualify':True,'postconditions':conditions},
            lib=self._qualify_lib(self._reached(flow),authored,verified,calls),learn=learn)
        self.assertEqual(['author','smoke'],[c[0] for c in calls])
        self.assertEqual(flow,calls[0][1]['flow'])
        # The persisted revision is graded against the task's own postconditions, not the flow's.
        self.assertEqual(('saved',3,conditions),(calls[1][1]['workflow_id'],calls[1][1]['version'],calls[1][1]['postconditions']))
        self.assertEqual('verified_success',r['signals']['outcome'])
        self.assertEqual('qualified',r['signals']['qualification'])
        self.assertTrue(r['signals']['replay_eligible'])
        self.assertEqual({'workflow_id':'saved','version':3},r['signals']['qualified_workflow'])
        self.assertIn(('preference',{'option_id':'saved@3'}),learn.notes)

    def test_qualification_failure_records_avoid_and_never_claims_success(self):
        flow={'nodes':[{'id':'open','action':{'type':'ACTION_TYPE_NAVIGATE','navigate':{'url':'http://x'}}}]}
        authored={'status':'ok','signals':{'outcome':'verified_success','workflow_id':'saved','version':3},'evidence':['adhoc:e1']}
        failed={'status':'failed','signals':{'outcome':'failed'},'errors':[{'class':'assertion_failed'}],'evidence':['execution:e2']}
        calls=[];learn=Learn()
        r=self.run_program('do-task',{'task':'read inbox','navigation_id':'nav','project_id':'p','qualify':True,'postconditions':[{'selector':'#inbox','mode':'exists'}]},
            lib=self._qualify_lib(self._reached(flow),authored,failed,calls),learn=learn)
        self.assertEqual('unknown',r['signals']['outcome'])
        self.assertEqual('partial',r['status'])
        self.assertFalse(r['signals']['replay_eligible'])
        self.assertIn(('avoid',{'option_id':'saved@3','fingerprint':'assertion_failed'}),learn.notes)
        self.assertNotIn(('preference',{'option_id':'saved@3'}),learn.notes)

    def test_qualify_requires_postconditions_before_repeating_any_effect(self):
        calls=[]
        r=self.run_program('do-task',{'task':'send once','navigation_id':'nav','qualify':True},
            lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:calls.append(kw) or Handle([]))))
        self.assertEqual('failed',r['status'])
        self.assertEqual('invalid_input',r['errors'][0]['class'])
        self.assertEqual([],calls)

    def test_qualification_never_downgrades_an_already_verified_task(self):
        """The task succeeded on the live page; failing to earn a durable flow is a separate outcome."""
        flow={'nodes':[{'id':'open','action':{'type':'ACTION_TYPE_NAVIGATE','navigate':{'url':'http://x'}}}]}
        nav={'status':'ok','signals':{'outcome':'verified_success','navigation_id':'nav','candidate_flow':flow,'output':{'subjects':['a']}},'evidence':['navigation:nav']}
        authored={'status':'failed','signals':{'outcome':'failed'},'evidence':[]}
        calls=[]
        r=self.run_program('do-task',{'task':'read inbox','navigation_id':'nav','project_id':'p','qualify':True,'postconditions':[{'selector':'#inbox','mode':'exists'}]},
            lib=self._qualify_lib(nav,authored,{'status':'ok','signals':{'outcome':'verified_success'},'evidence':['x']},calls))
        self.assertEqual('verified_success',r['signals']['outcome'])
        self.assertEqual({'subjects':['a']},r['signals']['output'])
        self.assertEqual(['author'],[c[0] for c in calls])
        self.assertIn('qualification_failed',[e['class'] for e in r['errors']])

    def test_unqualified_candidate_explains_how_to_qualify(self):
        flow={'nodes':[{'id':'submit','action':{'type':'ACTION_TYPE_CLICK','click':{'selector':'#send'}}}]}
        r=self.run_program('do-task',{'task':'send once','navigation_id':'nav'},
            lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([self._reached(flow)]))))
        self.assertEqual('candidate',r['signals']['qualification'])
        self.assertIn('qualify=true',[e['detail'] for e in r['errors']][0])


if __name__=='__main__':unittest.main()

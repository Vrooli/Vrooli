"""Promotion and orchestration invariants with bounded owner responses."""
import contextlib
import io
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT=Path(__file__).parents[1]
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
    def __init__(self, preferences=(), avoids=()):
        self.started = False; self.preferences = list(preferences); self.avoids = list(avoids)
        self.notes = []; self.outcomes = []; self.graded = []; self.chosen = []; self.keys = []
    def task(self, **kwargs): self.started = True; self.keys.append(kwargs.get('key')); return {'attempt_id': 'attempt-current', 'task_id': 'task-current', 'key': kwargs.get('key')}
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
    def run_program(self,name,inputs,api=None,lib=None,learn=None,ai=None):
        self.learn = learn or Learn()
        scope={'inputs':inputs,'browser_automation_studio':api,'lib':lib,'program':Program(inputs),'learn':self.learn,'ai':ai,
               'search_hub':NS(query=NS(query=lambda **kw:Handle([]))),'workflow_health':NS(workflows=NS(search=lambda **kw:Handle([])))}
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
        r,calls=self.author('EXECUTION_STATUS_COMPLETED');self.assertEqual('ok',r['status']);self.assertEqual(1,len(calls));self.assertEqual('saved',r['signals']['workflow_id'])
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

    def test_reached_navigation_is_authored_verified_and_remembered(self):
        calls={}
        flow={'nodes':[{'id':'step-1','action':{'type':'ACTION_TYPE_NAVIGATE','navigate':{'url':'http://x'}}}],'edges':[]}
        def navigate(**kw): calls['navigate']=kw; return Handle([{'status':'ok','signals':{'outcome':'reached','navigation_id':'nav-1','step_count':3,'candidate_flow':flow},'evidence':['navigation:nav-1']}])
        def author(**kw): calls['author']=kw; return Handle([{'status':'ok','signals':{'outcome':'verified_success','workflow_id':'wf-new','version':1},'evidence':['execution:a','workflow:wf-new:1']}])
        def smoke(**kw): calls['smoke']=kw; return Handle([{'status':'ok','signals':{'outcome':'verified_success'},'evidence':['execution:b']}])
        learn=Learn()
        r=self.run_program('do-task',{'task':'open settings','navigation_id':'nav-1','wait_millis':5000,'project_id':'p'},
                           lib=NS(browser_automation_studio=NS(navigate_intent=navigate,author_flow=author,smoke_flow=smoke)),learn=learn)
        self.assertEqual('nav-1',calls['navigate']['navigation_id']);self.assertEqual(5000,calls['navigate']['wait_millis'])
        self.assertEqual(flow,calls['author']['flow']);self.assertEqual('wf-new',calls['smoke']['workflow_id'])
        self.assertEqual('verified_success',r['signals']['outcome'])
        self.assertIn(('preference',{'option_id':'wf-new@1'}),learn.notes)
        self.assertTrue(any(k=='target-note' for k,_ in learn.notes))
        self.assertEqual({'reused_workflow':False},learn.outcomes[-1][2])

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
        self.assertEqual(['step-1','step-2','step-3','step-4'],ids)
        self.assertEqual(['ACTION_TYPE_NAVIGATE','ACTION_TYPE_CLICK','ACTION_TYPE_INPUT','ACTION_TYPE_ASSERT'],[n['action']['type'] for n in flow['nodes']])
        self.assertEqual({'url':'http://x/login'},flow['nodes'][0]['action']['navigate'])
        self.assertEqual({'selector':'#q','value':'hello'},flow['nodes'][2]['action']['input'])
        self.assertEqual('ASSERTION_MODE_EXISTS',flow['nodes'][3]['action']['assert']['mode'])
        self.assertEqual([('step-1','step-2'),('step-2','step-3'),('step-3','step-4')],[(e['source'],e['target']) for e in flow['edges']])
        self.assertTrue(all(e['type']=='WORKFLOW_EDGE_TYPE_SMOOTHSTEP' for e in flow['edges']))

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

if __name__=='__main__':unittest.main()

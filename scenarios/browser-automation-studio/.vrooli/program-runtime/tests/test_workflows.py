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

class Programs(unittest.TestCase):
    def run_program(self,name,inputs,api=None,lib=None):
        scope={'inputs':inputs,'browser_automation_studio':api,'lib':lib}
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
        for name in ['do-task','author-flow','smoke-flow']:
            contract=json.loads((ROOT/(name+'.json')).read_text())
            metadata=contract['learning_task']
            self.assertEqual('bas-usage',metadata['scope'])
            self.assertEqual('signals.outcome',metadata['outcome']['status_path'])
            self.assertEqual(['evidence'],metadata['outcome']['evidence_paths'])
            self.assertEqual({x:x for x in ['verified_success','failed','unavailable','unknown']},metadata['outcome']['mapping'])

    def test_repair_requires_version_before_any_binding(self):
        r=self.run_program('author-flow',{'flow':{'nodes':[{}]},'workflow_id':'existing'})
        self.assertEqual('invalid_input',r['errors'][0]['class'])

    def test_search_does_not_speculatively_execute(self):
        lib=NS(browser_automation_studio=NS(find_flows=lambda **kw:Handle([{'status':'ok','signals':{'candidates':[{'id':'plausible'}]}}])))
        r=self.run_program('do-task',{'task':'open settings'},lib=lib)
        self.assertEqual('selection_required',r['errors'][0]['class'])

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

    def test_completed_navigation_remains_unknown(self):
        child={'status':'ok','signals':{'outcome':'reached'},'evidence':['navigation:1']}
        lib=NS(browser_automation_studio=NS(navigate_intent=lambda **kw:Handle([child])))
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

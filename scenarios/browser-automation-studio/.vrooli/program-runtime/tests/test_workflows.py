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

class Programs(unittest.TestCase):
    def run_program(self,name,inputs,api=None,lib=None):
        scope={'inputs':inputs,'browser_automation_studio':api,'lib':lib}
        with contextlib.redirect_stdout(io.StringIO()):exec(compile((ROOT/(name+'.py')).read_text(),name,'exec'),scope)
        return scope['envelope']

    def author(self,status):
        calls=[]
        def create(**kw):calls.append(kw);return Handle([{'workflow':{'id':'saved','version':1}}])
        api=NS(workflows=NS(validate=lambda **kw:Handle([{'result':{'valid':True}}]),execute_adhoc=lambda **kw:Handle([{'executionId':'evidence','status':status}]),create=create))
        r=self.run_program('author-flow',{'flow':{'nodes':[{'id':'verify'}]},'project_id':'test-project'},api)
        return r,calls

    def test_failed_and_pending_candidates_do_not_persist(self):
        for status in ['EXECUTION_STATUS_FAILED','EXECUTION_STATUS_RUNNING','EXECUTION_STATUS_PENDING']:
            r,calls=self.author(status);self.assertNotEqual('ok',r['status']);self.assertEqual([],calls)

    def test_completed_candidate_persists_once(self):
        r,calls=self.author('EXECUTION_STATUS_COMPLETED');self.assertEqual('ok',r['status']);self.assertEqual(1,len(calls));self.assertEqual('saved',r['signals']['workflow_id'])

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
        self.assertTrue(r['signals']['capture_required'])

if __name__=='__main__':unittest.main()

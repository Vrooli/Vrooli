"""Recorded response probes for owner handoff, partial evidence, and measured learning."""
import contextlib
import io
import json
import unittest
from pathlib import Path
from types import SimpleNamespace as NS

ROOT=Path(__file__).parents[1]
class Handle:
    def __init__(self,rows=None,meta=None):self.rows=rows or [];self.metadata=meta or {}
    def head(self,n):return self.rows[:n]
    def count(self):return len(self.rows)
    def meta(self):return self.metadata

def run(name,inputs,**scope):
    scope.update(inputs=inputs)
    with contextlib.redirect_stdout(io.StringIO()) as stream:
        exec(compile((ROOT/(name+'.py')).read_text(),name,'exec'),scope)
    assert len(stream.getvalue().splitlines())==1
    parsed=json.loads(stream.getvalue())
    assert parsed==scope['envelope']
    return parsed

class Research(unittest.TestCase):
    def test_complete_answer_preserves_owner_citations(self):
        brief={'summary':'Supported answer','citations':[{'url':'https://primary.example/a','title':'A'}]}
        h=Handle(meta={'status':'ok','answerKind':'stored_finding','brief':brief,'findingIds':['f'],'liveCalls':0})
        calls=[]
        def answer(**kw):calls.append(kw);return h
        r=run('research',{'query':'q','max_age_seconds':60},web_search=NS(research=NS(answer=answer)))
        self.assertEqual(brief,r['signals']['brief']);self.assertEqual(0,r['signals']['live_calls']);self.assertEqual(1,len(calls))
    def test_raw_hits_retain_urls(self):
        h=Handle([{'url':'https://primary.example/a','title':'A'}],{'status':'ok','answerKind':'raw_hits'})
        r=run('research',{'query':'q','effort':'l0'},web_search=NS(research=NS(answer=lambda **kw:h)))
        self.assertEqual('https://primary.example/a',r['signals']['results'][0]['url'])
    def test_incomplete_comparison_preserves_healthy_answer(self):
        def answer(**kw):
            if kw['query']=='broken':raise RuntimeError('upstream unavailable')
            return Handle(meta={'status':'ok','answerKind':'cited_synthesis','brief':{'summary':'verified','citations':[{'url':'https://primary.example/a'}]}})
        r=run('compare-sources',{'questions':['healthy','broken']},web_search=NS(research=NS(answer=answer)))
        self.assertEqual('partial',r['status']);self.assertEqual(['broken'],r['signals']['unresolved']);self.assertEqual('verified',r['signals']['answers'][0]['brief']['summary'])
    def test_changed_facts_and_conflicts_remain_visible(self):
        calls=[]
        def answer(**kw):
            calls.append(kw)
            if kw['query']=='changed':
                return Handle(meta={'status':'ok','answerKind':'cited_synthesis','brief':{'summary':'new value','citations':[{'url':'https://primary.example/new'}]},'checkedAt':'2026-09-05T00:00:00Z'})
            return Handle(meta={'status':'partial','answerKind':'none','abstained':True,'gaps':['contradictory_sources'],'brief':{}})
        r=run('compare-sources',{'questions':['changed','conflict'],'source_domains':['primary.example']},web_search=NS(research=NS(answer=answer)))
        self.assertEqual('partial',r['status']);self.assertEqual(['conflict'],r['signals']['unresolved'])
        self.assertEqual('new value',r['signals']['answers'][0]['brief']['summary'])
        self.assertEqual(['contradictory_sources'],r['signals']['answers'][1]['gaps'])
        self.assertTrue(all(c['max_age_seconds']==0 and c['source_domains']==['primary.example'] for c in calls))
    def test_empty_learning_window_is_not_success(self):
        r=run('learning-read',{},vrooli_memory=NS(learning=NS(measure=lambda **kw:Handle(meta={'reliable':False,'reason':'empty_window'}))))
        self.assertEqual('partial',r['status']);self.assertFalse(r['signals']['reliable']);self.assertIsNone(r['signals']['targets'])
    def test_verified_success_requires_quality_evidence(self):
        r=run('record-attempt',{'attempt':{'outcome':'verified_success','evidence_refs':['run-1']}},vrooli_memory=NS())
        self.assertEqual('failed',r['status']);self.assertEqual('invalid_input',r['errors'][0]['class'])
    def test_outcome_and_quality_are_one_idempotent_write(self):
        calls=[]
        def record(**kw):calls.append(kw);return Handle(meta={'entryId':'receipt','existing':len(calls)>1})
        inputs={'attempt':{'attempt_id':'stable','outcome':'verified_success','evidence_refs':['source-check'],'approach':'compare'},'quality':{k:True for k in ['citation_support','freshness','coverage','contradictions_resolved']},'used_finding_ids':['f1']}
        for _ in range(2):
            r=run('record-attempt',inputs,vrooli_memory=NS(learning=NS(record=record)))
            self.assertEqual('ok',r['status']);self.assertEqual(['f1'],r['signals']['contributing_finding_ids'])
        self.assertEqual(calls[0],calls[1]);self.assertIn('citation_support',calls[0]['attempt']['approach']);self.assertIn('f1',calls[0]['attempt']['approach']);self.assertTrue(r['signals']['existing'])
    def test_wait_timeout_keeps_exact_execution_without_restart(self):
        calls=[]
        def wait(**kw):calls.append(kw);return Handle(meta={'runId':'e','status':'running','timedOut':True})
        r=run('research-l3',{'run_id':'e'},web_search=NS(research=NS(wait=wait)))
        self.assertEqual('partial',r['status']);self.assertEqual('e',r['signals']['run_id']);self.assertEqual(1,len(calls));self.assertTrue(r['signals']['next_action']['do_not_poll'])
    def test_abstention_is_not_answered(self):
        h=Handle(meta={'status':'complete','result':{'status':'abstained','summary':'thin sources'}})
        r=run('research-l3',{'run_id':'e'},web_search=NS(research=NS(wait=lambda **kw:h)))
        self.assertEqual('partial',r['status'])
    def test_setpoint_row_retains_unknown_reason(self):
        import ast
        tree=ast.parse((ROOT/'setpoint-read.py').read_text())
        row=next(n for n in tree.body if isinstance(n,ast.FunctionDef) and n.name=='row')
        scope={'envelope':{'signals':{'rows':[],'unavailable':0,'readable':0},'evidence':[]}}
        exec(compile(ast.Module(body=[row],type_ignores=[]),'row','exec'),scope)
        scope['row']('cache',None,None,None,unavailable=True,reason='pending_telemetry')
        self.assertEqual({'row':'cache','reading':None,'target':None,'in_band':None,'unavailable':True,'reason':'pending_telemetry'},scope['envelope']['signals']['rows'][0])
if __name__=='__main__':unittest.main()

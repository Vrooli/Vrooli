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
    if name=='record-attempt' and 'lib' not in scope:
        def finish(**request):
            shared={'inputs':request,'vrooli_memory':scope.get('vrooli_memory')}
            path=ROOT.parents[2]/'vrooli-memory/.vrooli/program-runtime/finish-attempt.py'
            with contextlib.redirect_stdout(io.StringIO()):
                exec(compile(path.read_text(),str(path),'exec'),shared)
            return Handle([shared['envelope']])
        scope['lib']=NS(vrooli_memory=NS(finish_attempt=finish))
    scope.update(inputs=inputs)
    with contextlib.redirect_stdout(io.StringIO()) as stream:
        exec(compile((ROOT/(name+'.py')).read_text(),name,'exec'),scope)
    assert len(stream.getvalue().splitlines())==1
    parsed=json.loads(stream.getvalue())
    assert parsed==scope['envelope']
    return parsed

class Research(unittest.TestCase):
    def attempt(self,identity):
        return {'attempt_id':identity,'task_id':'research-task','operation':'research',
                'context_key':'research/v1','started_at':'2026-09-01T00:00:00Z',
                'finished_at':'2026-09-01T00:01:00Z','task_started_at':'2026-09-01T00:00:00Z',
                'attempt_number':1,'recall_status':'no_match','provenance':'test',
                'trigger':'compare sources','approach':'compare','outcome':'verified_success',
                'evidence_refs':['source-check']}
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
    def test_comparison_expands_explicit_cells(self):
        calls=[]
        def answer(**kw):
            calls.append(kw)
            return Handle(meta={'status':'ok','answerKind':'cited_synthesis','brief':{'summary':'verified','citations':[{'url':'https://primary.example/a'}]}})
        r=run('compare-sources',{'subjects':['A','B'],'questions':['price','support'],'dimensions':['current'],'temporal_scope':'2026'},web_search=NS(research=NS(answer=answer)))
        self.assertEqual('ok',r['status']);self.assertEqual(4,len(r['signals']['cells']));self.assertEqual(4,len(calls))
        self.assertTrue(all('A' in c['query'] or 'B' in c['query'] for c in calls))

    def test_change_investigation_preserves_unresolved_change_states(self):
        r=run('change-investigation',{'prior_investigation_id':'prior-1','prior_cells':[{'cell_id':'same','claim':'x'},{'cell_id':'dates','claim':'x','effective_date':'2025'}],'current_cells':[{'cell_id':'same','claim':'x'},{'cell_id':'dates','claim':'x','effective_date':'2026'},{'cell_id':'new','claim':'y'}]})
        states={x['cell_id']:x for x in r['signals']['results']}
        self.assertEqual('partial',r['status']);self.assertEqual('unchanged',states['same']['status']);self.assertEqual('incomparable',states['dates']['status']);self.assertEqual('unverified',states['new']['status'])

    def test_comparison_cell_budget_bounds_maximum_input(self):
        calls=[]
        def answer(**kw):
            calls.append(kw)
            return Handle(meta={'status':'ok','answerKind':'cited_synthesis','brief':{'summary':'ok','citations':[]}})
        r=run('compare-sources',{'subjects':['A','B','C','D'],'questions':['q1','q2','q3','q4'],'dimensions':['d1','d2'],'temporal_scope':'current'},web_search=NS(research=NS(answer=answer)))
        self.assertEqual('ok',r['status']);self.assertEqual(32,len(r['signals']['cells']));self.assertEqual(32,len(calls));self.assertLess(len(json.dumps(r).encode()),65536)
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
    def test_quality_refusal_never_calls_shared_finish(self):
        def forbidden(**kw):self.fail('Invalid quality reached capture')
        for value in [False,None,'true']:
            quality={k:True for k in ['citation_support','freshness','coverage','contradictions_resolved']}
            quality['coverage']=value
            result=run('record-attempt',{'attempt':self.attempt('quality'),'quality':quality},
                       lib=NS(vrooli_memory=NS(finish_attempt=forbidden)))
            self.assertEqual('invalid_input',result['errors'][0]['class'])

    def test_generated_attempt_identity_links_quality_and_retries_stably(self):
        calls=[]
        observations=[]
        def record(**kw):calls.append(kw);return Handle(meta={'entryId':'receipt'})
        def observe(**kw):observations.append(kw);return Handle(meta={'entryId':'quality'})
        attempt=self.attempt('generated');del attempt['attempt_id']
        inputs={'attempt':attempt,'quality':{k:True for k in ['citation_support','freshness','coverage','contradictions_resolved']}}
        for _ in range(2):
            result=run('record-attempt',inputs,vrooli_memory=NS(learning=NS(record=record,observe=observe)))
            self.assertEqual('ok',result['status'])
        self.assertEqual(calls[0],calls[1]);self.assertEqual(observations[0],observations[1])
        self.assertEqual(calls[0]['attempt']['attempt_id'],observations[0]['observation']['attempt_id'])
        self.assertNotIn('attempt_id',attempt)
    def test_outcome_and_quality_are_separate_idempotent_writes(self):
        calls=[]
        observations=[]
        def record(**kw):calls.append(kw);return Handle(meta={'entryId':'receipt','existing':len(calls)>1})
        def observe(**kw):observations.append(kw);return Handle(meta={'entryId':'observation-receipt','existing':len(observations)>1})
        inputs={'attempt':self.attempt('stable'),'quality':{k:True for k in ['citation_support','freshness','coverage','contradictions_resolved']},'used_finding_ids':['f1']}
        for _ in range(2):
            r=run('record-attempt',inputs,vrooli_memory=NS(learning=NS(record=record,observe=observe)))
            self.assertEqual('ok',r['status']);self.assertEqual(['f1'],r['signals']['contributing_finding_ids'])
        self.assertEqual(calls[0],calls[1]);self.assertEqual(calls[0]['attempt']['approach'],'compare');self.assertEqual(observations[0],observations[1]);self.assertTrue(r['signals']['existing']);self.assertTrue(r['signals']['observation_existing'])
        assessment=json.loads(observations[0]['observation']['correction'])
        self.assertEqual(inputs['quality'],assessment['quality'])
        self.assertEqual(['f1'],assessment['contributing_finding_ids'])
        self.assertEqual('complete',r['signals']['capture_status'])
    def test_observation_delivery_failure_is_bounded_partial_state(self):
        calls=[]
        def record(**kw):
            calls.append(kw)
            return Handle(meta={'entryId':'receipt','existing':False})
        def observe(**kw):
            raise RuntimeError('memory observation unavailable')
        inputs={'attempt':self.attempt('partial'),'quality':{k:True for k in ['citation_support','freshness','coverage','contradictions_resolved']}}
        r=run('record-attempt',inputs,vrooli_memory=NS(learning=NS(record=record,observe=observe)))
        self.assertEqual('partial',r['status'])
        self.assertEqual('capture_failed',r['errors'][0]['class'])
        self.assertEqual('receipt',r['signals']['entry_id'])
        self.assertEqual(['source-check'],r['evidence'])
        self.assertEqual(1,len(calls))
        self.assertEqual('verified_success',r['signals']['task_outcome'])
        self.assertEqual('capture_failed',r['signals']['capture_status'])
        retry=r['signals']['retry_inputs']
        self.assertEqual(calls[0]['attempt'],retry['attempt'])
        self.assertEqual('web-search-usage',retry['scope'])
        self.assertEqual('observe',r['errors'][0]['where'])

    def test_partial_quality_is_unknown_and_false_is_contradicted(self):
        observations=[]
        def record(**kw):return Handle(meta={'entryId':'receipt'})
        def observe(**kw):observations.append(kw['observation']);return Handle(meta={'entryId':'quality'})
        for quality,expected in [({'coverage':True},'unknown'),
                                 ({'coverage':False},'contradicted'),
                                 ({'coverage':None},'unknown')]:
            attempt=self.attempt('partial-quality')
            attempt['outcome']='unknown'
            result=run('record-attempt',{'attempt':attempt,'quality':quality},
                       vrooli_memory=NS(learning=NS(record=record,observe=observe)))
            self.assertEqual('ok',result['status'])
            self.assertEqual(expected,observations[-1]['disposition'])
            self.assertEqual(quality,json.loads(observations[-1]['correction'])['quality'])

    def test_malformed_optional_inputs_do_not_reach_capture(self):
        def forbidden(**kw):self.fail('Malformed input reached capture')
        for field,value in [('quality',[]),('quality',None),('used_finding_ids',{}),
                            ('attempt',[])]:
            inputs={'attempt':self.attempt('invalid'),field:value}
            result=run('record-attempt',inputs,lib=NS(vrooli_memory=NS(finish_attempt=forbidden)))
            self.assertEqual('invalid_input',result['errors'][0]['class'])
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

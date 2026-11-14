"""[REQ:KO-KB-002] Incomplete evidence must never become verified success."""
import contextlib
import io
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT=Path(__file__).parents[1]
class Handle:
    def __init__(self,rows=(),meta=None): self.rows=list(rows);self.fields=meta or {}
    def head(self,n):return self.rows[:n]
    def count(self):return len(self.rows)
    def meta(self):return self.fields

class KnowledgeWorkflows(unittest.TestCase):
    def run_program(self,name,inputs,api=None,memory=None,runtime=None,hub=None):
        scope={'inputs':inputs,'knowledge_observatory':NS(knowledge_base=api),'vrooli_memory':memory,'program_runtime':runtime,'search_hub':hub}
        def compare(**selectors):
            shared={'inputs':selectors,'vrooli_memory':memory}
            path=ROOT.parents[2]/'vrooli-memory/.vrooli/program-runtime/compare-outcomes.py'
            with contextlib.redirect_stdout(io.StringIO()):
                exec(compile(path.read_text(),str(path),'exec'),shared)
            return Handle([shared['envelope']])
        scope['lib']=NS(vrooli_memory=NS(compare_outcomes=compare))
        with contextlib.redirect_stdout(io.StringIO()) as output:
            exec(compile((ROOT/(name+'.py')).read_text(),name,'exec'),scope)
        self.assertEqual(1,len(output.getvalue().splitlines()))
        self.assertLess(len(output.getvalue().encode()),65536)
        return scope['envelope']
    def test_invalid_inputs_call_no_bindings(self):
        for name in ['collect-context','prepare-change','verify-change','learning-read','setpoint-read']:
            self.assertEqual('invalid_input',self.run_program(name,{'unexpected':True})['errors'][0]['class'])
    def test_no_match_is_not_success(self):
        result=self.run_program('collect-context',{'query':'missing'},NS(search=lambda **kw:Handle()))
        self.assertEqual('no_match',result['errors'][0]['class']);self.assertFalse(result['signals']['task_success_verified'])
    def test_context_keeps_unknown_authority_and_bounded_follow(self):
        seen=[]
        def inspect(**kw):
            seen.append(kw['path'])
            return Handle([{'path':'docs/b.md','kind':'local','exists':True}],{'path':kw['path'],'sha256':'a'*64,'content':'source','truncated':True,'metadata':{'knowledge_status':'historical'}})
        result=self.run_program('collect-context',{'query':'rule','scope':'path','target':'docs','follow_references':1},NS(search=lambda **kw:Handle([{'path':'docs/a.md'}],{'method':'text'}),inspect=inspect))
        self.assertEqual(['docs/a.md','docs/b.md'],seen);self.assertEqual('ok',result['status']);self.assertTrue(result['signals']['gaps']);self.assertFalse(result['signals']['task_success_verified'])
    def test_reference_follow_never_leaves_scope(self):
        seen=[]
        def inspect(**kw):seen.append(kw['path']);return Handle([{'path':'scenarios/other/docs/b.md','kind':'local','exists':True}],{'path':kw['path'],'sha256':'a'*64})
        self.run_program('collect-context',{'query':'rule','scope':'path','target':'docs'},NS(search=lambda **kw:Handle([{'path':'docs/a.md'}]),inspect=inspect))
        self.assertEqual(['docs/a.md'],seen)
    def test_partial_inspection_preserves_prior_evidence(self):
        def inspect(**kw):
            if kw['path']=='docs/b.md':raise RuntimeError('scenario_not_running')
            return Handle(meta={'path':'docs/a.md','sha256':'a'*64})
        result=self.run_program('collect-context',{'query':'rule'},NS(search=lambda **kw:Handle([{'path':'docs/a.md'},{'path':'docs/b.md'}]),inspect=inspect))
        self.assertEqual('partial',result['status']);self.assertEqual(1,len(result['signals']['documents']))
    def verification(self,hit='docs/a.md',revision_error=False,content_findings=None):
        def inspect(**kw):
            if revision_error:raise RuntimeError('failed_precondition: source revision changed')
            return Handle(meta={'sha256':'a'*64,'path':'docs/a.md','metadata':{'knowledge_status':'accepted'}})
        api=NS(inspect=inspect,search=lambda **kw:Handle([{'path':hit}],{'method':'text'}),health=lambda **kw:Handle(meta={'contentFindings':content_findings or []}))
        inputs={'expectations':[{'path':'docs/a.md','sha256':'a'*64,'knowledge_status':'accepted'}],'queries':[{'query':'rule','expected_paths':['docs/a.md']}],'base_path':'docs','mode':'text'}
        return self.run_program('verify-change',inputs,api)
    def test_revision_conflict_is_not_retried(self):self.assertEqual('revision_changed',self.verification(revision_error=True)['errors'][0]['class'])
    def test_missing_retrieval_or_broken_link_fails_verification(self):
        self.assertEqual('verification_failed',self.verification(hit='docs/wrong.md')['errors'][0]['class'])
        self.assertEqual('verification_failed',self.verification(content_findings=[{'code':'broken_link'}])['errors'][0]['class'])
    def test_verified_checks_do_not_claim_task_completion(self):
        result=self.verification();self.assertEqual('ok',result['status']);self.assertFalse(result['signals']['task_success_verified'])
    def test_truncated_review_cannot_authorize_retirement(self):
        api=NS(review=lambda **kw:Handle([{'path':'docs/artifacts/a.md','sha256':'a'*64,'metadata':{'knowledge_status':'supplemental'}}],{'truncated':True,'filesChecked':1}),search=lambda **kw:Handle(),health=lambda **kw:Handle())
        result=self.run_program('prepare-change',{'paths':['docs/artifacts/a.md'],'base_path':'docs','intent':'clean'},api)
        self.assertEqual('partial',result['status']);self.assertIn('plan owner',result['signals']['decisions_required'][0]['review']);self.assertEqual('a'*64,result['signals']['preconditions'][0]['sha256'])
    def test_learning_does_not_combine_incomparable_cohorts(self):
        memory=NS(learning=NS(measure=lambda **kw:Handle([{'operation':'read'},{'operation':'verify'}],{'reliable':True})))
        result=self.run_program('learning-read',{},memory=memory)
        for row in result['signals']['rows']:self.assertEqual('unreliable:cohort_sample',row['reason']);self.assertIsNone(row['in_band'])
        self.assertEqual(6,len(result['signals']['rows']))
        self.assertEqual('partial',result['status'])
        self.assertTrue(result['signals']['reliability']['cohorts_truncated'])

    def test_learning_preserves_full_identity_and_resolved_window(self):
        cohort={'operation':'operation-'*12,'contextKey':'context-'*20,'attempts':2}
        memory=NS(learning=NS(measure=lambda **kw:Handle([cohort],{'reliable':True,'eligibleAttempts':2})))
        result=self.run_program('learning-read',{'from':'2026-09-01T00:00:00Z','to':'2026-09-02T00:00:00Z'},memory=memory)
        self.assertEqual('ok',result['status'])
        self.assertEqual('2026-09-01T00:00:00Z',result['inputs']['from'])
        actual=result['signals']['rows'][0]['reading']['cohorts'][0]
        self.assertEqual(cohort['operation'],actual['operation'])
        self.assertEqual(cohort['contextKey'],actual['context'])
    def test_oversized_unicode_evidence_returns_explicit_bounded_gap(self):
        api=NS(search=lambda **kw:Handle([{'path':'docs/a.md'}]),inspect=lambda **kw:Handle(meta={'path':'docs/a.md','sha256':'a'*64,'content':'😀'*20000}))
        result=self.run_program('collect-context',{'query':'large'},api)
        self.assertEqual('partial',result['status']);self.assertTrue(result['signals']['output_truncated'])
    def test_memory_unavailable_is_not_zero(self):
        def unavailable(**kw):raise RuntimeError('scenario_not_running')
        result=self.run_program('learning-read',{},memory=NS(learning=NS(measure=unavailable)))
        self.assertEqual('unavailable',result['status'])
        for row in result['signals']['rows']:self.assertIsNone(row['reading'])

class MaintenanceWorkflows(unittest.TestCase):
    run_program = KnowledgeWorkflows.run_program
    """[REQ:KO-KB-005] Reviewed knowledge survives moves and debt stays visible."""
    def inputs(self):
        return {'expectations':[{'path':'docs/new.md','sha256':'a'*64}], 'queries':[{'query':'portable setup','expected_paths':['docs/new.md'],'forbidden_paths':['docs/old.md']}], 'base_path':'docs','absent_paths':['docs/old.md'],'preservation':[{'id':'setup','path':'docs/new.md','excerpt':'Portable setup','review':'preserved','evidence_ref':'path:docs/new.md#setup'}]}
    def api(self,missing=True,content='Portable setup',findings=(),hits=None):
        def inspect(**kw):
            return Handle(meta={'path':kw['path'],'missing':missing}) if kw.get('allow_missing') else Handle(meta={'path':kw['path'],'sha256':'a'*64,'content':content})
        return NS(inspect=inspect,search=lambda **kw:Handle(hits or [{'path':'docs/new.md'}]),health=lambda **kw:Handle(findings))
    def test_move_preserves_reviewed_knowledge(self):
        result=self.run_program('verify-change',self.inputs(),self.api())
        self.assertEqual('ok',result['status']); self.assertTrue(result['signals']['editorial_review_complete']);self.assertFalse(result['signals']['task_success_verified'])
    def test_missing_claim_or_unretired_source_fails(self):
        for api in [self.api(content='Linux only'),self.api(missing=False),self.api(hits=[{'path':'docs/new.md'},{'path':'docs/old.md'}])]:
            self.assertEqual('failed',self.run_program('verify-change',self.inputs(),api)['status'])
    def test_unresolved_editorial_review_is_partial(self):
        inputs=self.inputs();inputs['preservation'][0]['review']='unresolved'
        self.assertEqual('editorial_review_pending',self.run_program('verify-change',inputs,self.api())['errors'][0]['class'])
    def test_existing_debt_is_visible_new_debt_fails(self):
        old={'code':'missing','path':'docs/unrelated.md'};new={'code':'missing','path':'docs/new.md'}
        inputs=self.inputs();inputs['reference_baseline']={'base_path':'docs','checks':['links','refs'],'skip_external_links':True,'complete':True,'reference_findings':[old],'content_findings':[]}
        result=self.run_program('verify-change',inputs,self.api(findings=[old]))
        self.assertEqual('ok',result['status']);self.assertEqual(1,result['signals']['reference_delta']['retained_count'])
        changed=self.run_program('verify-change',inputs,self.api(findings=[old,new]))
        self.assertEqual('failed',changed['status']);self.assertEqual([{'kind':'reference','finding':new}],changed['signals']['reference_delta']['new_findings'])
        inputs['reference_baseline']['base_path']='other'
        self.assertEqual('invalid_input',self.run_program('verify-change',inputs)['errors'][0]['class'])
    def test_incomplete_baseline_rejected_and_overflow_not_success(self):
        inputs=self.inputs();inputs['reference_baseline']={'base_path':'docs','checks':['links','refs'],'skip_external_links':True,'complete':False,'reference_findings':[],'content_findings':[]}
        self.assertEqual('invalid_input',self.run_program('verify-change',inputs)['errors'][0]['class'])
        self.assertEqual('failed',self.run_program('verify-change',self.inputs(),self.api(findings=[{'code':str(i)} for i in range(101)]))['status'])
    def test_prepare_outputs_dispositions_and_baseline(self):
        inputs={'paths':['docs/old.md'],'base_path':'docs','intent':'consolidate portable setup','dispositions':[{'source':'docs/old.md','action':'relocate','targets':['docs/new.md'],'reason':'Canonical task location','preserve':['Portable setup and Linux qualification'],'decision':'resolved','evidence_refs':['path:docs/old.md#setup']}]}
        api=NS(review=lambda **kw:Handle([{'path':'docs/old.md','sha256':'a'*64}],{'filesChecked':2}),search=lambda **kw:Handle(),health=lambda **kw:Handle())
        result=self.run_program('prepare-change',inputs,api)
        self.assertEqual('ok',result['status']);self.assertEqual('a'*64,result['signals']['dispositions'][0]['source_sha256']);self.assertTrue(result['signals']['reference_baseline']['complete'])
        del inputs['dispositions']
        result=self.run_program('prepare-change',inputs,api)
        self.assertEqual('unresolved',result['signals']['dispositions'][0]['decision']);self.assertFalse(result['signals']['editorial_review_complete'])

if __name__=='__main__':unittest.main()

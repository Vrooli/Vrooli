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
        api=NS(review=lambda **kw:Handle([{'path':'docs/artifacts/a.md','sha256':'a'*64,'metadata':{'knowledge_status':'supplemental'}}],{'truncated':True,'filesChecked':1}),search=lambda **kw:Handle())
        result=self.run_program('prepare-change',{'paths':['docs/artifacts/a.md'],'base_path':'docs','intent':'clean'},api)
        self.assertEqual('partial',result['status']);self.assertIn('plan owner',result['signals']['decisions_required'][0]['review']);self.assertEqual('a'*64,result['signals']['preconditions'][0]['sha256'])
    def test_learning_does_not_combine_incomparable_cohorts(self):
        memory=NS(learning=NS(measure=lambda **kw:Handle([{'operation':'read'},{'operation':'verify'}],{'reliable':True})))
        result=self.run_program('learning-read',{},memory=memory)
        for row in result['signals']['rows']:self.assertEqual('unreliable:cohort_sample',row['reason']);self.assertIsNone(row['in_band'])
    def test_oversized_unicode_evidence_returns_explicit_bounded_gap(self):
        api=NS(search=lambda **kw:Handle([{'path':'docs/a.md'}]),inspect=lambda **kw:Handle(meta={'path':'docs/a.md','sha256':'a'*64,'content':'😀'*20000}))
        result=self.run_program('collect-context',{'query':'large'},api)
        self.assertEqual('partial',result['status']);self.assertTrue(result['signals']['output_truncated'])
    def test_memory_unavailable_is_not_zero(self):
        def unavailable(**kw):raise RuntimeError('scenario_not_running')
        result=self.run_program('learning-read',{},memory=NS(learning=NS(measure=unavailable)))
        self.assertEqual('partial',result['status'])
        for row in result['signals']['rows']:self.assertIsNone(row['reading'])

if __name__=='__main__':unittest.main()

import contextlib, io, json, pathlib, sys
from types import SimpleNamespace as N
class Handle:
    def __init__(self, meta=None, rows=None): self.m=meta or {};self.r=rows or []
    def meta(self): return self.m
    def head(self,n): return self.r[:n]
source=pathlib.Path(sys.argv[1]).read_text()
def evaluate(family_id='', graph_revision='7', foreign=False):
    execution={'id':'execution','planId':'plan','baselineSet':{'receiptId':'receipt'}}
    family={'familyId':'family','graph':{'revision':'7'},'members':[{'executionId':'foreign' if foreign else 'execution','planId':'plan'}]}
    receipt_calls=[]
    def receipt(**kw):
        receipt_calls.append(kw)
        return Handle({'receipt':{'receiptId':'receipt','state':'RECEIPT_STATE_QUEUED'}})
    owner=N(exec=N(status=lambda **kw:Handle({'execution':execution,'step':{'stepKind':'baseline_receipt_pending','nextActions':[{'id':'wait','argv':['test-genie','validation','wait','receipt']}]}})),families=N(get=lambda **kw:Handle({'family':family}),frontier=lambda **kw:Handle({'graphRevision':graph_revision,'batches':[]})))
    out=io.StringIO()
    with contextlib.redirect_stdout(out):exec(compile(source,sys.argv[1],'exec'),{'inputs':{'execution_id':'execution','family_id':family_id},'plan_manager':owner,'test_genie':N(validation=N(get=receipt)),'gather':lambda *fs:tuple(f() for f in fs)})
    assert len(out.getvalue().encode())<=4096
    return json.loads(out.getvalue()),receipt_calls
single,calls=evaluate()
assert single['status']=='ok' and single['signals']['validation']['state']=='RECEIPT_STATE_QUEUED'
assert single['signals']['next_action']['argv'][-1]=='receipt' and calls==[{'receipt_id':'receipt'}]
assert evaluate('family')[0]['status']=='ok'
assert evaluate('family','8')[0]['status']=='unavailable'
assert evaluate('family',foreign=True)[0]['errors'][0]['class']=='identity_mismatch'
print('execution board: standalone receipt, family identity, and concurrent graph change passed')

import contextlib, io, json, pathlib, sys
from types import SimpleNamespace as N
class Handle:
    def __init__(self,meta=None,rows=None):self.m=meta or {};self.r=rows or []
    def meta(self):return self.m
    def head(self,n):return self.r[:n]
    def count(self):return len(self.r)
def run(name,inputs,watch):
    path=pathlib.Path(sys.argv[1])/f'{name}.py'
    out=io.StringIO()
    with contextlib.redirect_stdout(out):exec(compile(path.read_text(),str(path),'exec'),{'inputs':inputs,'agent_manager':N(watch=watch),'gather':lambda *fs:tuple(f() for f in fs)})
    assert len(out.getvalue().encode())<=4096
    return json.loads(out.getvalue())
inspection={'watch':{'watchId':'watch','revision':'4','spec':{'policyVersion':'candidate'},'lastDecision':{'decisionId':'decision','classification':'stalled'}},'subjectStates':[{'runId':'child','status':'complete','terminal':True}],'cursorResetRequired':True}
coverage={'outcomes':19,'assessedOutcomes':0,'families':1,'population':'retained nonsuperseded outcomes'}
watch=N(inspect=lambda **kw:Handle(inspection),actions=lambda **kw:Handle(rows=[{'actionId':'action','decisionId':'decision','state':'WATCH_ACTION_STATE_APPLIED','targetRunId':'child'}]),policy_outcomes=lambda **kw:Handle({'coverage':coverage},[{'observedClass':'','familyExecutionId':'family'}]),policy_get=lambda **kw:Handle({'policy':{'version':'candidate'},'digest':'pinned'}))
case=run('supervision-case-read',{'watch_id':'watch'},watch)
assert case['status']=='ok' and case['signals']['current_children'][0]['terminal']
assert case['signals']['unassessed_sample']==1 and case['signals']['next_action']=='reconcile_cursor'
assert case['signals']['classification']=='stalled'  # historical decision is not overwritten by later completion
experiment=run('supervision-experiment-read',{'policy_version':'candidate'},watch)
assert experiment['status']=='ok' and experiment['signals']['evaluation'] is None
assert experiment['signals']['next_action']=='collect_assessed_evidence' and experiment['signals']['coverage']['families']==1
inspection['watch']['watchId']='other'
assert run('supervision-case-read',{'watch_id':'watch'},watch)['status']=='failed'
print('supervision boards: current vs historical state, missing assessment, denominators, and identity passed')

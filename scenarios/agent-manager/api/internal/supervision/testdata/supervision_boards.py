import contextlib, io, json, pathlib, sys
from types import SimpleNamespace as N
sys.path.insert(0, str(pathlib.Path(sys.argv[1]).resolve().parents[2] / "program-runtime" / "kernel"))
from host.program_helper import ProgramHelper
class Handle:
    def __init__(self,meta=None,rows=None):self.m=meta or {};self.r=rows or []
    def meta(self):return self.m
    def head(self,n):return self.r[:n]
    def count(self):return len(self.r)
def run(name,inputs,watch):
    path=pathlib.Path(sys.argv[1])/f'{name}.py'
    out=io.StringIO()
    environment={'inputs':inputs,'agent_manager':N(watch=watch),'gather':lambda *fs:tuple(f() for f in fs)}
    helper=ProgramHelper()
    helper._bind(environment)
    environment['program']=helper
    with contextlib.redirect_stdout(out):exec(compile(path.read_text(),str(path),'exec'),environment)
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
compact=run('supervision-observation-read',{'owner_observation':{'coverage':'complete','quotaObservations':[{'provider':'openai','pool':'primary','window':'daily','evidence_ref':'quota:1'}],'efforts':[{'id':'effort:a','targetRevision':'accepted','evidenceRevision':'changed','priorAssessment':{'id':'assessment:prior'},'changedEvidence':['checkpoint:2'],'usage':{'partial':True},'quotaObservations':[{'provider':'openai','pool':'primary','window':'run','evidence_ref':'quota:row'}],'namedWaits':['owner:wait:1'],'repairLinks':[{'work_ref':'swarm-manager:backlog/chore/adoption','state':'assigned'}],'detailRefs':['agent-manager:GetEffortBoard:effort:a','checkpoint:2','owner:wait:1']}]},'selected_effort_refs':['effort:a']},watch)
assert compact['status']=='ok' and compact['signals']['projected_count']==1
assert compact['signals']['efforts'][0]['priorAssessment']['id']=='assessment:prior'
assert compact['signals']['efforts'][0]['namedWaits']==['owner:wait:1']
assert compact['signals']['efforts'][0]['repairLinks'][0]['work_ref']=='swarm-manager:backlog/chore/adoption'
assert compact['signals']['quotaObservations'][0]['evidence_ref']=='quota:1'
assert compact['signals']['efforts'][0]['quotaObservations'][0]['evidence_ref']=='quota:row'
assert 'owner:wait:1' in compact['signals']['detail_refs']
inspection['watch']['watchId']='other'
assert run('supervision-case-read',{'watch_id':'watch'},watch)['status']=='failed'
print('supervision boards: current vs historical state, missing assessment, denominators, and identity passed')

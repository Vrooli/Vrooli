"""Read external binding health and outcome-linked learning; never invent a baseline."""
import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {'program':'command-center.setpoint-read','version':'1','status':'ok','phase':'collect','inputs':{},'signals':{'rows':[]},'errors':[],'evidence':[]}
def row(name,reading,target,in_band=None,reason=None):
    envelope['signals']['rows'].append({'row':name,'reading':reading,'target':target,'in_band':in_band,'unavailable':reason is not None,'reason':reason})
def guarded(call):
    def run():
        try: return call()
        except Exception as exc: return exc
    return run
results = gather(guarded(lambda:program_runtime.bindings.condition(scenario='command-center',window_seconds=604800,rows='conditions')),
                 guarded(lambda:vrooli_memory.learning.measure(scope='command-center-usage',operation='vision-walk-prep',rows='cohorts')))
envelope['phase']='classify'
for name,result in zip(['binding-health','learning'],results):
    if isinstance(result,Exception):
        envelope['status']='partial'
        reason='scenario_unreachable' if any(v in str(result).lower() for v in ['unreachable','connection refused','bridge unavailable']) else 'unreliable:binding_error'
        row(name,None,None,None,reason)
        envelope['errors'].append({'class':'binding_error','where':name,'detail':str(result)[:180]})
    elif name=='binding-health':
        total=result.count(); bad=result.filter(lambda r:r.get('status')!='CONDITION_STATUS_HEALTHY').count()
        row(name,{'total':total,'not_healthy':bad},'all declared bindings healthy',bad==0 if total else None,None if total else 'unreliable:no_bindings')
        envelope['evidence'].append('program-runtime/bindings/condition')
    else:
        meta=result.meta(); reliable=meta.get('reliable',False)
        row(name,{'eligible_attempts':meta.get('eligibleAttempts',0),'reliable':reliable,'reason':meta.get('reason'),
                  'cohorts':result.head(8),'truncated':meta.get('truncated',False)},None,None,
            None if reliable and not meta.get('truncated',False) else 'unreliable:'+str(meta.get('reason') or 'capped_or_empty').removeprefix('unreliable:'))
        envelope['evidence'].append('vrooli-memory/learning/measure')
row('external-friction',None,None,None,'read_elsewhere:agent-manager.friction-digest')
row('briefing-quality',None,'12 phases; exact checkpoint; named source gaps',None,'read_elsewhere:command-center.vision-walk-prep')
row('operator-usefulness',None,None,None,'pending_telemetry')
envelope['phase']='report'
print(json.dumps(envelope,separators=(',',':')))

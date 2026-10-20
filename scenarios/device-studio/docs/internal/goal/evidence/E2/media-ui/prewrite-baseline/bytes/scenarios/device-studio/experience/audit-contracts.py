"""Check design-contract coherence. This audits files; it never executes UI journeys."""
import hashlib,json,re
from pathlib import Path
root=Path(__file__).resolve().parent
scenario=root.parent
index=json.loads((root/'index.json').read_text())
reqs={}
for p in (scenario/'requirements').glob('*/module.json'):
 for r in json.loads(p.read_text()).get('requirements',[]):reqs[r['id']]=r
prd=(scenario/'PRD.md').read_text()
common={'idle','pending','retrying','default','loading','submitting','saving','syncing','refreshing','empty','partial','stale','success','validation-error','request-error','permission-denied','offline','retry'}
pages={};claims=[];checks=[]
def check(name,ok,detail):
 checks.append({'check':name,'passed':bool(ok),'detail':detail})
check('all-index-entries-draft',all(r['status']=='draft' for key in ['pages','journeys'] for r in index[key]),'Draft route status is intentionally distinct from actual template route existence.')
check('exact-page-file-parity',{p.stem for p in (root/'pages').glob('*.json')}=={r['id'] for r in index['pages']},'No orphan page documents or missing index entries.')
check('exact-journey-file-parity',{p.stem for p in (root/'journeys').glob('*.json')}=={r['id'] for r in index['journeys']},'No orphan journey documents or missing index entries.')
for r in index['pages']:
 d=json.loads((root/r['path']).read_text());pages[r['id']]=d;claims+=d.get('claims',[])
 check(f"{r['id']}:common-intended-states",common <= {s['id'] for s in d['states']},'Declared intended state vocabulary only; no runtime setup or coverage proof.')
 check(f"{r['id']}:operational-targets",all(x in prd for x in d['page']['prd_refs']),'Every OT id appears in populated PRD.')
 check(f"{r['id']}:requirement-ids",all(x in reqs for x in d.get('x-requirements',[])),'All optional x-requirements ids resolve to actual requirement modules.')
 if r['id'] not in ['dashboard','notes','settings']:
  check(f"{r['id']}:no-fake-runtime-bindings",not any(k in d for k in ['bindings','regions','elements']) and all('setup' not in s for s in d['states']),'Future controls use x-design metadata; no invented selectors, grounded regions or state hooks.')
for r in index['journeys']:
 d=json.loads((root/r['path']).read_text());claims+=d.get('claims',[])
 check(f"{r['id']}:journey-links",all(s['page'] in pages and s.get('state','default') in {x['id'] for x in pages[s['page']]['states']} for s in d['steps']),'Every planned step references a declared page and intended state.')
 check(f"{r['id']}:operational-targets",all(x in prd for x in d['journey']['prd_refs']),'Every journey OT id appears in populated PRD.')
check('all-claims-aspirational',all(c['tier']=='aspirational' for c in claims),'No runtime machine claim or manual attestation has been manufactured.')
check('template-notes-real-binding',pages['notes']['bindings']['regions']['notes']['testid']=='notes-surface' and 'notes-surface' in (scenario/'ui/src/consts/selectors.ts').read_text(),'Inherited Notes testid is present in actual selector registry; no runtime interaction claimed.')
check('template-major-export-resolves-pin','1.0.3' in (scenario.parents[1]/'packages/react-component-library/dist/exports/ExperienceSurface/1.js').read_text() and pages['notes']['regions'][0]['component']['library']['version']=='1.0.3','Actual ExperienceSurface/1 package export resolves to pinned1.0.3 canonical contract.')
check('actual-template-routes-only',all(x in (scenario/'ui/src/app/routes.tsx').read_text() for x in ['DashboardPage','NotesPage','SettingsPage']) and 'PLACEHOLDER:home-surface' in (scenario/'ui/src/pages/DashboardPage.tsx').read_text(),'Read-only source inspection: template router and home placeholder remain; no product route implemented.')
covered={x for d in pages.values() for x in d['page'].get('prd_refs',[])}
check('all-p0-targets-covered',{f'OT-P0-{i:03}' for i in range(1,15)}<=covered,'Design references cover all14 committed P0 target groups, not only first photographic milestone.')
check('no-attestation-files',not any((root/'attestations').glob('*')) if (root/'attestations').exists() else True,'No attestations created; visual/operator approval remains absent.')
files={str(p.relative_to(scenario)):hashlib.sha256(p.read_bytes()).hexdigest() for p in [scenario/'DESIGN.md',root/'index.json',*sorted((root/'pages').glob('*.json')),*sorted((root/'journeys').glob('*.json'))]}
out={'kind':'design-contract-audit','status':'PASSED' if all(c['passed'] for c in checks) else 'FAILED','scope':'Read-only semantic and source-link audit, not UI tests, journey execution, accessibility or visual acceptance.','counts':{'pages':len(pages),'real_template_pages':3,'planned_product_pages':len(pages)-3,'journeys':len(index['journeys']),'aspirational_claims':len(claims),'runtime_machine_claims':0,'manual_attestations':0,'checks':len(checks)},'checks':checks,'artifact_sha256':files,'remaining':['Ratified concept and paired desktop/mobile mockups','Product routes and deterministic state setup/bindings','Real BAS structure/capture and interactive journey evidence','Accessibility, responsive, renderer and final-look acceptance','Real worker, credit, privacy and cancellation integration proofs']}
(root/'validation').mkdir(exist_ok=True)
(root/'validation/authoring-audit.json').write_text(json.dumps(out,indent=2)+'\n')
print(json.dumps(out['counts']));print(out['status'])
if out['status']!='PASSED':raise SystemExit(1)

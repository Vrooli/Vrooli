from pathlib import Path
import json,hashlib,zipfile,re
from PIL import Image
import fitz
ROOT=Path(__file__).resolve().parent.parent
def sha(p): return hashlib.sha256(p.read_bytes()).hexdigest()
def info(p): return {'path':str(p.relative_to(ROOT)),'bytes':p.stat().st_size,'sha256':sha(p)}
pdf=ROOT/'Marketing experience 0.3 - exact native review.pdf'
atlas=ROOT/'Marketing experience 0.3 - full-size visual atlas.pdf'
export=json.loads((ROOT/'qa/native-final-export.json').read_text())
states=json.loads((ROOT/'current-build-visuals.json').read_text())
pages=json.loads((ROOT/'atlas-pages.json').read_text())
lookup={s:p['page'] for p in pages for s in p['sources']}
old=json.loads((ROOT/'history/visual-catalogue-0.2.json').read_text())
old['catalogue_revision']='0.3';old['as_of']='2026-10-04'
old['role']='Living project-local visual version catalogue; current selection plus preserved exact prior records, not an execution queue'
old['owner_decisions']='Task-specific C direction selected as starting point; exact 0.3 specification/screens and remaining consequential choices await scoped owner review.'
old['current_candidate']='0.3'
old['historical_artifact_resolution']={
 'candidate_0_2':{'source_package_file_id':'1637HJzN3a54J0dLp9-_PV73hoIlC7gsw','source_package_sha256':'deb08376b9ce5b18af5d3e3fb691c3461ce7057fe6f2ea39f6d8964e645909e9','pdf_file_id':'1pXlEctQBoX6olsvrGPwl6rRL09uiT7f-','pdf_sha256':'7b24d7d76b8e5209b6e911ae311092d1d6d877f115f2a182a82416d89e358118','atlas_file_id':'1r71CliiJ074mdWOye7RGD0H-MlpGlSx9','atlas_sha256':'124be8b33e43fd44b3f55e842efb301b984089bcd2d8dd9a7ac2241adf522a4a','state':'historical preserved baseline; use these immutable targets for earlier candidate_0_2 current/delivery records'},
 'candidate_0_1':{'state':'Existing original record IDs and earlier source package remain unchanged; no 0.3 approval transfers to them'}}
source_map={'M01':'draw_workspaces.py','M02':'draw_operations.py','M03':'draw_operations.py','M04':'draw_workspaces.py','M05':'draw_workspaces.py','M06':'draw_exemplar.py','M07':'draw_operations.py','M08':'draw_workspaces.py','M09':'draw_operations.py','M10':'draw_state_specimens.py','M11':'draw_operations.py','M12':'draw_workspaces.py','M13':'draw_operations.py','M14':'draw_operations.py','M15':'draw_workspaces.py','M16':'draw_state_specimens.py','M17':'draw_operations.py','M18':'draw_state_specimens.py','M19':'draw_state_specimens.py','M20':'draw_state_specimens.py','M21':'draw_adaptive.py','M22':'draw_adaptive.py','M23':'draw_handoffs.py','M24':'draw_handoffs.py','M25':'draw_handoffs.py'}
chmap={};chapter=None
for line in (ROOT/'source/dossier.md').read_text().splitlines():
 m=re.match(r'## (\d+) ·',line)
 if m:chapter=int(m[1])
 m=re.match(r'\[FIG:(M\d+)\]',line)
 if m:chmap.setdefault(m[1],[]).append(chapter)
current=[]
for a in states:
 png=Path(a['png']);vector=Path(a['pdf']);family=a['family'];source=ROOT/'source'/source_map[family]
 current.append({'version_id':'0.3/'+a['name'],'family':family,'chapters':chmap.get(family,[]),'png':info(png),'vector':info(vector),'original_raster_dimensions':list(Image.open(png).size),'logical_viewport':[a['width'],a['height']],'classification':'conceptual diagram' if family in ('M10','M18') else 'proposed static product/state specimen','authoring_source':info(source),'state_brief':'source/screen-briefs.json','final_compatibility_basis':{'manuscript_sha256':sha(ROOT/'source/dossier.md'),'check':'post-production compatibility; not a claim of captured generation-time manuscript hash'},'production':'rendered and corrected','design_review':'independent reviews with bounded scopes in qa; final review identifies exactly sampled states','owner_acceptance':'direction starting point selected; exact state unaccepted','freshness':'checked to captured source/design basis; runtime unverified','retirement':'current review selection','implementation':'not started or verified','delivery':{'source_package_file_id':'1rsy4cSNIbh3H_wjJxWiGom057s49Y7io','atlas_file_id':'15UXSK90NG3YHEqL5C-lya_RahD7kitPt','atlas_page':lookup[a['name']]}})
selected={Path(a['png']).name for a in states}
history=[{**info(p),'version_id':'0.3/'+p.stem,'state':'superseded production pixels retained; not current selection','owner_acceptance':'unaccepted'} for p in sorted((ROOT/'screens').glob('*.png')) if p.name not in selected]
old['candidate_0_3']={'schema':'marketing-project.visual-catalogue/0.3','source_of_selection':'this existing live catalogue','current':current,'prior_production_pixels':history,'native_dossier_integration':'complete;15 chapters,17 choice histories,34 preserved anchors,45 images','native_revision':export['native_revision'],'current_source_manuscript':info(ROOT/'source/dossier.md'),'chosen_reference_manifest':info(ROOT/'references/concept-lineage.json'),'complete_method_read':'Design dossier1.2.1 plus access-only1.2.2 clarification from verified owned source; all seven references consumed, loader failure not silently treated as passed','review_record':'qa/final-independent-review.md','limits':['Static design only; no native source application or implementation','Live Google Docs/iOS, runtime reflow/accessibility and account actions remain unverified','Separate M25 private reflection must not resolve unknown A7','Raw repository bundle remains separately owner-only and is excluded']}
delivery={'candidate':'0.3','native_document_id':'1LlO2CTsIwHvSlPpjVmdLfjXc2EpulEldwhIS_RVuPko','native_tab_id':'t.0','native_revision':export['native_revision'],'native_pdf':{**info(pdf),'file_id':'1Qb5AdNnwHzNoejA-NLGYOSRgIyiMOM-W','pages':export['pages']},'atlas':{**info(atlas),'file_id':'15UXSK90NG3YHEqL5C-lya_RahD7kitPt','pages':len(pages),'states':len(states)},'source_package_file_id':'1rsy4cSNIbh3H_wjJxWiGom057s49Y7io','visual_catalogue_file_id':'1e3Uf-neFK15Jkra2M-Ye_nlkYt3UtRty','review':'qa/final-independent-review.md','exact_export_check':{'first_95_pages_identical_to_inspected_first_export':True,'changed_pages_inspected':[96,97],'inline_images_pixel_identical':45,'valid_internal_links':57},'readability_limit':'Desktop figures are overview compositions; phone/focused figures and native prose carry readable controls. Actual live Docs/iOS viewer unverified.','delivery_set':'Separate native Doc/PDF/atlas plus this source ZIP and live catalogue. PDF/atlas hashes are pinned here; their bytes are served by their existing adjacent IDs, not duplicated in the ZIP.','authority':'Whole redesign commissioned; exact product choices and implementation remain unaccepted. No publication/accounts/grants/spending.'}
old['delivery_record']='delivery-record.json in current source package; the PDF/atlas are adjacent same-ID outputs'
old['candidate_0_3']['delivery']=delivery
(ROOT/'delivery-record.json').write_text(json.dumps(delivery,ensure_ascii=False,indent=2))
(ROOT/'visual-catalogue.json').write_text(json.dumps(old,ensure_ascii=False,indent=2))
allowed_qa=['initial-contract-challenge.md','initial-journey-challenge.md','exemplar-review.md','exemplar-review-v04.md','full-visual-review.md','final-independent-review.md','author-delivery-review.md','native-final-parity.json','native-final-export.json','token-contrast.json','atlas-parity-quantified.json']
files=[ROOT/'README.md',ROOT/'delivery-record.json',ROOT/'visual-catalogue.json',ROOT/'current-build-visuals.json',ROOT/'native-image-plan.json',ROOT/'atlas-pages.json']
files += [p for p in (ROOT/'source').glob('*') if p.is_file() and p.suffix in('.py','.md','.json','.txt') and not p.name.startswith(('native-write','native-content'))]
current_vectors={Path(a['pdf']).name for a in states}
files += [p for p in (ROOT/'screens').glob('*') if p.suffix=='.png' or p.name in current_vectors]
files += [p for p in (ROOT/'references').rglob('*') if p.is_file()]
files += [p for p in (ROOT/'history').rglob('*') if p.is_file()]
files += [ROOT/'qa'/name for name in allowed_qa]
files=list(dict.fromkeys(files))
manifest={'role':'Exact packaged-file inventory; not a competing live visual selection','files':[info(p) for p in sorted(files)],'excluded':['raw71-file repository JSON and extracted repository files','transient tool request/results and private operational notes','duplicate native PDF/atlas bytes, separately pinned','superseded intermediate vector exports; original PNG history retained']}
(ROOT/'package-manifest.json').write_text(json.dumps(manifest,indent=2));files.append(ROOT/'package-manifest.json')
path=ROOT/'Marketing experience 0.3 - reviewed design source.zip'
with zipfile.ZipFile(path,'w',zipfile.ZIP_DEFLATED,compresslevel=9) as z:
 for p in files:z.write(p,str(p.relative_to(ROOT)))
with zipfile.ZipFile(path) as z:
 assert z.testzip() is None
 for a in manifest['files']:assert hashlib.sha256(z.read(a['path'])).hexdigest()==a['sha256']
receipt={'path':path.name,'bytes':path.stat().st_size,'sha256':sha(path),'file_count':len(files),'all_manifest_hashes_checked':True}
(ROOT/'qa/source-package-build.json').write_text(json.dumps(receipt,indent=2));print(receipt)

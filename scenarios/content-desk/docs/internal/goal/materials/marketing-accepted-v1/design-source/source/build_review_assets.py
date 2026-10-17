from pathlib import Path
import json,re,fitz,hashlib,subprocess
ROOT=Path(__file__).resolve().parent.parent
SCREENS=ROOT/'screens';OUT=ROOT/'document-images';OUT.mkdir(exist_ok=True)
groups={}
for f in SCREENS.glob('*.png'):
 m=re.match(r'(M\d+-.*?)-v(\d+)(.*)\.png',f.name)
 if not m:continue
 key=m.group(1)+m.group(3)
 if key not in groups or int(m.group(2))>groups[key][0]:groups[key]=(int(m.group(2)),f)
states=[]
for key,(v,f) in sorted(groups.items()):
 pdf=f.with_suffix('.pdf');doc=fitz.open(pdf);r=doc[0].rect
 states.append({'name':f.stem,'family':f.name[:3],'png':str(f),'pdf':str(pdf),'width':r.width,'height':r.height,'png_sha256':hashlib.sha256(f.read_bytes()).hexdigest(),'pdf_sha256':hashlib.sha256(pdf.read_bytes()).hexdigest()})
(ROOT/'current-build-visuals.json').write_text(json.dumps(states,indent=2))

def combine(name,items):
 w=sum(a['width'] for a in items)+20*(len(items)-1);h=max(a['height'] for a in items)
 doc=fitz.open();page=doc.new_page(width=w,height=h);x=0
 for a in items:
  src=fitz.open(a['pdf']);page.show_pdf_page(fitz.Rect(x,0,x+a['width'],a['height']),src,0);x+=a['width']+20
 pdf=OUT/(name+'.pdf');doc.save(pdf,garbage=4,deflate=True)
 return {'name':name,'pdf':str(pdf),'width':w,'height':h}

def raster(a):
 path=OUT/(a['name']+'.png')
 # Direct vector derivative. Keep provider-observed max dimension <=2500.
 scale=min(2,2500/max(a['width'],a['height']))
 fitz.open(a['pdf'])[0].get_pixmap(matrix=fitz.Matrix(scale,scale),alpha=False).save(path)
 return str(path)

family={}
for a in states:family.setdefault(a['family'],[]).append(a)
native=[]
for f,items in family.items():
 desk=[a for a in items if a['width']>500];phone=[a for a in items if a['width']<500]
 if not desk and len(phone)>1:
  a=combine(f+'-readable-phone-details',phone);native.append({'family':f,'path':raster(a),'width_pt':504,'height_pt':504*a['height']/a['width'],'kind':'readable related phone states','source_names':[b['name'] for b in phone]})
 else:
  for a in desk:
   w=430 if f=='M10' else 410 if f=='M18' else 504
   native.append({'family':f,'path':raster(a),'width_pt':w,'height_pt':w*a['height']/a['width'],'kind':'diagram' if f in ('M10','M18') else 'desktop layout overview','source_names':[a['name']]})
  if len(phone)>1:
   a=combine(f+'-readable-phone-details',phone);native.append({'family':f,'path':raster(a),'width_pt':504,'height_pt':504*a['height']/a['width'],'kind':'readable phone reading and change states','source_names':[b['name'] for b in phone]})
  elif phone:
   a=phone[0];native.append({'family':f,'path':raster(a),'width_pt':260,'height_pt':260*a['height']/a['width'],'kind':'readable phone detail','source_names':[a['name']]})
# Readable main-document detail of the exact failed-save control.
a=next(a for a in states if a['family']=='M20');s=fitz.open(a['pdf']);d=fitz.open();p=d.new_page(width=663,height=296);p.show_pdf_page(p.rect,s,0,clip=fitz.Rect(747,472,1410,768));fp=OUT/'M20-save-unknown-detail.pdf';d.save(fp,deflate=True);a={'name':'M20-save-unknown-detail','pdf':str(fp),'width':663,'height':296}
native.append({'family':'M20','path':raster(a),'width_pt':504,'height_pt':504*296/663,'kind':'readable failed-save control detail','source_names':['M20-states-v01-desktop']})
for i,a in enumerate(native):
 a['slot']=f'IMG{i+1:02d}';a['sha256']=hashlib.sha256(Path(a['path']).read_bytes()).hexdigest()
(ROOT/'native-image-plan.json').write_text(json.dumps(native,indent=2))

# Full-size vector collection; pairs retain true logical viewports.
atlas=fitz.open();pages=[]
def add_page(f,items,desc):
 w=sum(a['width'] for a in items)+24*(len(items)-1)+40;h=max(a['height'] for a in items)+106
 p=atlas.new_page(width=w,height=h);p.insert_text((20,31),f+'  |  '+desc,fontsize=19,color=(.07,.18,.8));x=20
 for a in items:
  src=fitz.open(a['pdf']);p.show_pdf_page(fitz.Rect(x,61,x+a['width'],61+a['height']),src,0);x+=a['width']+24
 footer='Marketing 0.3 | Proposed static design | Native owners and implementation limits remain as specified'
 spare=p.insert_textbox(fitz.Rect(20,h-35,w-20,h-7),footer,fontsize=9,color=(.35,.38,.43))
 assert spare>=0, ('Atlas footer does not fit',f,w,h,spare)
 pages.append({'page':len(atlas),'family':f,'sources':[a['name'] for a in items]})
for f,items in family.items():
 desk=[a for a in items if a['width']>500];phone=[a for a in items if a['width']<500]
 if desk:
  for i,a in enumerate(desk):add_page(f,[a]+([phone[0]] if i==0 and phone else []),'Desktop and phone' if i==0 and phone else 'Full-size specimen')
  if len(phone)>1:add_page(f,phone[1:],'Additional phone state')
 else:add_page(f,phone,'Readable phone states')
atlas.set_metadata({'title':'Marketing experience 0.3 — full-size visual atlas','subject':'Proposed product screens, state specimens and conceptual ownership; not implementation proof','author':'Prepared for owner review'})
atlas_path=ROOT/'Marketing experience 0.3 - full-size visual atlas.pdf';atlas.save(atlas_path,garbage=4,deflate=True)
(ROOT/'atlas-pages.json').write_text(json.dumps(pages,indent=2))
print(json.dumps({'states':len(states),'native_images':len(native),'atlas_pages':len(pages),'atlas_bytes':atlas_path.stat().st_size}))

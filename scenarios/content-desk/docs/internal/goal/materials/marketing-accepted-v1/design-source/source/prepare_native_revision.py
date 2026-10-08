from pathlib import Path
import json,re
ROOT=Path(__file__).resolve().parent.parent;TAB='t.0';DOC='1LlO2CTsIwHvSlPpjVmdLfjXc2EpulEldwhIS_RVuPko'
baseline=json.loads((ROOT/'research/native-baseline/document-result.json').read_text())['structuredContent']
tab=baseline['tabs'][0];old=[]
for e in tab['body']['content']:
 p=e.get('paragraph')
 if p:
  text=''.join(a.get('textRun',{}).get('content','') for a in p['elements']).rstrip('\n')
  old.append({'text':text,'start':e.get('startIndex',0),'end':e['endIndex'],'style':p.get('paragraphStyle',{}).get('namedStyleType','NORMAL_TEXT'),'heading':p.get('paragraphStyle',{}).get('headingId')})
imgs=json.loads((ROOT/'native-image-plan.json').read_text());byfamily={}
for a in imgs:byfamily.setdefault(a['family'],[]).append(a)
source=(ROOT/'source/dossier.md').read_text();lines=[]
for line in source.splitlines():
 if not line.strip():continue
 m=re.fullmatch(r'\[FIG:(M\d+)\]',line)
 if m:
  for i,a in enumerate(byfamily[m.group(1)]):
   lines.append('[IMAGE:'+a['slot']+']')
   label=a['kind'].capitalize()
   lines.append(f'Figure {m.group(1)} · {label}. Proposed design; [open full-size visual atlas](https://drive.google.com/file/d/15UXSK90NG3YHEqL5C-lya_RahD7kitPt/view).')
 else:lines.append(line)

def ulen(s):return len(s.encode('utf-16-le'))//2
paras=[];links=[];chips=[];image_slots=[];pos=1;alltext=''
for raw in lines:
 style='NORMAL_TEXT';line=raw;islist=False
 for prefix,named in [('### ','HEADING_2'),('## ','HEADING_1'),('# ','TITLE')]:
  if line.startswith(prefix):style=named;line=line[len(prefix):];break
 if line.startswith('Vrooli marketing experience and delivery plan'):style='SUBTITLE'
 if re.match(r'^\d+\. ',line):line=re.sub(r'^\d+\. ','',line);islist=True
 im=re.fullmatch(r'\[IMAGE:(IMG\d+)\]',line)
 if im:
  line='@';image_slots.append({'index':pos,**next(a for a in imgs if a['slot']==im.group(1))})
 else:
  # Compile semantic links and dates into one-unit native-chip placeholders.
  tokens=[]
  for m in re.finditer(r'\[([^\]]+)\]\((https://[^)]+)\)|Oct [34], 2026',line):tokens.append(m)
  out='';cur=0
  for m in tokens:
   out+=line[cur:m.start()];start=pos+ulen(out)
   if m.group(0).startswith('Oct '):
    chips.append({'index':start,'kind':'DATE','value':'2026-10-0'+m.group(0)[4]});out+='@'
   else:
    label,url=m.group(1),m.group(2)
    if re.match(r'https://docs.google.com/(document|spreadsheets|presentation)/d/',url):
     canonical=re.sub(r'[?#].*','',url);canonical=re.sub(r'/edit/?$','',canonical)+'/edit'
     chips.append({'index':start,'kind':'LINK','value':canonical});out+='@'
    else:
     out+=label;links.append({'start':start,'end':start+ulen(label),'url':url})
   cur=m.end()
  line=out+line[cur:]
 end=pos+ulen(line+'\n');paras.append({'start':pos,'end':end,'text':line,'style':style,'list':islist,'image':bool(im),'caption':raw.startswith('Figure M')});alltext+=line+'\n';pos=end

# Retain every existing numbered chapter and every existing Chapter14 review anchor.
oldanchors=[];inch14=False
for p in old:
 if p['style']=='HEADING_1':inch14=p['text'].startswith('14 ·');oldanchors.append(p)
 elif inch14 and p['style']=='HEADING_2':oldanchors.append(p)
newanchors=[];inch14=False
for p in paras:
 if p['style']=='HEADING_1':inch14=p['text'].startswith('14 ·');newanchors.append(p)
 elif inch14 and p['style']=='HEADING_2':newanchors.append(p)
assert [p['text'] for p in oldanchors]==[p['text'] for p in newanchors],([p['text'] for p in oldanchors],[p['text'] for p in newanchors])
def uslice(a,b):return alltext.encode('utf-16-le')[(a-1)*2:(b-1)*2].decode('utf-16-le')
oldend=tab['body']['content'][-1]['endIndex']-1
segments=[];ostart=nstart=1
for o,n in zip(oldanchors,newanchors):segments.append((ostart,o['start'],nstart,n['start']));ostart=o['end'];nstart=n['end']
segments.append((ostart,oldend,nstart,pos-1))
requests=[]
for a,b,c,d in reversed(segments):
 if b>a:requests.append({'deleteContentRange':{'range':{'startIndex':a,'endIndex':b,'tabId':TAB}}})
 content=uslice(c,d)
 if content:requests.append({'insertText':{'location':{'index':a,'tabId':TAB},'text':content}})

def ran(a,b):return {'startIndex':a,'endIndex':b,'tabId':TAB}
def rgb(h):return {'red':int(h[0:2],16)/255,'green':int(h[2:4],16)/255,'blue':int(h[4:6],16)/255}
requests.append({'updateTextStyle':{'range':ran(1,pos-1),'textStyle':{'weightedFontFamily':{'fontFamily':'Arial'},'fontSize':{'magnitude':10.5,'unit':'PT'},'bold':False,'italic':False,'foregroundColor':{'color':{'rgbColor':rgb('171B2E')}}},'fields':'weightedFontFamily,fontSize,bold,italic,foregroundColor'}})
requests.append({'updateParagraphStyle':{'range':ran(1,pos-1),'paragraphStyle':{'spaceAbove':{'magnitude':0,'unit':'PT'},'spaceBelow':{'magnitude':7,'unit':'PT'},'lineSpacing':112,'keepLinesTogether':True,'keepWithNext':False,'alignment':'START','pageBreakBefore':False},'fields':'spaceAbove,spaceBelow,lineSpacing,keepLinesTogether,keepWithNext,alignment,pageBreakBefore'}})
requests.append({'deleteParagraphBullets':{'range':ran(1,pos-1)}})
for p in paras:
 st={'namedStyleType':p['style']};fields=['namedStyleType']
 if p['style'] in ('TITLE','SUBTITLE','HEADING_1','HEADING_2'):
  st.update({'keepWithNext':True,'spaceAbove':{'magnitude':15 if p['style']=='HEADING_1' else 8,'unit':'PT'},'spaceBelow':{'magnitude':7,'unit':'PT'}});fields+=['keepWithNext','spaceAbove','spaceBelow']
 if p['image']:st.update({'alignment':'CENTER','keepWithNext':True,'spaceAbove':{'magnitude':6,'unit':'PT'},'spaceBelow':{'magnitude':4,'unit':'PT'}});fields+=['alignment','keepWithNext','spaceAbove','spaceBelow']
 requests.append({'updateParagraphStyle':{'range':ran(p['start'],p['end']),'paragraphStyle':st,'fields':','.join(fields)}})
 if p['style'] in ('TITLE','SUBTITLE','HEADING_1','HEADING_2') or p['caption']:
  size={'TITLE':29,'SUBTITLE':13,'HEADING_1':19,'HEADING_2':12.5}.get(p['style'],9)
  col='1230CD' if p['style'] in ('TITLE','HEADING_1') else '171B2E' if p['style']=='HEADING_2' else '5B6070'
  requests.append({'updateTextStyle':{'range':ran(p['start'],p['end']-1),'textStyle':{'fontSize':{'magnitude':size,'unit':'PT'},'bold':p['style'] in ('TITLE','HEADING_1','HEADING_2'),'foregroundColor':{'color':{'rgbColor':rgb(col)}}},'fields':'fontSize,bold,foregroundColor'}})
for a in links:requests.append({'updateTextStyle':{'range':ran(a['start'],a['end']),'textStyle':{'link':{'url':a['url']},'foregroundColor':{'color':{'rgbColor':rgb('1230CD')}},'underline':False},'fields':'link,foregroundColor,underline'}})
listparas=[p for p in paras if p['list']]
if listparas:requests.append({'createParagraphBullets':{'range':ran(listparas[0]['start'],listparas[-1]['end']),'bulletPreset':'NUMBERED_DECIMAL_ALPHA_ROMAN'}})
# Chip replacement preserves all precomputed indexes (one unit replaces one unit).
for a in sorted(chips,key=lambda a:a['index'],reverse=True):
 requests.append({'deleteContentRange':{'range':ran(a['index'],a['index']+1)}})
 if a['kind']=='DATE':requests.append({'insertDate':{'location':{'index':a['index'],'tabId':TAB},'dateElementProperties':{'timestamp':a['value']+'T00:00:00Z','locale':'en','dateFormat':'DATE_FORMAT_MONTH_DAY_YEAR_ABBREVIATED','timeFormat':'TIME_FORMAT_DISABLED'}}})
 else:requests.append({'insertRichLink':{'location':{'index':a['index'],'tabId':TAB},'richLinkProperties':{'uri':a['value']}}})
model={'text':alltext,'paragraphs':paras,'links':links,'chips':chips,'images':image_slots,'end':pos,'preserved_anchors':[{'text':o['text'],'heading':o['heading'],'new_start':n['start']} for o,n in zip(oldanchors,newanchors)]}
(ROOT/'source/native-content.json').write_text(json.dumps(model,ensure_ascii=False))
(ROOT/'source/native-write-prepared.json').write_text(json.dumps({'document_id':DOC,'write_control':{'requiredRevisionId':baseline['revisionId']},'requests':requests},ensure_ascii=False))
print(json.dumps({'requests':len(requests),'paragraphs':len(paras),'images':len(image_slots),'chips':len(chips),'preserved_anchors':len(oldanchors),'end':pos,'bytes':(ROOT/'source/native-write-prepared.json').stat().st_size}))

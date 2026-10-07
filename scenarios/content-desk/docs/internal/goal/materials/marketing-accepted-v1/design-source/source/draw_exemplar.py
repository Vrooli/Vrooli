from pathlib import Path
from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.lib.colors import HexColor
import subprocess, json, hashlib

ROOT=Path(__file__).resolve().parent.parent
OUT=ROOT/'screens'; OUT.mkdir(exist_ok=True)
FONTS=ROOT/'source'/'fonts'; FONTS.mkdir(exist_ok=True)
for name,src in [('Sans','Regular'),('Medium','Semibold'),('Bold','Bold')]:
 pdfmetrics.registerFont(TTFont(name,'/usr/share/fonts/truetype/open-sans/OpenSans-'+src+'.ttf'))
pdfmetrics.registerFont(TTFont('Serif','/usr/share/fonts/truetype/liberation/LiberationSerif-Regular.ttf'))
pdfmetrics.registerFont(TTFont('SerifI','/usr/share/fonts/truetype/liberation/LiberationSerif-Italic.ttf'))
PAPER='#FCFBF7'; WHITE='#FFFFFF'; INK='#171B2E'; BLUE='#1230CD'; MUTED='#5B6070'; LINE='#D6D7D9'; PALE='#EDF1FF'; RED='#B12626'; ROSE='#FDECEC'; SHADE='#F4F2ED'

class D:
 def __init__(self,name,w=1440,h=900):
  self.name=name;self.w=w;self.h=h;self.c=canvas.Canvas(str(OUT/(name+'.pdf')),pagesize=(w,h));self.rect(0,0,w,h,PAPER)
 def rect(self,x,y,w,h,fill,stroke=None,r=0):
  self.c.setFillColor(HexColor(fill));self.c.setStrokeColor(HexColor(stroke or fill));self.c.setLineWidth(.7)
  if r:self.c.roundRect(x,self.h-y-h,w,h,r,stroke=bool(stroke),fill=1)
  else:self.c.rect(x,self.h-y-h,w,h,stroke=bool(stroke),fill=1)
 def line(self,x,y,w,color=LINE):self.rect(x,y,w,.8,color)
 def text(self,x,y,s,size=16,font='Sans',color=INK,width=None,leading=None):
  s=s.replace('▾','').replace('≡','Menu').replace('→','›')
  self.c.setFont(font,size);self.c.setFillColor(HexColor(color));leading=leading or size*1.4;lines=[]
  for p in s.split('\n'):
   if not width:lines.append(p);continue
   line=''
   for word in p.split():
    n=(line+' '+word).strip()
    if line and pdfmetrics.stringWidth(n,font,size)>width:lines.append(line);line=word
    else:line=n
   lines.append(line)
  for i,l in enumerate(lines):self.c.drawString(x,self.h-y-size-i*leading,l)
  return y+len(lines)*leading
 def button(self,x,y,label,w=None,primary=False,disabled=False):
  w=w or pdfmetrics.stringWidth(label,'Medium',14)+30
  self.rect(x,y,w,44,BLUE if primary else PAPER,BLUE if primary else LINE,5)
  self.text(x+15,y+11,label,14,'Medium','#FFFFFF' if primary else (MUTED if disabled else BLUE));return w
 def tag(self,x,y,label,color=BLUE,bg=PALE):
  w=pdfmetrics.stringWidth(label,'Medium',12)+18;self.rect(x,y,w,25,bg,r=3);self.text(x+9,y+5,label,12,'Medium',color)
 def shell(self,nav='Review',mobile=False):
  if mobile:
   self.text(18,15,'Content Desk',18,'Bold');self.button(278,4,'Content',94);self.line(0,52,self.w)
   self.rect(0,785,self.w,59,PAPER);self.line(0,785,self.w)
   for i,n in enumerate(['Today','Plan','Create','Review','Learn']):
    x=13+i*76;self.text(x,804,n,13,'Bold' if n==nav else 'Sans',BLUE if n==nav else MUTED)
    if n==nav:self.rect(x,833,45,3,BLUE)
  else:
   self.text(24,17,'Content Desk',24,'Bold');self.text(196,25,'Illustrative workspace',11,color=MUTED)
   for i,n in enumerate(['Today','Plan','Create','Review','Learn']):
    x=485+i*90;self.text(x,23,n,15,'Bold' if n==nav else 'Sans',BLUE if n==nav else INK)
    if n==nav:self.rect(x,56,48,3,BLUE)
   self.text(1035,24,'Vrooli · All work',14,'Medium');self.c.setStrokeColor(HexColor(INK));self.c.setLineWidth(1.3);self.c.lines([(1178,self.h-30,1183,self.h-35),(1183,self.h-35,1188,self.h-30)]);self.text(1220,24,'Content',14,'Medium',BLUE);self.text(1323,24,'Search',14,color=MUTED);self.line(0,60,self.w)
 def finish(self):
  footer='Proposed design 0.3 · synthetic Aster fixture · static specimen'
  if self.name.startswith(('M13','M16-article','M21')):footer='Proposed design 0.3 · existing private article fixture · static specimen'
  if self.name.startswith(('M10','M18')):footer='Proposed design 0.3 · conceptual diagram · no runtime validation'
  if self.w>500:self.text(24,self.h-23,footer,10,color=MUTED)
  self.c.save();subprocess.run(['pdftoppm','-png','-singlefile','-scale-to-x',str(self.w*2),'-scale-to-y',str(self.h*2),str(OUT/(self.name+'.pdf')),str(OUT/self.name)],check=True,stderr=subprocess.DEVNULL)

COPY=['The notes, comparison and release checklist now sit beside the decision they support.','We have not measured a speed improvement.','What context do you usually have to go looking for during a review?']

def review_desktop():
 d=D('M06-review-v04-desktop');d.shell()
 d.rect(0,61,245,813,SHADE);d.text(22,88,'Your decisions',18,'Bold');d.text(211,90,'2',16,color=MUTED)
 d.rect(0,133,245,137,PALE);d.rect(0,133,4,137,BLUE);d.text(21,150,'Aster · LinkedIn',16,'Bold');d.text(21,181,'Opening changed',14,'Medium');d.text(21,207,'Saved draft r2',13,color=MUTED);d.text(21,235,'Editorial decision requested',12,color=MUTED)
 d.line(20,285,203);d.text(21,305,'Existing dot article',16,'Bold');d.text(21,336,'Full article review',14);d.text(21,362,'Independent request',13,color=MUTED)
 d.text(21,775,'Waiting on others',14,'Medium');d.text(21,804,'No action required from you',12,color=MUTED)
 d.text(274,83,'Review',28,'Bold',BLUE);d.text(419,94,'Content (2)',15,'Bold',BLUE);d.text(557,94,'Publication',15);d.text(693,94,'Outcomes',15);d.rect(419,128,96,3,BLUE);d.line(245,131,1195)
 d.text(276,151,'Aster review story',19,'Bold');d.text(491,155,'LinkedIn · text only · account not selected',13,color=MUTED);d.tag(1188,150,'Synthetic revision')
 d.text(277,197,'Compare',14,'Bold',BLUE);d.text(383,197,'Read final',14);d.text(499,197,'Evidence · 3 sources',14);d.rect(277,230,64,3,BLUE);d.line(245,233,1195)
 d.text(278,254,'Previous saved wording · r1',14,'Medium');d.text(865,254,'Current saved draft · r2',14,'Medium');d.rect(837,251,1,384,LINE)
 d.rect(277,293,534,87,ROSE,r=4);d.text(294,307,'−',22,'Bold',RED);d.text(329,309,'We moved three review files into one view.',24,'Serif',width=455,leading=30)
 d.rect(864,293,548,87,PALE,r=4);d.text(881,307,'+',22,'Bold',BLUE);d.text(916,309,'Put the evidence beside the decision.',24,'Serif',width=466,leading=30)
 for x,w in [(282,502),(869,513)]:
  y=407
  for s in COPY:y=d.text(x,y,s,22,'Serif',width=w,leading=29)+25
 d.line(245,646,1195);d.text(276,667,'Evidence for this decision',15,'Bold');d.text(277,701,'Aster review story · source r3',14,'Medium');d.text(277,730,'Structure supported · Inspect source →',13,'Medium',BLUE)
 d.text(852,701,'Speed improvement unestablished',14,'Medium');d.text(852,730,'No timing study. This limit remains in r2.',13,color=MUTED)
 d.rect(245,775,1195,96,WHITE);d.rect(245,775,1195,2,BLUE);d.text(277,791,'Accept editorial wording for saved draft r2',15,'Bold');d.text(277,819,'No account or release permission is granted.',12,color=MUTED)
 d.button(899,796,'Accept wording',155,True);d.button(1066,796,'Request changes',161);d.text(1246,810,'Return to r1…',13,'Medium',BLUE)
 d.finish()

def review_phone():
 d=D('M06-review-v04-phone',390,844);d.shell(mobile=True)
 d.text(18,69,'‹ Decisions',13,'Medium',BLUE);d.text(281,69,'2 decisions',12,color=MUTED)
 d.text(18,104,'Aster · LinkedIn',23,'Bold');d.text(18,141,'Saved draft r2 · synthetic example',13,color=MUTED)
 d.text(18,180,'Read final',14,'Bold',BLUE);d.text(128,180,'Changes (1)',14);d.text(262,180,'Evidence (3)',14);d.rect(18,207,74,3,BLUE);d.line(0,210,390)
 y=d.text(24,234,'Put the evidence beside the decision.',25,'Serif',width=340,leading=31)+22
 for s in COPY:y=d.text(24,y,s,20,'Serif',width=339,leading=27)+22
 d.line(18,576,354);d.text(20,594,'Aster review story · r3 · Inspect →',13,'Medium',BLUE);d.text(20,619,'No timing study; speed gain unestablished.',12,color=MUTED);d.text(20,651,'View the change / restore r1  →',14,'Medium',BLUE)
 d.rect(0,687,390,98,WHITE);d.rect(0,687,390,2,BLUE);d.text(18,699,'Wording r2 only · no publication',12,'Medium');d.button(18,729,'Accept wording',167,True);d.button(199,729,'Request changes',174)
 d.finish()

def review_changes_phone():
 d=D('M06-review-v04-phone-changes',390,844);d.shell(mobile=True)
 d.text(18,71,'‹ Back to final reading',14,'Medium',BLUE);d.text(18,108,'One opening change',24,'Bold');d.text(18,146,'Aster · saved r2 · synthetic example',12,color=MUTED)
 d.text(18,196,'Previous saved wording · r1',13,'Medium');d.rect(18,230,354,112,ROSE,r=4);d.text(34,248,'We moved three review files into one view.',23,'Serif',width=320,leading=30)
 d.text(18,373,'Current saved draft · r2',13,'Medium');d.rect(18,407,354,106,PALE,r=4);d.text(34,425,'Put the evidence beside the decision.',23,'Serif',width=320,leading=30)
 d.text(18,544,'The remaining three paragraphs are unchanged.',14,width=346);d.text(18,594,'Restoring r1 creates a new draft revision. It keeps r2 in history and requires a new review.',14,width=346)
 d.button(18,693,'Preview restore as new draft',354,False);d.text(18,749,'Opening this view changes no wording.',12,color=MUTED);d.finish()

if __name__=='__main__':
 review_desktop();review_phone();review_changes_phone()
 records=[]
 for p in sorted(OUT.glob('M06-review-v04-*')):
  records.append({'path':str(p),'bytes':p.stat().st_size,'sha256':hashlib.sha256(p.read_bytes()).hexdigest()})
 (ROOT/'qa'/'exemplar-artifacts.json').write_text(json.dumps(records,indent=2))

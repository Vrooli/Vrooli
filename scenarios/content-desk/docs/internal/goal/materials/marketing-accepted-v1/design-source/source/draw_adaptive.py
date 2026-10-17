from draw_state_specimens import *

def adaptive():
 d=D('M21-adaptive-art-v02-desktop');d.text(27,26,'The image follows the work',32,'Bold',BLUE);d.text(27,83,'Same article and next action · three deliberate presentation states',15,color=MUTED)
 for i,(title,sub) in enumerate([('Actual artifact preview','Exported cover · source pin retained'),('Curated context art','Vector context · not evidence or post media'),('No image available','Deliberate typographic fallback')]):
  x=28+i*469;d.text(x,137,title,22,'Bold');d.text(x,177,sub,12,color=MUTED)
  if i==0:d.c.drawImage(str(ROOT/'references/article/original-pdf-image-1.png'),x,d.h-456,width=432,height=243,mask='auto')
  if i==1:
   d.rect(x,213,432,243,BLUE)
   for j in range(3):
    d.rect(x+49+j*107,253+j*12,93,142,PAPER);d.rect(x+65+j*107,280+j*12,60,5,'#D6D7D9');d.rect(x+65+j*107,301+j*12,45,5,'#D6D7D9');d.rect(x+65+j*107,325+j*12,62,5,'#D6D7D9')
   d.text(x+27,424,'Context art · inspect provenance ›',12,'Medium',WHITE)
  if i==2:
   d.rect(x,213,432,243,SHADE);d.text(x+26,245,'The next decision\nshould arrive\nbefore I ask',28,'Serif',width=382,leading=39);d.text(x+26,408,'Article section heading · source pin',12,color=MUTED)
  d.text(x,493,'Review the existing\ndot article',28,'Bold',BLUE,leading=34);d.text(x,592,'Owner review pending. Preserve the current prose and images.',16,'Serif',width=421,leading=24);d.button(x,684,'Open exact article',259,True)
 d.line(27,773,1380,BLUE);d.text(28,796,'One action, same scope. Permission changes remove the preview. No new generation is required.',16,'Medium');d.finish()

def dark_editor():
 dark='#121829';sur='#1C2439';ink='#F4F1E8';mut='#BAC3D5';accent='#A9BEFF';sel='#2B3B63'
 d=D('M22-dark-editor-v02-desktop');d.rect(0,0,1440,900,dark);d.text(25,22,'Content Desk',23,'Bold',ink);d.text(547,27,'Today',14,color=ink);d.text(637,27,'Plan',14,color=ink);d.text(727,27,'Create',14,'Bold',accent);d.text(817,27,'Review',14,color=ink);d.text(907,27,'Learn',14,color=ink);d.rect(727,57,52,3,accent);d.text(1100,27,'Vrooli · all work',13,color=mut);d.text(1290,27,'Content',14,'Medium',accent);d.line(0,65,1440,'#46516B')
 d.text(25,93,'Aster · LinkedIn draft r1',22,'Bold',ink);d.text(25,135,'Dark appearance · same saved source and optional proposal',13,color=mut);d.line(0,180,1440,'#46516B');d.rect(0,181,294,648,sur)
 d.text(23,204,'Sources (3)',18,'Bold',ink);d.text(23,252,'Aster review story · r3',15,'Medium',accent);d.text(23,295,'Comparison notes · r2',14,color=mut);d.text(23,338,'Release checklist · r1',14,color=mut);d.text(23,433,'No timing study.',17,'Bold','#FFB3B3');d.text(23,480,'Structural evidence is not a measured speed gain.',15,color=mut,width=247);d.text(23,591,'Inspect source ›',14,'Medium',accent)
 d.rect(327,224,1077,74,sel,r=4);d.text(348,240,'We moved three review files into one view.',29,'Serif',ink,width=1000)
 y=344
 for s in COPY:y=d.text(348,y,s,27,'Serif',ink,width=1020,leading=37)+27
 d.rect(327,702,1077,111,sur, '#46516B',5);d.text(347,719,'Optional opening proposal · based on r1',14,'Medium',accent);d.text(347,755,'Put the evidence beside the decision.',23,'Serif',ink);d.rect(1119,733,259,53,accent,r=5);d.text(1144,749,'Compare proposal',16,'Medium',dark)
 d.text(25,851,'Static appearance specimen · same authority and save semantics · contrast checked separately',12,color=mut);d.c.save();subprocess.run(['pdftoppm','-png','-singlefile','-scale-to-x','2880','-scale-to-y','1800',str(OUT/(d.name+'.pdf')),str(OUT/d.name)],check=True,stderr=subprocess.DEVNULL)

if __name__=='__main__':adaptive();dark_editor()

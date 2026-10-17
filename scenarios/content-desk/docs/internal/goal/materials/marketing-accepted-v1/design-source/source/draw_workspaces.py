"""Deterministic static product specimens. No app/API implementation."""
from draw_exemplar import *

def tabs(d,y,labels,active,x=26):
 for name in labels:
  d.text(x,y,name,14,'Bold' if name==active else 'Sans',BLUE if name==active else INK)
  width=pdfmetrics.stringWidth(name,'Sans',14)
  if name==active:d.rect(x,y+31,width,3,BLUE)
  x+=width+37

def phone_top(d,title,sub,nav):
 d.shell(nav,True);d.text(18,71,title,25,'Bold',BLUE);d.text(18,110,sub,12,color=MUTED)

def row(d,x,y,w,title,sub,tail=None):
 d.line(x,y,w);d.text(x+1,y+17,title,16,'Medium');d.text(x+1,y+46,sub,13,color=MUTED,width=w-20)
 if tail:d.text(x+w-130,y+20,tail,13,'Medium',BLUE)

def today():
 d=D('M01-today-v05-desktop');d.shell('Today')
 d.text(32,84,'Marketing,',64,'Bold',BLUE);d.text(32,157,'with something to say.',64,'Bold',BLUE)
 d.text(33,249,'Write a simple post  +',18,'Medium',BLUE);d.text(302,249,'Bring in an article  →',18,'SerifI',BLUE)
 # A content preview, not decorative proof. Strong art direction without generated scenery.
 d.rect(32,312,365,401,BLUE);d.text(54,337,'SOURCE STORY · ASTER · R3',12,'Medium',WHITE)
 d.text(53,391,'THREE\nFILES.\nONE VIEW.',48,'Bold',WHITE,leading=51)
 for i,label in enumerate(['Notes','Comparison','Checklist']):
  d.rect(57+i*96,584,85,54,PAPER);d.text(65+i*96,602,label,10,'Medium',INK)
 d.text(55,670,'Synthetic artifact preview · inspect →',11,'Medium',WHITE)
 d.text(435,319,'SUGGESTED NEXT · ABOUT 20 MIN',12,'Medium',RED)
 d.text(435,360,'Turn the review story\ninto a useful post.',38,'Bold',BLUE,leading=43)
 d.text(435,470,'The structural change is supported.\nA time-saving claim is not.',21,'Serif',leading=29)
 d.text(435,549,'LinkedIn draft r1 · source r3 · 3 sources',13,color=MUTED)
 d.button(435,586,'Open LinkedIn draft',238,True);d.text(696,600,'Why this next?',14,'Medium',BLUE)
 d.text(435,652,'AI proposes. Rules constrain. You decide.',14,color=MUTED)
 d.rect(1022,311,386,403,'#FFF6D8');d.rect(1022,311,2,403,BLUE)
 d.text(1048,334,'On your desk',28,'Bold',BLUE)
 row(d,1049,397,330,'Existing dot article','Independent review · owner decision',None);d.text(1049,474,'Review the article →',14,'Medium',BLUE)
 row(d,1049,520,330,'Personal X adaptation','After LinkedIn adaptation · 15 min',None)
 d.text(1049,626,'Vrooli subreddit',16,'Medium');d.text(1049,655,'Setup and moderation decision',13,color=MUTED)
 d.line(32,757,1376,BLUE);d.text(32,778,'One publication view',27,'Bold',BLUE);d.text(438,791,'3 intentions · 0 confirmed scheduled',16,'Medium');d.text(1125,791,'Open plan →',15,'Medium',BLUE)
 d.text(32,832,'35 min of suggested work · estimates only · Personal Planner owns reservations',13,color=MUTED);d.finish()
 p=D('M01-today-v05-phone',390,844);phone_top(p,'Something worth saying.','Vrooli · all work · illustrative', 'Today')
 p.text(18,151,'Write a post +',14,'Medium',BLUE);p.text(179,151,'Bring in an article ›',14,'Medium',BLUE)
 p.rect(18,198,354,145,BLUE);p.text(37,216,'ASTER · SOURCE STORY R3',11,'Medium',WHITE);p.text(37,247,'Three files.\nOne view.',33,'Bold',WHITE,leading=35);p.text(221,301,'Inspect ›',12,'Medium',WHITE)
 p.text(18,370,'LinkedIn adaptation',24,'Bold');p.text(18,408,'Draft r1 · about 20 min · 3 sources',13,color=MUTED);p.text(18,445,'Keep the structural claim. No timing study supports a speed gain.',17,'Serif',width=352)
 p.button(18,508,'Open LinkedIn draft',354,True);p.text(18,575,'Why this?  AI + rules · inspect ›',14,'Medium',BLUE)
 row(p,18,626,354,'Existing article: review','Independent request · owner decision')
 p.text(18,724,'3 intentions · 0 confirmed scheduled ›',13,'Medium',BLUE);p.finish()

def editor():
 d=D('M04-editor-v03-desktop');d.shell('Create')
 d.text(25,82,'‹ Content',14,'Medium',BLUE);d.text(155,82,'Aster review story / LinkedIn',17,'Medium');d.text(502,86,'Saved r1',13,color=MUTED);d.text(628,86,'Text only',13,color=MUTED);d.button(1174,72,'Preview',108,True);d.button(1294,72,'More',119)
 d.line(0,128,1440);d.rect(0,129,284,720,SHADE);tabs(d,149,['Sources','Assets','Help'],'Sources',22)
 d.text(22,211,'3 sources',17,'Bold');row(d,22,252,238,'Aster review story','r3 · selected source');row(d,22,345,238,'Comparison notes','r2 · structural context');row(d,22,438,238,'Release checklist','r1 · availability basis')
 d.rect(20,563,244,242,WHITE,LINE,4);d.tag(35,580,'Content fact');d.text(35,622,'Three review files now sit in one view.',18,'Serif',width=215);d.text(35,688,'No timing study.',14,'Bold',RED);d.text(35,721,'Do not claim a speed gain.',13,color=MUTED,width=205);d.text(35,775,'Inspect source ›',13,'Medium',BLUE)
 d.rect(1088,129,1,720,LINE);d.text(318,153,'Paragraph    B    I    Link    Quote',14);d.text(942,153,'Focus',14,'Medium',BLUE);d.line(284,189,804)
 d.text(323,224,'POST TEXT · OPENING SELECTED',11,'Medium',MUTED);d.rect(314,261,740,75,PALE,r=3);d.text(335,273,'We moved three review files into one view.',26,'Serif',width=690,leading=34)
 y=368
 for s in COPY:y=d.text(335,y,s,24,'Serif',width=676,leading=33)+28
 d.rect(314,695,740,135,WHITE,LINE,5);d.text(333,711,'Optional proposal · replace opening · based on r1',13,'Medium',BLUE);d.text(333,747,'Put the evidence beside the decision.',19,'Serif');d.button(333,780,'Compare',104);d.button(450,780,'Apply to draft',159,True);d.text(627,795,'Keep original',13,'Medium',BLUE);d.text(803,795,'Creates r2',12,color=MUTED)
 d.text(1114,152,'Preview',18,'Bold');d.text(1114,189,'LinkedIn · no account selected',12,color=MUTED,width=296)
 y=251
 for s in ['We moved three review files into one view.']+COPY:y=d.text(1114,y,s,18,'Serif',width=287,leading=25)+24
 d.text(25,850,'Saved to Content Desk · draft r1 · no approval or publication',12,color=MUTED);d.finish()
 p=D('M04-editor-v03-phone',390,844);p.shell('Create',True);p.text(18,69,'‹ Content',13,'Medium',BLUE);p.text(18,102,'Aster · LinkedIn',23,'Bold');p.text(18,141,'Saved r1 · synthetic fixture',12,color=MUTED);tabs(p,174,['Draft','Sources (3)','Help'],'Draft',18)
 p.rect(18,232,354,93,PALE,r=4);p.text(30,246,'We moved three review files into one view.',22,'Serif',width=327,leading=29)
 y=351
 for s in COPY:y=p.text(23,y,s,19,'Serif',width=342,leading=26)+19
 p.line(18,644,354);p.text(18,662,'1 proposal for selected opening',14,'Medium',BLUE);p.button(18,704,'Compare proposal',184);p.button(216,704,'Preview',156,True);p.text(18,761,'Apply happens only after the comparison.',11,color=MUTED);p.finish()

def plan():
 d=D('M05-plan-v04-desktop');d.shell('Plan');d.text(23,83,'Plan',30,'Bold',BLUE);tabs(d,92,['Strategy','Content mix','Publication','Work'],'Publication',161);d.button(1246,80,'Review plan',163,True);d.line(0,136,1440)
 d.text(24,157,'‹    ›    12–18 October 2026',19,'Medium');d.text(24,189,'America/New_York · illustrative week',12,color=MUTED);d.button(432,155,'Week',90,True);d.button(525,155,'Agenda',105);d.text(800,174,'3 proposed     0 queued     0 provider confirmed',14,'Medium');d.line(0,218,1440)
 d.rect(0,219,244,447,SHADE);d.text(22,241,'Without a date',17,'Bold');d.text(22,282,'Vrooli subreddit',16,'Medium');d.text(22,315,'Setup and moderation\ndecision needed',13,color=MUTED);d.button(22,372,'Open readiness',184)
 d.text(22,459,'Needs preparation',17,'Bold');d.text(22,501,'LinkedIn: review needed',13);d.text(22,538,'Silent demo: not recorded',13);d.text(22,604,'Add intention +',14,'Medium',BLUE)
 x0=337;cw=153
 for i,label in enumerate(['Mon 12','Tue 13','Wed 14','Thu 15','Fri 16','Sat 17','Sun 18']):
  x=x0+i*cw;d.text(x+12,239,label,13,'Medium');d.rect(x,280,1,373,LINE)
 for y,label in [(282,'Morning'),(407,'Afternoon'),(532,'Evening')]:d.line(244,y,1180);d.text(254,y+24,label,12,'Medium')
 for i,y,title,sub in [(3,420,'LinkedIn','Basis changed: r1→r2'),(4,294,'Personal X','Manual handoff'),(5,294,'YouTube','Route unverified')]:
  x=x0+i*cw+5;d.rect(x,y,143,104,PALE if i==3 else WHITE,BLUE if i==3 else LINE,5);d.text(x+10,y+12,title,14,'Bold');d.text(x+10,y+43,sub,12,width=120);d.text(x+10,y+79,'Proposed',11,'Medium',BLUE)
 d.rect(0,666,1440,182,WHITE);d.rect(0,666,1440,2,BLUE);d.text(25,688,'LinkedIn · proposed target',20,'Bold');d.text(25,725,'Thursday afternoon · America/New_York',15);d.text(25,758,'Draft is now r2. Intention preview still references r1.',14,'Medium',RED);d.text(25,794,'Compare before updating its basis. This does not schedule a release.',13,color=MUTED)
 d.button(999,732,'Compare r1 and r2',192,True);d.button(1204,732,'Keep target',199);d.text(999,798,'Account eligibility unverified',12,color=MUTED);d.finish()
 p=D('M05-plan-v04-phone',390,844);phone_top(p,'Publication agenda','Oct 12–18 · America/New_York · example','Plan');tabs(p,153,['Publication','Work','Actual'],'Publication',18)
 for y,date,title,sub in [(218,'THU 15 · AFTERNOON','LinkedIn','Proposed · basis changed r1→r2'),(351,'FRI 16 · MORNING','Personal X','Proposed · manual handoff'),(484,'SAT 17 · MORNING','Silent demo','Proposed · route unverified')]:
  p.line(18,y,354);p.text(18,y+15,date,11,'Medium',MUTED);p.text(18,y+44,title,19,'Bold');p.text(18,y+78,sub,12,'Medium',BLUE)
 p.text(18,639,'0 confirmed scheduled',16,'Medium');p.text(18,678,'180 min of work is provisional.\nNo Planner time is reserved.',13,color=MUTED);p.button(18,729,'Inspect selected target',354,True);p.finish()

def learn():
 d=D('M08-learning-v04-desktop');d.shell('Learn');d.text(28,87,'Learn',30,'Bold',BLUE);d.text(28,130,'Evidence and experiments',14,color=MUTED);d.button(1050,88,'Add observation',190,True);d.button(1250,88,'Import evidence',160)
 d.line(0,168,1440);d.text(28,191,'Scope  Aster review story',15,'Medium');d.text(429,191,'Period  All recorded evidence',14);d.text(840,191,'Sources  All permitted',14);tabs(d,241,['Observations','Experiments','Coverage'],'Observations',28);d.text(1030,243,'No publication outcomes recorded',12,color=MUTED)
 cols=[29,430,662,1007,1225]
 for x,t in zip(cols,['Evidence','Kind','Source','Observed','Status']):d.text(x,302,t,12,'Medium',MUTED)
 for i,values in enumerate([['Aster review structure','Content fact','Source r3','Fixture source','Available'],['LinkedIn publication','Publication outcome','Not provided','Unknown','Missing'],['Reader understanding','Audience observation','Not recorded','Unknown','Missing'],['Faster review','Outcome claim','No timing study','Unknown','Unsupported']]):
  y=341+i*62
  if i==0:d.rect(22,y-8,1386,61,PALE);d.rect(22,y-8,3,61,BLUE)
  for x,t in zip(cols,values):d.text(x,y+9,t,14,'Medium' if x==29 else 'Sans',RED if t=='Unsupported' else INK)
  d.line(22,y+53,1386)
 d.text(28,604,'Selected evidence · Aster review story r3',12,'Medium',BLUE);d.text(28,641,'Three files, one view',29,'Bold');d.text(28,688,'Establishes the structural change.\nDoes not establish speed or audience understanding.',18,'Serif',leading=27);d.text(28,776,'Inspect source ›',14,'Medium',BLUE)
 d.rect(714,599,1,250,LINE);d.text(749,604,'Linked question · proposed',12,'Medium',BLUE);d.text(749,641,'Does the explanation stand on its own?',26,'Bold',width=648);d.text(749,692,'Needs exact post/version, attributed feedback and an observation date.',15,width=608);d.text(749,753,'Not run · no results',14,'Medium');d.button(1118,787,'Save experiment idea',285);d.finish()
 p=D('M08-learning-v04-phone',390,844);phone_top(p,'Evidence notebook','Aster · illustrative workspace','Learn');p.button(18,151,'Add observation',354,True);tabs(p,216,['Evidence','Questions'],'Evidence',18)
 row(p,18,274,354,'Review structure ›','Content fact · source r3 · inspect');row(p,18,374,354,'LinkedIn publication ›','Outcome missing · no receipt');row(p,18,474,354,'Reader understanding ›','No audience observation');row(p,18,574,354,'Faster review ›','Unsupported · no timing study');p.line(18,674,354)
 p.text(18,698,'1 proposed question · not run',14,'Medium');p.text(18,740,'Open question and evidence needed ›',14,'Medium',BLUE);p.finish()

def content():
 d=D('M15-content-v01-desktop');d.shell('Create');d.text(28,85,'Content',30,'Bold',BLUE);d.text(28,131,'The same work, through every stage.',14,color=MUTED);d.button(1221,85,'Write a post +',185,True)
 d.rect(26,179,771,51,WHITE,LINE,5);d.text(43,194,'Search titles, opening words or permitted sources…',15,color=MUTED);d.button(818,182,'All states',146);d.button(977,182,'All formats',149);d.button(1140,182,'Projects',144)
 tabs(d,256,['Recent','All content','Archived'],'All content',28);d.text(29,312,'Title / opening',12,'Medium',MUTED);d.text(595,312,'Owning scope',12,'Medium',MUTED);d.text(827,312,'Editorial state',12,'Medium',MUTED);d.text(1128,312,'Publication',12,'Medium',MUTED)
 for i,vals in enumerate([('Evidence beside the decision','Aster · D1/r2','Awaiting review','1 proposed target'),('Evidence beside the decision','Founder writing · D9/r1','Private draft','No intention'),('How I’m trying to get more useful work…','Founder writing · article','Owner review pending','Not published'),('A useful design change','Aster · personal X variant','Draft','Manual route only')]):
  y=356+i*81
  if i==0:d.rect(22,y-7,1386,79,PALE);d.rect(22,y-7,3,79,BLUE)
  for x,t in zip([29,595,827,1128],vals):d.text(x,y+14,t,14,'Medium' if x==29 else 'Sans',width=530 if x==29 else 210)
  d.line(22,y+72,1386)
 d.line(26,721,1380,BLUE);d.text(29,746,'Selected: D1 · one native draft',18,'Bold');d.text(29,783,'Aster family F1 · primary campaign C1 · source S1/r3',14);d.text(722,746,'Relationships are not duplicate records',18,'Bold');d.text(722,783,'Revision, review, intention and receipt history stay linked.',14);d.text(1182,838,'Open record ›',14,'Medium',BLUE);d.finish()
 p=D('M15-content-v01-phone',390,844);phone_top(p,'All content','Vrooli · all states · synthetic records','Create');p.rect(18,155,354,47,WHITE,LINE,4);p.text(31,169,'Search: evidence beside',14)
 row(p,18,232,354,'Evidence beside the decision','Aster · D1/r2 · awaiting review');p.text(18,314,'1 proposed target · open record ›',13,'Medium',BLUE)
 row(p,18,368,354,'Evidence beside the decision','Founder writing · D9/r1 · private');p.text(18,450,'No publication intention · open ›',13,'Medium',BLUE)
 row(p,18,505,354,'Existing dot article','Founder writing · owner review');p.text(18,590,'Open exact article record ›',13,'Medium',BLUE)
 p.text(18,678,'Names can match. Native identities and owning scopes stay distinct.',14,width=349);p.text(18,743,'Include archived work ›',13,'Medium',BLUE);p.finish()

def simple():
 d=D('M12-simple-v05-desktop');d.shell('Create');d.text(26,84,'‹ Content',14,'Medium',BLUE);d.text(184,84,'Untitled thought',18,'Medium');d.text(28,133,'Unassigned · private draft',13,color=MUTED);d.button(1180,81,'Copy text',113);d.button(1303,81,'Preview',108,True);d.line(0,174,1440)
 d.text(245,225,'Start with what you actually noticed.',31,'SerifI',MUTED);d.text(246,316,'Putting the evidence beside the decision feels like\na useful way to reduce context hunting. I want to\nsee whether it helps the next review.',29,'Serif',leading=43)
 d.text(246,521,'No campaign or channel is required to keep a thought.',14,color=MUTED);d.line(244,604,944);d.text(246,629,'Add source',14,'Medium',BLUE);d.text(389,629,'Add media',14,'Medium',BLUE);d.text(531,629,'Ask for help',14,'Medium',BLUE)
 d.rect(244,723,944,90,PALE,r=4);d.text(263,741,'Prepared here · not yet saved to Content Desk',16,'Medium');d.text(263,773,'Saving to Content Desk is unavailable. Keep writing or copy your text.',13,color=MUTED);d.finish()
 p=D('M12-simple-v05-phone',390,844);p.shell('Create',True);p.text(18,73,'New thought',23,'Bold');p.text(18,111,'Unassigned · private · example',12,color=MUTED);p.text(18,161,'Putting the evidence beside the decision feels like a useful way to reduce context hunting.',23,'Serif',width=350,leading=31);p.text(18,304,'I want to see whether it helps the next review.',23,'Serif',width=350,leading=31)
 p.text(18,437,'Saved on this device only',13,'Medium',BLUE);p.text(18,466,'Saving to Content Desk unavailable.',12,color=MUTED);p.button(18,505,'Copy text',152);p.button(181,505,'Preview',191,True)
 p.rect(0,579,390,265,'#E5E6EA');p.text(20,594,'Illustrative keyboard-open state',11,'Medium',MUTED)
 for j,letters in enumerate(['Q W E R T Y U I O P','A S D F G H J K L','Z X C V B N M']):
  x=13+j*12
  for a in letters.split():p.rect(x,628+j*48,31,39,WHITE,r=4);p.text(x+10,638+j*48,a,16);x+=36
 p.rect(86,773,203,43,WHITE,r=4);p.text(160,786,'space',13,color=MUTED);p.finish()

if __name__=='__main__':
 for f in [today,editor,plan,learn,content,simple]:f()

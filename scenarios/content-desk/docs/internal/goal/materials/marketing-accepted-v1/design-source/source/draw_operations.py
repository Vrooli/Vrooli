"""Static supporting journeys and state specimens in the selected C language."""
from draw_workspaces import *

def strategy():
 d=D('M02-strategy-v03-desktop');d.shell('Plan');d.text(26,85,'Choose the useful work',30,'Bold',BLUE);tabs(d,144,['Strategy','Content mix','Publication','Work'],'Content mix',26);d.text(817,93,'Aster · developer audience · synthetic plan',13,color=MUTED)
 d.rect(26,205,1386,80,SHADE);d.text(45,220,'Audience job',12,'Medium',MUTED);d.text(45,246,'Understand one real design change without reconstructing three files.',18,'Serif');d.text(1013,240,'Source r3 · no speed evidence',13,'Medium',RED)
 d.text(28,322,'Coordinated explanation',27,'Bold');d.text(28,362,'180 min estimated · 7 distinct work items',14,color=MUTED);d.text(760,322,'A smaller useful outcome',27,'Bold');d.text(760,362,'90 min estimated · 3 work items',14,color=MUTED);d.rect(713,320,1,407,LINE)
 for i,(label,minutes) in enumerate([('Proof / demo / destination check',30),('Core explanation',45),('LinkedIn adaptation',20),('Personal X adaptation',15),('Silent demo',40),('Exact review',15),('Response observation',15)]):
  y=405+i*40;d.text(29,y,label,15);d.text(615,y,str(minutes)+' min',14,'Medium');d.line(28,y+31,645)
 for i,(label,minutes) in enumerate([('Proof / destination check',30),('Core explanation',45),('Exact review',15)]):
  y=405+i*49;d.text(762,y,label,16);d.text(1308,y,str(minutes)+' min',14,'Medium');d.line(759,y+39,640)
 d.text(761,593,'Deferred deliberately',16,'Bold');d.text(761,628,'Video, channel adaptations and their targets.\nNo invented spare capacity. No subreddit date.',15,width=635)
 d.line(27,740,1380,BLUE);d.text(29,758,'Why these?',16,'Bold');d.text(29,788,'AI proposes reader fit; rules check capacity, sources and channel limits. You select the set.',14);d.button(1028,769,'Choose 90-minute plan',377,True);d.text(29,835,'Accepting content work does not reserve Planner time or authorize publication.',12,color=MUTED);d.finish()
 p=D('M02-strategy-v03-phone',390,844);phone_top(p,'Choose useful work','Aster · plan proposal · synthetic','Plan');p.text(18,160,'One real design change.\nNo measured speed claim.',22,'Serif',width=351,leading=29)
 p.line(18,253,354);p.text(18,273,'180-minute plan',24,'Bold');p.text(18,314,'Core story + two adaptations + silent demo, with proof, review and observation.',15,width=351);p.text(18,396,'7 distinct work items · inspect ›',13,'Medium',BLUE)
 p.line(18,445,354);p.text(18,465,'90-minute plan',24,'Bold');p.text(18,506,'30 min proof + 45 min story + 15 min review.',15,width=351);p.text(18,570,'Defer adaptations, video and targets.',13,color=MUTED);p.button(18,641,'Review 90-minute plan',354,True);p.text(18,713,'No time reserved. No release authorized.',13,color=MUTED);p.finish()

def evidence():
 d=D('M03-evidence-v03-desktop');d.shell('Create');d.text(27,84,'Aster · inspect one claim',25,'Bold');d.text(27,124,'Unsaved failure branch based on r1 · synthetic',13,color=MUTED);d.line(0,171,1440)
 d.text(29,198,'The wording',14,'Medium',MUTED);d.rect(28,239,648,125,ROSE,r=4);d.text(49,264,'Our review system is 40% faster.',29,'Serif',width=603);d.tag(49,322,'Needs evidence',RED,ROSE)
 d.text(29,410,'A narrower supported alternative',20,'Bold');d.text(29,458,'We moved three review files into one view.',28,'Serif',width=613,leading=37);d.text(29,562,'Preserve the useful fact; remove the unsupported number.',15,width=622)
 d.button(29,658,'Preview wording repair',266,True);d.button(310,658,'Add evidence',197);d.text(29,739,'The proposal will not replace text until you apply it.',13,color=MUTED)
 d.rect(720,172,1,642,LINE);d.text(750,199,'Source: Aster review story',22,'Bold');d.text(750,240,'Revision 3 · permitted private use · fixture',13,color=MUTED);d.rect(750,292,658,222,SHADE);d.text(773,318,'“The notes, comparison and release checklist are now together beside the decision.”',28,'Serif',width=607,leading=38);d.text(750,555,'What this supports',14,'Bold');d.text(750,586,'A structural change to the review surface.',17,'Serif');d.text(750,644,'What it does not establish',14,'Bold',RED);d.text(750,675,'Speed, audience understanding or customer outcomes.',17,'Serif',width=635);d.text(750,755,'Open exact source passage ›',14,'Medium',BLUE);d.finish()
 p=D('M03-evidence-v03-phone',390,844);phone_top(p,'Check this sentence','Aster · unsaved failure branch · example','Create');p.rect(18,154,354,110,ROSE,r=4);p.text(31,175,'Our review system is 40% faster.',23,'Serif',width=328,leading=30)
 p.text(18,298,'Aster review story · source r3',15,'Bold');p.text(18,337,'“The notes, comparison and release checklist are now together beside the decision.”',20,'Serif',width=350,leading=27)
 p.text(18,472,'No timing study supports 40%.',14,'Medium',RED);p.text(18,521,'Suggested repair',14,'Bold');p.text(18,556,'We moved three review files into one view.',22,'Serif',width=351,leading=29);p.button(18,659,'Preview wording repair',354,True);p.text(18,725,'Inspect source · add evidence ›',14,'Medium',BLUE);p.finish()

def outcome():
 d=D('M07-outcome-v04-desktop');d.shell('Review');d.text(28,83,'Outcome needs verification',30,'Bold',BLUE);d.tag(1068,92,'Future connected-route example');tabs(d,149,['Content','Publication','Outcomes'],'Outcomes',28)
 d.rect(27,215,1382,83,ROSE);d.text(47,233,'The request may have reached the provider.',23,'Bold',RED);d.text(47,267,'Do not create another attempt until the original is reconciled.',14)
 d.text(28,340,'LinkedIn · selected personal identity',24,'Bold');d.text(28,383,'Attempt A7 · release package RP2 · draft D1/r2',14,color=MUTED);d.text(28,430,'Last confirmed events',16,'Bold')
 for i,(a,b) in enumerate([('01  Package validated','Exact wording, identity and permission scope'),('02  Dispatch started','Provider response was not received'),('03  Outcome unknown','No post ID or publication receipt established')]):
  y=474+i*85;d.text(30,y,a,17,'Medium');d.text(30,y+30,b,14,color=MUTED);d.line(28,y+65,654)
 d.rect(718,331,1,406,LINE);d.text(749,342,'What you can do',23,'Bold');d.button(750,400,'Check supported status',274,True);d.text(750,466,'Uses the existing attempt identity.',14,color=MUTED);d.button(750,518,'Record owner observation',298);d.text(750,581,'A reported URL remains owner-reported until independently verified.',15,width=611);d.text(750,663,'Retry unavailable',16,'Bold',RED);d.text(750,699,'Requires proven non-admission or safe idempotent replay.',14,width=610)
 d.line(27,781,1382,BLUE);d.text(28,802,'Responsible operator: owner · unresolved item remains on Today',15,'Medium');d.finish()
 p=D('M07-outcome-v04-phone',390,844);phone_top(p,'Outcome unknown','Future connected-route example','Review');p.text(18,158,'LinkedIn · personal identity',20,'Bold');p.text(18,197,'A7 · RP2 · draft r2',13,color=MUTED);p.rect(18,244,354,113,ROSE);p.text(33,260,'The provider may have received the request. No receipt is established.',18,'Serif',width=324,leading=25)
 p.text(18,396,'Do not submit again yet.',20,'Bold',RED);p.button(18,451,'Check supported status',354,True);p.button(18,513,'Record owner observation',354);p.text(18,582,'Owner observations retain their evidence label; they are not provider receipts.',14,width=350);p.text(18,666,'Retry requires reconciliation.',14,'Medium');p.text(18,721,'Owner action · remains on Today',13,color=MUTED);p.finish()

def recovery():
 d=D('M09-recovery-v04-desktop');d.shell('Create');d.text(27,83,'Keep both versions. Choose the result.',29,'Bold',BLUE);d.text(27,130,'Aster draft · synthetic concurrent-edit case · nothing overwritten',13,color=MUTED);d.line(0,181,1440)
 boxes=[(28,'Base · saved r2','Put the evidence beside the decision.','What context do you usually have to go looking for during a review?'),(507,'Server · newer r3','Put the evidence beside the decision.','Which source is usually missing from a handoff?'),(985,'This device · based on r2','Keep the evidence beside the decision.','What context do you usually have to go looking for during a review?')]
 for x,title,opening,ending in boxes:
  d.text(x,209,title,17,'Bold');d.rect(x,254,425,177,PALE if x==985 else SHADE,r=4);d.text(x+19,276,opening,25,'Serif',width=386,leading=33);d.text(x,478,ending,23,'Serif',width=418,leading=31)
 d.line(28,623,1382,BLUE);d.text(29,649,'Composed result preview',22,'Bold');d.text(29,692,'Use this device’s opening and the server’s ending. Unchanged paragraphs retain r2’s text and evidence.',16,width=1040);d.text(29,757,'Both branches remain in history. Saving creates a new revision and reopens affected review.',14,color=MUTED);d.button(1050,711,'Preview new revision',359,True);d.text(1177,788,'Keep both and return later',13,'Medium',BLUE);d.finish()
 p=D('M09-recovery-v04-phone',390,844);phone_top(p,'Two versions to keep','Aster · conflict example · no overwrite','Create');p.text(18,160,'Server r3 changed the ending.',18,'Bold');p.text(18,204,'“Which source is usually missing from a handoff?”',21,'Serif',width=350,leading=28);p.line(18,312,354)
 p.text(18,333,'This device changed the opening.',18,'Bold',width=350);p.text(18,386,'“Keep the evidence beside the decision.”',21,'Serif',width=350,leading=28);p.text(18,489,'Base r2 and both branches are retained.',14,width=350);p.button(18,555,'Preview combined wording',354,True);p.button(18,619,'Keep both; return later',354);p.text(18,697,'No save occurs until you inspect the composed result.',13,color=MUTED,width=347);p.finish()

def contributor():
 d=D('M11-contributor-v03-desktop');d.text(28,25,'Vrooli · contribution packet',23,'Bold');d.text(941,30,'Fictional Alex · proposed recipient view',13,color=MUTED);d.line(0,70,1440)
 d.text(29,104,'Your words. Your choice.',40,'Bold',BLUE);d.text(29,171,'Optional feedback on one review workflow. Criticism is welcome.',19,'Serif');d.line(28,219,1380)
 d.text(29,248,'The request',21,'Bold');d.text(29,294,'Try the described review and tell us what was confusing. You can decline or ask a question. There is no required praise, comment or social post.',20,'Serif',width=629,leading=29)
 d.text(29,440,'Permitted packet',16,'Bold');d.text(29,477,'One task brief · selected source excerpt · no private project history',15,width=625);d.text(29,562,'Your draft response',16,'Bold');d.rect(27,602,641,150,WHITE,LINE,4);d.text(45,622,'I understood the decision, but had to look twice to find the source.',23,'Serif',width=595,leading=31)
 d.rect(716,241,1,514,LINE);d.text(752,248,'Allow this exact quote?',23,'Bold');d.text(752,293,'Relationship shown: friend who tried the example.',15,width=636)
 d.rect(750,349,657,163,SHADE,r=4);d.text(770,370,'Included use',13,'Medium',MUTED);d.text(770,405,'Quote in the founder article only.',23,'Serif');d.text(770,459,'No personal-account post or broader reuse.',14)
 d.text(752,555,'You can edit this wording or decline. A later destination or wording change needs your new permission.',16,width=627);d.text(752,668,'No credentials or whole-project account needed.',14,color=MUTED)
 d.line(27,788,1380,BLUE);d.button(750,806,'Submit words and this permission',392,True);d.button(1156,806,'Decline',129);d.text(29,820,'Prepared design fixture · no message has been sent',13,color=MUTED);d.finish()
 p=D('M11-contributor-v03-phone',390,844);p.text(18,22,'Vrooli · contribution',18,'Bold');p.text(18,65,'Fictional Alex · recipient example',12,color=MUTED);p.line(0,97,390);p.text(18,119,'Your words.\nYour choice.',31,'Bold',BLUE,leading=34)
 p.text(18,222,'Optional feedback. Criticism is welcome; no social post is requested.',16,width=350);p.text(18,316,'Your response',14,'Bold');p.rect(18,351,354,133,WHITE,LINE,4);p.text(32,368,'I understood the decision, but had to look twice to find the source.',20,'Serif',width=325,leading=26)
 p.text(18,524,'Permit this exact quote',18,'Bold');p.text(18,561,'Founder article only. Relationship: friend who tried the example. No broader reuse.',14,width=350);p.button(18,652,'Submit words + this permission',354,True);p.button(18,714,'Decline or ask a question',354);p.finish()

def intake():
 d=D('M13-intake-v06-desktop');d.shell('Create');d.text(28,84,'Bring in the work already written',29,'Bold',BLUE);d.text(28,132,'1 Source     2 Inspect extraction     3 Map destination     4 Reopen result',14,'Medium');d.line(0,181,1440)
 d.text(28,208,'Selected source · pinned Google Doc',13,'Medium',MUTED);d.text(28,250,'How I’m trying to get more useful\nwork from my ChatGPT dot',34,'Serif',leading=41);d.text(28,365,'Existing article + 3 visual placements + companion variants',15,color=MUTED)
 for i,(a,b) in enumerate([('Article text and links','Ready to save'),('Image 1 · cover','Candidate media'),('Image 2 · figure','Unavailable in this simulated branch'),('Image 3 · figure','Candidate media'),('X / LinkedIn companion','Separate variant mappings')]):
  y=418+i*66;d.line(27,y,767);d.text(28,y+18,a,15,'Medium');d.text(385,y+20,b,13,color=RED if i==2 else MUTED)
 d.rect(834,205,1,544,LINE);d.text(865,212,'Destination and action',24,'Bold')
 for i,(a,b) in enumerate([('Project','Vrooli / Founder writing'),('Campaign','None'),('Producer','External assistant / Drive'),('Method','Founder article guide 0.6'),('Requested effect','Save as unapproved draft')]):
  y=271+i*78;d.text(866,y,a,12,'Medium',MUTED);d.text(866,y+28,b,17,'Medium')
 d.text(866,700,'The original stays available. No regeneration.',14,'Medium',BLUE);d.line(27,797,1381,BLUE);d.button(28,816,'Save text only',225,True);d.button(271,816,'Resolve image 2',221);d.text(867,826,'Prepared here · not saved yet',14,color=MUTED);d.finish()
 p=D('M13-intake-v06-phone',390,844);phone_top(p,'Bring in this article','Simulated partial-media branch','Create');p.text(18,160,'How I’m trying to get more useful work from my ChatGPT dot',25,'Serif',width=350,leading=31);p.text(18,288,'Source pinned · original retained',13,color=MUTED)
 row(p,18,334,354,'Text and source links','Prepared');row(p,18,429,354,'3 image placements','Image 2 unavailable in this example');row(p,18,535,354,'Founder writing · no campaign','Method 0.6 · save as draft only');p.button(18,655,'Save text only',354,True);p.text(18,719,'Not saved to Content Desk yet.',13,'Medium',MUTED);p.finish()

def readiness():
 d=D('M14-readiness-v04-desktop');d.shell('Plan');d.text(28,86,'Choose the actual destination',30,'Bold',BLUE);d.text(28,132,'Account readiness is action-specific. A complete profile is not publication permission.',14,color=MUTED);d.line(0,182,1440)
 d.rect(0,183,372,662,SHADE);d.text(25,210,'Illustrative identities',16,'Bold')
 for i,(a,b) in enumerate([('LinkedIn · personal','Selected · ownership unverified'),('LinkedIn · organization','Separate role and audience'),('Personal X','Link-only · private drafting'),('Vrooli subreddit','Not created · no destination')]):
  y=262+i*122
  if i==0:d.rect(16,y-10,339,106,PALE)
  d.text(28,y,a,18,'Bold');d.text(28,y+40,b,13,color=MUTED,width=311)
 d.text(409,211,'LinkedIn · personal identity',25,'Bold');d.text(410,251,'Requested action: publish one selected text-only package',14,color=MUTED)
 for i,(a,b) in enumerate([('Public identity / ownership','Unverified · inspect supplied profile'),('Your actor role','Not established'),('Strategy and post type','Activation must be confirmed'),('Publication route','Not qualified'),('Current eligibility / limits','Unverified'),('Release permission','Not granted')]):
  y=304+i*75;d.line(409,y,991);d.text(410,y+19,a,16,'Medium');d.text(923,y+21,b,13,color=MUTED,width=460)
 d.button(410,787,'Review identity evidence',263,True);d.button(689,787,'See setup decisions',229);d.text(944,803,'Private drafting stays available.',13,color=MUTED);d.finish()
 p=D('M14-readiness-v04-phone',390,844);phone_top(p,'Account readiness','LinkedIn · personal · example','Plan');p.text(18,158,'One identity, one action',23,'Bold');p.text(18,199,'Text-only publication of selected package',13,color=MUTED)
 for y,a,b in [(246,'Ownership','Unverified'),(337,'Your role','Not established'),(428,'Type and strategy','Activation needs confirmation'),(519,'Route / limits','Not qualified / unverified')]:row(p,18,y,354,a,b)
 p.text(18,635,'Release permission not granted',16,'Bold',RED);p.button(18,676,'Review identity evidence',354,True);p.text(18,741,'Return to private draft ›',14,'Medium',BLUE);p.finish()

def work():
 d=D('M17-work-v02-desktop');d.shell('Plan');d.text(28,85,'The plan is saved. Time is still unreserved.',29,'Bold',BLUE);d.text(28,132,'Synthetic partial-admission result · resume only the unresolved operation',13,color=MUTED);tabs(d,172,['Strategy','Content mix','Publication','Work'],'Work',28)
 for i,(title,status,body) in enumerate([('Content plan CP1','Saved','Accepted 90-minute plan; native content work identities retained.'),('Earlier intentions','Retained · at risk','Existing targets need new dates because adaptations are deferred.'),('Planner allocation','Not admitted','Request failed before confirmed admission. Source work remains saved.')]):
  x=28+i*468;d.rect(x,244,445,171,PALE if i<2 else ROSE,r=4);d.text(x+20,264,title,22,'Bold');d.text(x+20,306,status,15,'Medium',BLUE if i<2 else RED);d.text(x+20,347,body,14,width=403)
 d.text(29,462,'Source-owned work',22,'Bold');d.text(913,470,'Personal Planner allocations',16,'Bold')
 for i,(title,time,sub) in enumerate([('W1 · proof check',30,'Shared by D1 and D2; one work identity'),('W2 · core story',45,'Source constraint revision 2'),('W3 · exact review',15,'Depends on W1 and W2')]):
  y=514+i*81;d.line(28,y,1380);d.text(29,y+17,title,17,'Medium');d.text(530,y+19,str(time)+' min',15);d.text(671,y+19,sub,13,color=MUTED);d.text(1190,y+19,'Unreserved',14,'Medium',RED)
 d.button(1037,795,'Review allocation request',369,True);d.text(29,809,'Do not recreate CP1, W1–W3 or the saved intentions.',15,'Medium');d.finish()
 p=D('M17-work-v02-phone',390,844);phone_top(p,'Saved plan, open step','Partial-admission example','Plan');row(p,18,157,354,'Content plan CP1','Saved · 90-minute scope');row(p,18,268,354,'Earlier intentions','Retained · at risk · review dates');row(p,18,379,354,'Planner allocation','Not admitted · work remains unreserved')
 p.text(18,509,'Resume this step only.',23,'Bold');p.text(18,551,'W1 proof 30 + W2 story 45 + W3 review 15 = 90 minutes. No duplicate work records.',15,width=350);p.button(18,663,'Review allocation request',354,True);p.text(18,731,'Native owner receipts stay separate.',12,color=MUTED);p.finish()

if __name__=='__main__':
 for f in [strategy,evidence,outcome,recovery,contributor,intake,readiness,work]:f()

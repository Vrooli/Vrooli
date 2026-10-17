from draw_adaptive import *

def assistance():
 d=D('M23-assistance-v01-desktop');d.shell('Create');d.text(28,85,'Help with this opening',30,'Bold',BLUE);d.text(28,133,'Aster · draft r1 · selected first paragraph · proposed request',13,color=MUTED);d.line(0,181,1440)
 d.text(28,213,'What the helper would see',22,'Bold');d.rect(27,260,661,132,PALE,r=4);d.text(47,283,'We moved three review files into one view.',25,'Serif',width=613,leading=34);d.text(47,350,'Replace opening only · base r1',13,'Medium',BLUE)
 for i,(a,b) in enumerate([('Sources','Aster story r3; comparison r2; checklist r1'),('Method','Adopted post-type guidance and adapter pin'),('Excluded','Other projects, private chat, contributor packets')]):
  y=432+i*88;d.text(29,y,a,14,'Bold');d.text(29,y+32,b,15,width=649)
 d.rect(719,207,1,533,LINE);d.text(750,212,'Choose a route deliberately',24,'Bold')
 d.rect(749,261,659,139,SHADE,r=4);d.text(770,281,'Keep the good work you have',23,'Bold');d.text(770,330,'Continue manually or bring in existing dot-authored material. No regeneration is requested.',16,width=618)
 d.text(750,441,'Optional agent generation',21,'Bold');d.text(750,485,'Runner / provider: not selected',15);d.text(750,524,'Inference location: unknown',15);d.text(750,563,'Cost basis / enforceable limit: unknown',15,'Medium',RED);d.text(750,612,'Starting stays unavailable until the route, context and required resource permission are established.',15,width=631)
 d.rect(750,690,281,45,SHADE,LINE,5);d.text(765,702,'Start unavailable',14,'Medium',MUTED);d.text(1052,703,'Review route details ›',14,'Medium',BLUE)
 d.line(27,788,1380,BLUE);d.text(28,812,'Your draft remains editable if assistance is unavailable.',15);d.button(1138,804,'Keep writing',271,True);d.finish()
 p=D('M23-assistance-v01-phone',390,844);phone_top(p,'Help with this opening','Aster · draft r1 · proposed request','Create');p.text(18,160,'Selected paragraph only',18,'Bold');p.text(18,199,'“We moved three review files into one view.”',21,'Serif',width=350,leading=28);p.text(18,288,'3 selected sources · inspect context ›',13,'Medium',BLUE);p.line(18,338,354)
 p.text(18,359,'Keep existing work',20,'Bold');p.text(18,399,'Continue writing or bring in the dot’s prepared article. No regeneration is needed.',15,width=350);p.line(18,482,354);p.text(18,504,'Optional agent route',20,'Bold');p.text(18,546,'Provider, inference location and cost basis are not selected.',14,width=350);p.text(18,609,'Start unavailable until the route and required permission are established.',14,color=RED,width=350);p.button(18,709,'Keep writing',354,True);p.finish()

def release():
 d=D('M24-release-v03-desktop');d.shell('Review');d.text(28,86,'One exact release',30,'Bold',BLUE);d.tag(929,94,'Future qualified-route fixture · no real account action');tabs(d,149,['Content','Publication','Outcomes'],'Publication',28)
 d.text(29,215,'Release package RP2 · draft r2',16,'Bold');y=261
 for s in ['Put the evidence beside the decision.']+COPY:y=d.text(31,y,s,25,'Serif',width=633,leading=33)+25
 d.text(29,673,'Text-only · exact reviewed body',14,'Medium');d.text(29,715,'Inspect sources, package and operator approval ›',13,'Medium',BLUE);d.rect(720,213,1,566,LINE)
 d.text(750,214,'Destination and route',24,'Bold');d.text(750,260,'Owner · personal LinkedIn identity',17,'Medium');d.text(750,298,'Identity and publisher role verified in this fixture',13,color=MUTED);d.text(750,347,'Route: connected dispatch',16,'Medium',BLUE);d.text(750,386,'Other qualified routes: manual / provider schedule',13,color=MUTED)
 d.text(750,437,'Allowed window',14,'Bold');d.text(750,471,'Thu 15 Oct 2026 · 14:00–17:00 · America/New_York',18,'Medium');d.text(750,512,'No later dispatch without a new decision.',13,color=MUTED)
 d.text(750,563,'Human operator approval: RP2 · fixture receipt',14,'Medium');d.text(750,601,'Claims / media / eligibility: validated for this fixture',13);d.text(750,639,'Permission scope: one release, this identity and window',13,width=632)
 d.line(28,783,1380,BLUE);d.text(29,809,'Admission returns an actual queue/provider receipt. It is not publication evidence.',14);d.button(1070,802,'Authorize one release',337,True);d.finish()
 p=D('M24-release-v03-phone',390,844);phone_top(p,'One exact release','Future qualified-route fixture','Review');p.text(18,157,'RP2 · saved wording r2',20,'Bold');p.text(18,197,'Read complete final text + sources ›',14,'Medium',BLUE);p.line(18,245,354)
 p.text(18,268,'Personal LinkedIn identity',19,'Bold');p.text(18,309,'Verified actor/role in this example',13,color=MUTED);p.text(18,359,'Connected dispatch',18,'Medium',BLUE);p.text(18,406,'Thu 15 Oct 2026 · 14:00–17:00\nAmerica/New_York',18,'Medium',leading=27);p.text(18,486,'Human operator approval for RP2 is recorded in this synthetic fixture.',14,width=347);p.text(18,567,'Scope: one release, this identity and window. A queue receipt is not a published post.',14,width=347)
 p.button(18,685,'Authorize one release',354,True);p.text(18,747,'Static specimen · no real account action',11,color=MUTED);p.finish()

def outcome_forms():
 p=D('M25-manual-outcome-v02-phone',390,844);p.shell('Review',True);p.text(18,71,'Manual handoff',25,'Bold',BLUE);p.text(18,112,'Example · MP2 · personal LinkedIn',12,color=MUTED);p.text(18,159,'Copied does not mean posted.',22,'Bold',width=347)
 p.text(18,226,'What happened at the destination?',17,'Medium',width=350)
 for y,label in [(279,'Posted'),(342,'Scheduled in provider'),(405,'Not posted'),(468,'Still unsure')]:p.button(18,y,label,354,primary=label=='Still unsure')
 p.text(18,546,'A URL and actual time can be recorded after you know. Any external text changes stay visible.',15,width=349);p.text(18,640,'Still unsure keeps this outcome open; it creates no publication receipt.',14,width=347);p.button(18,723,'Save this observation',354,True);p.finish()
 p=D('M25-observation-v02-phone',390,844);p.shell('Learn',True);p.text(18,73,'Add an observation',25,'Bold',BLUE);p.text(18,114,'Illustrative private note · not owner feedback',11,color=MUTED)
 p.text(18,160,'Context',13,'Bold');p.rect(18,192,354,51,WHITE,LINE,4);p.text(31,207,'Aster · draft r2 · not published',14)
 p.text(18,278,'Kind: personal reflection',15,'Medium');p.text(18,325,'What did you actually notice?',14,'Bold');p.rect(18,359,354,158,WHITE,LINE,4);p.text(31,378,'The opening is clearer to me. I have not shown it to readers.',21,'Serif',width=327,leading=29)
 p.text(18,551,'Basis: manual entry',14);p.text(18,585,'Observed time: not entered',14);p.text(18,620,'Incomplete note; no audience outcome is inferred.',14,width=350);p.text(18,682,'Private to this project',13,'Medium',BLUE);p.button(18,721,'Save private observation',354,True);p.finish()

if __name__=='__main__':assistance();release();outcome_forms()

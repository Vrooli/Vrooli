from draw_operations import *

def ownership():
 d=D('M10-ownership-v05',720,940);d.text(28,25,'One experience, distinct owners',29,'Bold',BLUE);d.text(28,80,'Conceptual ownership · commands and receipts to each owner',13,color=MUTED)
 d.rect(28,132,664,90,BLUE);d.text(49,149,'Marketing experience in Content Desk',24,'Bold',WHITE);d.text(49,190,'Today · Plan · Create · Review · Learn · Content',15,color=WHITE)
 blocks=[('Content Desk','Drafts, revisions, campaigns, claims, review and content plans'),('Channel Manager','Identity, eligibility, release admission, attempts and outcomes'),('Personal Planner','Accepted human allocations and global availability'),('Asset Studio / Offer Desk','Media release/provenance and commercial/release truth'),('Optional authoring','Existing dot-authored material or permitted Prose/agent work')]
 for i,(a,b) in enumerate(blocks):
  y=247+i*110;d.rect(36,y,656,96,SHADE,LINE,4);d.text(53,y+12,a,22,'Bold');d.text(53,y+49,b,16,width=617,leading=22)
  d.c.setStrokeColor(HexColor(BLUE));d.c.setLineWidth(1);d.c.lines([(16,d.h-211,16,d.h-y-47),(16,d.h-y-47,35,d.h-y-47),(29,d.h-y-43,35,d.h-y-47),(29,d.h-y-51,35,d.h-y-47)])
 d.text(29,817,'New connections require native source work.',20,'Bold',BLUE);d.text(29,855,'The view keeps each owner’s records and permissions authoritative.',16,width=657,leading=23);d.finish()

def sequence():
 d=D('M18-journey-v02',720,1030);d.text(28,25,'One draft through the whole loop',27,'Bold',BLUE);d.text(28,71,'Synthetic state trace · no commands executed',13,color=MUTED)
 stages=[('1  Source and draft','S1/r3 supports D1/r1. Existing article or simple post can enter here.','Content Desk'),('2  Apply a scoped edit','New saved draft r2. Unapproved. Source limit stays in the text.','Author / Content Desk'),('3  Make the review decision','Exact wording/package assessed. Native human approval remains explicit.','Reviewer / human operator'),('4  Plan and allocate separately','Save intention I1. Request work allocation W1. Keep separate receipts.','Content Desk / Planner'),('5  Admit an exact release','Bind package, identity and allowed window. Return actual admitted state.','Channel Manager'),('6  Reconcile and learn','Receipt or unknown result → attributed observation → next question.','Channel Manager / Content Desk')]
 for i,(a,b,c) in enumerate(stages):
  y=126+i*135;d.rect(28,y,664,113,PALE if i in (2,4) else SHADE,r=4);d.text(46,y+12,a,20,'Bold');d.text(46,y+46,b,16,width=624,leading=22);d.text(46,y+89,c,12,'Medium',BLUE)
  if i<5:d.text(340,y+111,'↓',18,'Serif',BLUE)
 d.text(28,953,'Unknown result: hold, inspect the original operation, then recover.',16,'Medium',RED);d.finish()

def formats():
 d=D('M16-article-v03-desktop');d.shell('Create');d.text(26,83,'‹ Content',14,'Medium',BLUE);d.text(179,82,'Founder article · illustrated draft',19,'Medium');d.button(1240,77,'Read final',170,True);d.line(0,137,1440)
 d.rect(0,138,239,708,SHADE);d.text(22,163,'Outline',17,'Bold')
 for i,s in enumerate(['The article moment','What needed me','A clearer handoff','What remains unproven']):d.text(22,212+i*51,s,14,width=204)
 d.text(22,490,'Images (3)',17,'Bold');d.text(22,531,'Order and alt text',13,'Medium',BLUE);d.text(22,574,'Source / method 0.6',13,'Medium',BLUE)
 d.text(285,173,'PRIVATE REVIEW ARTICLE · EXISTING SOURCE',11,'Medium',MUTED);d.text(285,214,'How I’m trying to get more\nuseful work from my ChatGPT dot',36,'Serif',leading=42)
 d.c.drawImage(str(ROOT/'references/article/original-pdf-image-1.png'),360,d.h-661,width=608,height=342,mask='auto')
 d.text(286,681,'Conceptual illustration from the article. Not a live product screenshot.',12,color=MUTED);d.text(286,717,'My new ChatGPT dot was working on an article. Other proposals had been prepared too. Yet I still had to ask why only one piece of work was moving and which decisions needed me.',20,'Serif',width=759,leading=27)
 d.rect(1088,138,1,708,LINE);d.text(1111,167,'Selected figure',18,'Bold');d.text(1111,216,'Candidate media',14,'Medium',RED);d.text(1111,258,'Not released for publication.',14,width=287);d.text(1111,338,'Alt text',14,'Bold');d.rect(1109,372,301,148,WHITE,LINE,4);d.text(1123,387,'A conceptual coral monocle character presents undecided work cards beside a checked manuscript.',14,width=271);d.text(1111,558,'Caption · Replace · Move',14,'Medium',BLUE);d.text(1111,609,'Provenance / rights ›',14,'Medium',BLUE);d.text(26,853,'Illustrative intake state with actual article excerpt and its exported cover image.',12,color=MUTED);d.finish()
 d=D('M16-thread-v01-desktop');d.shell('Create');d.text(26,82,'Aster · private thread adaptation',24,'Bold');d.text(26,126,'X remains link-only · account limits unverified · no dispatch controls',13,color=MUTED);d.line(0,175,1440)
 for i,(title,copy) in enumerate([('Item T1','A small design change: put the evidence beside the decision.'),('Item T2','The notes, comparison and release checklist now sit in one view. We have not measured a speed improvement.'),('Item T3','Which piece of context do you normally have to go looking for?')]):
  y=211+i*191;d.text(28,y,title,16,'Bold');d.rect(154,y-8,911,149,WHITE,LINE,4);d.text(176,y+11,copy,25,'Serif',width=865,leading=33);d.text(176,y+105,'Move up / down · split · inspect source',12,'Medium',BLUE)
 d.rect(1110,188,1,597,LINE);d.text(1140,215,'Format readiness',19,'Bold');d.text(1140,262,'Per-item count is calculated from the actual account/format contract.',15,width=265);d.text(1140,377,'Limit unknown',16,'Bold',RED);d.text(1140,416,'Private editing remains useful. Qualification is required before release.',15,width=264);d.text(26,824,'Ordered items retain native identities; a partial result cannot be represented by one root URL.',14);d.finish()
 d=D('M16-storyboard-v01-desktop');d.shell('Create');d.text(26,84,'A silent demonstration',28,'Bold',BLUE);d.text(26,129,'Aster · storyboard candidate · no recording or runtime proof',13,color=MUTED);d.button(1150,83,'Preview sequence',261);d.line(0,181,1440)
 for x,t in zip([28,122,473,823,1150],['Scene','Shot / action','On-screen explanation','Evidence','Media']):d.text(x,208,t,13,'Medium',MUTED)
 rows=[('01','Show the three source files','“Notes, comparison, checklist”','Selected source r3','Not recorded'),('02','Show the proposed single view','“Context beside the decision”','Design specimen only','Not recorded'),('03','Name the limit and question','“No timing study yet”','No performance proof','Caption planned')]
 for i,values in enumerate(rows):
  y=261+i*156;d.line(28,y,1380)
  for x,w,t in zip([28,122,473,823,1150],[75,311,310,284,240],values):d.text(x,y+26,t,18,'Medium' if x==28 else 'Sans',width=w,leading=26)
 d.line(28,729,1380,BLUE);d.text(28,754,'Reordering preserves scene ID, caption, source and media relationships.',17,'Medium');d.text(28,800,'A rendered video is a separate Asset Studio artifact. A storyboard does not prove the app works.',14,color=MUTED);d.finish()

def review_forms():
 p=D('M19-request-changes-v01-phone',390,844);p.shell('Review',True);p.text(18,74,'‹ Review r2',14,'Medium',BLUE);p.text(18,112,'Request a change',26,'Bold');p.text(18,151,'Synthetic editorial request · Aster',12,color=MUTED);p.text(18,201,'Anchor: opening paragraph',14,'Bold');p.rect(18,238,354,119,PALE,r=4);p.text(32,256,'Put the evidence beside the decision.',22,'Serif',width=328,leading=29)
 p.text(18,395,'What needs changing?',14,'Bold');p.rect(18,432,354,142,WHITE,LINE,4);p.text(32,449,'Make it clear this describes the review structure, not a measured improvement.',18,'Serif',width=328,leading=25);p.text(18,614,'Returns this exact package to its author. It does not edit the text or publish.',14,width=350);p.button(18,708,'Send change request',354,True);p.finish()
 p=D('M19-source-detail-v01-phone',390,844);p.shell('Review',True);p.text(18,73,'‹ Back to wording r2',14,'Medium',BLUE);p.text(18,115,'Aster review story',24,'Bold');p.text(18,154,'Source r3 · synthetic private fixture',12,color=MUTED);p.text(18,209,'Selected source passage',14,'Bold');p.rect(18,251,354,189,SHADE,r=4);p.text(32,270,'The notes, comparison and release checklist are now together beside the decision.',23,'Serif',width=326,leading=31);p.text(18,480,'Supports',14,'Bold');p.text(18,514,'Structural change only.',18,'Serif');p.text(18,565,'Does not establish',14,'Bold',RED);p.text(18,599,'Faster work, reader understanding or customer outcomes.',18,'Serif',width=348);p.button(18,708,'Return to exact wording',354,True);p.finish()

def fallback_states():
 d=D('M20-states-v01-desktop');d.text(27,25,'Distinct quiet, loading and failure states',29,'Bold',BLUE);d.text(27,76,'Focused state specimens · same visual language · no live status claims',13,color=MUTED)
 states=[(28,126,'Nothing needs you now','All known decisions are clear. Capture an idea or leave the workspace.','Write a thought +',PAPER),(747,126,'No matching content','Your filter found no matching records. Existing work has not disappeared.','Clear filters',PAPER),(28,472,'Loading this record','D1 is selected. Its last observed revision is r2; current state is not yet known.','Keep editing local copy',SHADE),(747,472,'Save outcome unknown','Your local wording is retained. Check the original save before trying again.','Check save status',ROSE)]
 for x,y,title,body,action,bg in states:
  d.rect(x,y,663,296,bg,LINE,5);d.text(x+23,y+25,title,27,'Bold',BLUE if bg!=ROSE else RED);d.text(x+23,y+89,body,19,'Serif',width=613,leading=28);d.button(x+23,y+211,action,360,True)
 d.text(28,833,'No-art views preserve typography, hierarchy and useful action. Unknown is neither empty nor failed.',14,color=MUTED);d.finish()

if __name__=='__main__':
 for f in [ownership,sequence,formats,review_forms,fallback_states]:f()

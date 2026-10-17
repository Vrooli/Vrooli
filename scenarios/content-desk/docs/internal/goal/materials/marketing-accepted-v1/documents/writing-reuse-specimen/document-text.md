# When a save message refers to an older edit

- Document ID: 1EN3yoimBSjVmpI--nyFGIUvS2_h6Hvbo4swTndR42gg
- Revision ID: AHj4eMT3S07GJ-dE1i3G2co9GHMXC9WMfqv2WJfYd9xEDWGasayd2kIYOHNi7dOR9VedrWUGZfhedX0hkkGDGC472Wi18zXk7jbyZnR2UQ
- Selected tab: all
- Protected controls: 0
- Opaque controls: 0
- Authoritative dropdowns: 0

Protected-control annotations are preservation instructions. Do not insert their displayed placeholder text to recreate a native control.

## Tab 1 (t.0)

[P00001 | 1:45 | TITLE]
When a save message refers to an older edit

[P00002 | 45:242 | NORMAL_TEXT]
For someone editing a launch announcement, “Saved” is permission to move on. Beside newer text, that message can imply the newer text is safe, even when the reply only confirms an earlier version.

[P00003 | 242:540 | NORMAL_TEXT]
Consider a hypothetical editor. You write “Launch on Friday” and click Save. That sends revision A. Before the reply returns, you change the sentence to “Launch on Monday,” revision B. Then A’s success message arrives. The editor now shows Monday and “Saved.” The confirmed copy still says Friday.

[P00004 | 540:614 | NORMAL_TEXT]
The save succeeded for A. The status now makes a broader promise about B.

[P00005 | 614:890 | NORMAL_TEXT]
For the engineer building the editor, the important choice happens when that reply arrives: may it clear the unsaved-change state? Only if it confirms the content currently shown. If the draft has changed since the request, preserve it and keep the newer edit marked unsaved.

[P00006 | 890:892 | NORMAL_TEXT]
[INLINE_OBJECT i.0]

[P00007 | 892:1060 | NORMAL_TEXT]
Figure 1. Intended behavior. Editing continues while the save is in flight. A’s reply leaves B unsaved. This conceptual example does not depict or test a live product.

[P00008 | 1060:1355 | NORMAL_TEXT]
In this example, B has not been sent, so the truthful result is “Earlier edit saved. Newer edit not saved yet.” The user can continue editing or explicitly save B. Identifying the saved snapshot gives the interface enough information to make that distinction; a success response alone does not.

[P00009 | 1355:1357 | NORMAL_TEXT]
⟦EMPTY PARAGRAPH⟧

[P00010 | 1357:1370 | HEADING_1]
Article plan

[P00011 | 1370:1395 | HEADING_2]
Reader and useful result

[P00012 | 1395:1733 | NORMAL_TEXT]
Technical founders and product engineers who own editor behavior. The timely need is the moment a misleading save status could lose someone’s work; it is an evergreen problem, without a claimed product launch or trend. The reader should identify the exact snapshot a success message confirms and know when an unsaved indicator may clear.

[P00013 | 1733:1748 | HEADING_2]
Angle decision

[P00014 | 1748:2211 | NORMAL_TEXT]
The title promises to explain how a truthful save reply can mislead beside a newer edit. The opening delivers that promise with Friday becoming Monday, then gives the rule: preserve the newer draft and its unsaved state. This specific timing problem is more useful than a generic story about unreliable software. A retry tutorial would miss the case because A did save. Out-of-order writes, autosave or a real product postmortem would require different evidence.

[P00015 | 2211:2228 | HEADING_2]
Article sequence

[P00016 | 2228:2292 | NORMAL_TEXT | LIST id=kix.iys8dqcp8vgm level=0]
Open with Friday becoming Monday while a save reply is pending.

[P00017 | 2292:2383 | NORMAL_TEXT | LIST id=kix.iys8dqcp8vgm level=0]
Use the two-lane sequence to separate the visible draft, the saved snapshot and the reply.

[P00018 | 2383:2534 | NORMAL_TEXT | LIST id=kix.iys8dqcp8vgm level=0]
Develop the decision rule with the actual A/B text. Preserve B; retain its unsaved state. A later user-initiated save of B needs its own confirmation.

[P00019 | 2534:2661 | NORMAL_TEXT | LIST id=kix.iys8dqcp8vgm level=0]
State the boundary: truthful status alone does not solve competing writes, multi-user conflicts or durable-storage guarantees.

[P00020 | 2661:2784 | NORMAL_TEXT | LIST id=kix.iys8dqcp8vgm level=0]
End with a review question: if the user edits again before success returns, which version does your interface say is safe?

[P00021 | 2784:2801 | HEADING_2]
Scope and visual

[P00022 | 2801:3263 | NORMAL_TEXT]
This specimen supplies only the opening and one explanatory figure. Its visual explains editing during a pending request and the identity of the returned acknowledgment, using labels as well as color. It is a hypothetical interface, so product branding and current-event imagery would add a false association. The R1 revision clarifies audience relevance and title-promise fit in this plan; it does not change or rerun the original independent version 0.3 test.


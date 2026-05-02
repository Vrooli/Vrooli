## Practice focus: Idea Workshop

Collaborative methodology for transforming vague intuitions into coherent, structured ideas through iterative conversation. The goal is to help the user externalize and shape a raw concept without disrupting their creative flow, then hand off the refined idea to swarm-manager's idea agent pipeline for formal hardening.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

---

### **1. When to Use This Methodology**

Use Idea Workshop when:
- The user is brainstorming or workshopping a new idea
- The user starts describing a vague concept, problem, or "what if"
- The user says things like "I've been thinking about...", "what if we built...", "I want to spitball..."
- The user wants to create a new scenario but hasn't fully articulated it yet

**Do NOT use** for:
- Refining an already-articulated idea (use swarm-manager's clarify/suggest/enhance directly)
- Implementation planning for a defined task (use plan-skill-discovery)
- Debugging or fixing existing code
- Quick one-off questions

**Proactive trigger:** This skill should be loaded automatically when the conversation pattern suggests brainstorming. Do not wait for the user to request it.

---

### **2. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                         IDEA WORKSHOP PROCESS                                │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────┐     ┌───────────┐     ┌──────────┐     ┌──────────┐          │
│   │ LISTEN  │ ──▶ │ SYNTHESIZE│ ──▶ │ SHARPEN  │ ──▶ │ CONVERGE │          │
│   │& ABSORB │     │& PLAY BACK│     │(questions)│     │          │          │
│   └─────────┘     └───────────┘     └────┬─────┘     └────┬─────┘          │
│                                          │                  │                │
│                         ┌────────────────┘                  │                │
│                         ▼                                   │                │
│                   User answers &                            │                │
│                   adds new info?                            │                │
│                    │      │                                  │                │
│               YES  │      │  NO (stable)                    │                │
│                    ▼      ▼                                  │                │
│              Return to    Proceed to ────────────────────────┘                │
│              SYNTHESIZE   CONVERGE                                            │
│                                          │                                   │
│                                          ▼                                   │
│                                    ┌───────────┐                             │
│                                    │  HANDOFF  │                             │
│                                    │(swarm-mgr)│                             │
│                                    └───────────┘                             │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Listen & Absorb**

**Entry criteria:** User has started describing an idea, problem, or concept.

**Actions:**
1. **Let the user talk.** Do not interrupt with questions or structure prematurely.
2. **Acknowledge understanding** — Confirm you're following without redirecting.
3. **If the user asks "are you following?" or similar** — Play back the core tension or insight in 1-2 sentences to prove comprehension, then let them continue.
4. **Note signals for later:**
   - Hedging language ("not sure", "maybe", "or something") — marks decision points to revisit
   - Emotional emphasis — marks what the user cares about most
   - Contradictions — marks areas needing resolution
   - Unstated assumptions — marks things to validate

**Exit criteria:**
- [ ] User has finished their initial dump (they ask a question, pause for input, or explicitly hand off)
- [ ] You have a mental model of what they're describing, even if incomplete

**Anti-pattern:** Jumping to solutions or structure too early. The user is in a creative state — protect that.

---

### **Phase 2: Synthesize & Play Back**

**Entry criteria:** User has finished an initial explanation (or a round of new information).

**Actions:**
1. **Restructure what they said** into a coherent model:
   - Identify the core data types / entities
   - Identify the key interactions / workflows
   - Identify the user experience goals
   - Identify the problem being solved
2. **Play it back concisely** — Use structured format (bullet points, short sections) but keep it conversational.
3. **Name the unnamed** — If the user described concepts without naming them, propose names. Naming makes things concrete and discussable.
4. **Highlight what's clear vs. what's fuzzy** — Don't ask questions yet, just flag areas that need refinement.

**Exit criteria:**
- [ ] User confirms the synthesis is accurate ("yes", "exactly", "right")
- [ ] OR user corrects misunderstandings (loop back to absorb the correction, then re-synthesize)

**Anti-pattern:** Adding your own ideas unprompted. This phase is about reflecting, not creating.

---

### **Phase 3: Sharpen**

**Entry criteria:** User has confirmed the synthesis is on track.

**Actions:**
1. **Ask targeted questions** that resolve the fuzzy areas identified in Phase 2.
2. **Prioritize questions by impact:**
   - Questions that change the fundamental shape of the idea (ask first)
   - Questions about specific UX/behavior details (ask second)
   - Questions about technical implementation (ask last or skip — these are for the plan phase)
3. **Batch questions** — Ask 3-7 at a time, numbered. Don't drip-feed one at a time.
4. **Include "or" options when possible** — "Should X be A or B?" is easier to answer than "What should X be?"
5. **After answers, check for new information that changes the model** — If yes, return to Phase 2 for a mini re-synthesis before asking more questions.

**Decision table — question priority:**

| Question Type | Priority | Example |
|---|---|---|
| Changes fundamental shape | Ask immediately | "Is this one app or multiple?" |
| Resolves ambiguity in data model | Ask early | "Can thoughts span sessions?" |
| Clarifies UX behavior | Ask mid | "Filter: hide or dim?" |
| Technical implementation | Defer or skip | "Which canvas library?" |
| Nice-to-have details | Skip | "What shade of blue?" |

**Exit criteria:**
- [ ] No more high-impact questions remain
- [ ] The idea is coherent enough that you could explain it to someone else without the user present
- [ ] User is satisfied with the level of detail

**Anti-pattern:** Asking implementation questions during brainstorming. Keep the focus on what and why, not how.

---

### **Phase 4: Converge**

**Entry criteria:** Idea is stable — no more major open questions.

**Actions:**
1. **Produce a final consolidated summary** incorporating all rounds of discussion.
2. **Present it to the user** for final review.
3. **Ask explicitly:** "Does this capture everything? Anything to add or change before we submit?"
4. **Make final adjustments** based on feedback.

**Exit criteria:**
- [ ] User approves the consolidated summary
- [ ] Summary is detailed enough to serve as a swarm-manager backlog item description

---

### **Phase 5: Handoff**

**Entry criteria:** User has approved the final summary.

**Actions:**
1. **Create the swarm-manager backlog idea:**
   ```bash
   swarm-manager backlog create --data '{
     "name": "<kebab-case-name>",
     "title": "<Title>",
     "kind": "idea",
     "priority": <1-5>,
     "tags": [<relevant-tags>],
     "description": "<full consolidated summary>"
   }'
   ```
2. **Kick off the clarify phase** to begin formal hardening:
   ```bash
   swarm-manager backlog research --kind idea --name <name> --data '{"mode":"clarify"}'
   ```
3. **Inform the user** of next steps: the idea agent will generate clarifying questions, then suggest improvements, then produce an enhanced specification.

**Exit criteria:**
- [ ] Backlog item created
- [ ] Clarify phase initiated
- [ ] User knows what happens next

---

### **3. Convergence Patterns**

#### **Iteration Detection**

Use this to decide whether to loop back or move forward:

| Signal | Action |
|---|---|
| User says "actually..." or "wait, I changed my mind" | Return to Phase 2 (re-synthesize) |
| User answers questions and adds new info | Return to Phase 2 (mini re-synthesis) |
| User answers questions cleanly, no new info | Proceed to next batch or Phase 4 |
| User says "yeah that's it" or "I think that covers it" | Proceed to Phase 4 (converge) |
| User seems fatigued or wants to wrap up | Proceed to Phase 4 with what you have |

#### **Depth vs. Breadth**

| Situation | Approach |
|---|---|
| User has a clear vision, just needs help articulating | Fewer questions, more synthesis |
| User has a vague feeling, exploring possibilities | More questions, more options presented |
| User keeps going deeper on one aspect | Follow them deep, capture details, broaden later |
| User is jumping between topics | Synthesize frequently to maintain coherence |

---

### **4. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| **Premature structuring** | Kills creative flow, user thinks shallowly | Let them dump first, structure after |
| **Asking implementation questions** | Wrong level of abstraction for brainstorming | Focus on what and why, defer how |
| **Adding unsolicited features** | Derails the user's vision with your ideas | Reflect their ideas, don't inject yours |
| **One question at a time** | Slow, frustrating, breaks user's flow | Batch 3-7 questions |
| **Not playing back** | User doesn't know if you understood | Always synthesize before questioning |
| **Skipping the handoff** | Ideas stay in chat, never become actionable | Always submit to swarm-manager |
| **Over-polishing before handoff** | Diminishing returns; swarm-manager does this better | Get to "coherent enough" then hand off |

---

### **5. Boundaries**

This methodology covers the **pre-ideation phase**: shaping raw thoughts into a coherent idea description.

**Does NOT cover:**
- **Formal idea refinement** — That's swarm-manager's clarify/suggest/enhance pipeline (this skill hands off to it)
- **Implementation planning** — Use plan-skill-discovery after the idea is refined
- **Technical architecture decisions** — Those come during planning, not brainstorming
- **Evaluating whether an idea is worth pursuing** — That's a product/business decision, not a methodology question

---

### **6. Output Expectations**

When applying Idea Workshop, you **must** produce:

1. **At least one synthesis** played back to the user for confirmation
2. **Targeted questions** that resolve ambiguity (batched, prioritized)
3. **A consolidated summary** approved by the user
4. **A swarm-manager backlog item** created via CLI with the full description
5. **Clarify phase initiated** on the backlog item

You **should** also:
- Name unnamed concepts to make them concrete
- Flag areas where the user hedged for explicit resolution
- Keep the conversation natural and low-friction — the user should feel like they're thinking out loud, not filling out a form

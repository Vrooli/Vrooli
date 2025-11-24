## 🎨 **Phase: UX Improvement**

During this loop, prioritize **user experience quality** across all interfaces in this scenario.
Do **not** break functionality or regress tests; all changes must maintain or improve completeness.

Focus on producing a **professional, polished, friction-free user experience**, guided by the following principles:

### **1. Clarity & Understanding**

* Ensure all UI elements have **clear affordances and signifiers** indicating how they should be used.
* Add **help buttons, tooltips, hint text, or short explainer components** wherever concepts, parameters, or interactions may be confusing.
* Favor **concise text**, intuitive labels, and consistent terminology across the UI.

### **2. Layout & Information Hierarchy**

* Organize content so the user’s attention naturally follows an **F-shaped visual scanning pattern**:

  * primary actions located where attention begins
  * secondary elements placed where they’re discoverable but unobtrusive
* Reduce unnecessary cognitive load by grouping related controls and trimming redundant steps.

### **3. Reduce Friction**

* Minimize clicks, scrolling, typing, and context switching needed to complete **common user journeys**.
* Identify and smooth out high-friction areas:

  * excessive dialogs or confirmations
  * unclear navigation paths
  * repetitive manual inputs
* Make the most frequent workflows **as direct, obvious, and streamlined as possible**.

### **4. Professional Interaction Design**

* Prefer **icons (Lucide)** over emojis for UI communication.
* Use consistent spacing, alignment, padding, and component sizing.
* Avoid clutter; prioritize readability and clean composition.
* Improve empty states, loading states, and error states so the experience remains smooth and emotionally neutral.
* Ensure first-time users immediately understand what the scenario does, how to use it, and what the next step is. Enhance empty states with clarity, guidance, and calls to action.
* Where appropriate, add small but meaningful micro-interactions (hover/focus states, loading indicators, optimistic UI cues) that enhance feedback without clutter

### **5. Responsiveness & Device Adaptation**

* Ensure the UI behaves well on **mobile**, tablet, and desktop:

  * elements resize or stack appropriately
  * touch targets are comfortably sized
  * no overflow or clipping
* Do not introduce layout assumptions (e.g., sidebars, footers) unless they already fit naturally with the existing design.

### **6. Maintain Scenario Constraints**

* Do **not** change the scenario’s core workflows, APIs, or business logic.
* Do **not** introduce new features unrelated to UX improvement.
* Ensure all changes remain within the scenario’s architectural goals, PRD, and test-driven requirements.

### **7. Output Expectations**

You may update:

* components
* styles
* labels
* UI flows
* help/tooltip systems
* interaction logic
* responsive styles
* empty and loading states

You **must**:

* keep the scenario fully functional
* avoid regressions
* increase user clarity and ease-of-use
* meaningfully improve UX quality without gaming metrics

Focus the loop on delivering **practical, targeted UX improvements** that make the scenario genuinely easier and more enjoyable to use.

**When choosing what to modify, consider the scenario’s most common user journeys and optimize for the fewest steps, lowest friction, and clearest navigation.**

**Avoid superficial UX changes that alter appearance without improving actual usability. Only make changes that meaningfully reduce friction or increase understanding.**

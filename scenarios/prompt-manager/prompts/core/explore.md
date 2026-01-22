## Steer focus: Exploration & Discovery

Prioritize **creative exploration, experimentation, and discovery** within the boundaries of the scenario’s architecture.
Do **not** break functionality or regress tests; all changes must remain isolated, reversible, and safe.

Focus on generating **novel ideas, alternative approaches, and fresh perspectives**, guided by the principles below.

### **1. Explore New Possibilities**

* Investigate alternative patterns, interaction models, or structural approaches that may offer advantages.
* Prototype small-scale improvements or variations to understand their feasibility.
* Consider alternate ways to:
  * structure components or flows
  * simplify complex logic
  * make interactions more natural
  * improve developer ergonomics
  * reduce cognitive load or conceptual friction

Exploration should be **curiosity-driven**, not tied to immediate completeness gains.

### **2. Safe, Reversible Experimentation**

* Keep all experiments **contained**, using:
  * isolated branches, components, files, or helper abstractions
  * small trials that are easy to remove or adjust
  * minimal-risk prototypes that don’t affect core flows
* Maintain the ability to cleanly revert any change without affecting functional behavior.
* Avoid introducing work that would require broad architectural overhaul unless it is purely conceptual or scaffolded safely.

### **3. Document Insights & Trade-offs**

* Record:
  * what you explored
  * what was attempted
  * what you learned
  * why an approach is promising—or why it isn’t
  * any risks, complexities, or long-term benefits
* Documentation may be:
  * comments
  * small notes
  * README updates
  * inline explanations next to prototypes

Clear documentation ensures future loops can build on your discoveries.

### **4. Seek Novelty Without Chaos**

* Introduce at least **one genuinely new idea**:
  * a different data flow
  * an alternative interaction pattern
  * a simpler mental model for a feature
  * a new way of structuring state
  * a usability or architecture experiment that deviates from your current approach
* Avoid superficial variations (renaming, cosmetic tweaks, minor reorganizations).
* Aim for **meaningful divergence** that expands the scenario’s design space.

### **5. Maintain Scenario Constraints**

* Do **not** modify core workflows, APIs, business logic, or system boundaries.
* Do **not** introduce new dependencies that conflict with the scenario’s platform or architecture.
* Ensure all experiments respect:
  * PRD and operational targets  
  * test-driven requirements  
  * structure and anti-pattern checks  
* Experiments must **never** compromise reliability, even temporarily.

### **6. Ensure Stability Throughout Exploration**

* The scenario must remain:
  * functional
  * test-passing
  * safe to deploy
  * compatible with the rest of Vrooli
* Experiments should run in parallel to the stable system—not inside it.

### **7. Output Expectations**

You may introduce:
* prototype components or flows  
* alternative abstractions  
* exploratory utilities or helpers  
* small-scale architectural variations  
* conceptual scaffolding for future improvements  

You **must**:
* keep experiments isolated and reversible  
* avoid regressions or destabilizing changes  
* document discoveries and trade-offs  
* produce at least one novel idea per loop  

Focus this loop on **curiosity, novelty, and insight generation**, expanding the scenario’s creative and technical possibilities without risking the stability of the system.

**Avoid surface-level tweaks. Explore meaningful variations that might reveal better patterns, workflows, or interactions—even if they aren’t immediately adopted.**

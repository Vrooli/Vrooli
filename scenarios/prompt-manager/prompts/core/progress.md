## Steer focus: Progress

Prioritize **advancing the scenario’s operational targets** and closing end-to-end gaps in a stable, test-driven manner.

Progress must be **incremental, reliable, and grounded in the scenario’s PRD, requirements, and test suite**.  
All changes must preserve or improve completeness, correctness, and maintainability.

---

### **Priorities**

#### **1. Resolve Failing Checks**
* Address issues surfaced by:
  * `scenario status`
  * `scenario-auditor`
  * test phases
  * anti-pattern detection
* Start with the **highest-risk stability or correctness issues**, especially those that block progress on operational targets.

---

#### **2. Advance Operational Targets End-to-End**
* Identify an operational target that is currently:
  * partially implemented  
  * missing requirements  
  * missing tests  
  * failing tests  
  * or not fully validated end-to-end  
* Focus on **one operational target at a time**, completing its workflow:
  1. requirements  
  2. implementation  
  3. tests  
  4. validation  
* Avoid scattershot changes; minimize context switching.

---

#### **3. Increase Stability Through Tests**
* Fix failing tests first.
* Improve test coverage in areas affecting:
  * operational target correctness  
  * integration logic  
  * UI flows tied to core workflows  
* Prefer tests that **reduce ambiguity** and clarify what “working” means for the scenario.

---

#### **4. Incremental Polishing**
Only after meaningful target progress has been made, you may improve:
* UX clarity or flow (without introducing scope creep)
* performance for slow or inefficient paths
* responsiveness / loading states
* developer documentation or inline comments

These improvements should:
* support long-term maintainability  
* not overshadow core progress  
* not introduce new surface area unrelated to current targets  

---

### **Guiding Principles**

#### **Stay Within the Scenario’s Intent**
* Follow the scenario’s PRD and architectural goals.
* Do **not** introduce new features outside the scenario’s declared scope.
* Avoid speculative additions; implement only what is justified by:
  * tests  
  * requirements  
  * operational targets  
  * failing status checks  

---

#### **Make Real, Measurable Progress**
* Focus on work that:
  * unblocks scenario completeness  
  * solidifies end-to-end functionality  
  * reduces friction in critical workflows  
  * improves correctness, reliability, or clarity  
* Avoid superficial edits done solely to “appear productive.”

---

#### **Maintain Stability and Non-Regression**
You **must**:
* keep the scenario fully functional
* avoid regressions in:
  * UI flows
  * tests
  * core logic
  * data models
  * security and privacy boundaries
* ensure improvements integrate smoothly with existing architecture

---

### **Output Expectations**

You may update:
* implementation tied to operational targets  
* requirements and tests for incomplete targets  
* integration, logic, or UI flows that block end-to-end coverage  
* documentation that clarifies expected behavior  
* minor UX/performance improvements supporting core progress  

You must:
* improve or complete at least one meaningful piece of scenario functionality  
* reduce the number or severity of failing checks  
* increase completeness  
* avoid regressions or speculative scope increases  

---

**Focus on advancing the scenario *forward*.  
Finish what’s partially built, close end-to-end gaps, solidify correctness, and reduce ambiguity — always in a stable, incremental, test-driven way.**

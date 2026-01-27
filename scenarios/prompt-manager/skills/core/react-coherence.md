## Steer focus: React Coherence

Prioritize **maintaining architectural coherence** across this React codebase.

Your goal is to ensure the codebase remains **well-organized, consistent, and free from duplication** - preventing the entropy that accumulates when features are developed in isolation without understanding the bigger picture.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve overall structure and consistency.

---

### **0. Why This Skill Exists**

When React UIs are developed over time, common issues emerge:
- State scattered across components instead of being centralized in Zustand stores
- Duplicate implementations of error handling, API operations, forms, modals
- Components that should be shared are defined locally in pages (or accidentally extracted multiple times)
- Styling inconsistencies and excessive variants
- Features developed without understanding how they fit the larger architecture

The root cause: **No shared mental model for code organization decisions.**

This skill provides concrete architectural patterns that ensure agents across multiple sessions conceptualize the app the same way.

---

### **1. State Architecture (Zustand-Centric)**

Use this decision flow for ALL state management:

```
┌─────────────────────────────────────────────────────┐
│                    Components                        │
│         (consume state via selectors, never define  │
│          shared state with useState)                │
└─────────────────────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
  ┌────────────┐  ┌────────────┐  ┌────────────┐
  │ Local UI   │  │  Zustand   │  │  Server    │
  │ State      │  │  Stores    │  │  State     │
  │ (useState) │  │            │  │  (cache)   │
  └────────────┘  └────────────┘  └────────────┘
       │                │                │
  ONLY for:        Shared app       React Query
  - Form inputs    state:           or Zustand
  - Open/closed    - User prefs     with async
  - Hover/focus    - UI state       patterns
  - Ephemeral UI   - Domain data
```

#### **State Location Decision Table**

| Question | If YES | If NO |
|----------|--------|-------|
| Is it used by 2+ components? | Zustand store | Continue... |
| Will it persist across navigation? | Zustand store | Continue... |
| Is it fetched from an API? | Server state (React Query or Zustand async) | Continue... |
| Is it purely ephemeral UI? | Local useState | Zustand store |

**When in doubt, use Zustand.** It's easier to move state from Zustand to local than to discover scattered useState calls later.

#### **Zustand Store Patterns**

```typescript
// ✅ GOOD: Focused store with selectors
interface AuthStore {
  user: User | null;
  isAuthenticated: boolean;
  login: (credentials: Credentials) => Promise<void>;
  logout: () => void;
}

const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  isAuthenticated: false,
  login: async (credentials) => {
    const user = await authService.login(credentials);
    set({ user, isAuthenticated: true });
  },
  logout: () => set({ user: null, isAuthenticated: false }),
}));

// Usage with selectors (prevents unnecessary re-renders)
const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
const login = useAuthStore((s) => s.login);
```

```typescript
// ❌ BAD: God store with everything
interface AppStore {
  user: User | null;
  theme: Theme;
  notifications: Notification[];
  currentProject: Project | null;
  executionStatus: ExecutionStatus;
  // ... 50 more fields mixing unrelated concerns
}
```

#### **Store Organization**

```
src/shared/stores/
├── authStore.ts        # Authentication state
├── settingsStore.ts    # User preferences
├── uiStore.ts          # Global UI state (modals, toasts)
└── index.ts            # Re-exports all stores
```

**Rule: One concern per store.** If a store exceeds ~200 lines, it's doing too much. Split it.

#### **Form State Anti-Pattern**

```tsx
// ❌ BAD: Multiple useState for related form state
const [name, setName] = useState("");
const [email, setEmail] = useState("");
const [loading, setLoading] = useState(false);
const [error, setError] = useState<string | null>(null);
const [submitted, setSubmitted] = useState(false);
// ... 10 more useState calls = hard to reason about, hard to test

// ✅ GOOD: Zustand store for form state
interface FormStore {
  values: FormValues;
  errors: Record<string, string>;
  status: 'idle' | 'submitting' | 'success' | 'error';
  setField: (field: string, value: string) => void;
  submit: () => Promise<void>;
  reset: () => void;
}
```

---

### **2. Sharing Decision Tree**

Before implementing ANY new functionality, follow this decision flow:

```
                    Implementing a feature?
                              │
                              ▼
         ┌────── Search: Does similar code exist? ──────┐
         │        (grep for similar patterns)           │
         │                                              │
       FOUND                                        NOT FOUND
         │                                              │
         ▼                                              │
  Use existing impl,                                    │
  extend if needed                                      │
         │                                              │
         └──────────────────┬───────────────────────────┘
                            │
                            ▼
              Is this used in 2+ places already?
                            │
              ┌─────────────┴─────────────┐
            YES                           NO
              │                            │
              ▼                            ▼
     Extract to shared/           Is this conceptually
     immediately                  domain-agnostic?
                                        │
                            ┌───────────┴───────────┐
                          YES                       NO
                            │                        │
                            ▼                        ▼
                    Put in shared/          Keep in feature/
                    from the start          (local scope)
```

#### **Before Writing New Code, Always Search**

```bash
# Find similar components
rg "function.*Button|const.*Button" --type tsx

# Find similar hooks
rg "function use.*Form|const use.*Form" --type tsx

# Find similar stores
rg "create<.*Store>" --type ts

# Find similar utilities
rg "function format|function parse|function validate" --type ts
```

**If you find existing code:** Use it. Extend it if needed. Do NOT create a duplicate.

---

### **3. Code Organization**

```
src/
├── shared/
│   ├── components/     # Reusable UI (Button, Card, Modal, etc.)
│   ├── hooks/          # Reusable hooks (useDebounce, useLocalStorage)
│   ├── stores/         # Zustand stores (authStore, settingsStore)
│   ├── services/       # API services (from react-stability)
│   ├── controllers/    # Business logic (from react-stability)
│   ├── schemas/        # Zod validation schemas
│   ├── lib/            # Pure utilities (formatDate, cn, etc.)
│   └── ui/             # Base design system components
│
├── surfaces/           # Feature areas (or "pages", "routes", "domains")
│   ├── dashboard/
│   │   ├── components/ # Dashboard-specific components
│   │   ├── hooks/      # Dashboard-specific hooks
│   │   └── DashboardPage.tsx
│   └── settings/
│       └── ...
```

#### **What Goes Where**

| Location | Criteria | Examples |
|----------|----------|----------|
| `shared/components/` | Used by 2+ surfaces | Button, Card, Modal, ErrorBoundary |
| `shared/hooks/` | Domain-agnostic utilities | useDebounce, useLocalStorage, useMediaQuery |
| `shared/stores/` | App-wide state | authStore, settingsStore, toastStore |
| `shared/ui/` | Design system primitives | Typography, spacing, color tokens |
| `shared/lib/` | Pure utility functions | formatDate, cn, parseError |
| `surfaces/X/components/` | Feature-specific | DashboardChart, SettingsForm |
| `surfaces/X/hooks/` | Feature-specific logic | useDashboardData, useSettingsForm |

#### **Migration Pattern: Extracting to Shared**

When moving code to `shared/`:

1. **Move the file** to appropriate shared directory
2. **Update all imports** across the codebase
3. **Remove feature-specific logic** (parameterize it via props/options)
4. **Add to index.ts** for discoverability
5. **Verify no circular dependencies** were introduced

---

### **4. Styling Coherence**

**Stack: Tailwind CSS + CSS Variables + CVA (Class Variance Authority)**

```
CSS Variables (design tokens)
        │
        ▼
Tailwind Config (maps tokens to utilities)
        │
        ▼
CVA Variants (component-level variants)
        │
        ▼
Components (compose variants, never raw styles)
```

#### **Design Token Pattern**

```css
/* index.css - Single source of truth for design tokens */
:root {
  --color-primary: 59 130 246;      /* rgb values for opacity support */
  --color-secondary: 107 114 128;
  --color-destructive: 239 68 68;

  --spacing-sm: 0.5rem;
  --spacing-md: 1rem;
  --spacing-lg: 1.5rem;

  --radius-sm: 0.25rem;
  --radius-md: 0.5rem;
  --radius-lg: 0.75rem;
}

[data-theme="dark"] {
  --color-primary: 96 165 250;
  /* ... dark overrides */
}
```

#### **Component Variant Pattern (CVA)**

```tsx
// ✅ GOOD: Explicit variants via CVA
import { cva, type VariantProps } from "class-variance-authority";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md font-medium transition-colors",
  {
    variants: {
      variant: {
        primary: "bg-primary text-white hover:bg-primary/90",
        secondary: "bg-secondary text-white hover:bg-secondary/90",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        destructive: "bg-destructive text-white hover:bg-destructive/90",
      },
      size: {
        sm: "h-8 px-3 text-sm",
        md: "h-10 px-4",
        lg: "h-12 px-6 text-lg",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  }
);
```

#### **Styling Rules**

1. **Never use arbitrary color values** - always reference design tokens
   ```tsx
   // ❌ BAD
   <div className="bg-[#f5f5f5]">

   // ✅ GOOD
   <div className="bg-surface-secondary">
   ```

2. **Limit variants per component** - max 3-4 variants per prop

3. **Compose, don't configure** - wrap components instead of adding style props
   ```tsx
   // ❌ BAD: Arbitrary styling props
   <Card padding={24} margin="10px 0" backgroundColor="#f5f5f5" />

   // ✅ GOOD: Semantic variants
   <Card variant="elevated" size="lg" />
   ```

4. **One source for each UI element** - Button in shared/ui, not reimplemented per feature

---

### **5. Common Mechanism Patterns**

Establish ONE canonical pattern for each common concern:

| Mechanism | Pattern | Location | Notes |
|-----------|---------|----------|-------|
| **Error handling** | ErrorBoundary + toast | `shared/components/ErrorBoundary` | Boundary catches, toast notifies |
| **Loading states** | Skeleton + Suspense | `shared/components/Skeleton` | Use Suspense where possible |
| **API calls** | Service layer (see react-stability) | `shared/services/` | Services return typed data |
| **Forms** | Zustand form store | `shared/stores/` or feature store | NOT multiple useState calls |
| **Modals** | Modal store + component | `shared/stores/modalStore` | Centralized modal management |
| **Toasts** | Toast store + component | `shared/stores/toastStore` | One notification system |
| **Validation** | Zod schemas | `shared/schemas/` | Colocate with API types |

**Before implementing any of these:** Search for existing implementation first!

---

### **6. Coherence Audit**

Run this audit when joining a project, before major work, or periodically for maintenance.

#### **Step 1: State Inventory**

```bash
# Find all useState usage with counts per file
rg "useState\(" --type tsx -c | sort -t: -k2 -nr | head -20

# Find Zustand stores
rg "create<.*>" --type ts -l

# Find Context usage
rg "createContext|useContext" --type tsx -l
```

**Red flags:**
- [ ] Files with 10+ useState calls → candidate for Zustand store
- [ ] Similar state shapes in multiple components → missing shared store
- [ ] Context used for frequently-changing state → should be Zustand

#### **Step 2: Duplication Check**

```bash
# Find duplicate component names
rg "function (Button|Card|Modal|Input|Form)" --type tsx -l | sort | uniq -d

# Find duplicate hook patterns
rg "function use(Form|Fetch|Auth|Modal)" --type tsx -l

# Find duplicate utility functions
rg "function (format|parse|validate)" --type ts -l
```

**Red flags:**
- [ ] Same component name in multiple locations → consolidate to shared/
- [ ] Similar hooks with slight variations → extract common logic
- [ ] Utility functions duplicated → move to shared/lib/

#### **Step 3: Styling Assessment**

```bash
# Find arbitrary color values (hex, rgb, hsl)
rg "#[0-9a-fA-F]{3,6}|rgb\(|hsl\(" --type tsx --type css

# Find inconsistent spacing patterns
rg "p-\[|m-\[|gap-\[" --type tsx  # Arbitrary values

# Check for CVA usage
rg "cva\(" --type tsx -c
```

**Red flags:**
- [ ] Hardcoded hex colors → should use design tokens
- [ ] Arbitrary spacing values `p-[14px]` → use standard scale
- [ ] No CVA usage → missing variant system

#### **Step 4: Architecture Alignment**

Verify the codebase follows the expected structure:

```bash
# Check if shared/ exists and has expected subdirs
ls -la src/shared/

# Find components outside shared that might belong there
rg "export (function|const) (Button|Card|Modal|Input)" --type tsx -l | grep -v shared
```

**Questions to answer:**
- [ ] Do all surfaces follow the same structure?
- [ ] Is shared/ actually shared, or duplicated per feature?
- [ ] Are services/controllers following react-stability patterns?

#### **Step 5: Document Findings**

After running the audit, create or update `docs/internal/COHERENCE_NOTES.md`:

```markdown
# Coherence Audit - [Date]

## State Management
- Current pattern: [Zustand/local state/mixed]
- Stores found: [list stores]
- Components with excessive useState (10+): [list with counts]

## Duplication
- Duplicate components: [list]
- Duplicate hooks: [list]
- Consolidation candidates: [list]

## Styling
- Design token usage: [good/partial/none]
- CVA adoption: [yes/partial/no]
- Inconsistencies found: [list]

## Priority Actions
1. [Highest impact fix]
2. [Second priority]
3. [Third priority]
```

---

### **7. Memory Management with Visited Tracker**

To ensure **systematic coverage without repetition**, use `visited-tracker`:

**At the start of each iteration:**
```bash
visited-tracker least-visited \
  --location scenarios/{{TARGET}}/ui \
  --pattern "**/*.{ts,tsx}" \
  --tag react-coherence \
  --name "{{TARGET}} - React Coherence" \
  --limit 5
```

**After analyzing each file:**
```bash
visited-tracker visit <file-path> \
  --location scenarios/{{TARGET}}/ui \
  --tag react-coherence \
  --note "<summary: what was consolidated, what patterns were aligned>"
```

**When a file follows all patterns correctly:**
```bash
visited-tracker exclude <file-path> \
  --location scenarios/{{TARGET}}/ui \
  --tag react-coherence \
  --reason "Already follows coherence patterns"
```

---

### **8. Relationship to Other Skills**

| Skill | Focus | When to Use Together |
|-------|-------|---------------------|
| react-stability | Crash prevention | Coherence builds on stability's architecture (Components→Hooks→Controllers→Services) |
| experience-architecture-audit | UX flows | Coherence is internal code quality; UX is user-facing experience |
| refactor | General cleanup | Use after coherence audit identifies structural issues |
| code-cleanup | Dead code removal | Coherence may reveal dead/duplicate code to clean |

Optional reading:
- `prompt-manager skills read react-stability experience-architecture-audit refactor code-cleanup`

**Recommended sequence for major work:**
1. **react-coherence** (audit) → understand current state
2. **react-stability** → ensure crash resistance
3. **experience-architecture-audit** → improve UX
4. **refactor** → clean up identified issues

---

### **9. Scenario Constraints**

* Do **not** change the scenario's core workflows, APIs, or business logic
* Do **not** introduce new features unrelated to architectural coherence
* Do **not** refactor everything at once - prioritize high-impact consolidations
* Prefer **incremental improvements** over ambitious rewrites

---

### **10. Output Expectations**

You may update:
* State management to use Zustand stores instead of scattered useState
* Component locations to consolidate duplicates into shared/
* Styling to use design tokens and CVA variants
* Code organization to follow the prescribed structure
* Import patterns to use shared/ components/hooks/stores

You **must**:
* Keep the scenario fully functional and non-regressed
* Reduce duplication and improve discoverability
* Align code with established patterns
* Document significant consolidations in commit messages

**Avoid superficial changes that only rename things or move code without genuinely improving coherence.**

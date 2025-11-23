# Auto Steer: Guided Multi-Dimensional Scenario Improvement

## Executive Summary

Auto Steer transforms the ecosystem-manager from a pure "completeness score optimizer" into a multi-dimensional improvement system. By introducing configurable "steering modes" and sequential profiles, agents can focus on specific improvement dimensions (UX, testing, refactoring, etc.) while maintaining the structural safety of operational target validation.

**Core Innovation**: Instead of agents min-maxing a single completeness metric, they optimize different dimensions in orchestrated sequences, producing scenarios that are complete, polished, well-tested, and maintainable.

---

## Problem Statement

### Current Limitations
1. **Single Optimization Target**: Agents optimize only for operational target completion percentage
2. **Gaming Behavior**: Agents exploit validation loopholes to maximize score with minimal work
3. **Loss of Creativity**: Structured approach prevents the interesting, unique UIs that emerged from the older unstructured approach
4. **Missing Quality Dimensions**: No explicit focus on UX, refactoring, comprehensive testing, or polish

### What We're Solving
- Enable multi-dimensional quality optimization
- Preserve structural safety (operational targets still validate completeness)
- Restore agent creativity within guardrails
- Create tunable, data-driven improvement sequences
- Build institutional knowledge about effective improvement strategies

---

## System Architecture

### 1. Data Models

#### Auto Steer Profile
```typescript
interface AutoSteerProfile {
  id: string;
  name: string;
  description: string;
  phases: SteerPhase[];
  qualityGates: QualityGate[];
  created: Date;
  updated: Date;
  tags: string[];  // e.g., "balanced", "rapid-mvp", "production-ready"
}
```

#### Steer Phase
```typescript
interface SteerPhase {
  id: string;
  mode: SteerMode;
  stopConditions: StopCondition[];
  maxIterations: number;  // Hard cap
  description?: string;
}

type SteerMode =
  | 'progress'      // Default: operational target completion
  | 'ux'            // Accessibility, user flows, design, responsiveness
  | 'refactor'      // Code quality, standards, complexity reduction
  | 'test'          // Coverage, edge cases, test quality
  | 'explore'       // Experimentation, novel approaches
  | 'polish'        // Final touches, error messages, loading states
  | 'integration'   // Connect with other scenarios/resources
  | 'performance'   // Profiling, optimization, caching
  | 'security';     // Vulnerability scanning, input validation
```

#### Stop Condition
```typescript
interface StopCondition {
  type: 'compound';
  operator: 'AND' | 'OR';
  conditions: (SimpleCondition | StopCondition)[];
}

interface SimpleCondition {
  type: 'simple';
  metric: string;  // e.g., 'loops', 'accessibility_score', 'ui_test_coverage'
  operator: '>' | '<' | '>=' | '<=' | '==' | '!=';
  value: number;
}

// Example: "(loops > 10) OR (loops > 3 AND accessibility > 90 AND ui_test_coverage > 80)"
const exampleCondition: StopCondition = {
  type: 'compound',
  operator: 'OR',
  conditions: [
    { type: 'simple', metric: 'loops', operator: '>', value: 10 },
    {
      type: 'compound',
      operator: 'AND',
      conditions: [
        { type: 'simple', metric: 'loops', operator: '>', value: 3 },
        { type: 'simple', metric: 'accessibility_score', operator: '>', value: 90 },
        { type: 'simple', metric: 'ui_test_coverage', operator: '>', value: 80 }
      ]
    }
  ]
};
```

#### Quality Gate
```typescript
interface QualityGate {
  name: string;
  condition: StopCondition;
  failureAction: 'halt' | 'skip_phase' | 'warn';
  message: string;
}

// Example: Can't start UX phase if build is broken
const buildGate: QualityGate = {
  name: 'build_health',
  condition: {
    type: 'compound',
    operator: 'AND',
    conditions: [
      { type: 'simple', metric: 'build_status', operator: '==', value: 1 },  // 1 = passing
      { type: 'simple', metric: 'operational_targets_passing', operator: '>=', value: 1 }
    ]
  },
  failureAction: 'halt',
  message: 'Cannot proceed: build is broken or no operational targets are passing'
};
```

#### Profile Execution State
```typescript
interface ProfileExecutionState {
  taskId: string;
  profileId: string;
  currentPhaseIndex: number;
  currentPhaseIteration: number;
  phaseHistory: PhaseExecution[];
  metrics: MetricsSnapshot;
  startedAt: Date;
  lastUpdated: Date;
}

interface PhaseExecution {
  phaseId: string;
  mode: SteerMode;
  iterations: number;
  startMetrics: MetricsSnapshot;
  endMetrics: MetricsSnapshot;
  commits: string[];  // Git commits made during this phase
  startedAt: Date;
  completedAt?: Date;
  stopReason: 'max_iterations' | 'condition_met' | 'quality_gate_failed';
}
```

#### Metrics Snapshot
```typescript
interface MetricsSnapshot {
  timestamp: Date;

  // Universal metrics
  loops: number;
  build_status: 0 | 1;  // 0 = failing, 1 = passing
  operational_targets_total: number;
  operational_targets_passing: number;
  operational_targets_percentage: number;

  // Mode-specific metrics
  ux?: UXMetrics;
  refactor?: RefactorMetrics;
  test?: TestMetrics;
  performance?: PerformanceMetrics;
  security?: SecurityMetrics;
}

interface UXMetrics {
  accessibility_score: number;  // 0-100
  ui_test_coverage: number;     // 0-100
  responsive_breakpoints: number;
  user_flows_implemented: number;
  loading_states_count: number;
  error_handling_coverage: number;
}

interface RefactorMetrics {
  cyclomatic_complexity_avg: number;
  duplication_percentage: number;
  standards_violations: number;
  tidiness_score: number;  // From tidiness-manager
  tech_debt_items: number;
}

interface TestMetrics {
  unit_test_coverage: number;
  integration_test_coverage: number;
  ui_test_coverage: number;
  edge_cases_covered: number;
  flaky_tests: number;
  test_quality_score: number;  // From test-genie
}

interface PerformanceMetrics {
  bundle_size_kb: number;
  initial_load_time_ms: number;
  lcp_ms: number;  // Largest Contentful Paint
  fid_ms: number;  // First Input Delay
  cls_score: number;  // Cumulative Layout Shift
}

interface SecurityMetrics {
  vulnerability_count: number;
  input_validation_coverage: number;
  auth_implementation_score: number;
  security_scan_score: number;
}
```

#### Historical Performance
```typescript
interface ProfilePerformance {
  profileId: string;
  scenarioName: string;
  executionId: string;
  startMetrics: MetricsSnapshot;
  endMetrics: MetricsSnapshot;
  phaseBreakdown: PhasePerformance[];
  totalIterations: number;
  totalDuration: number;  // milliseconds
  userFeedback?: {
    rating: 1 | 2 | 3 | 4 | 5;
    comments: string;
    submittedAt: Date;
  };
  executedAt: Date;
}

interface PhasePerformance {
  mode: SteerMode;
  iterations: number;
  metricDeltas: Record<string, number>;  // metric_name -> change
  duration: number;
  effectiveness: number;  // Calculated score based on improvement vs iterations
}
```

---

## 2. Mode Definitions & Agent Instructions

### Mode: Progress (Default)
**Goal**: Complete operational targets and advance feature implementation

**Agent Instructions**:
```
Focus on implementing missing operational targets and progressing incomplete features.
Prioritize breadth over depth - get features working end-to-end.

Success Criteria:
- Operational target completion percentage increases
- All implemented features have basic UI, API, and database components
- No regression in existing operational targets

Available Tools:
- All standard scenario resources
- scenario-dependency-analyzer (find related capabilities)
```

### Mode: UX
**Goal**: Improve user experience, accessibility, and design quality

**Agent Instructions**:
```
Focus on user experience improvements and accessibility. Make the application
delightful to use, accessible to all users, and visually polished.

Key Areas:
- Accessibility: ARIA labels, keyboard navigation, screen reader support
- Responsive design: Test and optimize for all breakpoints
- User flows: Ensure smooth, intuitive paths for all operational targets
- Visual polish: Consistent spacing, typography, color usage
- Loading states: All async operations show appropriate feedback
- Error handling: Clear, helpful error messages with recovery paths
- Performance UX: Perceived performance, optimistic updates

Follow F-layout principles and modern UX best practices.

Available Tools:
- browser-automation-studio (test user flows, validate accessibility)
- react-component-library (use consistent, accessible components)
- tidiness-manager (ensure code organization doesn't hinder UX work)

Success Criteria:
- Accessibility score > 90%
- All operational targets have complete user flows
- Responsive across mobile, tablet, desktop
- Zero UX-breaking errors
```

### Mode: Refactor
**Goal**: Improve code quality, reduce complexity, enforce standards

**Agent Instructions**:
```
Focus on code quality and maintainability. Reduce technical debt, improve
code organization, and ensure adherence to project standards.

Key Areas:
- Reduce cyclomatic complexity (target: < 10 per function)
- Eliminate code duplication (target: < 5%)
- Enforce naming conventions and project standards
- Improve code organization and module boundaries
- Add/improve code documentation
- Remove dead code and unused dependencies

Available Tools:
- tidiness-manager (identify issues, validate improvements)
- golangci-lint / eslint (automated quality checks)
- ast-grep (structural code analysis and refactoring)

Success Criteria:
- Tidiness score improves
- Standards violations decrease
- No regression in operational targets or tests
- Code is more maintainable (subjective, agent judgment)
```

### Mode: Test
**Goal**: Maximize test coverage and quality

**Agent Instructions**:
```
Focus on comprehensive testing across all layers. Write high-quality tests
that catch real issues and provide confidence in changes.

Key Areas:
- Unit tests: Cover edge cases, error paths, boundary conditions
- Integration tests: Validate component interactions
- UI tests: Verify user flows and accessibility
- Test quality: Clear, maintainable tests that fail meaningfully
- Reduce flakiness: Eliminate non-deterministic test failures

Available Tools:
- test-genie (generate high-quality tests, identify gaps)
- browser-automation-studio (UI automation and testing)
- Existing test frameworks (vitest, bats, go test)

Success Criteria:
- Unit coverage > 80%
- Integration coverage > 70%
- UI coverage > 60%
- Zero flaky tests
- All tests have clear failure messages
```

### Mode: Explore
**Goal**: Experiment with novel approaches and creative solutions

**Agent Instructions**:
```
This is your creative sandbox. Experiment with interesting approaches,
novel UIs, innovative features, or alternative implementations.

Key Areas:
- Try unconventional UI patterns that might improve UX
- Explore alternative architectural approaches
- Experiment with new libraries or techniques
- Implement "nice to have" features that add delight
- Think outside the box on problem-solving

Guidelines:
- Don't break existing functionality
- Document experiments clearly
- If an experiment succeeds, integrate it properly
- If it fails, revert cleanly

Success Criteria:
- At least one novel/interesting addition per iteration
- Experiments are documented
- Build remains stable
```

### Mode: Polish
**Goal**: Final touches and production-readiness

**Agent Instructions**:
```
Focus on the details that make a production-ready application. This is
the final 10% that takes 90% of the time.

Key Areas:
- Error messages: Clear, actionable, user-friendly
- Loading states: Every async operation has feedback
- Empty states: Helpful guidance when no data exists
- Edge cases: Handle all boundary conditions gracefully
- Validation messages: Clear, specific, helpful
- Tooltips and help text: Guide users proactively
- Confirmation dialogs: Prevent destructive actions
- Success feedback: Confirm actions completed

Available Tools:
- browser-automation-studio (verify edge cases)
- User flow validation

Success Criteria:
- Zero edge cases without proper handling
- All user actions have clear feedback
- Application feels "complete" and professional
```

### Mode: Integration
**Goal**: Connect with other scenarios and resources

**Agent Instructions**:
```
Focus on integrating with other scenarios and resources in the ecosystem.
Expand the capability surface by composing existing tools.

Key Areas:
- Identify relevant scenarios that could enhance functionality
- Integrate with additional resources (databases, AI services, etc.)
- Enable this scenario to be used BY other scenarios (API design)
- Cross-scenario workflows and data sharing
- Resource composition for emergent capabilities

Available Tools:
- scenario-dependency-analyzer (find integration opportunities)
- All available resources and scenarios
- deployment-manager (understand dependencies)

Success Criteria:
- At least one meaningful integration per iteration
- Integrations are documented
- APIs are designed for external use
```

### Mode: Performance
**Goal**: Optimize speed, resource usage, and scalability

**Agent Instructions**:
```
Focus on performance optimization across frontend and backend.

Key Areas:
- Frontend: Bundle size, code splitting, lazy loading
- Frontend: Render optimization, memoization
- Backend: Query optimization, caching, indexing
- Resource usage: Memory, CPU, database connections
- Scalability: Handle increased load gracefully

Tools:
- Browser DevTools (profiling, network analysis)
- Backend profiling tools
- Database query analyzers

Success Criteria:
- Initial load time < 3s
- Core Web Vitals in "Good" range
- Database queries optimized (no N+1 queries)
- Appropriate caching strategy implemented
```

### Mode: Security
**Goal**: Identify and fix security vulnerabilities

**Agent Instructions**:
```
Focus on security best practices and vulnerability remediation.

Key Areas:
- Input validation: All user inputs sanitized
- Authentication: Secure implementation, proper session handling
- Authorization: Proper access controls
- SQL injection prevention
- XSS prevention
- CSRF protection
- Dependency vulnerabilities
- Secrets management

Tools:
- Security scanning tools
- secrets-manager integration
- Dependency audit tools

Success Criteria:
- Zero high-severity vulnerabilities
- All inputs validated
- Auth/authz properly implemented
- Dependencies up to date with no known CVEs
```

---

## 3. Backend Implementation

### Database Schema

```sql
-- Auto Steer Profiles
CREATE TABLE auto_steer_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB NOT NULL,  -- Full profile configuration
    tags TEXT[],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Profile Executions (historical tracking)
CREATE TABLE profile_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID REFERENCES auto_steer_profiles(id),
    task_id UUID NOT NULL,  -- Links to ecosystem-manager task
    scenario_name VARCHAR(255) NOT NULL,
    start_metrics JSONB,
    end_metrics JSONB,
    phase_breakdown JSONB,
    total_iterations INTEGER,
    total_duration_ms BIGINT,
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    user_comments TEXT,
    user_feedback_at TIMESTAMP,
    executed_at TIMESTAMP DEFAULT NOW()
);

-- Profile Execution State (active executions)
CREATE TABLE profile_execution_state (
    task_id UUID PRIMARY KEY,
    profile_id UUID REFERENCES auto_steer_profiles(id),
    current_phase_index INTEGER NOT NULL,
    current_phase_iteration INTEGER NOT NULL,
    phase_history JSONB,
    metrics JSONB,
    started_at TIMESTAMP DEFAULT NOW(),
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_profile_executions_profile_id ON profile_executions(profile_id);
CREATE INDEX idx_profile_executions_scenario ON profile_executions(scenario_name);
CREATE INDEX idx_profile_executions_executed_at ON profile_executions(executed_at DESC);
```

### API Endpoints

```
# Profile Management
POST   /api/auto-steer/profiles              Create profile
GET    /api/auto-steer/profiles              List all profiles
GET    /api/auto-steer/profiles/:id          Get profile by ID
PUT    /api/auto-steer/profiles/:id          Update profile
DELETE /api/auto-steer/profiles/:id          Delete profile

# Profile Templates
GET    /api/auto-steer/templates             Get built-in profile templates

# Execution Management
GET    /api/auto-steer/execution/:taskId     Get current execution state
POST   /api/auto-steer/execution/:taskId/advance  Advance to next phase (manual)
POST   /api/auto-steer/execution/:taskId/halt     Halt execution

# Metrics
GET    /api/auto-steer/metrics/:taskId       Get current metrics for a task
POST   /api/auto-steer/metrics/:taskId       Update metrics (called by agents)

# Historical Performance
GET    /api/auto-steer/history               List all executions (with filters)
GET    /api/auto-steer/history/:executionId  Get specific execution details
POST   /api/auto-steer/history/:executionId/feedback  Submit user feedback
GET    /api/auto-steer/analytics/:profileId  Get aggregated analytics for profile
```

### Core Services

#### ProfileService
```typescript
class ProfileService {
  async createProfile(profile: AutoSteerProfile): Promise<AutoSteerProfile>;
  async getProfile(id: string): Promise<AutoSteerProfile | null>;
  async listProfiles(filters?: ProfileFilters): Promise<AutoSteerProfile[]>;
  async updateProfile(id: string, updates: Partial<AutoSteerProfile>): Promise<AutoSteerProfile>;
  async deleteProfile(id: string): Promise<void>;
  async getTemplates(): Promise<AutoSteerProfile[]>;  // Built-in templates
}
```

#### ExecutionEngine
```typescript
class ExecutionEngine {
  /**
   * Initialize profile execution for a task
   */
  async startExecution(taskId: string, profileId: string): Promise<ProfileExecutionState>;

  /**
   * Get current execution state
   */
  async getExecutionState(taskId: string): Promise<ProfileExecutionState | null>;

  /**
   * Called after each iteration to evaluate stop conditions
   * Returns: { shouldStop: boolean, reason?: string, nextPhase?: number }
   */
  async evaluateIteration(
    taskId: string,
    metrics: MetricsSnapshot
  ): Promise<IterationEvaluation>;

  /**
   * Advance to next phase (when current phase stops)
   */
  async advancePhase(taskId: string): Promise<PhaseAdvanceResult>;

  /**
   * Evaluate quality gates before phase transition
   */
  async evaluateQualityGates(
    taskId: string,
    gates: QualityGate[]
  ): Promise<QualityGateResult[]>;

  /**
   * Complete execution and archive to history
   */
  async completeExecution(taskId: string): Promise<ProfilePerformance>;
}
```

#### MetricsCollector
```typescript
class MetricsCollector {
  /**
   * Collect all available metrics for a scenario
   */
  async collectMetrics(scenarioName: string): Promise<MetricsSnapshot>;

  /**
   * Collect mode-specific metrics
   */
  async collectUXMetrics(scenarioName: string): Promise<UXMetrics>;
  async collectRefactorMetrics(scenarioName: string): Promise<RefactorMetrics>;
  async collectTestMetrics(scenarioName: string): Promise<TestMetrics>;
  async collectPerformanceMetrics(scenarioName: string): Promise<PerformanceMetrics>;
  async collectSecurityMetrics(scenarioName: string): Promise<SecurityMetrics>;

  /**
   * Parse metrics from various sources
   */
  private parseOperationalTargets(scenarioName: string): Promise<OperationalTargetMetrics>;
  private parseTidinessReport(scenarioName: string): Promise<RefactorMetrics>;
  private parseTestCoverage(scenarioName: string): Promise<TestMetrics>;
  private parseLighthouseReport(scenarioName: string): Promise<PerformanceMetrics>;
}
```

#### ConditionEvaluator
```typescript
class ConditionEvaluator {
  /**
   * Evaluate a stop condition against current metrics
   */
  evaluate(condition: StopCondition, metrics: MetricsSnapshot): boolean;

  /**
   * Evaluate simple condition
   */
  private evaluateSimple(condition: SimpleCondition, metrics: MetricsSnapshot): boolean;

  /**
   * Evaluate compound condition
   */
  private evaluateCompound(condition: StopCondition, metrics: MetricsSnapshot): boolean;
}
```

#### HistoryService
```typescript
class HistoryService {
  /**
   * Store completed execution
   */
  async recordExecution(performance: ProfilePerformance): Promise<void>;

  /**
   * Get execution history with filters
   */
  async getHistory(filters: HistoryFilters): Promise<ProfilePerformance[]>;

  /**
   * Get specific execution
   */
  async getExecution(executionId: string): Promise<ProfilePerformance | null>;

  /**
   * Submit user feedback
   */
  async submitFeedback(
    executionId: string,
    rating: number,
    comments: string
  ): Promise<void>;

  /**
   * Get aggregated analytics for a profile
   */
  async getProfileAnalytics(profileId: string): Promise<ProfileAnalytics>;
}

interface ProfileAnalytics {
  profileId: string;
  totalExecutions: number;
  avgRating: number;
  avgTotalIterations: number;
  avgDuration: number;
  phaseEffectiveness: Record<SteerMode, PhaseEffectiveness>;
  metricImprovements: Record<string, MetricImprovement>;
  scenarioBreakdown: ScenarioStats[];
}
```

---

## 4. Frontend Implementation

### Settings Dialog: Auto Steer Profiles Tab

**Location**: `scenarios/ecosystem-manager/ui/src/components/SettingsDialog.tsx`

**New Tab**: "Auto Steer Profiles"

**Components**:

```typescript
// Main profile management component
function AutoSteerProfilesTab() {
  return (
    <div className="auto-steer-profiles">
      <div className="profiles-header">
        <h2>Auto Steer Profiles</h2>
        <Button onClick={handleCreateProfile}>Create New Profile</Button>
      </div>

      <div className="profiles-list">
        {profiles.map(profile => (
          <ProfileCard
            key={profile.id}
            profile={profile}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onDuplicate={handleDuplicate}
          />
        ))}
      </div>

      <div className="templates-section">
        <h3>Templates</h3>
        <TemplateGallery
          templates={templates}
          onSelect={handleTemplateSelect}
        />
      </div>
    </div>
  );
}

// Individual profile card
function ProfileCard({ profile, onEdit, onDelete, onDuplicate }) {
  return (
    <Card className="profile-card">
      <div className="profile-header">
        <h3>{profile.name}</h3>
        <div className="profile-tags">
          {profile.tags.map(tag => (
            <Badge key={tag}>{tag}</Badge>
          ))}
        </div>
      </div>

      <p className="profile-description">{profile.description}</p>

      <div className="profile-phases">
        <h4>Phases ({profile.phases.length})</h4>
        <div className="phase-timeline">
          {profile.phases.map((phase, idx) => (
            <PhaseChip key={idx} phase={phase} />
          ))}
        </div>
      </div>

      <div className="profile-stats">
        <StatItem label="Executions" value={profile.executionCount} />
        <StatItem label="Avg Rating" value={profile.avgRating} />
        <StatItem label="Avg Duration" value={formatDuration(profile.avgDuration)} />
      </div>

      <div className="profile-actions">
        <Button variant="secondary" onClick={() => onEdit(profile)}>Edit</Button>
        <Button variant="secondary" onClick={() => onDuplicate(profile)}>Duplicate</Button>
        <Button variant="danger" onClick={() => onDelete(profile)}>Delete</Button>
      </div>
    </Card>
  );
}

// Profile editor dialog
function ProfileEditorDialog({ profile, onSave, onCancel }) {
  const [name, setName] = useState(profile?.name || '');
  const [description, setDescription] = useState(profile?.description || '');
  const [phases, setPhases] = useState<SteerPhase[]>(profile?.phases || []);
  const [qualityGates, setQualityGates] = useState<QualityGate[]>(profile?.qualityGates || []);

  return (
    <Dialog open onClose={onCancel}>
      <DialogTitle>{profile ? 'Edit Profile' : 'Create Profile'}</DialogTitle>

      <DialogContent>
        <TextField
          label="Profile Name"
          value={name}
          onChange={e => setName(e.target.value)}
          fullWidth
        />

        <TextField
          label="Description"
          value={description}
          onChange={e => setDescription(e.target.value)}
          multiline
          rows={3}
          fullWidth
        />

        <Divider />

        <div className="phases-editor">
          <h3>Phases</h3>
          <PhaseList
            phases={phases}
            onChange={setPhases}
          />
          <Button onClick={handleAddPhase}>Add Phase</Button>
        </div>

        <Divider />

        <div className="quality-gates-editor">
          <h3>Quality Gates</h3>
          <QualityGateList
            gates={qualityGates}
            onChange={setQualityGates}
          />
          <Button onClick={handleAddGate}>Add Quality Gate</Button>
        </div>
      </DialogContent>

      <DialogActions>
        <Button onClick={onCancel}>Cancel</Button>
        <Button onClick={handleSave} variant="primary">Save</Button>
      </DialogActions>
    </Dialog>
  );
}

// Phase editor component
function PhaseEditor({ phase, onChange }) {
  return (
    <Card className="phase-editor">
      <Select
        label="Mode"
        value={phase.mode}
        onChange={mode => onChange({ ...phase, mode })}
      >
        <option value="progress">Progress</option>
        <option value="ux">UX</option>
        <option value="refactor">Refactor</option>
        <option value="test">Test</option>
        <option value="explore">Explore</option>
        <option value="polish">Polish</option>
        <option value="integration">Integration</option>
        <option value="performance">Performance</option>
        <option value="security">Security</option>
      </Select>

      <TextField
        label="Max Iterations"
        type="number"
        value={phase.maxIterations}
        onChange={e => onChange({ ...phase, maxIterations: parseInt(e.target.value) })}
      />

      <div className="stop-conditions">
        <h4>Stop Conditions</h4>
        <ConditionBuilder
          conditions={phase.stopConditions}
          onChange={conditions => onChange({ ...phase, stopConditions: conditions })}
        />
      </div>
    </Card>
  );
}

// Condition builder for complex boolean expressions
function ConditionBuilder({ conditions, onChange }) {
  // Supports building conditions like:
  // "(loops > 10) OR (loops > 3 AND accessibility > 90 AND ui_test_coverage > 80)"

  // Visual tree-based editor with:
  // - Add condition/group buttons
  // - Drag-to-reorder
  // - AND/OR operator selection
  // - Metric dropdown
  // - Operator dropdown (>, <, >=, <=, ==, !=)
  // - Value input

  return (
    <div className="condition-builder">
      <ConditionGroup
        condition={conditions[0]}
        onChange={updated => onChange([updated])}
      />
    </div>
  );
}
```

### Task Dialog: Profile Selection & Progress

**Location**: `scenarios/ecosystem-manager/ui/src/components/TaskDialog.tsx`

**Updates**:

```typescript
function TaskDialog({ task, onSave, onCancel }) {
  const [autoSteerEnabled, setAutoSteerEnabled] = useState(task?.autoSteerProfileId != null);
  const [selectedProfile, setSelectedProfile] = useState(task?.autoSteerProfileId || '');

  return (
    <Dialog open onClose={onCancel}>
      <DialogTitle>{task ? 'Edit Task' : 'Create Task'}</DialogTitle>

      <DialogContent>
        {/* Existing task fields */}

        <Divider />

        <div className="auto-steer-section">
          <FormControlLabel
            control={
              <Switch
                checked={autoSteerEnabled}
                onChange={e => setAutoSteerEnabled(e.target.checked)}
              />
            }
            label="Enable Auto Steer"
          />

          {autoSteerEnabled && (
            <>
              <Select
                label="Profile"
                value={selectedProfile}
                onChange={e => setSelectedProfile(e.target.value)}
                fullWidth
              >
                {profiles.map(profile => (
                  <option key={profile.id} value={profile.id}>
                    {profile.name}
                  </option>
                ))}
              </Select>

              <ProfilePreview profileId={selectedProfile} />
            </>
          )}
        </div>
      </DialogContent>

      <DialogActions>
        <Button onClick={onCancel}>Cancel</Button>
        <Button onClick={handleSave} variant="primary">Save</Button>
      </DialogActions>
    </Dialog>
  );
}

// Show profile details in task dialog
function ProfilePreview({ profileId }) {
  const profile = useProfile(profileId);

  if (!profile) return null;

  return (
    <Card className="profile-preview">
      <h4>{profile.name}</h4>
      <p>{profile.description}</p>

      <div className="phase-timeline-preview">
        {profile.phases.map((phase, idx) => (
          <div key={idx} className="phase-preview">
            <Badge>{phase.mode}</Badge>
            <span className="phase-max-iterations">
              up to {phase.maxIterations} iterations
            </span>
          </div>
        ))}
      </div>

      <div className="profile-stats-preview">
        <StatItem label="Avg Iterations" value={profile.avgIterations} />
        <StatItem label="Avg Rating" value={profile.avgRating} />
      </div>
    </Card>
  );
}
```

### Task List: Show Profile Progress

**Location**: `scenarios/ecosystem-manager/ui/src/components/TaskList.tsx`

**Updates**:

```typescript
function TaskCard({ task }) {
  const executionState = useExecutionState(task.id);
  const profile = useProfile(task.autoSteerProfileId);

  return (
    <Card className="task-card">
      {/* Existing task info */}

      {executionState && profile && (
        <div className="auto-steer-progress">
          <div className="current-phase">
            <Badge variant={getModeColor(executionState.currentMode)}>
              {executionState.currentMode}
            </Badge>
            <span>
              Iteration {executionState.currentPhaseIteration} / {executionState.currentPhaseMax}
            </span>
          </div>

          <ProgressBar
            current={executionState.currentPhaseIndex}
            total={profile.phases.length}
            label="Profile Progress"
          />

          <div className="phase-timeline">
            {profile.phases.map((phase, idx) => (
              <PhaseIndicator
                key={idx}
                phase={phase}
                state={getPhaseState(idx, executionState)}
              />
            ))}
          </div>

          {executionState.currentStopConditionProgress && (
            <div className="stop-condition-progress">
              <h5>Stop Condition Progress</h5>
              <ConditionProgressDisplay
                condition={executionState.currentStopCondition}
                metrics={executionState.currentMetrics}
              />
            </div>
          )}

          <Button
            onClick={() => setShowMetricsDialog(true)}
            variant="secondary"
            size="small"
          >
            View Metrics
          </Button>
        </div>
      )}
    </Card>
  );
}

// Visual indicator for stop condition evaluation
function ConditionProgressDisplay({ condition, metrics }) {
  // Shows each condition with current value and target
  // Highlights which conditions are met vs not met
  // For compound conditions, shows tree structure

  return (
    <div className="condition-progress">
      {renderCondition(condition, metrics)}
    </div>
  );
}
```

### System Logs & History: Profile Performance Tab

**Location**: `scenarios/ecosystem-manager/ui/src/components/SystemLogsDialog.tsx`

**New Tab**: "Profile Performance"

```typescript
function ProfilePerformanceTab() {
  const [filters, setFilters] = useState<HistoryFilters>({});
  const [selectedExecution, setSelectedExecution] = useState<string | null>(null);

  return (
    <div className="profile-performance">
      <div className="performance-filters">
        <Select
          label="Profile"
          value={filters.profileId || ''}
          onChange={e => setFilters({ ...filters, profileId: e.target.value })}
        >
          <option value="">All Profiles</option>
          {profiles.map(p => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </Select>

        <Select
          label="Scenario"
          value={filters.scenario || ''}
          onChange={e => setFilters({ ...filters, scenario: e.target.value })}
        >
          <option value="">All Scenarios</option>
          {scenarios.map(s => (
            <option key={s} value={s}>{s}</option>
          ))}
        </Select>

        <DateRangePicker
          value={filters.dateRange}
          onChange={range => setFilters({ ...filters, dateRange: range })}
        />
      </div>

      <div className="performance-list">
        <ExecutionTable
          executions={executions}
          onSelectExecution={setSelectedExecution}
        />
      </div>

      {selectedExecution && (
        <ExecutionDetailPanel
          executionId={selectedExecution}
          onClose={() => setSelectedExecution(null)}
        />
      )}
    </div>
  );
}

// Table showing all executions
function ExecutionTable({ executions, onSelectExecution }) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>Date</TableCell>
          <TableCell>Profile</TableCell>
          <TableCell>Scenario</TableCell>
          <TableCell>Iterations</TableCell>
          <TableCell>Duration</TableCell>
          <TableCell>Rating</TableCell>
          <TableCell>Improvement</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {executions.map(exec => (
          <TableRow
            key={exec.id}
            onClick={() => onSelectExecution(exec.id)}
            className="clickable"
          >
            <TableCell>{formatDate(exec.executedAt)}</TableCell>
            <TableCell>{exec.profileName}</TableCell>
            <TableCell>{exec.scenarioName}</TableCell>
            <TableCell>{exec.totalIterations}</TableCell>
            <TableCell>{formatDuration(exec.totalDuration)}</TableCell>
            <TableCell>
              {exec.userRating ? (
                <Rating value={exec.userRating} readOnly />
              ) : (
                <em>Not rated</em>
              )}
            </TableCell>
            <TableCell>
              <ImprovementBadge
                start={exec.startMetrics.operational_targets_percentage}
                end={exec.endMetrics.operational_targets_percentage}
              />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

// Detailed view of a single execution
function ExecutionDetailPanel({ executionId, onClose }) {
  const execution = useExecution(executionId);
  const [feedbackDialog, setFeedbackDialog] = useState(false);

  if (!execution) return null;

  return (
    <Panel className="execution-detail" onClose={onClose}>
      <div className="execution-header">
        <h2>Execution Details</h2>
        {!execution.userFeedback && (
          <Button onClick={() => setFeedbackDialog(true)}>
            Rate This Execution
          </Button>
        )}
      </div>

      <div className="execution-summary">
        <StatGrid>
          <StatItem label="Profile" value={execution.profileName} />
          <StatItem label="Scenario" value={execution.scenarioName} />
          <StatItem label="Date" value={formatDate(execution.executedAt)} />
          <StatItem label="Total Iterations" value={execution.totalIterations} />
          <StatItem label="Duration" value={formatDuration(execution.totalDuration)} />
        </StatGrid>
      </div>

      <div className="metrics-comparison">
        <h3>Metrics Improvement</h3>
        <MetricsComparisonTable
          startMetrics={execution.startMetrics}
          endMetrics={execution.endMetrics}
        />
      </div>

      <div className="phase-breakdown">
        <h3>Phase Breakdown</h3>
        <PhaseBreakdownChart phases={execution.phaseBreakdown} />

        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Phase</TableCell>
              <TableCell>Mode</TableCell>
              <TableCell>Iterations</TableCell>
              <TableCell>Duration</TableCell>
              <TableCell>Key Improvements</TableCell>
              <TableCell>Effectiveness</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {execution.phaseBreakdown.map((phase, idx) => (
              <TableRow key={idx}>
                <TableCell>{idx + 1}</TableCell>
                <TableCell>
                  <Badge variant={getModeColor(phase.mode)}>{phase.mode}</Badge>
                </TableCell>
                <TableCell>{phase.iterations}</TableCell>
                <TableCell>{formatDuration(phase.duration)}</TableCell>
                <TableCell>
                  <MetricDeltaList deltas={phase.metricDeltas} />
                </TableCell>
                <TableCell>
                  <EffectivenessScore score={phase.effectiveness} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {execution.userFeedback && (
        <div className="user-feedback">
          <h3>Your Feedback</h3>
          <Rating value={execution.userFeedback.rating} readOnly />
          <p>{execution.userFeedback.comments}</p>
          <small>Submitted {formatDate(execution.userFeedback.submittedAt)}</small>
        </div>
      )}

      {feedbackDialog && (
        <FeedbackDialog
          executionId={executionId}
          onSubmit={handleFeedbackSubmit}
          onCancel={() => setFeedbackDialog(false)}
        />
      )}
    </Panel>
  );
}

// Feedback submission dialog
function FeedbackDialog({ executionId, onSubmit, onCancel }) {
  const [rating, setRating] = useState<number>(3);
  const [comments, setComments] = useState('');

  return (
    <Dialog open onClose={onCancel}>
      <DialogTitle>Rate This Execution</DialogTitle>

      <DialogContent>
        <div className="rating-input">
          <label>How well did this profile work?</label>
          <Rating
            value={rating}
            onChange={setRating}
          />
        </div>

        <TextField
          label="Comments (optional)"
          value={comments}
          onChange={e => setComments(e.target.value)}
          multiline
          rows={4}
          fullWidth
          placeholder="What worked well? What could be improved?"
        />
      </DialogContent>

      <DialogActions>
        <Button onClick={onCancel}>Cancel</Button>
        <Button
          onClick={() => onSubmit(rating, comments)}
          variant="primary"
        >
          Submit Feedback
        </Button>
      </DialogActions>
    </Dialog>
  );
}

// Chart showing phase effectiveness over time
function PhaseBreakdownChart({ phases }) {
  // Bar chart showing:
  // - X-axis: Phase number
  // - Y-axis: Iterations spent
  // - Color: Mode type
  // - Height: Number of iterations
  // - Tooltip: Shows metrics improvements

  return (
    <div className="phase-chart">
      {/* Chart implementation */}
    </div>
  );
}

// Comparison table for metrics
function MetricsComparisonTable({ startMetrics, endMetrics }) {
  const deltas = calculateDeltas(startMetrics, endMetrics);

  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>Metric</TableCell>
          <TableCell>Start</TableCell>
          <TableCell>End</TableCell>
          <TableCell>Change</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {Object.entries(deltas).map(([metric, delta]) => (
          <TableRow key={metric}>
            <TableCell>{formatMetricName(metric)}</TableCell>
            <TableCell>{formatMetricValue(metric, startMetrics[metric])}</TableCell>
            <TableCell>{formatMetricValue(metric, endMetrics[metric])}</TableCell>
            <TableCell>
              <MetricDelta value={delta} metric={metric} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
```

---

## 5. Integration with ecosystem-manager Task Loop

### Current Task Loop Flow

```
1. Load task from queue
2. Collect current scenario state (operational targets, requirements, etc.)
3. Generate agent prompt with task context
4. Spawn agent
5. Agent makes changes
6. Collect results
7. Update task state
8. If task not complete, goto 2
```

### Enhanced Flow with Auto Steer

```
1. Load task from queue
2. Check if task has Auto Steer profile
   - If yes, initialize ProfileExecutionState
   - Load profile configuration
   - Set current phase to 0
3. Collect current scenario state
4. Collect current metrics (MetricsCollector)
5. Generate agent prompt:
   - Base task context
   - Current phase mode instructions
   - Mode-specific tool recommendations
   - Context from previous phases
6. Spawn agent with enhanced prompt
7. Agent makes changes
8. Collect results and new metrics
9. Evaluate stop conditions (ConditionEvaluator)
   - Check phase stop conditions
   - Check quality gates
10. Update ProfileExecutionState
11. If phase should stop:
    - Archive phase results
    - Evaluate quality gates
    - If quality gates pass and more phases exist:
      - Advance to next phase
      - Goto 3
    - Else:
      - Complete execution
      - Archive to history
      - Mark task complete
12. Else (phase continues):
    - Goto 3
```

### Agent Prompt Enhancement

```typescript
function generateAgentPrompt(
  task: Task,
  scenarioState: ScenarioState,
  executionState?: ProfileExecutionState
): string {
  let prompt = `# Task: ${task.title}\n\n${task.description}\n\n`;

  // Add scenario state
  prompt += `## Current Scenario State\n`;
  prompt += `- Operational Targets: ${scenarioState.operationalTargetsCompleted}/${scenarioState.operationalTargetsTotal}\n`;
  prompt += `- Build Status: ${scenarioState.buildStatus}\n`;
  // ... more state

  if (executionState) {
    const profile = getProfile(executionState.profileId);
    const currentPhase = profile.phases[executionState.currentPhaseIndex];

    prompt += `\n## Auto Steer: ${profile.name}\n`;
    prompt += `You are in **${currentPhase.mode.toUpperCase()} MODE**.\n\n`;

    // Add mode-specific instructions
    prompt += getModeInstructions(currentPhase.mode);

    // Add context from previous phases
    if (executionState.phaseHistory.length > 0) {
      prompt += `\n## Previous Phases Context\n`;
      for (const phase of executionState.phaseHistory) {
        prompt += `- **${phase.mode}**: ${phase.iterations} iterations, `;
        prompt += `completed with ${phase.stopReason}\n`;

        // Highlight key improvements from this phase
        const keyImprovements = getKeyImprovements(phase);
        if (keyImprovements.length > 0) {
          prompt += `  Improvements: ${keyImprovements.join(', ')}\n`;
        }
      }
    }

    // Add progress towards stop condition
    prompt += `\n## Current Phase Progress\n`;
    prompt += `Iteration ${executionState.currentPhaseIteration} / ${currentPhase.maxIterations}\n`;
    prompt += `\nStop Conditions:\n`;
    prompt += formatStopConditions(currentPhase.stopConditions, executionState.metrics);
  }

  return prompt;
}

function getModeInstructions(mode: SteerMode): string {
  // Return the detailed mode instructions defined in section 2
  const instructions = {
    progress: `Focus on implementing missing operational targets...`,
    ux: `Focus on user experience improvements and accessibility...`,
    refactor: `Focus on code quality and maintainability...`,
    test: `Focus on comprehensive testing across all layers...`,
    explore: `This is your creative sandbox...`,
    polish: `Focus on the details that make a production-ready application...`,
    integration: `Focus on integrating with other scenarios and resources...`,
    performance: `Focus on performance optimization...`,
    security: `Focus on security best practices...`
  };

  return instructions[mode];
}

function formatStopConditions(
  conditions: StopCondition[],
  currentMetrics: MetricsSnapshot
): string {
  // Format conditions with current values and evaluation status
  let output = '';

  for (const condition of conditions) {
    const evaluation = evaluateCondition(condition, currentMetrics);
    const status = evaluation ? '✓' : '✗';
    output += `${status} ${formatConditionHuman(condition, currentMetrics)}\n`;
  }

  return output;
}
```

---

## 6. Built-in Profile Templates

### Template: Balanced
**Description**: Balanced approach suitable for most scenarios
```json
{
  "name": "Balanced",
  "description": "Well-rounded improvement across all dimensions",
  "phases": [
    {
      "mode": "progress",
      "maxIterations": 10,
      "stopConditions": [{
        "type": "compound",
        "operator": "OR",
        "conditions": [
          { "type": "simple", "metric": "loops", "operator": ">", "value": 10 },
          { "type": "simple", "metric": "operational_targets_percentage", "operator": ">=", "value": 80 }
        ]
      }]
    },
    {
      "mode": "ux",
      "maxIterations": 10,
      "stopConditions": [{
        "type": "compound",
        "operator": "OR",
        "conditions": [
          { "type": "simple", "metric": "loops", "operator": ">", "value": 10 },
          {
            "type": "compound",
            "operator": "AND",
            "conditions": [
              { "type": "simple", "metric": "accessibility_score", "operator": ">", "value": 90 },
              { "type": "simple", "metric": "ui_test_coverage", "operator": ">", "value": 60 }
            ]
          }
        ]
      }]
    },
    {
      "mode": "refactor",
      "maxIterations": 8,
      "stopConditions": [{
        "type": "compound",
        "operator": "OR",
        "conditions": [
          { "type": "simple", "metric": "loops", "operator": ">", "value": 8 },
          { "type": "simple", "metric": "tidiness_score", "operator": ">", "value": 85 }
        ]
      }]
    },
    {
      "mode": "test",
      "maxIterations": 15,
      "stopConditions": [{
        "type": "compound",
        "operator": "OR",
        "conditions": [
          { "type": "simple", "metric": "loops", "operator": ">", "value": 15 },
          {
            "type": "compound",
            "operator": "AND",
            "conditions": [
              { "type": "simple", "metric": "unit_test_coverage", "operator": ">", "value": 80 },
              { "type": "simple", "metric": "integration_test_coverage", "operator": ">", "value": 70 },
              { "type": "simple", "metric": "flaky_tests", "operator": "==", "value": 0 }
            ]
          }
        ]
      }]
    }
  ],
  "qualityGates": [
    {
      "name": "build_health",
      "condition": {
        "type": "simple",
        "metric": "build_status",
        "operator": "==",
        "value": 1
      },
      "failureAction": "halt",
      "message": "Build must be passing to continue"
    }
  ]
}
```

### Template: Rapid MVP
**Description**: Quick path to working MVP
```json
{
  "name": "Rapid MVP",
  "description": "Fast iteration to minimum viable product",
  "phases": [
    {
      "mode": "progress",
      "maxIterations": 20,
      "stopConditions": [{
        "type": "simple",
        "metric": "operational_targets_percentage",
        "operator": ">=",
        "value": 70
      }]
    },
    {
      "mode": "test",
      "maxIterations": 5,
      "stopConditions": [{
        "type": "simple",
        "metric": "unit_test_coverage",
        "operator": ">",
        "value": 60
      }]
    },
    {
      "mode": "polish",
      "maxIterations": 3,
      "stopConditions": [{
        "type": "simple",
        "metric": "loops",
        "operator": ">=",
        "value": 3
      }]
    }
  ]
}
```

### Template: Production Ready
**Description**: Comprehensive quality for production deployment
```json
{
  "name": "Production Ready",
  "description": "Thorough quality assurance for production deployment",
  "phases": [
    {
      "mode": "progress",
      "maxIterations": 15,
      "stopConditions": [{
        "type": "simple",
        "metric": "operational_targets_percentage",
        "operator": ">=",
        "value": 100
      }]
    },
    {
      "mode": "ux",
      "maxIterations": 15,
      "stopConditions": [{
        "type": "compound",
        "operator": "AND",
        "conditions": [
          { "type": "simple", "metric": "accessibility_score", "operator": ">", "value": 95 },
          { "type": "simple", "metric": "ui_test_coverage", "operator": ">", "value": 80 },
          { "type": "simple", "metric": "responsive_breakpoints", "operator": ">=", "value": 3 }
        ]
      }]
    },
    {
      "mode": "test",
      "maxIterations": 20,
      "stopConditions": [{
        "type": "compound",
        "operator": "AND",
        "conditions": [
          { "type": "simple", "metric": "unit_test_coverage", "operator": ">", "value": 90 },
          { "type": "simple", "metric": "integration_test_coverage", "operator": ">", "value": 80 },
          { "type": "simple", "metric": "ui_test_coverage", "operator": ">", "value": 70 },
          { "type": "simple", "metric": "flaky_tests", "operator": "==", "value": 0 }
        ]
      }]
    },
    {
      "mode": "refactor",
      "maxIterations": 12,
      "stopConditions": [{
        "type": "compound",
        "operator": "AND",
        "conditions": [
          { "type": "simple", "metric": "tidiness_score", "operator": ">", "value": 90 },
          { "type": "simple", "metric": "cyclomatic_complexity_avg", "operator": "<", "value": 8 },
          { "type": "simple", "metric": "duplication_percentage", "operator": "<", "value": 3 }
        ]
      }]
    },
    {
      "mode": "security",
      "maxIterations": 10,
      "stopConditions": [{
        "type": "compound",
        "operator": "AND",
        "conditions": [
          { "type": "simple", "metric": "vulnerability_count", "operator": "==", "value": 0 },
          { "type": "simple", "metric": "input_validation_coverage", "operator": ">", "value": 95 }
        ]
      }]
    },
    {
      "mode": "performance",
      "maxIterations": 8,
      "stopConditions": [{
        "type": "compound",
        "operator": "AND",
        "conditions": [
          { "type": "simple", "metric": "initial_load_time_ms", "operator": "<", "value": 3000 },
          { "type": "simple", "metric": "lcp_ms", "operator": "<", "value": 2500 }
        ]
      }]
    },
    {
      "mode": "polish",
      "maxIterations": 5,
      "stopConditions": [{
        "type": "simple",
        "metric": "loops",
        "operator": ">=",
        "value": 5
      }]
    }
  ],
  "qualityGates": [
    {
      "name": "build_health",
      "condition": {
        "type": "simple",
        "metric": "build_status",
        "operator": "==",
        "value": 1
      },
      "failureAction": "halt",
      "message": "Build must be passing"
    },
    {
      "name": "no_regressions",
      "condition": {
        "type": "simple",
        "metric": "operational_targets_passing",
        "operator": ">=",
        "value": -1  // Special value meaning "no decrease from phase start"
      },
      "failureAction": "halt",
      "message": "Cannot decrease passing operational targets"
    }
  ]
}
```

### Template: Refactor & Test Focus
**Description**: For scenarios that work but need quality improvement
```json
{
  "name": "Refactor & Test Focus",
  "description": "Improve quality of existing functionality",
  "phases": [
    {
      "mode": "test",
      "maxIterations": 20,
      "stopConditions": [{
        "type": "compound",
        "operator": "AND",
        "conditions": [
          { "type": "simple", "metric": "unit_test_coverage", "operator": ">", "value": 85 },
          { "type": "simple", "metric": "integration_test_coverage", "operator": ">", "value": 75 }
        ]
      }]
    },
    {
      "mode": "refactor",
      "maxIterations": 15,
      "stopConditions": [{
        "type": "simple",
        "metric": "tidiness_score",
        "operator": ">",
        "value": 90
      }]
    },
    {
      "mode": "test",
      "maxIterations": 10,
      "stopConditions": [{
        "type": "simple",
        "metric": "flaky_tests",
        "operator": "==",
        "value": 0
      }]
    }
  ]
}
```

### Template: UX Excellence
**Description**: Focus on creating exceptional user experience
```json
{
  "name": "UX Excellence",
  "description": "Maximum focus on user experience and design",
  "phases": [
    {
      "mode": "progress",
      "maxIterations": 8,
      "stopConditions": [{
        "type": "simple",
        "metric": "operational_targets_percentage",
        "operator": ">=",
        "value": 70
      }]
    },
    {
      "mode": "explore",
      "maxIterations": 10,
      "stopConditions": [{
        "type": "simple",
        "metric": "loops",
        "operator": ">=",
        "value": 10
      }]
    },
    {
      "mode": "ux",
      "maxIterations": 20,
      "stopConditions": [{
        "type": "compound",
        "operator": "AND",
        "conditions": [
          { "type": "simple", "metric": "accessibility_score", "operator": ">", "value": 95 },
          { "type": "simple", "metric": "user_flows_implemented", "operator": ">=", "value": 10 },
          { "type": "simple", "metric": "loading_states_count", "operator": ">=", "value": 5 }
        ]
      }]
    },
    {
      "mode": "performance",
      "maxIterations": 10,
      "stopConditions": [{
        "type": "simple",
        "metric": "initial_load_time_ms",
        "operator": "<",
        "value": 2000
      }]
    },
    {
      "mode": "polish",
      "maxIterations": 8,
      "stopConditions": [{
        "type": "simple",
        "metric": "loops",
        "operator": ">=",
        "value": 8
      }]
    },
    {
      "mode": "test",
      "maxIterations": 10,
      "stopConditions": [{
        "type": "simple",
        "metric": "ui_test_coverage",
        "operator": ">",
        "value": 80
      }]
    }
  ]
}
```

---

## 7. Metrics Collection Implementation

### Integration Points

#### Operational Targets (Existing)
```bash
# Already collected from test outputs
# Located in: scenarios/<name>/requirements/
# Provides: operational_targets_total, operational_targets_passing
```

#### Build Status
```bash
# Check if scenario builds successfully
cd scenarios/<name>
make build

# Exit code 0 = passing, non-zero = failing
```

#### UX Metrics

**Accessibility Score**:
```bash
# Use browser-automation-studio with axe-core
npm install -g @axe-core/cli
axe http://localhost:<port> --stdout

# Parse JSON output for violations count
# Score = 100 - (violations_count * penalty_per_violation)
```

**UI Test Coverage**:
```typescript
// Extract from vitest coverage reports
// Located in: scenarios/<name>/ui/coverage/coverage-summary.json
const coverage = JSON.parse(fs.readFileSync('coverage/coverage-summary.json'));
const uiTestCoverage = coverage.total.lines.pct;
```

**Responsive Breakpoints**:
```typescript
// Scan UI code for responsive CSS/breakpoints
// Count unique breakpoint definitions
const breakpoints = await grep('@media', 'ui/src/**/*.{css,tsx}');
const uniqueBreakpoints = new Set(breakpoints.map(extractBreakpoint));
```

#### Refactor Metrics

**Tidiness Score**:
```bash
# Use tidiness-manager scenario
curl http://localhost:<tidiness-port>/api/scan/<scenario-name>
# Returns: tidiness_score, standards_violations, tech_debt_items
```

**Cyclomatic Complexity**:
```bash
# For Go code
gocyclo -avg scenarios/<name>/api/

# For TypeScript
npm install -g complexity-report
cr scenarios/<name>/ui/src/ --format json
```

**Code Duplication**:
```bash
# Use jscpd for JavaScript/TypeScript
npm install -g jscpd
jscpd scenarios/<name>/ui/src/ --format json

# Use gocognit for Go
go install github.com/uudashr/gocognit/cmd/gocognit@latest
gocognit scenarios/<name>/api/
```

#### Test Metrics

**Coverage by Type**:
```bash
# Unit tests (Go)
cd scenarios/<name>/api
go test -cover ./... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Unit + Integration tests (TypeScript)
cd scenarios/<name>/ui
npm run test:coverage
# Parse coverage/coverage-summary.json
```

**Flaky Test Detection**:
```bash
# Run tests multiple times and detect inconsistencies
for i in {1..10}; do
  npm test > "test-run-$i.log"
done

# Compare results - any test that passes sometimes and fails sometimes is flaky
```

#### Performance Metrics

**Lighthouse CI**:
```bash
npm install -g @lhci/cli

# Run lighthouse
lhci autorun --collect.url=http://localhost:<port>

# Parse JSON output for:
# - initial_load_time_ms
# - lcp_ms (Largest Contentful Paint)
# - fid_ms (First Input Delay)
# - cls_score (Cumulative Layout Shift)
```

**Bundle Size**:
```bash
# For Vite builds
cd scenarios/<name>/ui
npm run build

# Get size of dist folder
du -sk dist/ | awk '{print $1}'
```

#### Security Metrics

**Vulnerability Scanning**:
```bash
# npm audit for JavaScript
cd scenarios/<name>/ui
npm audit --json

# go-audit for Go
go install github.com/sonatype-nexus-community/nancy@latest
go list -json -m all | nancy sleuth
```

**OWASP Dependency Check** (if needed):
```bash
dependency-check --scan scenarios/<name> --format JSON
```

### MetricsCollector Implementation

```typescript
class MetricsCollector {
  async collectMetrics(scenarioName: string): Promise<MetricsSnapshot> {
    const [
      universal,
      ux,
      refactor,
      test,
      performance,
      security
    ] = await Promise.all([
      this.collectUniversalMetrics(scenarioName),
      this.collectUXMetrics(scenarioName),
      this.collectRefactorMetrics(scenarioName),
      this.collectTestMetrics(scenarioName),
      this.collectPerformanceMetrics(scenarioName),
      this.collectSecurityMetrics(scenarioName)
    ]);

    return {
      timestamp: new Date(),
      ...universal,
      ux,
      refactor,
      test,
      performance,
      security
    };
  }

  private async collectUniversalMetrics(scenarioName: string) {
    // Operational targets from requirements
    const targets = await this.parseOperationalTargets(scenarioName);

    // Build status
    const buildStatus = await this.checkBuildStatus(scenarioName);

    return {
      loops: 0,  // Tracked separately by execution engine
      build_status: buildStatus ? 1 : 0,
      operational_targets_total: targets.total,
      operational_targets_passing: targets.passing,
      operational_targets_percentage: (targets.passing / targets.total) * 100
    };
  }

  private async parseOperationalTargets(scenarioName: string) {
    // Parse requirements/index.json and test outputs
    const requirementsPath = `scenarios/${scenarioName}/requirements/index.json`;
    const requirements = JSON.parse(await fs.readFile(requirementsPath, 'utf-8'));

    let total = 0;
    let passing = 0;

    // Count operational targets from all modules
    for (const module of requirements.modules) {
      for (const target of module.operationalTargets || []) {
        total++;
        if (target.status === 'passing') passing++;
      }
    }

    return { total, passing };
  }

  private async checkBuildStatus(scenarioName: string): Promise<boolean> {
    try {
      const result = await exec(`cd scenarios/${scenarioName} && make build`, {
        timeout: 300000  // 5 minutes
      });
      return result.exitCode === 0;
    } catch (error) {
      return false;
    }
  }

  async collectUXMetrics(scenarioName: string): Promise<UXMetrics> {
    const port = await this.getScenarioPort(scenarioName);

    // Accessibility score from axe
    const accessibilityScore = await this.runAxe(`http://localhost:${port}`);

    // UI test coverage from vitest
    const uiTestCoverage = await this.getUICoverage(scenarioName);

    // Responsive breakpoints from CSS analysis
    const responsiveBreakpoints = await this.countBreakpoints(scenarioName);

    // User flows from playbook analysis
    const userFlows = await this.countUserFlows(scenarioName);

    // Loading states from code analysis
    const loadingStates = await this.countLoadingStates(scenarioName);

    // Error handling coverage
    const errorHandling = await this.assessErrorHandling(scenarioName);

    return {
      accessibility_score: accessibilityScore,
      ui_test_coverage: uiTestCoverage,
      responsive_breakpoints: responsiveBreakpoints,
      user_flows_implemented: userFlows,
      loading_states_count: loadingStates,
      error_handling_coverage: errorHandling
    };
  }

  async collectRefactorMetrics(scenarioName: string): Promise<RefactorMetrics> {
    // Tidiness score from tidiness-manager
    const tidinessScore = await this.getTidinessScore(scenarioName);

    // Cyclomatic complexity
    const complexity = await this.getComplexity(scenarioName);

    // Code duplication
    const duplication = await this.getDuplication(scenarioName);

    // Standards violations from tidiness-manager
    const violations = tidinessScore.violations || 0;

    // Tech debt items from code comments or tidiness-manager
    const techDebt = await this.countTechDebt(scenarioName);

    return {
      cyclomatic_complexity_avg: complexity,
      duplication_percentage: duplication,
      standards_violations: violations,
      tidiness_score: tidinessScore.score,
      tech_debt_items: techDebt
    };
  }

  async collectTestMetrics(scenarioName: string): Promise<TestMetrics> {
    // Coverage from go test and vitest
    const goCoverage = await this.getGoCoverage(scenarioName);
    const tsCoverage = await this.getTypeScriptCoverage(scenarioName);

    // Flaky test detection
    const flakyTests = await this.detectFlakyTests(scenarioName);

    // Test quality from test-genie if available
    const testQuality = await this.getTestQuality(scenarioName);

    return {
      unit_test_coverage: (goCoverage.unit + tsCoverage.unit) / 2,
      integration_test_coverage: (goCoverage.integration + tsCoverage.integration) / 2,
      ui_test_coverage: tsCoverage.ui,
      edge_cases_covered: testQuality.edgeCases || 0,
      flaky_tests: flakyTests,
      test_quality_score: testQuality.score || 0
    };
  }

  async collectPerformanceMetrics(scenarioName: string): Promise<PerformanceMetrics> {
    const port = await this.getScenarioPort(scenarioName);

    // Run Lighthouse
    const lighthouse = await this.runLighthouse(`http://localhost:${port}`);

    // Bundle size
    const bundleSize = await this.getBundleSize(scenarioName);

    return {
      bundle_size_kb: bundleSize,
      initial_load_time_ms: lighthouse.initialLoad,
      lcp_ms: lighthouse.lcp,
      fid_ms: lighthouse.fid,
      cls_score: lighthouse.cls
    };
  }

  async collectSecurityMetrics(scenarioName: string): Promise<SecurityMetrics> {
    // Vulnerability scanning
    const vulnerabilities = await this.scanVulnerabilities(scenarioName);

    // Input validation coverage from code analysis
    const inputValidation = await this.assessInputValidation(scenarioName);

    // Auth implementation score
    const authScore = await this.assessAuthImplementation(scenarioName);

    return {
      vulnerability_count: vulnerabilities.length,
      input_validation_coverage: inputValidation,
      auth_implementation_score: authScore,
      security_scan_score: vulnerabilities.length === 0 ? 100 : Math.max(0, 100 - vulnerabilities.length * 10)
    };
  }
}
```

---

## 8. Implementation Phases

### Phase 1: Foundation (Week 1-2)
**Goal**: Core data models and backend services

- [ ] Database schema
- [ ] Profile CRUD API
- [ ] Execution state tracking
- [ ] Basic condition evaluator
- [ ] Universal metrics collection

**Deliverables**:
- Profiles can be created and stored
- Execution state can be initialized and tracked
- Simple stop conditions can be evaluated

### Phase 2: Metrics Collection (Week 2-3)
**Goal**: Comprehensive metrics across all dimensions

- [ ] MetricsCollector implementation
- [ ] UX metrics integration
- [ ] Refactor metrics (tidiness-manager integration)
- [ ] Test metrics
- [ ] Performance metrics (Lighthouse)
- [ ] Security metrics

**Deliverables**:
- Full MetricsSnapshot can be collected for any scenario
- Metrics update after each iteration

### Phase 3: Execution Engine (Week 3-4)
**Goal**: Profile execution and phase management

- [ ] ExecutionEngine implementation
- [ ] Phase advancement logic
- [ ] Quality gate evaluation
- [ ] Agent prompt enhancement
- [ ] Integration with task loop

**Deliverables**:
- Profiles execute end-to-end
- Agents receive mode-specific instructions
- Phases advance based on stop conditions

### Phase 4: Frontend - Settings (Week 4-5)
**Goal**: Profile management UI

- [ ] Settings dialog Auto Steer tab
- [ ] Profile list and cards
- [ ] Profile editor dialog
- [ ] Phase editor
- [ ] Condition builder
- [ ] Template gallery

**Deliverables**:
- Users can create, edit, and delete profiles
- Templates are available
- Condition builder supports complex conditions

### Phase 5: Frontend - Task Management (Week 5-6)
**Goal**: Profile selection and progress visualization

- [ ] Task dialog profile selection
- [ ] Profile preview in task dialog
- [ ] Task card profile progress display
- [ ] Phase timeline visualization
- [ ] Stop condition progress display

**Deliverables**:
- Users can assign profiles to tasks
- Task execution shows current phase and progress
- Stop condition evaluation is visible

### Phase 6: Frontend - History & Analytics (Week 6-7)
**Goal**: Historical performance tracking

- [ ] System Logs Profile Performance tab
- [ ] Execution table with filters
- [ ] Execution detail panel
- [ ] Metrics comparison
- [ ] Phase breakdown chart
- [ ] Feedback submission

**Deliverables**:
- Historical executions are visible
- Users can analyze profile effectiveness
- Feedback can be submitted and viewed

### Phase 7: Built-in Templates & Mode Refinement (Week 7-8)
**Goal**: Production-ready templates and mode optimization

- [ ] Implement all 5 built-in templates
- [ ] Refine mode-specific instructions based on testing
- [ ] Tool recommendations per mode
- [ ] Quality gate presets
- [ ] Documentation

**Deliverables**:
- All templates work effectively
- Mode instructions produce desired agent behavior
- Documentation is complete

### Phase 8: Testing & Iteration (Week 8-10)
**Goal**: Validate system with real scenarios

- [ ] Test with 5-10 real scenarios
- [ ] Collect effectiveness data
- [ ] Refine stop conditions based on outcomes
- [ ] Optimize metrics collection performance
- [ ] Bug fixes and polish

**Deliverables**:
- System works reliably across diverse scenarios
- Profiles demonstrably improve multi-dimensional quality
- Performance is acceptable

---

## 9. Success Criteria

### System-Level
- [ ] Profiles execute reliably without manual intervention
- [ ] Metrics collection completes in < 2 minutes per scenario
- [ ] Phase transitions happen correctly based on conditions
- [ ] Quality gates prevent degradation

### User Experience
- [ ] Profile creation takes < 5 minutes
- [ ] Progress visibility is clear and informative
- [ ] Historical analytics provide actionable insights
- [ ] Templates work well for common use cases

### Agent Behavior
- [ ] Agents follow mode-specific instructions
- [ ] Agents produce work appropriate to current phase
- [ ] Creativity increases in Explore mode
- [ ] Quality improves across dimensions (not just operational targets)

### Quality Outcomes
- [ ] Scenarios improved with Auto Steer have higher overall quality
- [ ] No more "gaming" of single metrics
- [ ] UIs are more interesting and polished
- [ ] Test coverage and code quality improve measurably

---

## 10. Future Enhancements

### Adaptive Profiles
- Learn optimal phase sequences from historical data
- Automatically adjust iteration counts based on scenario complexity
- Suggest profile modifications based on outcomes

### Cross-Scenario Learning
- Identify which modes work best for which scenario types
- Share successful patterns across similar scenarios
- Build a knowledge base of effective improvement strategies

### Agent Collaboration
- Multiple agents working on different phases in parallel
- Agent handoffs with context preservation
- Peer review between agents

### Advanced Metrics
- Code maintainability index
- User experience score (composite metric)
- "Production readiness" score
- Technical debt ratio

### Integration Expansion
- More tool integrations per mode
- Automatic tool recommendations
- Tool effectiveness tracking

---

## Conclusion

Auto Steer transforms ecosystem-manager from a single-dimension optimizer into a sophisticated, multi-dimensional improvement system. By giving agents focused objectives within guardrails, we restore creativity while maintaining structural safety.

The system is designed to:
1. **Prevent gaming** through multi-dimensional optimization
2. **Enhance quality** across UX, testing, refactoring, and more
3. **Preserve safety** via operational targets and quality gates
4. **Enable learning** through historical performance tracking
5. **Support customization** via flexible profile configuration

With careful implementation and iteration, Auto Steer will produce scenarios that are not just complete, but polished, maintainable, accessible, well-tested, and delightful to use.

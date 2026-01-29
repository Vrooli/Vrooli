# Testing Seams in prompt-manager

This document describes the testing seams (interfaces and dependency injection points) in the prompt-manager API and UI. These seams enable unit testing without external dependencies like databases or the filesystem.

## Overview

The API is organized into domain-driven packages:
- `skills/` - Core skill management (primary domain)
- `metrics/` - Usage tracking and ratings
- `tags/` - Tag categorization
- `testing/` - LLM-based skill testing via Ollama

Each package follows the same pattern:
1. **Models** - Type definitions
2. **Interfaces** - Contracts for testing seams (in skills/interfaces.go)
3. **Repository/Store** - Data access implementations
4. **Query** - Domain logic for filtering and transformations
5. **Handlers** - HTTP request handling (depends on interfaces, not concrete types)

## Architecture Alignment

Recent seam and boundary refinements:
- **UI preview seam**: Markdown rendering is isolated under `ui/src/components/markdown`, separate from the markdown ↔ HTML conversion pipeline in `ui/src/services/content`.
- **Editor view modes**: `SkillContentEditor` owns the Edit/Preview/Split selection while editors (Monaco/TipTap) only handle editing concerns.
- **Link preview seam**: `fetchLinkPreview` in `ui/src/lib/api.ts` centralizes OG metadata fetching for both rich editor tooltips and markdown preview links.
- **Split resizing seam**: `useResizableSplitPanel` encapsulates split-view resizing behavior to keep layout logic out of editor rendering.
- **World placement seam**: `ui/src/lib/world.ts` owns placement/boundary math so dragging and click-to-place use the same rules without duplicating logic.
- **Ground texturing seam**: `ui/src/lib/groundTextures.ts` and `ui/src/lib/groundShader.ts` centralize procedural texture generation + shader tweaks, while `GroundSurface`/`GroundMaterial` handle rendering without leaking shader logic into `WorldScene`.

## Testing Seams

### 1. skills.SkillStore (Interface)

**Location:** `api/skills/interfaces.go`

**Purpose:** Interface for skill storage operations - the primary testing seam.

**Interface:**
```go
// SkillStore defines the interface for skill storage operations.
// Implementations: Store (file-based, production), MockStore (testing).
type SkillStore interface {
    GetAll() ([]Metadata, error)
    FindByID(id string) (*Metadata, string, error)
    LoadMetadata(folder string) ([]Metadata, error)
    SaveMetadata(folder string, skills []Metadata) error
    GetContent(folder, filename string) (string, error)
    SaveContent(folder, filename, content string) error
    DeleteContent(folder, filename string) error
}
```

**Implementation:** `api/skills/store.go` (production, file-based)

**Testing Strategy:**
- **Unit tests:** Create a mock implementing `SkillStore`
- **Integration tests:** Create a temp directory with test fixtures, pass to `NewStore(tempDir)`

**Example (Mock):**
```go
type MockStore struct {
    skills map[string][]skills.Metadata
}

func (m *MockStore) GetAll() ([]skills.Metadata, error) {
    var all []skills.Metadata
    for _, p := range m.skills {
        all = append(all, p...)
    }
    return all, nil
}
// ... implement other methods

func TestSkillHandlers(t *testing.T) {
    mockStore := &MockStore{skills: testData}
    mockMetrics := &MockMetricsService{}
    handlers := skills.NewHandlers(mockStore, mockMetrics)
    // Test handlers without filesystem
}
```

**Example (Integration):**
```go
func TestSkillCreate(t *testing.T) {
    tempDir := t.TempDir()
    store := skills.NewStore(tempDir)

    // Test skill creation with real filesystem
    // ...
}
```

### 2. skills.MetricsService (Interface)

**Location:** `api/skills/interfaces.go`

**Purpose:** Interface for metrics operations used by skill handlers - enables testing without database.

**Interface:**
```go
// MetricsService defines the interface for skill metrics operations.
// Implementations: MetricsAdapter (wraps metrics.Repository), MockMetricsService (testing).
type MetricsService interface {
    Get(skillID string) (*SkillMetrics, error)
    RecordUsage(skillID string) (int, time.Time, error)
    SetRating(skillID string, rating int, notes *string) error
    Delete(skillID string) error
}
```

**Implementation:** `api/skills/metrics_adapter.go` (production, wraps metrics.Repository)

**Testing Strategy:**
- Create a mock implementing `MetricsService`
- No database access needed for testing handler logic

**Example:**
```go
type MockMetricsService struct {
    metrics map[string]*skills.SkillMetrics
}

func (m *MockMetricsService) Get(skillID string) (*skills.SkillMetrics, error) {
    if pm, ok := m.metrics[skillID]; ok {
        return pm, nil
    }
    return nil, nil // Not found is not an error
}

func (m *MockMetricsService) RecordUsage(skillID string) (int, time.Time, error) {
    // ... mock implementation
}
```

### 3. metrics.Repository (Implementation)

**Location:** `api/metrics/repository.go`

**Purpose:** PostgreSQL storage for usage metrics

**Interface:**
```go
type Repository struct {
    db *sql.DB
}

func (r *Repository) Get(skillID string) (*SkillMetrics, error)
func (r *Repository) RecordUsage(skillID string) (int, time.Time, error)
func (r *Repository) SetRating(skillID string, rating int, notes *string) error
func (r *Repository) Delete(skillID string) error
```

**Testing Strategy:**
- Use `sqlmock` for unit tests
- Use test container for integration tests
- Inject mock `*sql.DB` into `NewRepository(mockDB)`

**Example:**
```go
func TestRecordUsage(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    mock.ExpectQuery("INSERT INTO skill_metrics").
        WithArgs("skill-1").
        WillReturnRows(sqlmock.NewRows([]string{"usage_count", "last_used"}).
            AddRow(1, time.Now()))

    repo := metrics.NewRepository(db)
    count, _, err := repo.RecordUsage("skill-1")

    assert.NoError(t, err)
    assert.Equal(t, 1, count)
}
```

### 4. tags.Repository

**Location:** `api/tags/repository.go`

**Purpose:** PostgreSQL storage for tags

**Interface:**
```go
type Repository struct {
    db *sql.DB
}

func (r *Repository) GetAll() ([]Tag, error)
func (r *Repository) Create(tag *Tag) error
```

**Testing Strategy:** Same as metrics.Repository

### 5. testing.Repository

**Location:** `api/testing/repository.go`

**Purpose:** PostgreSQL storage for test results

**Interface:**
```go
type Repository struct {
    db *sql.DB
}

func (r *Repository) Save(result *TestResult) error
func (r *Repository) GetHistory(skillID string, limit int) ([]TestResult, error)
```

**Testing Strategy:** Same as metrics.Repository

### 6. testing.OllamaClient

**Location:** `api/testing/client.go`

**Purpose:** HTTP client for Ollama API

**Interface:**
```go
type OllamaClient struct {
    baseURL    string
    httpClient *http.Client
}

func (c *OllamaClient) IsEnabled() bool
func (c *OllamaClient) Generate(model, prompt string, maxTokens int, temperature float64) (*OllamaResponse, float64, error)
```

**Testing Strategy:**
- Create a mock HTTP server
- Pass mock server URL to `NewOllamaClient(mockURL)`
- Or pass empty string to disable

**Example:**
```go
func TestSkillTesting(t *testing.T) {
    // Mock Ollama server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(testing.OllamaResponse{
            Response:  "Test response",
            EvalCount: 50,
        })
    }))
    defer server.Close()

    client := testing.NewOllamaClient(server.URL)
    // Test skill testing
}
```

## Dependency Wiring in main.go

The main.go file is a thin bootstrap layer that wires dependencies using the adapter pattern:

```go
// Initialize domain components (seams for testing)
skillStore := skills.NewStore(skillsDir)
metricsRepo := metrics.NewRepository(db)
tagsRepo := tags.NewRepository(db)
testingRepo := testing.NewRepository(db)
ollamaClient := testing.NewOllamaClient(ollamaURL)

// Create adapter to bridge metrics.Repository -> skills.MetricsService interface
metricsAdapter := skills.NewMetricsAdapter(metricsRepo)

// Initialize handlers with interface adapters
skillHandlers := skills.NewHandlers(skillStore, metricsAdapter)
tagsHandlers := tags.NewHandlers(tagsRepo)
testingHandlers := testing.NewHandlers(testingRepo, ollamaClient, skillStore)
```

**Key Design Decision:** The `skillHandlers` depends on interfaces (`SkillStore`, `MetricsService`), not concrete types. This enables:
- Unit testing handlers without filesystem or database
- Swapping implementations (e.g., in-memory store for testing)
- Clear boundary between handlers and data access

## Domain Logic (query.go)

**Location:** `api/skills/query.go`

Filtering and transformation logic is separated from handlers into dedicated domain functions:

```go
// FilterOptions defines criteria for filtering skills.
type FilterOptions struct {
    Tag    string
    Folder string
    Modes  []string
}

// Filter applies all filter criteria to a list of skills.
func Filter(skills []Metadata, opts FilterOptions) []Metadata

// Slugify converts a string to a URL-safe slug.
func Slugify(s string) string
```

**Why This Matters:**
- Domain logic is testable independently of HTTP handlers
- Handlers focus on HTTP concerns (parsing, validation, response)
- Filter logic can be reused across different entry points

**Example:**
```go
func TestFilterByTag(t *testing.T) {
    allSkills := []skills.Metadata{
        {ID: "1", Tags: []string{"skill"}},
        {ID: "2", Tags: []string{"template"}},
    }

    filtered := skills.Filter(allSkills, skills.FilterOptions{Tag: "skill"})

    assert.Len(t, filtered, 1)
    assert.Equal(t, "1", filtered[0].ID)
}
```

## Testing the Handlers

Handlers receive dependencies via constructor injection:

```go
// In tests, inject mocks
func TestSkillHandlers(t *testing.T) {
    store := NewMockStore()       // Mock skill store
    metricsRepo := NewMockRepo()  // Mock metrics repository

    handlers := skills.NewHandlers(store, metricsRepo)

    // Create test request
    req := httptest.NewRequest("GET", "/api/v1/skills", nil)
    w := httptest.NewRecorder()

    // Call handler
    handlers.List(w, req)

    // Assert response
    assert.Equal(t, http.StatusOK, w.Code)
}
```

## Best Practices

1. **Don't mock the HTTP layer** - Use `httptest.NewRecorder()` to test handlers directly
2. **Mock at the repository level** - This tests business logic without database access
3. **Use real filesystems for integration tests** - Use `t.TempDir()` for isolation
4. **Test error paths** - Mock failures to verify error handling
5. **Avoid over-mocking** - Integration tests with test containers are valuable

## UI Testing Seams

The UI has clear testing seams at the data access layer:

### 1. api Client

**Location:** `ui/src/lib/api.ts`

**Purpose:** HTTP client for the prompt-manager API

**Interface:**
```typescript
class ApiClient {
  async getSkills(filters?: SearchFilters): Promise<Skill[]>
  async getSkillsByFolder(folder: FolderType): Promise<Skill[]>
  async getSkill(id: string): Promise<Skill>
  async createSkill(skill: CreateSkillRequest): Promise<Skill>
  async updateSkill(id: string, data: UpdateSkillRequest): Promise<Skill>
  async deleteSkill(id: string): Promise<void>
  async testSkill(id: string, request: SkillTestRequest): Promise<SkillTestResult>
  async searchSkills(query: string, filters?: SearchFilters): Promise<Skill[]>
  async getFolders(): Promise<Folder[]>
  async healthCheck(): Promise<HealthResponse>
}
```

**Testing Strategy:**
- Use MSW (Mock Service Worker) to intercept HTTP requests
- Or mock the `api` export directly with jest.mock()

**Example:**
```typescript
import { rest } from 'msw'
import { setupServer } from 'msw/node'
import { api } from '@/lib/api'

const server = setupServer(
  rest.get('/api/v1/skills', (req, res, ctx) => {
    return res(ctx.json([{ id: '1', name: 'Test', folder: 'local', ... }]))
  })
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

test('fetches skills', async () => {
  const skills = await api.getSkills()
  expect(skills).toHaveLength(1)
})
```

### 2. useSkills Hook (State Management)

**Location:** `ui/src/hooks/useSkillsData.ts`

**Purpose:** Encapsulates all skill-related state management (selection, filtering, search)

**Interface:**
```typescript
interface UseSkillsReturn {
  // Data
  folders: Folder[]
  filteredSkills: Skill[]
  sidebarCounts: SidebarCounts

  // Selection state
  selectedFolder: Folder | null
  selectedSkill: Skill | null

  // Filter state
  viewFilter: ViewFilter
  searchQuery: string
  filterInfo: FilterInfo | null

  // Loading states
  isLoading: boolean
  foldersLoading: boolean

  // Actions
  setSelectedFolder: (folder: Folder | null) => void
  setSelectedSkill: (skill: Skill | null) => void
  setViewFilter: (filter: ViewFilter) => void
  setSearchQuery: (query: string) => void
  handleFilterChange: (filter: ViewFilter) => void

  // Computed
  showSkillList: boolean
}
```

**Testing Strategy:**
- Wrap in `QueryClientProvider` with mocked `api`
- Use `@testing-library/react-hooks` for isolated hook testing

**Example:**
```typescript
import { renderHook, waitFor } from '@testing-library/react'
import { useSkillsData } from '@/hooks/useSkillsData'

const wrapper = ({ children }) => (
  <QueryClientProvider client={testQueryClient}>
    {children}
  </QueryClientProvider>
)

test('filters skills by favorites', async () => {
  const favorites = new Set(['skill-1'])
  const { result } = renderHook(() => useSkillsData({ favorites }), { wrapper })

  act(() => {
    result.current.handleFilterChange('favorites')
  })

  await waitFor(() => {
    expect(result.current.filteredSkills).toEqual(
      expect.arrayContaining([expect.objectContaining({ id: 'skill-1' })])
    )
  })
})
```

### 3. Favorites (localStorage)

**Location:** `ui/src/hooks/use-favorites.tsx`

**Purpose:** Client-side favorite storage (UI preference, not persisted to API)

**Interface:**
```typescript
function useFavorites(): {
  favorites: Set<string>
  isFavorite: (id: string) => boolean
  toggleFavorite: (id: string) => void
}
```

**Testing Strategy:**
- Mock localStorage in test setup
- Or use the hook directly with @testing-library/react-hooks

**Example:**
```typescript
import { renderHook, act } from '@testing-library/react-hooks'
import { useFavorites } from '@/hooks/use-favorites'

beforeEach(() => {
  localStorage.clear()
})

test('toggles favorite', () => {
  const { result } = renderHook(() => useFavorites())

  expect(result.current.isFavorite('prompt-1')).toBe(false)

  act(() => {
    result.current.toggleFavorite('prompt-1')
  })

  expect(result.current.isFavorite('prompt-1')).toBe(true)
})
```

### 4. React Query (Data Fetching)

**Location:** Components use `useQuery` and `useMutation` from @tanstack/react-query

**Testing Strategy:**
- Wrap tests in `QueryClientProvider` with a fresh `QueryClient`
- Use `waitFor` to wait for query resolution
- Mock the api client at the network level (MSW) or module level

**Example:**
```typescript
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, waitFor } from '@testing-library/react'
import { App } from '@/App'

const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: { retry: false }
  }
})

function renderWithClient(ui: React.ReactElement) {
  const client = createTestQueryClient()
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>
  )
}

test('renders skills', async () => {
  const { getByText } = renderWithClient(<App />)
  await waitFor(() => expect(getByText('Test Skill')).toBeInTheDocument())
})
```

### 5. Markdown Preview Rendering

**Location:** `ui/src/components/markdown`

**Purpose:** Render markdown content for preview and split-on-desktop views without requiring HTML round-trip conversion.

**Testing Strategy:**
- Render `MarkdownRenderer` with representative markdown and verify DOM output.
- Validate code blocks and inline code behaviors independently of TipTap.
- Mock `fetchLinkPreview` to test link preview rendering in isolation.

**Example:**
```typescript
import { render, screen } from '@testing-library/react'
import { MarkdownRenderer } from '@/components/markdown'

test('renders markdown preview', () => {
  render(<MarkdownRenderer content={'# Title\\n\\nParagraph'} />)
  expect(screen.getByText('Title')).toBeInTheDocument()
  expect(screen.getByText('Paragraph')).toBeInTheDocument()
})
```

## Boundary of Responsibility

The UI and API have clear boundaries:

| Concern | Owner | Storage |
|---------|-------|---------|
| Skill content & metadata | API | File system (metadata.json + .md) |
| Usage metrics | API | PostgreSQL |
| Tags | API | PostgreSQL |
| Test results | API | PostgreSQL |
| Favorites | UI | localStorage |
| View preferences | UI | Component state |
| Theme | UI | localStorage |

This separation means:
- API handles all business data (skills, metrics, tags)
- UI handles all user preferences (favorites, theme, view mode)
- No need to sync favorites to server (they're personal UI state)

## 3D World Testing Seams

The 3D world visualization system has clear testing seams at multiple layers.

### 1. MemberProvider (Dependency Injection)

**Location:** `ui/src/components/world/MemberProvider.tsx`

**Purpose:** Enables swapping member implementations for testing or alternative visualizations.

**Interface:**
```typescript
interface MemberConfig {
  Component: React.ComponentType<MemberProps>
  preloadAssets?: () => Promise<void>
  displayName: string
  description?: string
}

type MemberRegistry = Record<string, MemberConfig>
```

**Testing Strategy:**
- Register mock member components via `registerMember()`
- Test different member implementations in isolation
- Verify DI context propagation

**Example:**
```typescript
import { registerMember, MemberProvider } from '@/components/world'

const MockMember = ({ onMemberClick }: MemberProps) => (
  <mesh onClick={onMemberClick}>
    <boxGeometry args={[1, 1, 1]} />
  </mesh>
)

registerMember('mock', {
  Component: MockMember,
  displayName: 'Test Member',
})

// Use in tests
<MemberProvider member="mock">{children}</MemberProvider>
```

### 2. Zustand Stores (State Injection)

**Locations:**
- `ui/src/stores/cameraStore.ts`
- `ui/src/stores/graphicsStore.ts`
- `ui/src/stores/interactionStore.ts`
- `ui/src/stores/accessoryStore.ts`
- `ui/src/stores/environmentStore.ts`

**Purpose:** Centralized state management with testing hooks.

**Testing Strategy:**
- Access stores directly for state manipulation
- Use store's `reset()` methods between tests
- Mock initial state via store creation

**Example:**
```typescript
import { useCameraStore } from '@/stores/cameraStore'

beforeEach(() => {
  // Reset to default state
  useCameraStore.setState({
    position: [0, 15, 15],
    target: [0, 0, 0],
    mode: 'freeform',
    focusedMemberId: null,
  })
})

test('zooms to member', () => {
  const { zoomToMember } = useCameraStore.getState()
  zoomToMember('member-1', [5, 0, 5])

  const state = useCameraStore.getState()
  expect(state.mode).toBe('zoomed-member')
  expect(state.focusedMemberId).toBe('member-1')
})
```

### 3. MaterialProvider (Material Cache)

**Location:** `ui/src/components/world/materials/MaterialProvider.tsx`

**Purpose:** Caches material instances for memory efficiency.

**Interface:**
```typescript
interface MaterialCache {
  getMaterial: (preset: PresetName, color: string) => THREE.Material
  clear: () => void
  stats: () => { count: number; presets: string[] }
}
```

**Testing Strategy:**
- Use `stats()` to verify caching behavior
- Use `clear()` for cleanup between tests
- Test outside provider for fallback behavior

### 4. Graphics Tier System

**Location:** `ui/src/stores/graphicsStore.ts`, `ui/src/config/graphics.ts`

**Purpose:** Configurable rendering quality tiers.

**Testing Strategy:**
- Set specific tiers to test feature flags
- Verify component behavior at each tier
- Test tier auto-detection logic

**Example:**
```typescript
import { useGraphicsStore } from '@/stores/graphicsStore'

test('disables post-processing on low tier', () => {
  useGraphicsStore.getState().setTier('low')
  const { config } = useGraphicsStore.getState()

  expect(config.postProcessing).toBe(false)
  expect(config.shadows).toBe(false)
})
```

### 5. React Three Fiber Testing

**Testing Strategy:**
- Use `@react-three/test-renderer` for isolated component tests
- Mock R3F hooks for unit tests
- Use actual Canvas for integration tests

**Example:**
```typescript
import ReactThreeTestRenderer from '@react-three/test-renderer'
import { GeometricMember } from '@/components/world'

test('renders member at position', async () => {
  const renderer = await ReactThreeTestRenderer.create(
    <GeometricMember
      position={[0, 0, 0]}
      cursorPosition={null}
      selectedNodes={[]}
      isAnimating={false}
    />
  )

  const group = renderer.scene.children[0]
  expect(group.position.toArray()).toEqual([0, 0, 0])
})
```

For detailed architecture documentation, see [DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md].

---

## Related Skills

- `seam-discovery-and-enforcement.md` - The skill that guided this architecture
- `boundary-of-responsibility-enforcement.md` - Ensuring clean separation
- `screaming-architecture-audit.md` - Making domain concepts visible in folder structure

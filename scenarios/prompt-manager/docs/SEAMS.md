# Testing Seams in prompt-manager

This document describes the testing seams (interfaces and dependency injection points) in the prompt-manager API and UI. These seams enable unit testing without external dependencies like databases or the filesystem.

## Overview

The API is organized into domain-driven packages:
- `prompts/` - Core prompt management (primary domain)
- `metrics/` - Usage tracking and ratings
- `tags/` - Tag categorization
- `testing/` - LLM-based prompt testing via Ollama

Each package follows the same pattern:
1. **Models** - Type definitions
2. **Interfaces** - Contracts for testing seams (in prompts/interfaces.go)
3. **Repository/Store** - Data access implementations
4. **Query** - Domain logic for filtering and transformations
5. **Handlers** - HTTP request handling (depends on interfaces, not concrete types)

## Testing Seams

### 1. prompts.PromptStore (Interface)

**Location:** `api/prompts/interfaces.go`

**Purpose:** Interface for prompt storage operations - the primary testing seam.

**Interface:**
```go
// PromptStore defines the interface for prompt storage operations.
// Implementations: Store (file-based, production), MockStore (testing).
type PromptStore interface {
    GetAll() ([]Metadata, error)
    FindByID(id string) (*Metadata, string, error)
    LoadMetadata(folder string) ([]Metadata, error)
    SaveMetadata(folder string, prompts []Metadata) error
    GetContent(folder, filename string) (string, error)
    SaveContent(folder, filename, content string) error
    DeleteContent(folder, filename string) error
}
```

**Implementation:** `api/prompts/store.go` (production, file-based)

**Testing Strategy:**
- **Unit tests:** Create a mock implementing `PromptStore`
- **Integration tests:** Create a temp directory with test fixtures, pass to `NewStore(tempDir)`

**Example (Mock):**
```go
type MockStore struct {
    prompts map[string][]prompts.Metadata
}

func (m *MockStore) GetAll() ([]prompts.Metadata, error) {
    var all []prompts.Metadata
    for _, p := range m.prompts {
        all = append(all, p...)
    }
    return all, nil
}
// ... implement other methods

func TestPromptHandlers(t *testing.T) {
    mockStore := &MockStore{prompts: testData}
    mockMetrics := &MockMetricsService{}
    handlers := prompts.NewHandlers(mockStore, mockMetrics)
    // Test handlers without filesystem
}
```

**Example (Integration):**
```go
func TestPromptCreate(t *testing.T) {
    tempDir := t.TempDir()
    store := prompts.NewStore(tempDir)

    // Test prompt creation with real filesystem
    // ...
}
```

### 2. prompts.MetricsService (Interface)

**Location:** `api/prompts/interfaces.go`

**Purpose:** Interface for metrics operations used by prompt handlers - enables testing without database.

**Interface:**
```go
// MetricsService defines the interface for prompt metrics operations.
// Implementations: MetricsAdapter (wraps metrics.Repository), MockMetricsService (testing).
type MetricsService interface {
    Get(promptID string) (*PromptMetrics, error)
    RecordUsage(promptID string) (int, time.Time, error)
    SetRating(promptID string, rating int, notes *string) error
    Delete(promptID string) error
}
```

**Implementation:** `api/prompts/metrics_adapter.go` (production, wraps metrics.Repository)

**Testing Strategy:**
- Create a mock implementing `MetricsService`
- No database access needed for testing handler logic

**Example:**
```go
type MockMetricsService struct {
    metrics map[string]*prompts.PromptMetrics
}

func (m *MockMetricsService) Get(promptID string) (*prompts.PromptMetrics, error) {
    if pm, ok := m.metrics[promptID]; ok {
        return pm, nil
    }
    return nil, nil // Not found is not an error
}

func (m *MockMetricsService) RecordUsage(promptID string) (int, time.Time, error) {
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

func (r *Repository) Get(promptID string) (*PromptMetrics, error)
func (r *Repository) RecordUsage(promptID string) (int, time.Time, error)
func (r *Repository) SetRating(promptID string, rating int, notes *string) error
func (r *Repository) Delete(promptID string) error
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

    mock.ExpectQuery("INSERT INTO prompt_metrics").
        WithArgs("prompt-1").
        WillReturnRows(sqlmock.NewRows([]string{"usage_count", "last_used"}).
            AddRow(1, time.Now()))

    repo := metrics.NewRepository(db)
    count, _, err := repo.RecordUsage("prompt-1")

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
func (r *Repository) GetHistory(promptID string, limit int) ([]TestResult, error)
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
func TestPromptTesting(t *testing.T) {
    // Mock Ollama server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(testing.OllamaResponse{
            Response:  "Test response",
            EvalCount: 50,
        })
    }))
    defer server.Close()

    client := testing.NewOllamaClient(server.URL)
    // Test prompt testing
}
```

## Dependency Wiring in main.go

The main.go file is a thin bootstrap layer that wires dependencies using the adapter pattern:

```go
// Initialize domain components (seams for testing)
promptStore := prompts.NewStore(promptsDir)
metricsRepo := metrics.NewRepository(db)
tagsRepo := tags.NewRepository(db)
testingRepo := testing.NewRepository(db)
ollamaClient := testing.NewOllamaClient(ollamaURL)

// Create adapter to bridge metrics.Repository -> prompts.MetricsService interface
metricsAdapter := prompts.NewMetricsAdapter(metricsRepo)

// Initialize handlers with interface adapters
promptHandlers := prompts.NewHandlers(promptStore, metricsAdapter)
tagsHandlers := tags.NewHandlers(tagsRepo)
testingHandlers := testing.NewHandlers(testingRepo, ollamaClient, promptStore)
```

**Key Design Decision:** The `promptHandlers` depends on interfaces (`PromptStore`, `MetricsService`), not concrete types. This enables:
- Unit testing handlers without filesystem or database
- Swapping implementations (e.g., in-memory store for testing)
- Clear boundary between handlers and data access

## Domain Logic (query.go)

**Location:** `api/prompts/query.go`

Filtering and transformation logic is separated from handlers into dedicated domain functions:

```go
// FilterOptions defines criteria for filtering prompts.
type FilterOptions struct {
    Tag    string
    Folder string
    Modes  []string
}

// Filter applies all filter criteria to a list of prompts.
func Filter(prompts []Metadata, opts FilterOptions) []Metadata

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
    prompts := []prompts.Metadata{
        {ID: "1", Tags: []string{"skill"}},
        {ID: "2", Tags: []string{"template"}},
    }

    filtered := prompts.Filter(prompts, prompts.FilterOptions{Tag: "skill"})

    assert.Len(t, filtered, 1)
    assert.Equal(t, "1", filtered[0].ID)
}
```

## Testing the Handlers

Handlers receive dependencies via constructor injection:

```go
// In tests, inject mocks
func TestPromptHandlers(t *testing.T) {
    store := NewMockStore()       // Mock prompt store
    metricsRepo := NewMockRepo()  // Mock metrics repository

    handlers := prompts.NewHandlers(store, metricsRepo)

    // Create test request
    req := httptest.NewRequest("GET", "/api/v1/prompts", nil)
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
  async getPrompts(filters?: SearchFilters): Promise<Prompt[]>
  async getPromptsByFolder(folder: FolderType): Promise<Prompt[]>
  async getPrompt(id: string): Promise<Prompt>
  async createPrompt(prompt: CreatePromptRequest): Promise<Prompt>
  async updatePrompt(id: string, data: UpdatePromptRequest): Promise<Prompt>
  async deletePrompt(id: string): Promise<void>
  async testPrompt(id: string, request: PromptTestRequest): Promise<PromptTestResult>
  async searchPrompts(query: string, filters?: SearchFilters): Promise<Prompt[]>
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
  rest.get('/api/v1/prompts', (req, res, ctx) => {
    return res(ctx.json([{ id: '1', name: 'Test', folder: 'local', ... }]))
  })
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

test('fetches prompts', async () => {
  const prompts = await api.getPrompts()
  expect(prompts).toHaveLength(1)
})
```

### 2. usePrompts Hook (State Management)

**Location:** `ui/src/hooks/use-prompts.tsx`

**Purpose:** Encapsulates all prompt-related state management (selection, filtering, search)

**Interface:**
```typescript
interface UsePromptsReturn {
  // Data
  folders: Folder[]
  filteredPrompts: Prompt[]
  sidebarCounts: SidebarCounts

  // Selection state
  selectedFolder: Folder | null
  selectedPrompt: Prompt | null

  // Filter state
  viewFilter: ViewFilter
  searchQuery: string
  filterInfo: FilterInfo | null

  // Loading states
  isLoading: boolean
  foldersLoading: boolean

  // Actions
  setSelectedFolder: (folder: Folder | null) => void
  setSelectedPrompt: (prompt: Prompt | null) => void
  setViewFilter: (filter: ViewFilter) => void
  setSearchQuery: (query: string) => void
  handleFilterChange: (filter: ViewFilter) => void

  // Computed
  showPromptList: boolean
}
```

**Testing Strategy:**
- Wrap in `QueryClientProvider` with mocked `api`
- Use `@testing-library/react-hooks` for isolated hook testing

**Example:**
```typescript
import { renderHook, waitFor } from '@testing-library/react'
import { usePrompts } from '@/hooks/use-prompts'

const wrapper = ({ children }) => (
  <QueryClientProvider client={testQueryClient}>
    {children}
  </QueryClientProvider>
)

test('filters prompts by favorites', async () => {
  const favorites = new Set(['prompt-1'])
  const { result } = renderHook(() => usePrompts({ favorites }), { wrapper })

  act(() => {
    result.current.handleFilterChange('favorites')
  })

  await waitFor(() => {
    expect(result.current.filteredPrompts).toEqual(
      expect.arrayContaining([expect.objectContaining({ id: 'prompt-1' })])
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

test('renders prompts', async () => {
  const { getByText } = renderWithClient(<App />)
  await waitFor(() => expect(getByText('Test Prompt')).toBeInTheDocument())
})
```

## Boundary of Responsibility

The UI and API have clear boundaries:

| Concern | Owner | Storage |
|---------|-------|---------|
| Prompt content & metadata | API | File system (metadata.json + .md) |
| Usage metrics | API | PostgreSQL |
| Tags | API | PostgreSQL |
| Test results | API | PostgreSQL |
| Favorites | UI | localStorage |
| View preferences | UI | Component state |
| Theme | UI | localStorage |

This separation means:
- API handles all business data (prompts, metrics, tags)
- UI handles all user preferences (favorites, theme, view mode)
- No need to sync favorites to server (they're personal UI state)

## Related Skills

- `seam-discovery-and-enforcement.md` - The skill that guided this architecture
- `boundary-of-responsibility-enforcement.md` - Ensuring clean separation
- `screaming-architecture-audit.md` - Making domain concepts visible in folder structure

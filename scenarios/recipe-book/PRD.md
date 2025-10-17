# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Recipe-book adds a comprehensive household culinary intelligence system that stores, organizes, and learns from recipe data. It provides semantic search, AI-powered recipe generation and modification, multi-tenant recipe management with privacy controls, and becomes the central culinary memory for household and lifestyle scenarios.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability provides agents with deep understanding of household food preferences, dietary restrictions (via contact-book integration), cooking patterns, and ingredient relationships. Agents can query semantic recipe knowledge, generate meal plans based on available ingredients, and learn from family cooking habits to make increasingly personalized suggestions.

### Recursive Value
**What new scenarios become possible after this exists?**
1. **grocery-optimizer** - Generates shopping lists based on planned meals and pantry inventory
2. **nutrition-tracker** - Calculates accurate nutrition from actual recipes cooked
3. **date-night-generator** - Suggests restaurants/recipes matching couple's shared favorites
4. **meal-prep-assistant** - Optimizes batch cooking based on recipe compatibility
5. **kitchen-inventory-manager** - Tracks ingredients and suggests recipes before expiration

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Store and retrieve recipes with full CRUD operations
  - [ ] Multi-tenant support with scenario-authenticator integration
  - [ ] Semantic recipe search using Qdrant embeddings
  - [ ] Basic privacy controls (public/private/shared recipes)
  - [ ] API endpoints for cross-scenario recipe access
  - [ ] CLI commands for all recipe operations
  - [ ] Beautiful, responsive UI with cozy cookbook aesthetic
  
- **Should Have (P1)**
  - [ ] AI-powered recipe generation via shared ollama.json workflow
  - [ ] Recipe modification (make vegan/keto/instant-pot friendly)
  - [ ] Family member taste profiles from contact-book
  - [ ] Recipe ratings and popularity tracking
  - [ ] Shopping list generation with smart grouping
  - [ ] Nutritional information calculation
  - [ ] Recipe photo upload and storage
  
- **Nice to Have (P2)**
  - [ ] Voice-controlled "Cook Mode" with large display
  - [ ] Meal planning calendar with household view
  - [ ] Recipe inheritance and family stories
  - [ ] Seasonal recipe recommendations
  - [ ] "Use up leftovers" suggestions
  - [ ] Recipe scheduling (auto-reveal after date)
  - [ ] Kid-safe view filtering

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for 95% of requests | API monitoring |
| Search Speed | < 500ms for semantic search | Qdrant query logs |
| Throughput | 100 operations/second | Load testing |
| Embedding Accuracy | > 85% relevance for semantic search | User feedback |
| Resource Usage | < 512MB memory, < 10% CPU | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with postgres, qdrant, scenario-authenticator
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store recipe data, user preferences, ratings
    integration_pattern: Direct database connection via API
    access_method: API database/sql package
    
  - resource_name: qdrant
    purpose: Semantic search via recipe embeddings
    integration_pattern: Vector similarity search
    access_method: resource-qdrant CLI commands
    
  - resource_name: scenario-authenticator
    purpose: Multi-tenant authentication and user profiles
    integration_pattern: API authentication middleware
    access_method: API endpoints for auth verification
    
optional:
  - resource_name: ollama
    purpose: AI recipe generation and modification
    fallback: Manual recipe entry only
    access_method: Shared workflow initialization/n8n/ollama.json
    
  - resource_name: minio
    purpose: Store recipe photos
    fallback: No photo support
    access_method: resource-minio CLI commands
    
  - resource_name: redis
    purpose: Cache popular recipes and search results
    fallback: Direct database queries
    access_method: resource-redis CLI commands
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: AI recipe generation and modification
    - workflow: embedding-generator.json
      location: initialization/automation/n8n/
      purpose: Generate embeddings for semantic search
  
  2_resource_cli:
    - command: resource-qdrant upsert
      purpose: Store recipe embeddings
    - command: resource-qdrant search
      purpose: Semantic recipe search
    - command: resource-minio upload
      purpose: Store recipe photos
  
  3_direct_api:
    - justification: Database operations require direct connection
      endpoint: postgres connection via database/sql

shared_workflow_criteria:
  - ollama.json already exists and handles LLM calls
  - embedding-generator.json will be shared across scenarios
  - Both are truly reusable for any text processing need
```

### Data Models
```yaml
primary_entities:
  - name: Recipe
    storage: postgres
    schema: |
      {
        id: UUID
        title: string
        description: string
        ingredients: []Ingredient
        instructions: []string
        prep_time: int (minutes)
        cook_time: int (minutes)
        servings: int
        tags: []string
        cuisine: string
        dietary_info: []string
        nutrition: NutritionInfo
        photo_url: string
        created_by: UUID (user_id)
        created_at: timestamp
        updated_at: timestamp
        visibility: enum (public, private, shared)
        shared_with: []UUID
        source: string (original, ai_generated, modified)
        parent_recipe_id: UUID (for modifications)
      }
    relationships: Links to User, Ratings, MealPlans
    
  - name: RecipeRating
    storage: postgres
    schema: |
      {
        id: UUID
        recipe_id: UUID
        user_id: UUID
        rating: int (1-5)
        notes: string
        cooked_date: timestamp
        anonymous: boolean
      }
    relationships: Links to Recipe, User
    
  - name: RecipeEmbedding
    storage: qdrant
    schema: |
      {
        recipe_id: UUID
        embedding: vector[768]
        metadata: {
          title: string
          tags: []string
          cuisine: string
          dietary: []string
        }
      }
    relationships: References Recipe by ID
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/recipes
    purpose: List recipes with filtering and pagination
    input_schema: |
      {
        user_id?: UUID
        visibility?: string
        tags?: []string
        cuisine?: string
        dietary?: []string
        limit?: int
        offset?: int
      }
    output_schema: |
      {
        recipes: []Recipe
        total: int
        has_more: boolean
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/recipes/search
    purpose: Semantic search for recipes
    input_schema: |
      {
        query: string
        user_id: UUID
        limit?: int
        filters?: {
          dietary?: []string
          max_time?: int
          ingredients?: []string
        }
      }
    output_schema: |
      {
        results: []RecipeSearchResult
        query_interpretation: string
      }
    sla:
      response_time: 500ms
      availability: 99%
      
  - method: POST
    path: /api/v1/recipes/generate
    purpose: AI-powered recipe generation
    input_schema: |
      {
        prompt: string
        user_id: UUID
        dietary_restrictions?: []string
        available_ingredients?: []string
        style?: string
      }
    output_schema: |
      {
        recipe: Recipe
        confidence: float
        alternatives: []RecipeSuggestion
      }
    sla:
      response_time: 5000ms
      availability: 95%
      
  - method: POST
    path: /api/v1/recipes/{id}/modify
    purpose: Modify recipe for dietary needs
    input_schema: |
      {
        modification_type: string (vegan, keto, gluten_free, etc)
        user_id: UUID
      }
    output_schema: |
      {
        modified_recipe: Recipe
        changes_made: []string
      }
    sla:
      response_time: 3000ms
      availability: 95%
```

### Event Interface
```yaml
published_events:
  - name: recipe.created
    payload: {recipe_id, user_id, tags}
    subscribers: [nutrition-tracker, grocery-optimizer]
    
  - name: recipe.cooked
    payload: {recipe_id, user_id, rating, date}
    subscribers: [meal-history-analyzer, nutrition-tracker]
    
consumed_events:
  - name: authenticator.user.created
    action: Initialize user recipe preferences
    
  - name: contact.dietary_restriction.updated
    action: Update recipe recommendations cache
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: recipe-book
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show recipe-book service status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: list
    description: List recipes with filters
    api_endpoint: /api/v1/recipes
    arguments:
      - name: --user
        type: string
        required: false
        description: Filter by user ID
    flags:
      - name: --private
        description: Show only private recipes
      - name: --dietary
        description: Filter by dietary restriction
      - name: --json
        description: Output as JSON
    output: List of recipes in table or JSON format
    
  - name: search
    description: Semantic recipe search
    api_endpoint: /api/v1/recipes/search
    arguments:
      - name: query
        type: string
        required: true
        description: Natural language search query
    flags:
      - name: --max-time
        description: Maximum cooking time in minutes
      - name: --dietary
        description: Dietary restrictions to apply
    output: Ranked list of matching recipes
    
  - name: generate
    description: Generate recipe from prompt
    api_endpoint: /api/v1/recipes/generate
    arguments:
      - name: prompt
        type: string
        required: true
        description: Recipe generation prompt
    flags:
      - name: --ingredients
        description: Available ingredients (comma-separated)
      - name: --style
        description: Cooking style preference
    output: Generated recipe with confidence score
    
  - name: cook
    description: Mark recipe as cooked and rate
    api_endpoint: /api/v1/recipes/{id}/cook
    arguments:
      - name: recipe-id
        type: string
        required: true
        description: Recipe UUID
      - name: rating
        type: int
        required: false
        description: Rating 1-5
    flags:
      - name: --notes
        description: Cooking notes
      - name: --anonymous
        description: Don't reveal who cooked it
    output: Confirmation with updated stats
```

### CLI-API Parity Requirements
- Every API endpoint has corresponding CLI command
- CLI uses kebab-case (list-recipes → /api/v1/recipes)
- All parameters available via arguments or flags
- JSON output via --json flag for scripting
- Authentication via RECIPE_BOOK_USER env var or --user flag

### Implementation Standards
```yaml
implementation_requirements:
  architecture: Thin wrapper over lib/ functions
  language: Go for consistency
  dependencies: Reuse API client libraries
  error_handling: Exit codes (0=success, 1=error)
  configuration: 
    - ~/.vrooli/recipe-book/config.yaml
    - Environment variables override
    - Command flags highest priority
  
installation:
  - Create symlink in ~/.vrooli/bin/
  - Add to PATH if needed
  - Set 755 permissions
  - Generate comprehensive --help
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **scenario-authenticator**: User profiles and multi-tenancy
- **contact-book**: Dietary restrictions and family member data
- **ollama (via shared workflow)**: AI recipe generation
- **embedding-generator workflow**: Vector embeddings for search

### Downstream Enablement
- **grocery-optimizer**: Shopping lists from meal plans
- **nutrition-tracker**: Accurate nutrition from cooked recipes
- **date-night-generator**: Restaurant suggestions matching tastes
- **meal-prep-assistant**: Batch cooking optimization
- **kitchen-inventory**: Recipe suggestions from available ingredients

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: nutrition-tracker
    capability: Actual recipes cooked with portions
    interface: API events and queries
    
  - scenario: grocery-optimizer
    capability: Ingredient lists from meal plans
    interface: API batch ingredient query
    
  - scenario: date-night-generator
    capability: Food preference profiles
    interface: API preference scoring
    
consumes_from:
  - scenario: contact-book
    capability: Dietary restrictions and preferences
    fallback: Manual dietary entry
    
  - scenario: scenario-authenticator
    capability: User authentication and profiles
    fallback: Single-user mode
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: creative
  inspiration: "Grandma's recipe box meets modern app"
  
  visual_style:
    color_scheme: custom
    primary_colors:
      - warm_terracotta: "#C65D00"
      - sage_green: "#87A96B"
      - parchment: "#FAF0E6"
      - deep_burgundy: "#800020"
    typography: 
      - headers: "Kalam or Caveat (handwritten)"
      - body: "Merriweather (clean serif)"
      - ui: "Inter (modern sans)"
    layout: spacious
    animations: playful
    
  personality:
    tone: friendly
    mood: cozy
    target_feeling: "Warm kitchen on a Sunday morning"

style_references:
  creative:
    - study-buddy: "Cozy, approachable aesthetic"
    - notes: "Clean, focused interface"
  unique_elements:
    - Recipe cards with worn edges
    - Hand-drawn vegetable illustrations
    - Polaroid photo frames
    - Animated cooking timers
    - Chef hat rating system
```

### Target Audience Alignment
- **Primary Users**: Home cooks, families, meal planners
- **User Expectations**: Pinterest-meets-cookbook aesthetic
- **Accessibility**: WCAG AA compliance, voice control option
- **Responsive Design**: Mobile-first for kitchen use

### Brand Consistency Rules
- Must feel warm and inviting, not clinical
- Should evoke nostalgia while being modern
- Professional enough for meal planning, fun enough for kids
- Consistent with Vrooli's "old-internet spirit" philosophy

## 💰 Value Proposition

### Business Value
- **Primary Value**: Central culinary intelligence for household
- **Revenue Potential**: $15K - $30K per deployment (family subscription service)
- **Cost Savings**: 2-3 hours/week meal planning, 20% grocery savings
- **Market Differentiator**: AI + privacy + family sharing unique combo

### Technical Value
- **Reusability Score**: 9/10 - Foundation for all food/lifestyle scenarios
- **Complexity Reduction**: Natural language to structured recipes
- **Innovation Enablement**: Enables entire ecosystem of food-related apps

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core recipe CRUD with multi-tenancy
- Semantic search via Qdrant
- Basic AI generation
- Beautiful responsive UI
- Full API/CLI interface

### Version 2.0 (Planned)
- Voice-controlled cooking mode
- Meal planning calendar
- Advanced nutrition tracking
- Recipe video support
- Social recipe sharing network

### Long-term Vision
- Becomes the "GitHub of recipes" for Vrooli deployments
- Recipe DNA - tracks recipe evolution and variations
- Predictive meal suggestions based on weather, events, mood
- Integration with smart kitchen appliances

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with metadata
    - All initialization files
    - Deployment scripts
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose
    - kubernetes: Helm charts
    - cloud: Serverless functions
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        - free: 50 recipes
        - family: $9.99/month unlimited
        - chef: $19.99/month with AI features
    - trial_period: 30 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: recipe-book
    category: lifestyle
    capabilities:
      - recipe storage and retrieval
      - semantic recipe search
      - AI recipe generation
      - multi-tenant management
      - nutritional analysis
    interfaces:
      - api: http://localhost:3250/api/v1
      - cli: recipe-book
      - events: recipe.*
      
  metadata:
    description: "Intelligent recipe management with AI search and generation"
    keywords: [recipes, cooking, meal planning, nutrition, AI, search]
    dependencies: [scenario-authenticator, contact-book]
    enhances: [nutrition-tracker, grocery-optimizer, date-night-generator]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes: []
  
  deprecations: []
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Qdrant unavailable | Medium | High | Fallback to postgres full-text search |
| AI generation fails | Medium | Medium | Manual recipe entry always available |
| Photo storage fails | Low | Low | Graceful degradation, recipes work without photos |

### Operational Risks
- **Data Privacy**: Strict visibility controls, no recipe data leakage
- **Performance**: Caching layer for popular recipes
- **Scaling**: Postgres partitioning by user for large deployments

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: recipe-book

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/recipe-book
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/recipe-generator.json
    - scenario-test.yaml
    - ui/index.html
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/automation/n8n
    - initialization/storage/postgres
    - ui

resources:
  required: [postgres, qdrant, scenario-authenticator]
  optional: [ollama, minio, redis]
  health_timeout: 60

tests:
  - name: "Postgres is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "API creates recipe"
    type: http
    service: api
    endpoint: /api/v1/recipes
    method: POST
    body:
      title: "Test Recipe"
      ingredients: ["flour", "water"]
      instructions: ["Mix", "Bake"]
    expect:
      status: 201
      body:
        id: "*"
        
  - name: "Semantic search works"
    type: http
    service: api
    endpoint: /api/v1/recipes/search
    method: POST
    body:
      query: "something sweet for dessert"
    expect:
      status: 200
      body:
        results: "*"
        
  - name: "CLI lists recipes"
    type: exec
    command: ./cli/recipe-book list --json
    expect:
      exit_code: 0
      output_contains: ["recipes"]
      
  - name: "Recipe schema exists"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'recipes'"
    expect:
      rows:
        - count: 1
```

### Performance Validation
- [ ] API responses < 200ms for CRUD operations
- [ ] Semantic search < 500ms
- [ ] UI loads in < 2 seconds
- [ ] Supports 100 concurrent users

### Integration Validation
- [ ] Authenticates via scenario-authenticator
- [ ] Reads dietary restrictions from contact-book
- [ ] Generates recipes via ollama workflow
- [ ] Stores embeddings in Qdrant
- [ ] CLI mirrors all API functionality

### Capability Verification
- [ ] Stores and retrieves recipes accurately
- [ ] Semantic search returns relevant results
- [ ] Multi-tenant privacy controls work
- [ ] AI generation produces valid recipes
- [ ] UI provides delightful user experience

## 📝 Implementation Notes

### Design Decisions
**Embedding Model**: Using text-embedding-ada-002 compatible vectors (768 dimensions)
- Alternative considered: Smaller models for speed
- Decision driver: Quality of semantic search critical
- Trade-offs: Slightly slower for much better relevance

**UI Framework**: Vanilla JavaScript with modern CSS
- Alternative considered: React/Vue
- Decision driver: Simplicity and no build step
- Trade-offs: More manual work for rich interactions

### Known Limitations
- **Recipe Import**: No automated import from external sites (V1)
  - Workaround: Manual entry or AI generation from description
  - Future fix: Web scraper in V2
  
- **Offline Mode**: Requires connection for AI features
  - Workaround: Local recipe cache for viewing
  - Future fix: Local LLM support in V2

### Security Considerations
- **Data Protection**: Recipes encrypted at rest, careful visibility controls
- **Access Control**: Scenario-authenticator handles all auth
- **Audit Trail**: All recipe modifications logged with user and timestamp

## 🔗 References

### Documentation
- README.md - User guide and quick start
- docs/api.md - Complete API specification
- docs/cli.md - CLI command reference
- docs/cookbook.md - Recipe sharing best practices

### Related PRDs
- scenarios/scenario-authenticator/PRD.md
- scenarios/contact-book/PRD.md
- scenarios/nutrition-tracker/PRD.md (future)
- scenarios/grocery-optimizer/PRD.md (future)

### External Resources
- Qdrant documentation for vector search
- OpenAI Cookbook for embedding best practices
- USDA nutrition database for food data

---

**Last Updated**: 2025-09-09  
**Status**: Draft  
**Owner**: AI Agent (Claude)  
**Review Cycle**: Weekly validation against implementation
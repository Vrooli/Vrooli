# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Adds intelligent, age-appropriate story generation and interactive reading capability with time-aware UI that adapts to create healthy bedtime routines. This provides a creative content generation system specifically optimized for children's developmental needs and parental peace of mind.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability establishes patterns for:
- Age-appropriate content filtering and generation
- Time-context-aware UI adaptations
- Interactive reading experiences with progress tracking
- Safe AI content generation with strict guardrails
- Parent-approved content libraries

### Recursive Value
**What new scenarios become possible after this exists?**
1. **educational-content-generator** - Builds on story generation to create age-appropriate lessons
2. **reading-progress-tracker** - Analyzes reading patterns to suggest difficulty progression
3. **family-activity-planner** - Uses bedtime routine data to optimize family schedules
4. **speech-therapy-assistant** - Leverages story reading for pronunciation practice
5. **creativity-workshop** - Extends story generation for collaborative family storytelling

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Generate age-appropriate bedtime stories using Ollama
  - [x] Store generated stories in PostgreSQL with metadata
  - [x] Display stories in book-like UI with page turning
  - [x] Time-aware room lighting (sunny/lamp/nightlight)
  - [x] Kid-friendly tag for kids-dashboard discovery
  - [x] Reading history tracking per story
  
- **Should Have (P1)**
  - [ ] Story themes/genres selection (adventure, animals, fantasy, etc.)
  - [ ] Character name customization
  - [ ] Reading time estimates
  - [ ] Bookmark/favorite stories feature
  - [ ] Story length options (5, 10, 15 minute stories)
  
- **Nice to Have (P2)**
  - [ ] Text-to-speech reading capability
  - [ ] Illustration generation for stories
  - [ ] Parent dashboard for content review
  - [ ] Export stories as PDF books

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Story Generation Time | < 5s for standard story | API response timing |
| UI Load Time | < 2s | Browser performance API |
| Story Retrieval | < 200ms | Database query timing |
| Memory Usage | < 200MB | System monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration with kids-dashboard verified
- [x] Age-appropriate content validation passes
- [x] Time-based UI transitions work correctly
- [x] Stories persist and retrieve correctly

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store stories, reading history, and user preferences
    integration_pattern: Direct SQL via Go database/sql
    access_method: Environment variables for connection
    
  - resource_name: ollama
    purpose: Generate bedtime stories with age-appropriate content
    integration_pattern: CLI via resource-ollama commands
    access_method: resource-ollama generate with custom prompts
    
optional:
  - resource_name: redis
    purpose: Cache frequently read stories for faster access
    fallback: Direct PostgreSQL queries
    access_method: Resource CLI when available
```

### Data Models
```yaml
primary_entities:
  - name: Story
    storage: postgres
    schema: |
      {
        id: UUID
        title: string
        content: text (markdown)
        age_group: enum (3-5, 6-8, 9-12)
        theme: string
        reading_time_minutes: integer
        character_names: jsonb
        created_at: timestamp
        times_read: integer
        last_read: timestamp
        is_favorite: boolean
      }
    relationships: Has many ReadingSessions
    
  - name: ReadingSession
    storage: postgres
    schema: |
      {
        id: UUID
        story_id: UUID (FK)
        started_at: timestamp
        completed_at: timestamp
        pages_read: integer
        total_pages: integer
      }
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/stories/generate
    purpose: Generate a new bedtime story
    input_schema: |
      {
        age_group: "3-5" | "6-8" | "9-12"
        theme?: string
        length?: "short" | "medium" | "long"
        character_names?: string[]
      }
    output_schema: |
      {
        story_id: UUID
        title: string
        content: string
        pages: number
        reading_time: number
      }
    sla:
      response_time: 5000ms
      availability: 99%
      
  - method: GET
    path: /api/v1/stories
    purpose: List all stories with filtering
    
  - method: GET
    path: /api/v1/stories/:id
    purpose: Retrieve a specific story
    
  - method: POST
    path: /api/v1/stories/:id/read
    purpose: Track reading session
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: bedtime-story
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show service health and stats
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: generate
    description: Generate a new bedtime story
    api_endpoint: /api/v1/stories/generate
    arguments:
      - name: age-group
        type: string
        required: true
        description: Age group (3-5, 6-8, 9-12)
    flags:
      - name: --theme
        description: Story theme (adventure, animals, etc.)
      - name: --length
        description: Story length (short, medium, long)
    output: Story title and ID
    
  - name: read
    description: Display a story for reading
    arguments:
      - name: story-id
        type: string
        required: true
    output: Formatted story content
    
  - name: list
    description: List available stories
    flags:
      - name: --age-group
        description: Filter by age group
      - name: --favorites
        description: Show only favorites
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: playful
  inspiration: Classic children's bedroom with storybook aesthetics
  
  visual_style:
    color_scheme: custom (time-adaptive)
    typography: playful (rounded, friendly fonts)
    layout: single-page (immersive room view)
    animations: playful (page turns, gentle transitions)
  
  personality:
    tone: friendly
    mood: calm and cozy
    target_feeling: Safe, magical, sleepy

time_based_themes:
  daytime: # 6am - 6pm
    lighting: bright, sunny
    colors: warm yellows, sky blues
    shadows: soft, natural
    
  evening: # 6pm - 9pm  
    lighting: warm lamp glow
    colors: orange, amber, soft browns
    shadows: longer, cozier
    
  nighttime: # 9pm - 6am
    lighting: nightlight only
    colors: deep blues, purples, soft greens
    shadows: minimal, dreamy
    
ui_elements:
  bookshelf:
    style: wooden, whimsical
    organization: by favorites, recent, age group
    interaction: click to pull book out
    
  book_viewer:
    style: realistic open book
    pages: textured paper appearance
    animations: page flip with sound option
    typography: large, clear, child-friendly
    
  room_decorations:
    toys: stuffed animals, blocks
    furniture: bed, reading chair, nightstand
    window: shows time-appropriate outdoor scene
```

### Target Audience Alignment
- **Primary Users**: Children ages 3-12 and their parents
- **User Expectations**: Safe, engaging, calming bedtime experience
- **Accessibility**: WCAG AA compliance, dyslexia-friendly fonts option
- **Responsive Design**: Tablet-first, with mobile and desktop support

## 💰 Value Proposition

### Business Value
- **Primary Value**: Healthy bedtime routines for families
- **Revenue Potential**: $5K - $15K per deployment (family subscription model)
- **Cost Savings**: 30-60 minutes saved per bedtime routine
- **Market Differentiator**: Time-aware UI with educational value

### Technical Value
- **Reusability Score**: High - story generation can power many education scenarios
- **Complexity Reduction**: Makes AI content generation safe for children
- **Innovation Enablement**: Foundation for personalized children's content

## 🔄 Integration Requirements

### Upstream Dependencies
- **kids-dashboard**: Must be discoverable via kid-friendly tag
- **ollama resource**: Required for story generation

### Downstream Enablement
- **educational-content-generator**: Can reuse story generation patterns
- **reading-progress-tracker**: Can analyze reading session data
- **text-to-speech scenarios**: Can use stories as content source

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: kids-dashboard
    capability: Interactive story reading experience
    interface: Web UI iframe embedding
    
  - scenario: education-tracker
    capability: Reading progress data
    interface: API/Database
    
consumes_from:
  - scenario: kids-dashboard
    capability: User session and preferences
    fallback: Standalone mode with defaults
```

## ✅ Validation Criteria

### Test Specification
```yaml
tests:
  - name: "Story generation completes successfully"
    type: api
    endpoint: /api/v1/stories/generate
    method: POST
    body:
      age_group: "6-8"
      theme: "adventure"
    expect:
      status: 201
      response_time: < 5000ms
      
  - name: "Time-based UI theme changes"
    type: ui
    validation: Screenshot verification at different times
    
  - name: "Kid-friendly tag present"
    type: config
    file: .vrooli/service.json
    expect:
      contains: "kid-friendly"
```

## 📝 Implementation Notes

### Design Decisions
**Direct Ollama Integration**: Chose CLI-based Ollama calls over n8n workflows for 10x faster response times and simpler architecture

**Time-based Theming**: Using system time vs user preference to naturally encourage bedtime routines

**PostgreSQL for Stories**: Chose PostgreSQL over file storage for better querying, history tracking, and future analytics

### Known Limitations
- **Story Illustrations**: V1 uses emoji and text only, images in V2
- **Offline Mode**: Requires Ollama connection for generation
- **Language**: English only in V1

### Security Considerations
- **Content Filtering**: Strict prompts ensure age-appropriate content
- **No User Accounts**: Stories stored locally only
- **Parent Controls**: No external content sources

---

**Last Updated**: 2025-01-06  
**Status**: In Development  
**Owner**: Vrooli Team  
**Review Cycle**: After each story generation update
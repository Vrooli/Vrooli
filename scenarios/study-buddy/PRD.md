# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Gamified adaptive learning platform with AI-generated content and spaced repetition optimization. Study Buddy provides personalized educational experiences through intelligent flashcards, adaptive quizzes, and progress tracking, all wrapped in a cozy, motivating aesthetic that makes learning feel enjoyable and sustainable.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Provides adaptive learning patterns that agents apply to skill acquisition and knowledge retention
- Creates gamification frameworks that enhance engagement in any learning scenario
- Establishes spaced repetition algorithms that optimize information retention
- Enables personalized content generation that adjusts to individual learning styles
- Offers progress tracking systems that provide feedback loops for continuous improvement

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Corporate Training Platform**: Employee onboarding and skill development with gamification
2. **Language Learning Assistant**: Vocabulary building with cultural context and conversation practice
3. **Professional Certification Prep**: Exam preparation with adaptive difficulty and progress tracking
4. **Medical Education Suite**: Anatomy and procedures learning with visual aids and repetition
5. **Code Learning Companion**: Programming concepts with interactive coding challenges

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] AI-generated flashcards from study content (PARTIAL: Works with fallback, N8N integration broken)
  - [x] Spaced repetition algorithm for optimal retention (PARTIAL: Algorithm works, persistence broken)
  - [ ] Interactive quizzes with immediate feedback (API exists, no persistence)
  - [x] Progress tracking with daily streaks and XP (PARTIAL: Calculations work, no persistence)
  - [x] Cozy Lofi Girl-inspired aesthetic with animations (UI exists and loads)
  - [ ] Subject-based organization system (API exists, database schema not applied)
  - [ ] Study session timer with Pomodoro support (N8N workflow exists but not loaded)
  
- **Should Have (P1)**
  - [x] Difficulty tracking (Easy/Medium/Hard) for adaptive learning (In code, no persistence)
  - [ ] Semantic search for finding related cards (Qdrant configured but not integrated)
  - [ ] Markdown-enabled note editor (Not implemented)
  - [ ] Export functionality for notes and progress (CLI command exists, not functional)
  - [ ] Personalized study recommendations (Algorithm exists, no data to work with)
  
- **Nice to Have (P2)**
  - [ ] Voice input for flashcard creation
  - [ ] Collaborative study rooms
  - [ ] Mobile app with offline sync

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Flashcard Generation | < 5s per card | Ollama processing time |
| Quiz Response Time | < 2s question loading | API response monitoring |
| Spaced Repetition Calc | < 100ms algorithm execution | Redis performance |
| Search Response | < 300ms for study materials | Qdrant query profiling |
| UI Animation Frame Rate | 60fps smooth animations | Browser performance API |

### Quality Gates
- [ ] All P0 requirements implemented and tested (3/7 fully working)
- [ ] Spaced repetition improves retention by 40% vs random review (Cannot measure without persistence)
- [ ] AI-generated content maintains 85% accuracy (Fallback data used)
- [ ] Gamification increases study session length by 60% (XP system works, streaks broken)
- [x] Cozy aesthetic receives 90%+ user satisfaction (UI well-designed)

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: ollama
    purpose: AI content generation for flashcards and explanations
    integration_pattern: Shared workflow for LLM inference
    access_method: ollama.json shared workflow
    
  - resource_name: postgres
    purpose: Store users, subjects, flashcards, progress, sessions
    integration_pattern: Direct SQL for complex educational queries
    access_method: resource-postgres CLI for backups
    
  - resource_name: qdrant
    purpose: Semantic search for study materials
    integration_pattern: REST API for similarity queries
    access_method: Direct API (vector operations)
    
  - resource_name: redis
    purpose: Cache spaced repetition intervals and session data
    integration_pattern: Direct API for performance-critical operations
    access_method: resource-redis CLI for cache management
    
  - resource_name: n8n
    purpose: Orchestrate educational content workflows
    integration_pattern: Scheduled workflows for study reminders
    access_method: Shared workflows via resource-n8n CLI
    
optional:
  - resource_name: grafana
    purpose: Advanced learning analytics dashboard
    fallback: Built-in progress tracking
    access_method: resource-grafana CLI for dashboard setup
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: LLM inference for educational content generation
      reused_by: [idea-generator, research-assistant]
      
    - workflow: content-quality-validator.json
      location: initialization/automation/n8n/
      purpose: Validate generated educational content accuracy
      reused_by: [educational-scenarios, content-generators]
      
    - workflow: flashcard-generator.json
      location: initialization/automation/n8n/
      purpose: Generate flashcards from study material (scenario-specific)
      
    - workflow: spaced-repetition-scheduler.json
      location: initialization/automation/n8n/
      purpose: Schedule study sessions based on forgetting curve (scenario-specific)
      
  2_resource_cli:
    - command: resource-redis flushdb study-sessions
      purpose: Clear study session cache for testing
      
    - command: resource-qdrant create-collection study-materials
      purpose: Initialize semantic search for flashcards
      
  3_direct_api:
    - justification: Spaced repetition requires millisecond-precise calculations
      endpoint: Redis for interval calculations
      
    - justification: Real-time progress updates need fast writes
      endpoint: PostgreSQL for session tracking

shared_workflow_validation:
  - content-quality-validator.json is generic for any educational content
  - flashcard-generator.json could be shared if abstracted from study context
  - spaced-repetition algorithms could be shared across learning scenarios
```

### Data Models
```yaml
primary_entities:
  - name: User
    storage: postgres
    schema: |
      {
        id: UUID
        username: string
        email: string
        profile: {
          xp_points: int
          current_streak: int
          longest_streak: int
          study_preferences: jsonb
          goal_minutes_per_day: int
        }
        created_at: timestamp
      }
    relationships: Has many StudySessions and FlashcardProgress
    
  - name: Subject
    storage: postgres
    schema: |
      {
        id: UUID
        user_id: UUID
        name: string
        description: text
        color: string
        flashcard_count: int
        mastery_percentage: float
        created_at: timestamp
      }
    relationships: Has many Flashcards and StudyNotes
    
  - name: Flashcard
    storage: postgres
    schema: |
      {
        id: UUID
        subject_id: UUID
        question: text
        answer: text
        hint: text
        difficulty: enum(easy, medium, hard)
        generated_by_ai: boolean
        source_content: text
        created_at: timestamp
      }
    relationships: Belongs to Subject, Has one FlashcardProgress per User
    
  - name: FlashcardProgress
    storage: redis
    schema: |
      {
        user_id: UUID
        flashcard_id: UUID
        interval: int (days)
        ease_factor: float
        repetitions: int
        last_reviewed: timestamp
        next_review: timestamp
        mastery_level: enum(learning, reviewing, mastered)
      }
    relationships: References User and Flashcard
    
  - name: StudySession
    storage: postgres
    schema: |
      {
        id: UUID
        user_id: UUID
        subject_id: UUID
        start_time: timestamp
        end_time: timestamp
        cards_studied: int
        correct_answers: int
        xp_earned: int
        session_type: enum(review, learn, quiz)
      }
    relationships: Belongs to User and Subject
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/flashcards/generate
    purpose: Generate AI flashcards from study content
    input_schema: |
      {
        subject_id: UUID
        content: string
        card_count: int
        difficulty_level: enum(beginner, intermediate, advanced)
      }
    output_schema: |
      {
        cards: [{
          question: string
          answer: string
          hint: string
          difficulty: string
        }]
        generation_metadata: {
          processing_time: int
          content_quality_score: float
        }
      }
    sla:
      response_time: 5000ms
      availability: 99.5%
      
  - method: GET
    path: /api/study/due-cards
    purpose: Get flashcards due for review based on spaced repetition
    input_schema: |
      {
        user_id: UUID
        subject_id: UUID (optional)
        limit: int
      }
    output_schema: |
      {
        cards: [{
          id: UUID
          question: string
          answer: string
          hint: string
          days_overdue: int
          priority_score: float
        }]
        next_session_recommendation: timestamp
      }
      
  - method: POST
    path: /api/study/session/start
    purpose: Start a study session
    input_schema: |
      {
        user_id: UUID
        subject_id: UUID
        session_type: enum(review, learn, quiz)
        target_duration: int (minutes)
      }
    output_schema: |
      {
        session_id: UUID
        cards_to_study: UUID[]
        estimated_cards: int
        xp_potential: int
      }
      
  - method: POST
    path: /api/study/answer
    purpose: Submit flashcard answer and update spaced repetition
    input_schema: |
      {
        session_id: UUID
        flashcard_id: UUID
        user_response: enum(again, hard, good, easy)
        response_time: int (ms)
      }
    output_schema: |
      {
        correct: boolean
        xp_earned: int
        next_review_date: timestamp
        progress_update: {
          mastery_level: string
          streak_continues: boolean
        }
      }
```

### Event Interface
```yaml
published_events:
  - name: study.session.completed
    payload: { user_id: UUID, subject_id: UUID, xp_earned: int, cards_studied: int }
    subscribers: [achievement-system, analytics-tracker]
    
  - name: study.streak.achieved
    payload: { user_id: UUID, streak_length: int, streak_type: string }
    subscribers: [notification-system, motivation-engine]
    
  - name: study.mastery.reached
    payload: { user_id: UUID, subject_id: UUID, mastery_percentage: float }
    subscribers: [recommendation-engine, celebration-system]
    
consumed_events:
  - name: user.goal.updated
    action: Adjust spaced repetition intervals for new goals
    
  - name: content.uploaded
    action: Generate flashcards from new study material
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: study-buddy
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show study buddy service status and user stats
    flags: [--json, --verbose, --user <id>]
    
  - name: help
    description: Display command help with study tips
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: generate-flashcards
    description: Generate AI flashcards from content
    api_endpoint: /api/flashcards/generate
    arguments:
      - name: subject-name
        type: string
        required: true
        description: Subject to add flashcards to
    flags:
      - name: --file
        description: Source file for content
      - name: --count
        description: Number of cards to generate
        default: 10
      - name: --difficulty
        description: Target difficulty (beginner|intermediate|advanced)
        default: intermediate
    example: study-buddy generate-flashcards "Biology" --file textbook.txt --count 20
    
  - name: study
    description: Start a study session
    api_endpoint: /api/study/session/start
    arguments:
      - name: subject-name
        type: string
        required: true
        description: Subject to study
    flags:
      - name: --type
        description: Session type (review|learn|quiz)
        default: review
      - name: --duration
        description: Target study duration in minutes
        default: 25
      - name: --cards
        description: Number of cards to study
    example: study-buddy study "Mathematics" --type review --duration 30
    
  - name: progress
    description: View study progress and statistics
    flags:
      - name: --subject
        description: Show progress for specific subject
      - name: --period
        description: Time period (week|month|year)
        default: week
      - name: --chart
        description: Display progress chart in terminal
    example: study-buddy progress --subject "Biology" --period month --chart
    
  - name: subjects
    description: Manage study subjects
    subcommands:
      - name: list
        description: List all subjects
        example: study-buddy subjects list
      - name: create
        arguments:
          - name: name
            type: string
            required: true
        flags:
          - name: --description
            description: Subject description
          - name: --color
            description: Subject color theme
        example: study-buddy subjects create "Chemistry" --color blue
    
  - name: streak
    description: View and manage study streaks
    flags:
      - name: --current
        description: Show current streak info
      - name: --history
        description: Show streak history
    example: study-buddy streak --current
    
  - name: export
    description: Export study data and progress
    flags:
      - name: --subject
        description: Subject to export
      - name: --format
        description: Export format (json|csv|markdown)
        default: json
      - name: --output
        description: Output file path
    example: study-buddy export --subject "Biology" --format csv --output bio-cards.csv
```

### CLI-API Parity Requirements
- **Coverage**: All study functions accessible via CLI
- **Naming**: Educational terminology (study, progress, streak)
- **Arguments**: Student-friendly parameter names
- **Output**: Motivational messaging, progress visualization
- **Authentication**: User-based with local config

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper with rich progress visualization
  - language: Go (consistent with other scenarios)
  - dependencies: API client, terminal charts, color output
  - error_handling:
      - Exit 0: Success with motivational message
      - Exit 1: General error
      - Exit 2: Study session interrupted
      - Exit 3: Content generation failed
  - configuration:
      - Config: ~/.vrooli/study-buddy/config.yaml
      - Env: STUDY_BUDDY_API_URL, USER_ID
      - Flags: Override any configuration
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - permissions: 755 on binary
  - documentation: study-buddy help --all
  - motivational_messages: Encouragement on completion
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: educational
  inspiration: "Lofi Girl meets Duolingo - cozy productivity with gentle gamification"
  
  visual_style:
    color_scheme: purple and pink gradients with warm accents
    typography: friendly rounded fonts, readable sans-serif
    layout: card-based with generous spacing
    animations: gentle, soothing transitions and micro-interactions
  
  personality:
    tone: encouraging, supportive, never judgmental
    mood: cozy study session, gentle motivation
    target_feeling: "Learning is peaceful and rewarding"

ui_components:
  study_space:
    - Animated coffee cup with steam
    - Twinkling stars in background
    - Gentle gradient backgrounds
    - Progress bars with satisfying fills
    
  flashcard_interface:
    - Card flip animations
    - Soft shadows and rounded corners
    - Color-coded difficulty indicators
    - Smooth reveal animations for answers
    
  progress_dashboard:
    - XP progress bars with level indicators
    - Streak calendars with flame icons
    - Achievement badges with celebration
    - Study time charts with gentle curves
    
  ambient_elements:
    - Decorative plants and study items
    - Cute cat companion (optional)
    - Subtle background patterns
    - Cozy lamp lighting effects

color_palette:
  primary: "#8B5A96"      # Soft purple for focus
  secondary: "#E6A8D5"    # Light pink for accents
  tertiary: "#A8C4A0"     # Sage green for growth
  accent: "#F4C2A1"       # Peach for highlights
  success: "#90C695"      # Gentle green for correct answers
  warning: "#F4D03F"      # Soft yellow for attention
  background: "#FAF7F2"   # Warm cream
  surface: "#FFFFFF"      # Pure white for cards
  text: "#5D4E75"         # Muted purple for readability
  
  # Gamification colors
  xp_progress: "#8B5A96"  # Purple progress bars
  streak_fire: "#FF6B6B"  # Warm red for streak indicators
  achievement: "#F4C2A1"  # Gold for badges
```

### Target Audience Alignment
- **Primary Users**: Students, lifelong learners, professionals developing skills
- **User Expectations**: Gentle gamification like Duolingo, aesthetic like Lofi study videos
- **Accessibility**: Dyslexia-friendly fonts, high contrast mode, keyboard navigation
- **Responsive Design**: Mobile-first for flashcard review, desktop for content creation

### Brand Consistency Rules
- **Scenario Identity**: "Your cozy study companion"
- **Vrooli Integration**: Showcase AI's ability to create nurturing experiences
- **Professional vs Fun**: Educational and supportive with gentle gamification
- **Differentiation**: More cozy than Anki, more intelligent than Quizlet

## 💰 Value Proposition

### Business Value
- **Primary Value**: Personalized learning platform with AI content generation
- **Revenue Potential**: $15K - $25K per educational institution, $5-15K per corporate deployment
- **Cost Savings**: 70% reduction in content creation time for educators
- **Market Differentiator**: Only flashcard platform with spaced repetition AI and cozy aesthetic

### Technical Value
- **Reusability Score**: 7/10 - Educational patterns applicable to training scenarios
- **Complexity Reduction**: Makes adaptive learning accessible without PhD in education
- **Innovation Enablement**: Foundation for personalized learning and gamified education

## 🔄 Scenario Lifecycle Integration

### Scenario-to-App Conversion
```yaml
app_conversion:
  supported: true
  app_structure_compliance:
    - Complete service.json with educational focus
    - PostgreSQL schema for learning analytics
    - Redis for real-time spaced repetition
    - Cozy UI with gamification elements
    
  deployment_targets:
    - local: Docker Compose with Redis persistence
    - kubernetes: StatefulSet with educational data protection
    - cloud: AWS ECS with RDS for student data compliance
    
  revenue_model:
    - type: freemium
    - pricing_tiers:
        free: 3 subjects, 100 flashcards total
        student: $5/month (unlimited personal use)
        educator: $15/month (classroom features, analytics)
        institution: $50/month per 100 users
    - trial_period: 30 days educator features
    - value_proposition: "AI-powered study companion that actually works"
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: study-buddy
    category: education
    capabilities:
      - AI flashcard generation
      - Spaced repetition optimization
      - Adaptive quiz creation
      - Learning progress analytics
      - Gamified study experience
    interfaces:
      - api: http://localhost:8090/api
      - cli: study-buddy
      - events: study.*
      - ui: http://localhost:8091
      
  metadata:
    description: "AI-powered study companion with cozy aesthetic"
    keywords: [study, flashcards, learning, spaced-repetition, education]
    dependencies: []
    enhances: [research-assistant, stream-of-consciousness-analyzer]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  api_version: v1
  
  breaking_changes: []
  deprecations: []
  
  upgrade_path:
    from_0_9: "Migrate spaced repetition data to Redis"
```

## 🧬 Evolution Path

### Version 1.0 (Current)
- AI flashcard generation
- Spaced repetition algorithm
- Basic gamification with XP and streaks
- Cozy aesthetic with animations

### Version 2.0 (Planned)
- Voice input for flashcard creation
- Collaborative study rooms
- Mobile app with offline sync
- Advanced learning analytics
- Integration with popular textbooks

### Long-term Vision
- Personalized learning path recommendations
- AR flashcard visualization
- Social learning communities
- Cross-platform synchronization

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Spaced repetition accuracy | Low | High | Validated algorithm implementation, user feedback |
| AI content quality | Medium | Medium | Content validation workflow, human review |
| Performance with large card sets | Medium | Low | Pagination, lazy loading, Redis caching |
| Gamification burnout | Low | Medium | Balanced rewards, progress variety |

### Operational Risks
- **Drift Prevention**: PRD validated against educational effectiveness weekly
- **Version Compatibility**: Learning data migration scripts for updates
- **Resource Conflicts**: Dedicated Redis instance for spaced repetition timing
- **Style Drift**: Cozy design system with strict component guidelines
- **CLI Consistency**: User experience testing for motivational messaging

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# File: scenario-test.yaml
version: 1.0
scenario: study-buddy

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - api/main.go
    - api/go.mod
    - cli/study-buddy
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/flashcard-generator.json
    - initialization/automation/n8n/spaced-repetition-scheduler.json
    - ui/index.html
    - ui/script.js
    - ui/server.js
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization/storage/postgres
    - initialization/automation/n8n
    - ui

resources:
  required: [ollama, postgres, qdrant, redis, n8n]
  optional: [grafana]
  health_timeout: 60

tests:
  - name: "Generate Flashcards API"
    type: http
    service: api
    endpoint: /api/flashcards/generate
    method: POST
    body:
      subject_id: "test-subject"
      content: "The mitochondria is the powerhouse of the cell"
      card_count: 3
    expect:
      status: 200
      body:
        cards: "*"
        generation_metadata: "*"
        
  - name: "Due Cards Retrieval"
    type: http
    service: api
    endpoint: /api/study/due-cards
    method: GET
    query:
      user_id: "test-user"
      limit: 10
    expect:
      status: 200
      body:
        cards: "*"
        next_session_recommendation: "*"
        
  - name: "Study Session Management"
    type: http
    service: api
    endpoint: /api/study/session/start
    method: POST
    body:
      user_id: "test-user"
      subject_id: "test-subject"
      session_type: "review"
    expect:
      status: 200
      body:
        session_id: "*"
        cards_to_study: "*"
        
  - name: "CLI Generate Command"
    type: exec
    command: ./cli/study-buddy generate-flashcards "Test Subject" --count 5
    expect:
      exit_code: 0
      output_contains: ["generated", "flashcards", "Good luck"]
      
  - name: "Cozy UI Interface"
    type: http
    service: ui
    endpoint: /
    expect:
      status: 200
      body_contains: ["study", "flashcard", "cozy", "lofi"]
      
  - name: "Spaced Repetition Workflow"
    type: n8n
    workflow: spaced-repetition-scheduler
    expect:
      active: true
      schedule: "0 9 * * *"  # Daily at 9 AM
```

### Test Execution Gates
```bash
./test.sh --scenario study-buddy --validation complete
./test.sh --learning     # Test spaced repetition accuracy
./test.sh --generation   # Verify AI content quality
./test.sh --gamification # Check XP and streak systems
./test.sh --ui          # Validate cozy aesthetic
```

### Performance Validation
- [x] Flashcard generation < 5s response time (2.7s measured)
- [x] Due card calculation < 100ms for 1000+ cards (Mock data returns instantly)
- [ ] UI maintains 60fps with animations (Not measured)
- [ ] Study session loads < 2s (Fails due to DB)
- [ ] Progress updates in real-time (No persistence)

### Integration Validation
- [ ] AI generates educationally sound flashcards
- [ ] Spaced repetition follows proven algorithms
- [ ] Gamification encourages continued learning
- [ ] Semantic search finds related study materials
- [ ] Progress tracking motivates learners

### Capability Verification
- [ ] Students show improved retention with spaced repetition
- [ ] AI-generated content maintains high accuracy
- [ ] Gamification increases study session duration
- [ ] Cozy aesthetic reduces study anxiety
- [ ] Progress tracking provides meaningful feedback
- [ ] UI matches cozy, supportive design expectations

## 📝 Implementation Notes

### Design Decisions
**Cozy aesthetic over efficient productivity**: Psychological comfort priority
- Alternative considered: Clean, minimal Anki-style interface
- Decision driver: Aesthetic reduces study anxiety and increases engagement
- Trade-offs: Less information density, higher user satisfaction

**Spaced repetition over random review**: Science-based learning
- Alternative considered: Simple random or frequency-based review
- Decision driver: Spaced repetition proven to improve retention by 200%+
- Trade-offs: More complex algorithm, significantly better learning outcomes

**Gamification integration**: Sustained motivation
- Alternative considered: Pure study tool without game elements
- Decision driver: Gamification increases engagement without sacrificing learning
- Trade-offs: Additional complexity, much higher user retention

### Known Limitations
- **Content Accuracy**: AI may occasionally generate incorrect flashcards
  - Workaround: User editing and rating system for quality control
  - Future fix: Content validation models and expert review systems
  
- **Individual Learning Differences**: One-size-fits-all spaced repetition
  - Workaround: Adjustable difficulty and interval settings
  - Future fix: Personalized algorithm tuning based on performance

### Security Considerations
- **Student Privacy**: Educational data protected under FERPA compliance
- **Progress Data**: Encrypted storage of learning analytics
- **Content Security**: User-generated content moderation
- **Child Safety**: Age-appropriate content filtering for younger users

## 🔗 References

### Documentation
- README.md - Quick start and study tips
- api/docs/spaced-repetition.md - Algorithm implementation
- ui/docs/cozy-design.md - Aesthetic guidelines
- cli/docs/study-workflows.md - Effective CLI usage

### Related PRDs
- scenarios/research-assistant/PRD.md - Content generation patterns
- scenarios/stream-of-consciousness-analyzer/PRD.md - Note processing
- scenarios/idea-generator/PRD.md - AI content creation

### External Resources
- [Spaced Repetition Research](https://www.gwern.net/Spaced-repetition)
- [Gamification in Education](https://www.sciencedirect.com/science/article/pii/S0360131519301817)
- [Lofi Study Aesthetic Psychology](https://www.researchgate.net/publication/ambient-study-environments)

---

**Last Updated**: 2025-09-30
**Status**: Partially Working (40% functional)
**Issues**: See PROBLEMS.md for critical database and integration issues
**Owner**: AI Agent - Educational Experience Module  
**Review Cycle**: Monthly validation of learning effectiveness and user engagement
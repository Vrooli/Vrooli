# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Provides a safe, curated, age-appropriate dashboard interface that allows children to access and interact with kid-friendly Vrooli scenarios through an engaging, playful UI. This establishes the infrastructure for user-segmented access control and multi-user household deployments.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Agents learn to categorize content by appropriateness and user segments
- Establishes patterns for role-based access control that other scenarios can leverage
- Creates feedback loops where kid interactions inform what scenarios work best for different age groups
- Enables agents to build age-appropriate variations of existing capabilities

### Recursive Value
**What new scenarios become possible after this exists?**
1. **parental-controls-manager** - Advanced controls for managing child access, time limits, content filtering
2. **educational-curriculum-tracker** - Track learning progress across kid-friendly educational scenarios  
3. **senior-dashboard** - Similar segmented interface for elderly users with accessibility features
4. **classroom-mode** - Multi-student dashboard for educational institutions
5. **family-calendar-hub** - Shared family scheduling that kids can safely interact with

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Dashboard UI that displays kid-friendly scenarios only
  - [ ] Category-based filtering using "kid-friendly" tag from scenario service.json files
  - [ ] Iframe-based scenario isolation for security
  - [ ] Fun, colorful, animated interface appropriate for ages 5-12
  - [ ] Large, touch-friendly buttons and navigation
  - [ ] Automatic discovery of new kid-friendly scenarios
  
- **Should Have (P1)**
  - [ ] Age range filtering (5-8, 9-12, 13+)
  - [ ] Mascot character for guidance and engagement
  - [ ] Sound effects and animations on interactions
  - [ ] Session time tracking for parental awareness
  - [ ] Favorites/recently used scenarios
  
- **Nice to Have (P2)**
  - [ ] Customizable themes (space, ocean, jungle, etc.)
  - [ ] Achievement badges for using different scenarios
  - [ ] Kid profiles with avatars
  - [ ] Simplified search with voice input

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 500ms for scenario loading | Frontend monitoring |
| Throughput | 10 concurrent child sessions | Load testing |
| Accessibility | WCAG 2.1 Level AA | Automated testing |
| Touch Target Size | Minimum 44x44px | UI validation |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Successfully filters and displays only kid-friendly scenarios
- [ ] UI tested with target age groups (if possible via screenshots)
- [ ] No access to system/development tools from dashboard
- [ ] Parent-friendly setup documentation complete

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: none
    purpose: Standalone UI that queries local scenario catalog
    integration_pattern: File-based discovery
    access_method: Read local catalog.json and service.json files
    
optional:
  - resource_name: redis
    purpose: Cache scenario metadata for faster loading
    fallback: Direct file reads if unavailable
    access_method: resource-redis CLI
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_local_discovery:
    - method: Read scenarios/*/service.json files
    - filter: category includes "kid-friendly"
    - purpose: Discover available kid-appropriate scenarios
  
  2_iframe_isolation:
    - method: Embed scenarios in sandboxed iframes
    - security: Restricted permissions, no parent access
    - purpose: Safe execution of child scenarios

shared_workflow_criteria:
  - No shared workflows needed initially
  - Future: parental-notification workflow for time limits
```

### Data Models
```yaml
primary_entities:
  - name: KidScenario
    storage: In-memory (from catalog)
    schema: |
      {
        id: string (scenario name)
        title: string
        description: string  
        icon: string (emoji or image path)
        color: string (hex color for tile)
        ageRange: [5-8, 9-12, 13+]
        category: string (games, learn, create)
        url: string (scenario URL)
        popularity: number (usage count)
      }
    relationships: Filtered subset of main scenario catalog
```

### API Contract
```yaml
endpoints:
  - method: GET
    path: /api/v1/kids/scenarios
    purpose: List all available kid-friendly scenarios
    input_schema: |
      {
        ageRange?: string // Optional age filter
        category?: string // Optional category filter
      }
    output_schema: |
      {
        scenarios: KidScenario[]
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/kids/launch
    purpose: Launch a scenario in the dashboard
    input_schema: |
      {
        scenarioId: string
      }
    output_schema: |
      {
        url: string // Iframe URL for scenario
        sessionId: string
      }
```

### Event Interface
```yaml
published_events:
  - name: kids.scenario.launched
    payload: { scenarioId, timestamp, sessionId }
    subscribers: parental-controls, usage-analytics
    
  - name: kids.session.started
    payload: { timestamp, profileId? }
    subscribers: time-tracker, parental-controls

consumed_events:
  - name: scenario.catalog.updated
    action: Refresh available scenarios list
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: kids-dashboard
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show dashboard operational status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: list-scenarios
    description: List available kid-friendly scenarios
    api_endpoint: /api/v1/kids/scenarios
    arguments:
      - name: age-range
        type: string
        required: false
        description: Filter by age range (5-8, 9-12, 13+)
    flags:
      - name: --json
        description: Output in JSON format
    output: List of available scenarios
    
  - name: add-scenario
    description: Mark a scenario as kid-friendly
    arguments:
      - name: scenario-name
        type: string
        required: true
        description: Name of scenario to add
    output: Confirmation message
```

## 🔄 Integration Requirements

### Upstream Dependencies
- **Scenario Catalog**: Must read catalog.json for scenario discovery
- **Service.json Standards**: Scenarios must properly tag themselves as kid-friendly

### Downstream Enablement
- **Parental Controls**: This dashboard becomes the controlled environment parents manage
- **Educational Tracking**: Learning scenarios can report progress through this interface
- **Multi-User Dashboards**: Establishes pattern for senior, developer, business dashboards

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: parental-controls-manager
    capability: Controlled environment to manage
    interface: API/Events
    
  - scenario: any-kid-friendly-scenario
    capability: Discovery and launch platform
    interface: Iframe embed

consumes_from:
  - scenario: all-scenarios
    capability: Metadata and categorization
    fallback: Only shows scenarios with explicit kid-friendly tag
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: playful
  inspiration: Nintendo Switch UI, PBS Kids website, Disney+
  
  visual_style:
    color_scheme: bright-and-colorful
    typography: rounded, friendly, large
    layout: grid-of-tiles
    animations: bouncy, playful, rewarding
  
  personality:
    tone: friendly, encouraging, safe
    mood: energetic, fun, welcoming
    target_feeling: excitement and safety

design_specifications:
  - colors:
      primary: "#FF6B6B"  # Friendly red
      secondary: "#4ECDC4" # Teal
      accent: "#FFE66D"    # Yellow
      success: "#95E77E"   # Green
      background: "#F7F9FB" # Soft white
      
  - typography:
      headings: "Fredoka One" or similar playful font
      body: "Open Sans" or similar readable font
      sizes: 
        minimum: 16px
        headers: 32px+
        buttons: 20px+
        
  - components:
      tiles: 
        - Rounded corners (16px radius)
        - Drop shadows for depth
        - Hover animations (scale 1.05)
        - Click animations (scale 0.95)
      buttons:
        - Minimum 48px height
        - Rounded (24px radius)
        - Clear active states
      mascot:
        - Friendly robot or animal character
        - Appears in corners with helpful tips
        - Celebrates successes
```

### Target Audience Alignment
- **Primary Users**: Children ages 5-12
- **User Expectations**: Fun, game-like, responsive, safe
- **Accessibility**: Large touch targets, high contrast, simple language
- **Responsive Design**: Tablet-first, works on desktop, basic phone support

## 💰 Value Proposition

### Business Value
- **Primary Value**: Enables family adoption of Vrooli platform
- **Revenue Potential**: $10K - $30K per deployment (educational institutions)
- **Cost Savings**: Replaces multiple kid software subscriptions
- **Market Differentiator**: First AI platform with true family-safe mode

### Technical Value
- **Reusability Score**: 10/10 - Pattern reusable for all user segments
- **Complexity Reduction**: Parents don't need to curate safe scenarios
- **Innovation Enablement**: Opens education and family market segments

## 🧬 Evolution Path

### Version 1.0 (Current)
- Basic dashboard with category filtering
- Iframe-based scenario display
- Fun, colorful UI
- Simple navigation

### Version 2.0 (Planned)
- Age-based filtering
- Parental controls integration
- Usage analytics
- Achievement system
- Voice navigation

### Long-term Vision
- AI tutor integration for educational guidance
- Multi-child household support with profiles
- Classroom mode for schools
- Adaptive difficulty based on child's progress
- Parent mobile app for monitoring

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with kid-friendly category
    - Simple UI suitable for children
    - No complex authentication required
    - Health check endpoints
    
  deployment_targets:
    - local: Standalone web interface
    - school: Multi-tenant deployment
    - home: Single family server
    
  revenue_model:
    - type: freemium
    - pricing_tiers:
        free: 3 scenarios
        family: $9.99/mo unlimited
        school: $99/mo per classroom
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: kids-dashboard
    category: interface
    capabilities: [user-segmentation, content-filtering, safe-browsing]
    interfaces:
      - api: http://localhost:PORT/api/v1/kids
      - cli: kids-dashboard
      - events: kids.*
      
  metadata:
    description: Safe, fun dashboard for children to access kid-friendly scenarios
    keywords: [kids, children, family, safe, dashboard, parental-controls]
    dependencies: []
    enhances: [all kid-friendly scenarios]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Inappropriate content exposure | Low | Critical | Strict category filtering, iframe sandboxing |
| Complex UI overwhelming kids | Medium | High | Extensive UI simplification, user testing |
| Performance with many scenarios | Low | Medium | Pagination, lazy loading |

### Operational Risks
- **Content Safety**: Only explicitly tagged scenarios appear
- **Age Appropriateness**: Age range metadata enforced
- **Session Management**: Automatic timeouts for safety
- **Data Privacy**: No personal data collection from children

## ✅ Validation Criteria

### Test Specification
```yaml
tests:
  - name: "Only kid-friendly scenarios appear"
    type: integration
    description: Verify filtering works correctly
    
  - name: "Iframe isolation prevents system access"
    type: security
    description: Ensure sandboxing is effective
    
  - name: "UI elements meet size requirements"
    type: ui
    description: All buttons >= 44x44px
    
  - name: "Page loads in under 500ms"
    type: performance
    description: Fast enough for impatient kids
```

### Capability Verification
- [ ] Successfully filters adult/system scenarios
- [ ] Children can navigate without reading skills
- [ ] Parents can understand setup instantly
- [ ] Scenarios launch safely in isolation
- [ ] New kid scenarios auto-appear

## 📝 Implementation Notes

### Design Decisions
**Category-based filtering**: Chose explicit tagging over ML classification
- Alternative considered: AI content analysis
- Decision driver: Reliability and parent trust
- Trade-offs: Manual tagging effort for scenario authors

**Iframe isolation**: Chose iframes over native integration
- Alternative considered: Direct component embedding
- Decision driver: Security and simplicity
- Trade-offs: Some performance overhead, limited interaction

### Known Limitations
- **Scenario modifications**: Can't modify scenario UIs for kid-friendliness
  - Workaround: Only include already kid-appropriate scenarios
  - Future fix: Scenario UI adaptation layer
  
- **Offline scenarios**: Some scenarios may require internet
  - Workaround: Clear marking of online requirements
  - Future fix: Offline mode for core scenarios

### Security Considerations
- **Data Protection**: No personal data collected from children (COPPA compliance)
- **Access Control**: No access to system functions or adult scenarios
- **Audit Trail**: Parent-accessible log of scenario usage

## 🔗 References

### Documentation
- README.md - Parent/admin setup guide
- docs/safety.md - Safety and privacy documentation
- docs/adding-scenarios.md - How to mark scenarios as kid-friendly

### Related PRDs
- scenarios/parental-controls-manager/PRD.md (future)
- scenarios/retro-game-launcher/PRD.md
- scenarios/picker-wheel/PRD.md

---

**Last Updated**: 2025-01-06  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: After each new kid-friendly scenario added
# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Provides serendipitous discovery and exploration of running scenarios through a StumbleUpon-like browsing experience, combined with streamlined issue reporting directly to app-issue-tracker. This creates the primary interface for discovering and using the growing ecosystem of Vrooli capabilities.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Discovery Loop**: Enables users and agents to find existing capabilities they weren't aware of, reducing duplicate work
- **Quality Feedback**: Creates a direct path from user experience to bug reports, improving all scenarios through crowdsourced testing  
- **Usage Patterns**: Tracks which scenarios are actually used, informing development priorities
- **Ecosystem Awareness**: Makes the compound intelligence visible and accessible

### Recursive Value
**What new scenarios become possible after this exists?**
1. **scenario-analytics** - Track usage patterns and scenario performance metrics discovered through surfing
2. **capability-recommender** - AI-powered suggestions based on user's browsing history and needs
3. **scenario-marketplace** - Curated collections and ratings system for business scenarios
4. **cross-scenario-orchestrator** - Workflows that chain multiple discovered scenarios together
5. **user-journey-optimizer** - Analyze how users move between scenarios to improve UX

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Browse running scenarios via iframe embedding with health status filtering
  - [ ] Navigation controls (back, random, scenario dropdown selector)
  - [ ] Category-based browsing modes (work, fun, dev, etc.) from service.json tags
  - [ ] One-click issue reporting with screenshot capture to app-issue-tracker
  - [ ] Minimal chrome design that doesn't compete with embedded scenarios
  
- **Should Have (P1)**
  - [ ] Favorites/bookmarks system for frequently used scenarios
  - [ ] Navigation history tracking and breadcrumbs
  - [ ] Quick scenario health status indicators
  - [ ] Keyboard shortcuts for navigation (space for random, escape for back)
  - [ ] Responsive design for different screen sizes
  
- **Nice to Have (P2)**
  - [ ] Scenario descriptions and metadata overlay
  - [ ] Usage analytics and personal dashboard
  - [ ] Social features (sharing interesting scenarios)
  - [ ] Integration with scenario-generator-v1 for testing new scenarios

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Scenario Discovery Time | < 500ms to load scenario list | API monitoring |
| Iframe Load Time | < 2s for scenario UI to display | Browser performance timing |
| Issue Report Submission | < 3s from click to submitted | End-to-end timing |
| Navigation Response | < 100ms for UI transitions | Frontend performance |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Can discover and embed all healthy running scenarios
- [ ] Issue reporting integrates successfully with app-issue-tracker
- [ ] Responsive design works on desktop and tablet
- [ ] Navigation is intuitive without instructions

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: app-issue-tracker
    purpose: Submit bug reports and issues discovered during surfing
    integration_pattern: API calls to issue creation endpoint
    access_method: HTTP POST to /api/v1/issues
    
optional:
  - resource_name: browserless
    purpose: Capture screenshots for issue reporting
    fallback: Users provide text description without screenshot
    access_method: resource-browserless screenshot command
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  2_resource_cli:        # Direct CLI for scenario discovery
    - command: vrooli scenario status --json
      purpose: Get real-time scenario health and port information
  
  3_direct_api:          # Direct API calls to scenarios
    - justification: Need to embed running scenario UIs in iframes
      endpoint: Individual scenario UI endpoints from status data

# No shared workflows needed - this is primarily a discovery/navigation tool
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: ScenarioInfo
    storage: runtime_only
    schema: |
      {
        name: string,
        status: "running" | "stopped" | "error" | "degraded",
        ports: { ui?: number, api?: number },
        url: string,
        category: string[],
        health: "healthy" | "degraded" | "unhealthy"
      }
    relationships: Derived from vrooli scenario status --json
  
  - name: NavigationHistory
    storage: localStorage
    schema: |
      {
        scenarios: string[],
        current_index: number,
        category_mode: string
      }
    relationships: Browser session persistence
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: GET
    path: /api/v1/scenarios/healthy
    purpose: List all healthy scenarios for programmatic discovery
    output_schema: |
      {
        scenarios: ScenarioInfo[],
        categories: string[]
      }
    sla:
      response_time: 500ms
      availability: 99%
  
  - method: POST
    path: /api/v1/issues/report
    purpose: Submit issue reports (proxy to app-issue-tracker)
    input_schema: |
      {
        scenario: string,
        title: string,
        description: string,
        screenshot?: string
      }
    sla:
      response_time: 2000ms
      availability: 99%
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: scenario-surfer
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show scenario-surfer health and available scenarios
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: discover
    description: List all discoverable scenarios
    api_endpoint: /api/v1/scenarios/healthy
    flags:
      - name: --category
        description: Filter by category (work, fun, dev, etc.)
      - name: --status
        description: Filter by health status
    output: JSON list of available scenarios
    
  - name: open
    description: Open scenario-surfer browser interface
    arguments:
      - name: category
        type: string
        required: false
        description: Start in specific category mode
    output: Opens web browser to scenario-surfer UI
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: minimalist
  inspiration: StumbleUpon's clean discovery interface meets modern web browsing
  
  visual_style:
    color_scheme: light
    typography: modern
    layout: single-page
    animations: subtle
  
  personality:
    tone: friendly
    mood: calm
    target_feeling: Serendipitous discovery and smooth exploration

# The chrome should be invisible - scenarios themselves are the content
chrome_design:
  - Thin top navigation bar (< 60px height)
  - Subtle, non-distracting colors (grays, whites)
  - Icons over text where possible
  - Fade in/out on mouse inactivity
  - Smooth transitions between scenarios
  - Clean typography that doesn't compete with scenario content
```

### Target Audience Alignment
- **Primary Users**: Vrooli power users exploring the ecosystem
- **User Expectations**: Smooth browsing experience similar to web browsing
- **Accessibility**: Full keyboard navigation support, screen reader friendly
- **Responsive Design**: Desktop primary, tablet secondary, mobile basic support

## 💰 Value Proposition

### Business Value
- **Primary Value**: Makes the growing Vrooli ecosystem discoverable and usable
- **Revenue Potential**: $5K - $15K per deployment (discovery platform for business scenarios)
- **Cost Savings**: Reduces duplicate scenario development by improving discoverability
- **Market Differentiator**: First-of-its-kind self-service discovery for AI capabilities

### Technical Value
- **Reusability Score**: High - other scenarios can embed scenario-surfer for capability discovery
- **Complexity Reduction**: Makes finding the right tool trivial instead of requiring documentation searches
- **Innovation Enablement**: Users discover unexpected combinations of capabilities

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **vrooli scenario status --json**: Core capability discovery mechanism
- **Running scenarios**: Content to surf and discover
- **app-issue-tracker**: Issue reporting destination

### Downstream Enablement
**What future capabilities does this unlock?**
- **Usage Analytics**: Understanding which capabilities are actually used
- **Capability Recommendations**: AI-powered scenario suggestions
- **Quality Feedback Loops**: Systematic improvement of all scenarios through user reports

## 🧬 Evolution Path

### Version 1.0 (Current)
- Basic scenario browsing and navigation
- Category-based filtering
- Issue reporting integration
- Minimal chrome interface

### Version 2.0 (Planned)
- Usage analytics and personal dashboard
- AI-powered scenario recommendations
- Social features and sharing
- Advanced navigation (search, filters, tags)

### Long-term Vision
- Becomes the primary interface for Vrooli interaction
- Intelligent workflow suggestions based on browsing patterns
- Marketplace features for scenario discovery and rating

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| CORS issues with iframe embedding | Medium | High | Implement proper headers, fallback to new window |
| Performance with many scenarios | Low | Medium | Lazy loading, pagination for large lists |
| Issue reporting failures | Low | Medium | Queue reports, retry mechanism |

### Operational Risks
- **Scenario Discovery Drift**: Regularly validate that all running scenarios are discoverable
- **UI Interference**: Chrome must remain minimal to avoid competing with scenario content
- **Navigation Confusion**: Intuitive UX critical since users will be focused on embedded content

## ✅ Validation Criteria

### Functional Validation
- [ ] Can browse all healthy running scenarios successfully
- [ ] Category filtering works for all service.json tag combinations
- [ ] Issue reporting creates tickets in app-issue-tracker
- [ ] Navigation history works (back/forward)
- [ ] Random scenario selection provides good variety
- [ ] Screenshots are captured and attached to issues

### Integration Validation
- [ ] Iframe embedding works for all scenario types
- [ ] No conflicts with scenario UI/UX (minimal chrome)
- [ ] Issue reporting API integration functions correctly
- [ ] CLI commands provide programmatic access

### User Experience Validation
- [ ] Navigation feels intuitive and responsive
- [ ] Chrome elements don't distract from scenario content
- [ ] Issue reporting process is smooth and quick
- [ ] Category modes provide meaningful content groupings

## 📝 Implementation Notes

### Design Decisions
**Iframe vs New Window**: Chosen iframe approach for seamless experience
- Alternative considered: Opening scenarios in new browser windows
- Decision driver: Maintain context and navigation state
- Trade-offs: CORS complexity for better UX

**Minimal Chrome**: Decided on very subtle navigation chrome
- Alternative considered: Rich navigation UI with metadata
- Decision driver: Scenarios are the content, not our interface
- Trade-offs: Less feature discoverability for cleaner experience

### Known Limitations
- **CORS Restrictions**: Some scenarios may not embed properly in iframes
  - Workaround: Fallback to new window opening
  - Future fix: Coordinate with scenario developers on iframe-friendly headers

### Security Considerations
- **Data Protection**: No sensitive data stored, only navigation preferences
- **Access Control**: Inherits access control from individual scenarios
- **Audit Trail**: Issue reports are logged with timestamps and user context

## 🔗 References

### Documentation
- README.md - User-facing overview and getting started guide
- docs/api.md - API specification for programmatic access
- docs/navigation.md - Keyboard shortcuts and navigation guide

### Related PRDs
- app-issue-tracker PRD - Issue reporting destination
- scenario-generator-v1 PRD - Source of new scenarios to discover

---

**Last Updated**: 2025-09-05  
**Status**: In Development  
**Owner**: AI Agent  
**Review Cycle**: Weekly validation against running scenario ecosystem
# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Brand Manager provides AI-powered brand generation and automated app integration platform. It generates comprehensive brand identities including logos, color palettes, typography, slogans, and marketing copy, then automatically applies these assets to existing applications and websites through Claude Code integration.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability provides brand intelligence that other scenarios can leverage to create cohesive, professional visual identities. Any scenario that builds user-facing applications can use this to ensure consistent branding. The brand asset generation and integration patterns become reusable knowledge for all future app development scenarios.

### Recursive Value
**What new scenarios become possible after this exists?**
- Marketing campaign generator with consistent brand application
- E-commerce store builder with professional branding from day one
- SaaS product generator that includes complete brand identity
- Website theme generator based on brand guidelines
- Social media content creator with brand-consistent visuals
- Corporate identity manager for multi-brand organizations

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Complete brand generation with logo, colors, typography, and copy
  - [ ] Professional web UI for brand asset management
  - [ ] Integration with ComfyUI for AI-powered logo generation
  - [ ] Automated Claude Code integration for applying brands to existing apps
  - [ ] Brand asset export and packaging system
  
- **Should Have (P1)**
  - [ ] Brand template library for different industries
  - [ ] Color palette analyzer and harmonizer
  - [ ] Brand guideline document generation
  - [ ] Multi-format asset generation (PNG, SVG, ICO)
  - [ ] Brand asset versioning and history
  
- **Nice to Have (P2)**
  - [ ] Brand consistency checker across multiple apps
  - [ ] A/B testing for brand variants
  - [ ] Brand performance analytics
  - [ ] Integration with external design tools

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Brand Generation Time | < 90 seconds complete brand | API monitoring |
| Asset Quality | > 95% approval rate for generated assets | User feedback |
| Integration Speed | < 5 minutes to apply brand to existing app | Automated testing |
| Resource Usage | < 4GB memory, < 50% CPU during generation | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Professional UI with brand asset management
- [ ] Claude Code integration working end-to-end
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: n8n
    purpose: Workflow orchestration for brand generation pipelines
    integration_pattern: Custom workflows for brand pipeline and Claude integration
    access_method: Webhooks for brand generation and integration triggers
    
  - resource_name: comfyui
    purpose: AI-powered logo and visual asset generation
    integration_pattern: Custom workflows for different asset types
    access_method: API for logo, icon, and favicon generation
    
  - resource_name: ollama
    purpose: AI-powered brand copy generation and style analysis
    integration_pattern: Via n8n workflows for content generation
    access_method: Text generation for slogans, descriptions, and marketing copy
    
  - resource_name: postgres
    purpose: Store brand data, assets metadata, and integration tracking
    integration_pattern: Direct database connection
    access_method: SQL via Go database/sql
    
  - resource_name: minio
    purpose: Asset storage for logos, icons, exports, and backups
    integration_pattern: S3-compatible API
    access_method: MinIO Go client for asset management
    
  - resource_name: claude-code
    purpose: Automated app integration agent for brand deployment
    integration_pattern: API calls to spawn Claude sessions
    access_method: REST API for integration requests

optional:
  - resource_name: vault
    purpose: Secure storage for API keys and integration tokens
    fallback: Environment variables if unavailable
    access_method: Vault API for secret management
    
  - resource_name: qdrant
    purpose: Store brand embeddings for similarity search
    fallback: Database-based brand matching without semantic understanding
    access_method: HTTP API for vector operations
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_workflow_orchestration:
    - workflow: brand-pipeline.json
      location: initialization/automation/n8n/
      purpose: Complete brand generation workflow
    - workflow: claude-spawner.json
      location: initialization/automation/n8n/
      purpose: Automated app integration via Claude Code
  
  2_asset_generation:
    - workflow: logo-generator.json
      location: initialization/configuration/comfyui-workflows/
      purpose: AI-powered logo generation
    - workflow: icon-creator.json
      location: initialization/configuration/comfyui-workflows/
      purpose: Icon and favicon generation
  
  3_direct_api:
    - justification: PostgreSQL requires direct connection for complex brand queries
      endpoint: postgres://localhost:5432/vrooli
    - justification: MinIO requires direct access for asset management
      endpoint: s3://localhost:9000/brand-assets
```

### Data Models
```yaml
primary_entities:
  - name: Brand
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        short_name: string
        slogan: string
        ad_copy: text
        description: text
        brand_colors: jsonb
        logo_url: string
        favicon_url: string
        assets: jsonb array
        metadata: jsonb
        created_at: timestamp
        updated_at: timestamp
      }
    relationships: Can have multiple exports and integration requests
    
  - name: IntegrationRequest
    storage: postgres
    schema: |
      {
        id: UUID
        brand_id: UUID (FK)
        target_app_path: string
        integration_type: string
        claude_session_id: string
        status: string
        request_payload: jsonb
        response_payload: jsonb
        created_at: timestamp
        completed_at: timestamp
      }
    relationships: Belongs to Brand
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/brands
    purpose: Generate a new brand with complete identity package
    input_schema: |
      {
        brand_name: string (required)
        short_name: string (optional)
        industry: string (required)
        template: string (optional)
        logo_style: string (optional)
        color_scheme: string (optional)
      }
    output_schema: |
      {
        message: string
        workflow_result: object
        polling_endpoint: string
        estimated_time: string
      }
    sla:
      response_time: 90000ms
      availability: 99%
      
  - method: GET
    path: /api/brands/{id}
    purpose: Retrieve complete brand with all assets
    output_schema: |
      {
        brand: Brand object with full asset URLs
      }
      
  - method: POST
    path: /api/integrations
    purpose: Start automated brand integration into existing app
    input_schema: |
      {
        brand_id: string (required)
        target_app_path: string (required)
        integration_type: string (optional)
        create_backup: boolean (optional)
      }
    output_schema: |
      {
        message: string
        workflow_result: object
        estimated_time: string
      }
```

## 🖥️ UI Design Specifications

### Design Philosophy
```yaml
style_profile:
  category: creative_professional
  inspiration: "Modern design tool (Figma/Canva) meets professional brand management"
  
  visual_style:
    color_scheme: 
      primary: "#2D1B69" # Deep purple for creativity
      secondary: "#FF6B35" # Vibrant orange for energy
      accent: "#4ECDC4" # Teal for balance
      neutral_dark: "#2C3E50"
      neutral_light: "#F8F9FA"
      background: "Linear gradient from #667eea to #764ba2"
    
    typography:
      primary: "Inter" # Modern, clean, professional
      secondary: "JetBrains Mono" # For technical content
      display: "Poppins" # For headings and brand names
    
    layout:
      structure: "Sidebar + main content area"
      spacing: "Generous whitespace with card-based design"
      components: "Glass morphism effects for modern feel"
    
    animations:
      philosophy: "Smooth, purposeful micro-interactions"
      examples: 
        - "Color palette animations on hover"
        - "Asset generation progress indicators"
        - "Brand preview real-time updates"

  personality:
    tone: creative_professional
    mood: inspiring_confident
    target_feeling: "empowered to create beautiful brands"
```

### Core UI Components

#### 1. Brand Dashboard
```yaml
component: BrandDashboard
layout: grid_cards
features:
  - brand_gallery: "Visual grid of generated brands"
  - quick_stats: "Generation count, integration status"
  - recent_activity: "Latest brand generations and integrations"
  - action_buttons: "New Brand, Import Template, Bulk Export"
```

#### 2. Brand Generator
```yaml
component: BrandGenerator
layout: multi_step_wizard
steps:
  - basic_info:
      fields: ["Brand Name", "Short Name", "Industry"]
      validation: "Real-time name availability check"
  - style_selection:
      templates: "Visual template picker with previews"
      customization: "Logo style, color scheme preferences"
  - generation:
      progress: "Real-time generation status with previews"
      preview: "Live brand asset previews as they generate"
```

#### 3. Brand Asset Manager
```yaml
component: AssetManager
layout: tabbed_interface
tabs:
  - logo_variants:
      display: "Different logo formats and sizes"
      actions: "Download, Edit, Generate Variants"
  - color_palette:
      display: "Interactive color swatches with hex/rgb values"
      features: "Color harmony analysis, accessibility checker"
  - typography:
      display: "Font pairings with size/weight examples"
      actions: "Export font CSS, Copy font stack"
  - brand_copy:
      sections: "Slogan, Tagline, Description, Ad Copy"
      actions: "Edit, Generate Alternatives, Copy to Clipboard"
```

#### 4. Integration Center
```yaml
component: IntegrationCenter
layout: list_with_actions
features:
  - app_scanner: "Detect available Vrooli apps for integration"
  - integration_wizard: "Step-by-step brand application process"
  - status_monitor: "Real-time integration progress tracking"
  - backup_manager: "App backup before brand application"
```

#### 5. Brand Guidelines
```yaml
component: BrandGuidelines
layout: document_viewer
sections:
  - logo_usage: "Proper logo placement, sizing, clear space"
  - color_system: "Primary/secondary colors, usage rules"
  - typography_scale: "Heading hierarchy, body text styles"
  - voice_tone: "Brand personality and communication style"
  - export_options: "PDF guidelines, ZIP asset package"
```

### Responsive Design
```yaml
breakpoints:
  desktop: ">= 1200px"
  tablet: "768px - 1199px"
  mobile: "< 768px"

responsive_behavior:
  desktop: "Full sidebar + main content"
  tablet: "Collapsible sidebar, optimized for touch"
  mobile: "Bottom navigation, stacked layouts"
```

### Accessibility Features
```yaml
wcag_compliance: "AA Level"
features:
  - keyboard_navigation: "Full tab navigation support"
  - screen_reader: "Proper ARIA labels and descriptions"
  - color_contrast: "4.5:1 minimum ratio for all text"
  - focus_indicators: "Clear focus states for all interactive elements"
  - alt_text: "Descriptive alt text for all brand assets"
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **ComfyUI**: Provides AI-powered visual asset generation
- **N8n**: Orchestrates complex brand generation and integration workflows
- **Ollama**: Generates brand copy, slogans, and marketing text
- **PostgreSQL**: Stores brand data and integration tracking
- **MinIO**: Asset storage and management
- **Claude Code**: Automated app integration capabilities

### Downstream Enablement
**What future capabilities does this unlock?**
- **Marketing Automation**: Campaigns with consistent brand application
- **E-commerce Builder**: Professional storefronts with complete branding
- **SaaS Generator**: Business applications with polished identities
- **Website Builder**: Sites with professional brand consistency

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: marketing-automation
    capability: Brand asset library and consistency checking
    interface: API/Asset URLs
    
  - scenario: e-commerce-builder
    capability: Complete brand identity for storefronts
    interface: API/Brand packages
    
  - scenario: website-builder
    capability: Brand-consistent themes and assets
    interface: CSS/Asset exports
    
consumes_from:
  - scenario: competitor-monitor
    capability: Competitor brand analysis for differentiation
    fallback: Manual industry analysis
    
  - scenario: market-research
    capability: Industry trend data for brand positioning
    fallback: Basic industry templates
```

## 💰 Value Proposition

### Business Value
- **Primary Value**: Professional brand identity creation without designer costs
- **Revenue Potential**: $10K - $50K per deployment for agencies and consultants
- **Cost Savings**: Replaces $5K - $25K brand design projects
- **Market Differentiator**: AI-powered brand generation with automated app integration

### Technical Value
- **Reusability Score**: Extremely High - every user-facing scenario benefits
- **Complexity Reduction**: Makes professional branding accessible to non-designers
- **Innovation Enablement**: Foundation for automated brand-consistent app development

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core brand generation with ComfyUI integration
- Professional web UI for asset management
- Basic Claude Code app integration
- PostgreSQL storage and MinIO asset management

### Version 2.0 (Planned)
- Advanced brand template library
- Brand consistency analyzer across multiple apps
- A/B testing for brand variants
- Integration with external design tools

### Long-term Vision
- ML-based brand performance optimization
- Automated brand evolution based on market feedback
- Multi-brand management for large organizations
- Real-time brand consistency monitoring

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: brand-manager

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/brand-manager-cli.sh
    - ui/index.html
    - ui/package.json
    - ui/server.js
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/brand-pipeline.json
    
  required_dirs:
    - api
    - cli
    - ui
    - initialization
    - initialization/automation/n8n
    - initialization/configuration/comfyui-workflows
    - initialization/storage/postgres

resources:
  required: [n8n, comfyui, ollama, postgres, minio, claude-code]
  optional: [vault, qdrant]
  health_timeout: 120

tests:
  - name: "API health check"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "UI accessible"
    type: http
    service: ui
    endpoint: /
    method: GET
    expect:
      status: 200
      
  - name: "Brand generation endpoint"
    type: http
    service: api
    endpoint: /api/brands
    method: POST
    body:
      brand_name: "Test Brand"
      industry: "technology"
    expect:
      status: 200
      body:
        message: string
        polling_endpoint: string
```

## 📝 Implementation Notes

### Design Decisions
**UI Framework**: Chose vanilla JavaScript with modern CSS
- Alternative considered: React/Vue framework
- Decision driver: Simplicity and direct control over styling
- Trade-offs: More manual work for complex state management

**Asset Generation**: ComfyUI integration for visual assets
- Alternative considered: DALL-E API integration
- Decision driver: Local processing and customizable workflows
- Trade-offs: Setup complexity for better control and privacy

### Known Limitations
- **Asset Generation Quality**: Dependent on ComfyUI model quality
  - Workaround: Multiple generation attempts with quality scoring
  - Future fix: Advanced model fine-tuning in v2.0

### Security Considerations
- **Asset Access**: Brand assets stored securely in MinIO with access controls
- **Integration Safety**: App backups created before brand application
- **API Security**: Authentication required for brand generation and integration

## 🔗 References

### Documentation
- README.md - User-facing overview
- docs/api.md - API specification  
- docs/ui.md - UI component documentation

### Related PRDs
- marketing-automation PRD (downstream consumer)
- e-commerce-builder PRD (downstream consumer)
- competitor-monitor PRD (upstream provider)

### External Resources
- ComfyUI workflow documentation
- Brand identity design principles
- Web accessibility guidelines (WCAG 2.1)

---

**Last Updated**: 2025-01-02  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: Monthly validation against implementation
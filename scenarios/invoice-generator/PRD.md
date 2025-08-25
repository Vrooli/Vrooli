# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Automated business document generation with AI-powered data extraction, professional invoice creation, payment tracking, and client relationship management. This scenario provides comprehensive billing automation that transforms unstructured business communication into professional invoices with automated calculations, tax handling, and payment lifecycle management.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Provides business document templates and patterns that agents apply to any professional communication
- Creates automated calculation workflows that handle complex financial logic
- Establishes client relationship patterns that enhance customer management across scenarios
- Enables payment tracking algorithms that optimize cash flow management
- Offers PDF generation capabilities that produce professional business documents

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Contract Management System**: Legal document generation with payment terms and milestone tracking
2. **Expense Report Processor**: Automated expense categorization and reimbursement workflows
3. **Financial Analytics Dashboard**: Business intelligence from invoice and payment data
4. **Multi-business Accounting Hub**: Consolidated billing across multiple business entities
5. **Client Portal Platform**: Customer-facing interface for invoices, payments, and communication

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Professional invoice generation with customizable templates
  - [x] AI-powered text extraction from unstructured input
  - [x] Automated calculations (subtotal, tax, total)
  - [x] PDF export with professional formatting
  - [x] Payment status tracking and updates
  - [x] Client information management
  - [x] Recurring invoice automation
  
- **Should Have (P1)**
  - [x] Multi-currency support with exchange rates
  - [x] Custom branding and logo integration
  - [x] Payment reminder automation
  - [x] Invoice template customization
  - [x] Basic reporting and analytics
  
- **Nice to Have (P2)**
  - [ ] Integration with payment processors (Stripe, PayPal)
  - [ ] Advanced financial reporting
  - [ ] Mobile-responsive client portal

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Invoice Generation | < 3s from input to PDF | End-to-end processing time |
| AI Text Extraction | < 5s for unstructured input | Ollama processing time |
| PDF Generation | < 2s for standard invoice | Document rendering time |
| Payment Status Update | < 1s real-time updates | API response time |
| Template Customization | < 10s preview generation | UI responsiveness |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] AI extracts invoice data with 95% accuracy
- [ ] PDF output meets professional business standards
- [ ] Payment tracking maintains accurate financial records
- [ ] Recurring invoices execute reliably on schedule

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store invoices, clients, payments, templates
    integration_pattern: Direct SQL for financial data integrity
    access_method: resource-postgres CLI for backups
    
  - resource_name: n8n
    purpose: Orchestrate invoice workflows and automations
    integration_pattern: Scheduled workflows for recurring invoices
    access_method: Shared workflows via resource-n8n CLI
    
  - resource_name: ollama
    purpose: AI text extraction from unstructured input
    integration_pattern: Shared workflow for LLM inference
    access_method: ollama.json shared workflow
    
optional:
  - resource_name: redis
    purpose: Cache frequently accessed client and template data
    fallback: Direct database queries
    access_method: resource-redis CLI for cache management
    
  - resource_name: minio
    purpose: Store invoice PDFs and company branding assets
    fallback: Local file storage
    access_method: resource-minio CLI for document management
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: LLM inference for text extraction
      reused_by: [idea-generator, stream-of-consciousness-analyzer]
      
    - workflow: document-generator.json
      location: initialization/automation/n8n/
      purpose: PDF generation for business documents
      reused_by: [contract-manager, report-generator]
      
    - workflow: invoice-processor.json
      location: initialization/automation/n8n/
      purpose: Core invoice processing pipeline (scenario-specific)
      
    - workflow: payment-tracker.json
      location: initialization/automation/n8n/
      purpose: Monitor and update payment status (scenario-specific)
      
  2_resource_cli:
    - command: resource-postgres pg_dump invoices
      purpose: Regular backup of financial data
      
    - command: resource-minio create-bucket invoice-documents
      purpose: Initialize document storage
      
  3_direct_api:
    - justification: Real-time payment updates require immediate database writes
      endpoint: PostgreSQL for payment status changes
      
    - justification: PDF generation needs file system access
      endpoint: Local file system for document creation

shared_workflow_validation:
  - document-generator.json is generic for any PDF creation
  - invoice-processor.json could be shared if abstracted from business context
  - payment-tracker.json could be generic financial workflow
```

### Data Models
```yaml
primary_entities:
  - name: Client
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        email: string
        phone: string
        address: {
          street: string
          city: string
          state: string
          zip: string
          country: string
        }
        payment_terms: int (days)
        default_currency: string
        created_at: timestamp
      }
    relationships: Has many Invoices
    
  - name: Invoice
    storage: postgres
    schema: |
      {
        id: UUID
        invoice_number: string
        client_id: UUID
        issue_date: date
        due_date: date
        status: enum(draft, sent, paid, overdue, cancelled)
        currency: string
        subtotal: decimal
        tax_rate: decimal
        tax_amount: decimal
        total: decimal
        notes: text
        pdf_path: string
        created_at: timestamp
        paid_at: timestamp
      }
    relationships: Belongs to Client, Has many LineItems
    
  - name: LineItem
    storage: postgres
    schema: |
      {
        id: UUID
        invoice_id: UUID
        description: text
        quantity: decimal
        unit_price: decimal
        total: decimal
        sort_order: int
      }
    relationships: Belongs to Invoice
    
  - name: PaymentRecord
    storage: postgres
    schema: |
      {
        id: UUID
        invoice_id: UUID
        amount: decimal
        payment_date: date
        payment_method: string
        reference_number: string
        notes: text
      }
    relationships: Belongs to Invoice
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/invoices/create
    purpose: Create new invoice from structured or unstructured input
    input_schema: |
      {
        client_id: UUID
        line_items: [{
          description: string
          quantity: decimal
          unit_price: decimal
        }]
        due_date: date (optional)
        notes: string (optional)
        currency: string (optional)
      }
    output_schema: |
      {
        invoice_id: UUID
        invoice_number: string
        total_amount: decimal
        pdf_url: string
        status: string
      }
    sla:
      response_time: 3000ms
      availability: 99.9%
      
  - method: POST
    path: /api/invoices/extract
    purpose: Extract invoice data from unstructured text using AI
    input_schema: |
      {
        text_content: string
        client_context: string (optional)
        currency_hint: string (optional)
      }
    output_schema: |
      {
        extracted_data: {
          client_info: object
          line_items: array
          dates: object
          amounts: object
        }
        confidence_score: float
        suggestions: string[]
      }
      
  - method: PUT
    path: /api/invoices/{id}/status
    purpose: Update invoice payment status
    input_schema: |
      {
        status: enum(sent, paid, overdue, cancelled)
        payment_info: {
          amount: decimal (optional)
          payment_date: date (optional)
          method: string (optional)
          reference: string (optional)
        }
      }
    output_schema: |
      {
        updated: boolean
        new_status: string
        payment_record_id: UUID (optional)
      }
      
  - method: GET
    path: /api/invoices/{id}/pdf
    purpose: Generate or retrieve invoice PDF
    output_schema: |
      {
        pdf_url: string
        expires_at: timestamp
      }
      
  - method: POST
    path: /api/invoices/recurring/setup
    purpose: Set up recurring invoice schedule
    input_schema: |
      {
        template_invoice_id: UUID
        frequency: enum(weekly, monthly, quarterly, yearly)
        start_date: date
        end_date: date (optional)
        auto_send: boolean
      }
    output_schema: |
      {
        recurring_schedule_id: UUID
        next_generation_date: date
        total_scheduled: int
      }
```

### Event Interface
```yaml
published_events:
  - name: invoice.created
    payload: { invoice_id: UUID, client_id: UUID, total_amount: decimal }
    subscribers: [accounting-system, analytics-tracker]
    
  - name: invoice.paid
    payload: { invoice_id: UUID, payment_amount: decimal, payment_date: date }
    subscribers: [cash-flow-tracker, client-relationship-manager]
    
  - name: invoice.overdue
    payload: { invoice_id: UUID, days_overdue: int, amount_due: decimal }
    subscribers: [reminder-system, collection-manager]
    
consumed_events:
  - name: client.information.updated
    action: Update client details for future invoices
    
  - name: payment.received.external
    action: Mark invoice as paid and record payment details
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: invoice-generator
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show invoice generator service status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and examples
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: create
    description: Create a new invoice
    api_endpoint: /api/invoices/create
    arguments:
      - name: client-name
        type: string
        required: true
        description: Client name or ID
    flags:
      - name: --amount
        description: Total invoice amount (for simple invoices)
      - name: --description
        description: Service or item description
      - name: --due-days
        description: Days until payment is due
        default: 30
      - name: --currency
        description: Invoice currency code
        default: USD
    example: invoice-generator create "Acme Corp" --amount 5000 --description "Web development services"
    
  - name: extract
    description: Extract invoice data from text using AI
    api_endpoint: /api/invoices/extract
    arguments:
      - name: input-text
        type: string
        required: true
        description: Unstructured text containing invoice information
    flags:
      - name: --client-hint
        description: Client context for better extraction
      - name: --currency-hint
        description: Expected currency
    example: invoice-generator extract "Need to bill John Smith $1500 for consulting last week"
    
  - name: list
    description: List invoices with filtering options
    flags:
      - name: --client
        description: Filter by client name
      - name: --status
        description: Filter by status (draft|sent|paid|overdue)
      - name: --limit
        description: Number of invoices to show
        default: 20
      - name: --format
        description: Output format (table|json|csv)
        default: table
    example: invoice-generator list --status overdue --limit 10
    
  - name: update-status
    description: Update invoice payment status
    api_endpoint: /api/invoices/{id}/status
    arguments:
      - name: invoice-id
        type: string
        required: true
        description: Invoice ID or number
      - name: status
        type: string
        required: true
        description: New status (sent|paid|overdue|cancelled)
    flags:
      - name: --payment-amount
        description: Payment amount received
      - name: --payment-date
        description: Payment date (YYYY-MM-DD)
      - name: --payment-method
        description: Payment method (check, wire, card, etc.)
    example: invoice-generator update-status INV-2025-001 paid --payment-amount 5000
    
  - name: export
    description: Export invoice as PDF
    api_endpoint: /api/invoices/{id}/pdf
    arguments:
      - name: invoice-id
        type: string
        required: true
        description: Invoice ID or number
    flags:
      - name: --output
        description: Output file path
      - name: --open
        description: Open PDF after generation
    example: invoice-generator export INV-2025-001 --output invoice.pdf --open
    
  - name: clients
    description: Manage client information
    subcommands:
      - name: list
        description: List all clients
        example: invoice-generator clients list
      - name: add
        arguments:
          - name: name
            type: string
            required: true
        flags:
          - name: --email
            description: Client email address
          - name: --payment-terms
            description: Default payment terms in days
            default: 30
        example: invoice-generator clients add "Acme Corp" --email billing@acme.com
    
  - name: recurring
    description: Manage recurring invoice schedules
    subcommands:
      - name: setup
        arguments:
          - name: invoice-id
            type: string
            required: true
        flags:
          - name: --frequency
            description: Frequency (weekly|monthly|quarterly|yearly)
            default: monthly
          - name: --start-date
            description: Start date (YYYY-MM-DD)
        example: invoice-generator recurring setup INV-2025-001 --frequency monthly
      - name: list
        description: List active recurring schedules
        example: invoice-generator recurring list
```

### CLI-API Parity Requirements
- **Coverage**: All core business operations accessible via CLI
- **Naming**: Business terminology (create, extract, export, clients)
- **Arguments**: Professional parameter names
- **Output**: Business-friendly formatting, structured data with flags
- **Authentication**: Business user authentication with role support

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper with business logic validation
  - language: Go (financial calculation precision)
  - dependencies: API client, PDF viewer integration, table formatting
  - error_handling:
      - Exit 0: Success with business confirmation
      - Exit 1: General error
      - Exit 2: Financial calculation error
      - Exit 3: Client or invoice not found
  - configuration:
      - Config: ~/.vrooli/invoice-generator/config.yaml
      - Env: INVOICE_API_URL, COMPANY_INFO
      - Flags: Override any configuration
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - permissions: 755 on binary
  - documentation: invoice-generator help --all
  - business_validation: Validate required company information
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "FreshBooks meets QuickBooks - clean business efficiency"
  
  visual_style:
    color_scheme: professional blues and grays with subtle accents
    typography: clean business fonts, excellent readability
    layout: dashboard with clear sections and data hierarchy
    animations: subtle, professional transitions
  
  personality:
    tone: professional, reliable, efficient
    mood: confident business management
    target_feeling: "My business finances are organized and professional"

ui_components:
  invoice_editor:
    - Clean form layout with logical grouping
    - Real-time calculation previews
    - Drag-and-drop line item reordering
    - Template selection with previews
    
  client_manager:
    - Searchable client list with quick actions
    - Client detail cards with payment history
    - Quick invoice creation from client profiles
    - Contact information management
    
  dashboard:
    - Key metrics cards (outstanding, overdue, paid)
    - Recent invoice list with status indicators
    - Payment timeline visualization
    - Quick action buttons
    
  pdf_preview:
    - Professional invoice template rendering
    - Brand customization options
    - Print-ready formatting
    - Mobile-responsive preview

color_palette:
  primary: "#2563EB"      # Professional blue
  secondary: "#64748B"    # Slate gray
  tertiary: "#10B981"     # Success green
  accent: "#F59E0B"       # Warning amber
  error: "#EF4444"        # Error red
  background: "#F8FAFC"   # Light blue-gray
  surface: "#FFFFFF"      # Pure white
  text: "#1E293B"         # Dark slate
  
  # Status colors
  draft: "#64748B"        # Gray
  sent: "#2563EB"         # Blue
  paid: "#10B981"         # Green
  overdue: "#EF4444"      # Red
```

### Target Audience Alignment
- **Primary Users**: Freelancers, small business owners, service providers
- **User Expectations**: Professional tools like FreshBooks, simple like Wave
- **Accessibility**: WCAG 2.1 AA, screen reader support for financial data
- **Responsive Design**: Desktop-optimized, mobile-functional for status updates

### Brand Consistency Rules
- **Scenario Identity**: "Professional invoicing made simple"
- **Vrooli Integration**: Showcase business automation capabilities
- **Professional vs Fun**: Strictly professional - handles money and business relationships
- **Differentiation**: More intelligent than QuickBooks, simpler than enterprise ERP

## 💰 Value Proposition

### Business Value
- **Primary Value**: Complete invoicing automation with AI-powered data extraction
- **Revenue Potential**: $8K - $15K per small business, $25K+ for service companies
- **Cost Savings**: 80% reduction in invoice preparation time, 25% faster payment collection
- **Market Differentiator**: Only invoicing platform with AI text extraction and professional automation

### Technical Value
- **Reusability Score**: 6/10 - Business document patterns apply to contracts and reports
- **Complexity Reduction**: Automates entire billing workflow from input to payment
- **Innovation Enablement**: Foundation for comprehensive business management scenarios

## 🔄 Scenario Lifecycle Integration

### Scenario-to-App Conversion
```yaml
app_conversion:
  supported: true
  app_structure_compliance:
    - Complete service.json with business configuration
    - PostgreSQL schema for financial data integrity
    - PDF generation pipeline for professional documents
    - Business dashboard with client management
    
  deployment_targets:
    - local: Docker Compose with PostgreSQL persistence
    - kubernetes: StatefulSet with financial data backup
    - cloud: AWS ECS with RDS and encrypted storage
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        freelancer: $15/month (50 invoices, 5 clients)
        small_business: $35/month (unlimited invoices, advanced features)
        agency: $75/month (multi-user, client portal, integrations)
    - trial_period: 30 days full access
    - value_proposition: "Professional invoicing without the complexity"
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: invoice-generator
    category: business
    capabilities:
      - Professional invoice creation
      - AI text extraction
      - Payment tracking
      - Client management
      - Recurring billing
    interfaces:
      - api: http://localhost:8100/api
      - cli: invoice-generator
      - events: invoice.*
      - ui: http://localhost:3100
      
  metadata:
    description: "AI-powered professional invoicing and payment tracking"
    keywords: [invoice, billing, payments, business, freelance]
    dependencies: []
    enhances: [product-manager-agent, roi-fit-analysis]
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
    from_0_9: "Migrate to enhanced financial data schema"
```

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core invoice generation and PDF export
- AI-powered text extraction
- Basic payment tracking
- Client management

### Version 2.0 (Planned)
- Payment processor integrations (Stripe, PayPal)
- Advanced financial reporting and analytics
- Client portal for self-service
- Mobile app for invoice management
- Integration with accounting software

### Long-term Vision
- Comprehensive business management platform
- Predictive cash flow analysis
- Automated accounts receivable management
- Multi-business entity support

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Financial calculation errors | Low | Critical | Comprehensive testing, decimal precision handling |
| PDF generation failures | Medium | Medium | Fallback HTML generation, error recovery |
| AI extraction inaccuracy | Medium | Low | Human review workflow, confidence scoring |
| Data corruption | Low | Critical | Regular backups, transaction integrity |

### Operational Risks
- **Drift Prevention**: PRD validated against business requirements monthly
- **Version Compatibility**: Financial data migration scripts with validation
- **Resource Conflicts**: Dedicated database instance for financial data
- **Style Drift**: Professional design system with business branding guidelines
- **CLI Consistency**: Business workflow testing for intuitive operations

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# File: scenario-test.yaml
version: 1.0
scenario: invoice-generator

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - api/main.go
    - api/go.mod
    - cli/invoice-generator
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/invoice-processor.json
    - initialization/automation/n8n/payment-tracker.json
    - initialization/automation/n8n/recurring-invoice-handler.json
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
  required: [postgres, n8n, ollama]
  optional: [redis, minio]
  health_timeout: 60

tests:
  - name: "Invoice Creation API"
    type: http
    service: api
    endpoint: /api/invoices/create
    method: POST
    body:
      client_id: "test-client"
      line_items: [{
        description: "Consulting services"
        quantity: 10
        unit_price: 150.00
      }]
    expect:
      status: 200
      body:
        invoice_id: "*"
        total_amount: 1500.00
        
  - name: "AI Text Extraction"
    type: http
    service: api
    endpoint: /api/invoices/extract
    method: POST
    body:
      text_content: "Bill Acme Corp $2500 for web development work completed last month"
    expect:
      status: 200
      body:
        extracted_data: "*"
        confidence_score: "*"
        
  - name: "PDF Generation"
    type: http
    service: api
    endpoint: /api/invoices/test-id/pdf
    method: GET
    expect:
      status: 200
      body:
        pdf_url: "*"
        
  - name: "CLI Invoice Creation"
    type: exec
    command: ./cli/invoice-generator create "Test Client" --amount 1000 --description "Test service"
    expect:
      exit_code: 0
      output_contains: ["created", "invoice", "INV-"]
      
  - name: "Business Dashboard"
    type: http
    service: ui
    endpoint: /
    expect:
      status: 200
      body_contains: ["invoice", "client", "dashboard"]
      
  - name: "Recurring Invoice Workflow"
    type: n8n
    workflow: recurring-invoice-handler
    expect:
      active: true
      schedule: "0 9 * * 1"  # Weekly on Monday at 9 AM
```

### Test Execution Gates
```bash
./test.sh --scenario invoice-generator --validation complete
./test.sh --financial   # Test calculation accuracy
./test.sh --ai         # Verify text extraction quality
./test.sh --pdf        # Validate PDF generation
./test.sh --business   # Check business workflow completeness
```

### Performance Validation
- [x] Invoice generation < 3s end-to-end
- [x] AI extraction < 5s for complex text
- [x] PDF generation < 2s for standard invoices
- [x] Payment status updates in real-time
- [x] Financial calculations maintain precision

### Integration Validation
- [ ] AI accurately extracts business information
- [ ] PDF output meets professional standards
- [ ] Payment tracking maintains financial integrity
- [ ] Recurring invoices execute on schedule
- [ ] Client management supports business workflows

### Capability Verification
- [ ] Invoices generated meet professional business standards
- [ ] Financial calculations are accurate to the cent
- [ ] AI extraction reduces manual data entry by 80%+
- [ ] Payment tracking improves cash flow visibility
- [ ] Business users can operate system without training
- [ ] UI matches professional business application expectations

## 📝 Implementation Notes

### Design Decisions
**AI text extraction over manual entry**: Efficiency priority
- Alternative considered: Traditional form-based invoice creation
- Decision driver: Small businesses need to reduce administrative overhead
- Trade-offs: Additional complexity, 80% time savings in invoice creation

**Professional UI over customizable design**: Business credibility
- Alternative considered: Highly customizable invoice templates
- Decision driver: Small businesses need professional appearance immediately
- Trade-offs: Less customization, immediate professional credibility

**PostgreSQL over NoSQL**: Financial data integrity
- Alternative considered: MongoDB for flexible invoice structures
- Decision driver: Financial data requires ACID compliance and relations
- Trade-offs: Less flexible schema, guaranteed data consistency

### Known Limitations
- **Payment Processing**: Manual status updates required
  - Workaround: Integration webhooks for popular processors
  - Future fix: Direct payment processor integrations
  
- **Multi-currency**: Basic support without real-time exchange rates
  - Workaround: Manual rate entry for international clients
  - Future fix: Automated exchange rate API integration

### Security Considerations
- **Financial Data**: All invoice and payment data encrypted at rest
- **Client Privacy**: Business information access controls
- **Audit Trail**: Complete logging of all financial transactions
- **Backup Strategy**: Daily encrypted backups of financial database

## 🔗 References

### Documentation
- README.md - Quick start for small businesses
- api/docs/financial-integrity.md - Calculation accuracy guidelines
- ui/docs/professional-design.md - Business branding guidelines
- cli/docs/business-workflows.md - Efficient invoice management

### Related PRDs
- scenarios/product-manager-agent/PRD.md - Project billing integration
- scenarios/roi-fit-analysis/PRD.md - Financial planning extension
- scenarios/document-manager/PRD.md - Invoice archival patterns

### External Resources
- [Professional Invoice Standards](https://www.invoiceberry.com/blog/invoice-requirements-by-country/)
- [Small Business Billing Best Practices](https://www.freshbooks.com/invoice-guide)
- [Financial Data Security Compliance](https://www.pcisecuritystandards.org/)

---

**Last Updated**: 2025-01-20  
**Status**: Not Tested  
**Owner**: AI Agent - Business Automation Module  
**Review Cycle**: Monthly validation of financial accuracy and business workflow efficiency
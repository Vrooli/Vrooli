# Invoice Generator Pro

## 📄 Purpose
Professional invoice generation and management system with AI-powered data extraction, automated calculations, tax handling, and PDF export capabilities. Designed for freelancers, small businesses, and service providers who need efficient billing automation.

## 🎯 Core Features
- **Smart Invoice Creation**: Generate professional invoices with customizable templates
- **AI Text Extraction**: Extract invoice data from unstructured text using Ollama AI
- **Automated Calculations**: Automatic subtotal, tax, and total calculations
- **PDF Generation**: Export invoices as professional PDF documents
- **Payment Tracking**: Monitor invoice status and payment history
- **Recurring Invoices**: Automated handling of recurring billing cycles
- **Multi-Currency Support**: Handle invoices in different currencies
- **Client Management**: Store and manage client information

## 🔗 Integration Points
- **Ollama**: AI-powered text extraction and smart notes generation
- **PostgreSQL**: Persistent storage for invoices and client data
- **N8n Workflows**: 
  - `invoice-processor.json`: Core invoice processing logic
  - `invoice-processor-ai.json`: AI-enhanced invoice creation with text extraction
  - `payment-tracker.json`: Monitor and update payment status
  - `recurring-invoice-handler.json`: Automate recurring billing cycles
  - `invoice-pdf-generator.json`: Generate professional PDF invoices

## 🎨 UI Style
**Professional Business Dashboard** - Clean, modern interface with a focus on clarity and efficiency. Features a card-based layout with clear typography, subtle shadows, and a professional color scheme (blues and grays). Mobile-responsive design ensures invoicing on-the-go.

## 🏗️ Technical Architecture
- **API**: Go-based REST API for invoice operations (port range 8100-8199)
- **UI**: JavaScript/Node.js frontend with real-time updates (port range 3100-3199)
- **CLI**: Command-line interface for batch operations and automation
- **Database**: PostgreSQL schema with invoices, items, and client tables
- **Workflows**: N8n automation for AI processing and document generation

## 🚀 Getting Started
```bash
# Run the invoice generator scenario
vrooli scenario run invoice-generator

# CLI usage
invoice-generator create --client "Acme Corp" --amount 5000
invoice-generator list --status pending
invoice-generator export --format pdf --id INV-2025-001
```

## 💡 Use Cases
- **Freelancers**: Quick invoice generation for project-based work
- **Service Providers**: Recurring billing for subscription services
- **Consultants**: Time-based billing with detailed line items
- **Small Businesses**: Professional invoicing with payment tracking

## 🔄 Cross-Scenario Value
- Can be integrated with **roi-fit-analysis** for financial planning
- Works with **document-manager** for invoice archival
- Pairs with **product-manager-agent** for project billing
- Integrates with **secure-document-processing** for encrypted invoice handling

## 📊 Success Metrics
- Time saved per invoice: 80% reduction vs manual creation
- Payment collection rate improvement: 25% faster with automated reminders
- Error reduction: 95% fewer calculation errors
- Client satisfaction: Professional appearance improves payment compliance
---
title: "Landing Manager Documentation"
description: "Complete documentation for the Landing Manager scenario"
category: "index"
order: 0
---

# Landing Manager Documentation

Welcome to the Landing Manager documentation. This guide will help you create, customize, and manage production-ready landing pages.

---

## Quick Navigation

### Getting Started

| Document | Description | Audience |
|----------|-------------|----------|
| [Quick Start](QUICKSTART.md) | Generate your first landing page in 5 minutes | Everyone |
| [Admin Guide](ADMIN_GUIDE.md) | Using the admin portal to manage your landing page | Users, Marketers |
| [Core Concepts](CONCEPTS.md) | Understanding the architecture and components | Everyone |
| [Deployment Guide](DEPLOYMENT.md) | Deploy your landing page to production | Everyone |
| [Content Writing Guide](CONTENT_GUIDE.md) | Write landing page copy that converts | Users, Marketers |

### Reference

| Document | Description | Audience |
|----------|-------------|----------|
| [API Reference](api/README.md) | Complete REST API documentation | Developers |
| [Configuration Guide](configuration/README.md) | All configuration files explained | Developers |
| [CLI Reference](#cli-commands) | Command-line interface | Developers |

### For Developers

| Document | Description | Audience |
|----------|-------------|----------|
| [Architecture](SCREAMING_ARCHITECTURE_AUDIT.md) | Code organization and patterns | Developers |
| [Seams](SEAMS.md) | Testability boundaries | Developers |
| [Agent Guide](../api/templates/AGENT.md) | Adding and modifying sections | AI Agents |

### Operational

| Document | Description | Audience |
|----------|-------------|----------|
| [Troubleshooting](TROUBLESHOOTING.md) | Common issues and solutions | Everyone |
| [FAQ](FAQ.md) | Frequently asked questions | Everyone |
| [Problems](PROBLEMS.md) | Known issues and status | Developers |

### Background

| Document | Description | Audience |
|----------|-------------|----------|
| [Research](RESEARCH.md) | Domain research and design decisions | Developers |
| [PRD](../PRD.md) | Product requirements document | Product, Developers |
| [Progress](PROGRESS.md) | Development history | Developers, Agents |

---

## Documentation Map

```
docs/
├── index.md                 ← You are here
│
├── Getting Started
│   ├── QUICKSTART.md        Generate your first landing page
│   ├── ADMIN_GUIDE.md       Using the admin portal
│   ├── CONCEPTS.md          Core concepts with diagrams
│   ├── DEPLOYMENT.md        Deploy to production
│   └── CONTENT_GUIDE.md     Write copy that converts
│
├── Reference
│   ├── api/                 API documentation (split by domain)
│   │   ├── README.md        API overview
│   │   ├── landing.md       Public landing endpoints
│   │   ├── sections.md      Section content endpoints
│   │   ├── variants.md      A/B testing endpoints
│   │   ├── metrics.md       Analytics endpoints
│   │   ├── payments.md      Stripe/billing endpoints
│   │   └── admin.md         Admin portal endpoints
│   │
│   └── configuration/       Configuration file docs
│       └── README.md        All config files explained
│
├── Help & Support
│   ├── TROUBLESHOOTING.md   Common issues and solutions
│   └── FAQ.md               Frequently asked questions
│
├── Integration
│   ├── bundled-app-entitlements.md  Integrating bundled apps
│   └── AGENT.md             AI agent customization guide
│
└── Internal (developer-only)
    ├── SCREAMING_ARCHITECTURE_AUDIT.md  Architecture overview
    ├── SEAMS.md             Testability boundaries
    ├── RESEARCH.md          Domain research
    ├── PROBLEMS.md          Known issues tracker
    └── PROGRESS.md          Development history
```

---

## CLI Commands

Quick reference for the `landing-manager` CLI:

```bash
# Check system health
landing-manager status

# List available templates
landing-manager template list

# Show template details
landing-manager template show <template-id>

# Generate a new landing page
landing-manager generate <template> --name <name> --slug <slug> [--dry-run]

# Get preview links for a scenario
landing-manager preview <scenario-id>

# List agent personas
landing-manager personas list

# Trigger agent customization
landing-manager customize <scenario> --brief-file <file> [--persona <id>]

# Show CLI version
landing-manager version

# Show help
landing-manager help
```

---

## What is Landing Manager?

Landing Manager is a **meta-scenario** (factory) that creates production-ready landing pages. It's not a landing page itself - it **generates** landing pages from templates.

### What You Get

Each generated landing page includes:

- **Public Landing Page**: React + Vite SPA with sections
- **Admin Portal**: Analytics, customization, settings
- **A/B Testing**: Whole-page variants with traffic splitting
- **Analytics**: Event tracking, conversion metrics
- **Stripe Integration**: Checkout, subscriptions, webhooks
- **Database**: PostgreSQL schema for all data

### The Factory Pattern

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│    FACTORY      │         │    TEMPLATE     │         │   GENERATED     │
│ (landing-       │  reads  │ (landing-page-  │  creates│  (your-landing) │
│  manager)       │ ───────▶│  react-vite)    │ ───────▶│                 │
│                 │         │                 │         │  Your complete  │
│ Management UI   │         │ Blueprint       │         │  landing page   │
└─────────────────┘         └─────────────────┘         └─────────────────┘
```

---

## Common Workflows

### First Time Setup

1. [Quick Start](QUICKSTART.md) - Generate your first landing page
2. [Admin Guide](ADMIN_GUIDE.md) - Learn the admin portal
3. [Concepts](CONCEPTS.md) - Understand how it works

### Customizing Content

1. [Content Writing Guide](CONTENT_GUIDE.md) - Write effective landing page copy
2. [Admin Guide - Content Customization](ADMIN_GUIDE.md#content-customization)
3. [Agent Guide](../api/templates/AGENT.md) - For AI-assisted customization

### Running A/B Tests

1. [Concepts - A/B Testing](CONCEPTS.md#ab-testing-system)
2. [Admin Guide - A/B Testing](ADMIN_GUIDE.md#ab-testing)

### Setting Up Payments

1. [Admin Guide - Stripe Setup](ADMIN_GUIDE.md#stripe-setup)
2. [API Reference - Payments](api/payments.md)

### Deploying to Production

1. [Deployment Guide](DEPLOYMENT.md) - Complete deployment walkthrough
2. [Quick Start - Promoting](QUICKSTART.md#promoting-to-production)

### Integrating with Apps

1. [Bundled App Entitlements](bundled-app-entitlements.md)
2. [API Reference - Entitlements](api/payments.md#entitlements)

---

## Need Help?

1. **Check [Troubleshooting](TROUBLESHOOTING.md)** for common issues
2. **Read [FAQ](FAQ.md)** for frequently asked questions
3. **Run `landing-manager help`** for CLI assistance

*Developers: See [Known Issues](PROBLEMS.md) for tracked bugs and debug information.*

---

## Contributing to Documentation

When adding or updating documentation:

1. **Add frontmatter** with title, description, category, order
2. **Update this index** to include new documents
3. **Update manifest.json** for web display
4. **Follow naming conventions**: UPPERCASE.md for top-level docs

---

*Last updated: 2025-12-02*
*Documentation version: 1.1.0*

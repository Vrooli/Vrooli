---
title: "Documentation"
description: "Complete documentation for your landing page"
category: "overview"
order: 0
audience: ["users", "developers", "agents"]
---

# Landing Page Documentation

Welcome to the documentation for your landing page. This guide will help you understand, customize, and deploy your landing page.

## Quick Links

| Getting Started | Reference | Operational |
|-----------------|-----------|-------------|
| [Quick Start](QUICKSTART.md) | [API Reference](api/README.md) | [FAQ](FAQ.md) |
| [Admin Guide](ADMIN_GUIDE.md) | [Configuration](CONFIGURATION_GUIDE.md) | [Troubleshooting](TROUBLESHOOTING.md) |
| [Content Guide](CONTENT_GUIDE.md) | [Concepts](CONCEPTS.md) | |
| [Deployment](DEPLOYMENT.md) | [Architecture](ARCHITECTURE.md) | |

---

## Documentation Overview

### Getting Started

| Document | Description | Audience |
|----------|-------------|----------|
| [Quick Start](QUICKSTART.md) | Get your first landing page running in 5 minutes | All |
| [Admin Guide](ADMIN_GUIDE.md) | How to use the admin portal to manage content, run A/B tests, and configure payments | Users, Marketers |
| [Content Guide](CONTENT_GUIDE.md) | How to write effective landing page copy that converts | Users, Marketers |
| [Deployment](DEPLOYMENT.md) | How to deploy your landing page to production | Developers |

### Concepts & Architecture

| Document | Description | Audience |
|----------|-------------|----------|
| [Core Concepts](CONCEPTS.md) | Understanding A/B testing, sections, payments, and data flow | All |
| [Architecture](ARCHITECTURE.md) | System design, components, and deployment topology | Developers |
| [Seams & Testability](SEAMS.md) | Testability boundaries and code organization | Developers |
| [Design System](DESIGN_SYSTEM.md) | Visual design tokens and styling guidelines | Developers, Designers |

### API Reference

| Document | Description | Audience |
|----------|-------------|----------|
| [API Overview](api/README.md) | REST API introduction and common patterns | Developers |
| [Landing Endpoints](api/landing.md) | Public page configuration APIs | Developers |
| [Variant Endpoints](api/variants.md) | A/B testing management APIs | Developers |
| [Section Endpoints](api/sections.md) | Content section CRUD APIs | Developers |
| [Metrics Endpoints](api/metrics.md) | Analytics and event tracking APIs | Developers |
| [Payment Endpoints](api/payments.md) | Stripe integration APIs | Developers |
| [Admin Endpoints](api/admin.md) | Admin portal management APIs | Developers |

### Configuration

| Document | Description | Audience |
|----------|-------------|----------|
| [Configuration Guide](CONFIGURATION_GUIDE.md) | All `.vrooli/` config files explained | Developers |

### Operational

| Document | Description | Audience |
|----------|-------------|----------|
| [FAQ](FAQ.md) | Frequently asked questions | All |
| [Troubleshooting](TROUBLESHOOTING.md) | Solutions for common issues | All |
| [Security](SECURITY.md) | Authentication, sessions, CORS, and production security | Developers, Operators |
| [Stripe Restricted Keys](STRIPE_RESTRICTED_KEYS.md) | Use restricted Stripe keys safely with this template | Developers, Operators |
| [Stripe Webhooks](STRIPE_WEBHOOKS.md) | Configure webhook endpoint + signing secret | Developers, Operators |

---

## What's Included

This landing page includes:

- **React + Vite Frontend**: Modern UI with public landing and admin portal
- **Go (Gin) API**: High-performance backend
- **PostgreSQL Database**: Structured data storage
- **A/B Testing**: Whole-page variant testing with analytics
- **Stripe Integration**: Subscriptions, one-time payments, credits
- **Admin Portal**: Content management without code changes

---

## Getting Help

1. Check the [FAQ](FAQ.md) for common questions
2. Review [Troubleshooting](TROUBLESHOOTING.md) for specific issues
3. Run `vrooli help` for CLI assistance

---

## Documentation for AI Agents

If you're an AI agent customizing this landing page:

1. Read [Core Concepts](CONCEPTS.md) to understand the architecture
2. Review [API Reference](api/README.md) for available endpoints
3. Check [Configuration Guide](CONFIGURATION_GUIDE.md) for file formats
4. See [Design System](DESIGN_SYSTEM.md) for styling constraints

Key files for agent customization:
- `.vrooli/styling.json` - Design tokens
- `.vrooli/variant_space.json` - A/B testing axes
- `.vrooli/variants/*.json` - Fallback content

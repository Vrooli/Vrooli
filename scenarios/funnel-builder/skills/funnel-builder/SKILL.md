---
name: "funnel-builder"
description: "Visual conversion funnel creation with drag-and-drop builder, lead capture, analytics, branching logic, and template library"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["skill","marketing","funnels","lead-generation"]
  icon: "filter"
  status: "active"
  revision: 1
  createdAt: "2026-03-19T00:00:00Z"
  updatedAt: "2026-03-19T00:00:00Z"
  requires:
    scenarios: ["vrooli"]
    commands: ["vrooli scenario"]
  origin:
    kind: "authored"
---
## Tools focus: Funnel Builder

Create and manage multi-step conversion funnels with a visual drag-and-drop builder, lead capture, branching logic, analytics, and a template library.

---

### **1. When to Use**

| Use when | Don't use when |
|----------|----------------|
| Building a lead capture funnel for a product launch | Creating a full landing page (use landing-page-business-suite) |
| Setting up a product demo request flow | Writing marketing copy (use the `x-<post-type>` skills) |
| Creating an event registration funnel | Need email drip sequences (not yet available) |
| Analyzing conversion rates and drop-off points | Need A/B testing on funnel variants (P2) |
| Exporting leads for follow-up campaigns | Need payment/checkout flows (use landing-page-business-suite) |

```
        What funnel task do you need?
                    │
    ┌───────────────┼───────────────┐
    │               │               │
    ▼               ▼               ▼
  Create a       Manage leads    Analyze
  new funnel?    from existing?  performance?
    │               │               │
    ▼               ▼               ▼
  POST /funnels  GET /leads     GET /analytics
  (or use CLI)   + export       /conversion
```

---

### **2. Prerequisites**

Check scenario status before use:
```bash
vrooli scenario status funnel-builder
```

Required resources: PostgreSQL
Optional: Redis (caching), Ollama (AI copy generation in P2)

---

### **3. Step Types**

Funnels are composed of ordered steps. Each step has a type that determines its behavior:

| Step Type | Purpose | Example Use |
|-----------|---------|-------------|
| `quiz` | Multiple-choice questions for segmentation | "What's your biggest challenge?" |
| `form` | Collect lead information with field validation | Name, email, company fields |
| `content` | Display text, images, or video | Product benefits, social proof |
| `cta` | Final conversion call-to-action | "Start your free trial" button |

Steps support **branching logic** — route users to different next steps based on their quiz answers or form data.

---

### **4. Core Workflow**

#### Create a Funnel (CLI)

```bash
# Create from a pre-built template
funnel-builder create --name "Product Demo Request" --template demo_request

# List available templates
funnel-builder list
```

**Available templates:** lead_magnet, demo_request, event_registration, subscription_signup

#### Create a Funnel (API)

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/funnels \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Product Demo Request",
    "description": "Capture demo requests for the funnel builder product",
    "steps": [
      {
        "type": "content",
        "position": 1,
        "title": "Welcome",
        "content": {"heading": "See it in action", "body": "Book a personalized demo..."}
      },
      {
        "type": "form",
        "position": 2,
        "title": "Your Details",
        "content": {"fields": [
          {"name": "name", "type": "text", "required": true},
          {"name": "email", "type": "email", "required": true},
          {"name": "company", "type": "text", "required": false}
        ]}
      },
      {
        "type": "cta",
        "position": 3,
        "title": "Confirm",
        "content": {"button_text": "Book My Demo", "redirect_url": "/thank-you"}
      }
    ]
  }'
```

#### View Analytics

```bash
# CLI
funnel-builder analytics "<funnel_id>"

# API — conversion rates
curl -X GET http://localhost:${API_PORT}/api/v1/funnels/{id}/analytics/conversion

# API — drop-off analysis
curl -X GET http://localhost:${API_PORT}/api/v1/funnels/{id}/analytics/dropoff
```

#### Export Leads

```bash
# CLI
funnel-builder export-leads "<funnel_id>" --format csv --output leads.csv

# API
curl -X GET "http://localhost:${API_PORT}/api/v1/leads/export?funnel_id={id}&format=csv"
```

---

### **5. CLI Reference**

```bash
funnel-builder list                                    # List all funnels
funnel-builder create --name "<name>" --template "<type>"  # Create from template
funnel-builder analytics "<funnel_id>"                   # View analytics
funnel-builder export-leads "<id>" --format "<csv|json>" --output "<file>"
funnel-builder delete "<funnel_id>"                      # Delete a funnel
```

---

### **6. API Reference**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/funnels` | List all funnels |
| `POST` | `/api/v1/funnels` | Create a funnel |
| `GET` | `/api/v1/funnels/{id}` | Get funnel details |
| `PUT` | `/api/v1/funnels/{id}` | Update a funnel |
| `DELETE` | `/api/v1/funnels/{id}` | Delete a funnel |
| `GET` | `/api/v1/funnels/{id}/steps` | Get funnel steps |
| `POST` | `/api/v1/funnels/{id}/start` | Start a funnel session |
| `GET` | `/api/v1/funnels/{id}/step/{stepId}` | Get current step |
| `POST` | `/api/v1/funnels/{id}/step/{stepId}/response` | Submit step response |
| `POST` | `/api/v1/funnels/{id}/complete` | Mark funnel complete |
| `GET` | `/api/v1/funnels/{id}/leads` | Get funnel leads |
| `GET` | `/api/v1/funnels/{id}/analytics` | Get funnel analytics |
| `GET` | `/api/v1/funnels/{id}/analytics/conversion` | Conversion rates |
| `GET` | `/api/v1/funnels/{id}/analytics/dropoff` | Drop-off analysis |
| `GET` | `/api/v1/leads/export` | Export leads (CSV/JSON) |
| `GET` | `/api/v1/templates` | List available templates |
| `POST` | `/api/v1/templates/{id}/apply` | Apply a template |

---

### **7. Verification**

After creating a funnel:
1. **Walk the funnel** — Use `POST /funnels/{id}/start` and step through each step to verify the flow
2. **Test branching** — If using branching logic, verify all paths lead to the correct next step
3. **Submit a test lead** — Complete the funnel end-to-end and verify the lead appears in `GET /funnels/{id}/leads`
4. **Check analytics** — Verify that step transitions and completion are being tracked

Before sharing a funnel URL:
- Ensure all step content is finalized (no placeholder text)
- Verify the form fields collect what you need
- Confirm the CTA redirect URL works

---

### **8. Planned Capabilities (Not Yet Available)**

- **AI copy generation** — Generate step content and CTA text using Ollama (P2)
- **A/B testing** — Test funnel variants against each other (P2)
- **Webhook integrations** — Trigger external actions on lead capture (P2)
- **Advanced segmentation** — Complex audience filtering on lead data (P2)
- **Multi-tenant authentication** — Role-based access via scenario-authenticator (P2)
- **Predictive analytics** — AI-powered conversion predictions (P2)

If you need any of these capabilities, log a decision or TODO noting what you needed, which feature would have helped, and why it matters for your current task.

---

### **9. Output Expectations**

You may:
- Create funnels via CLI or API, from templates or from scratch
- Analyze conversion and drop-off data to optimize funnel performance
- Export leads for use in other marketing tools
- Recommend funnel designs based on campaign objectives

You must:
- Verify the scenario is running before making API calls
- Test funnels end-to-end before deploying them publicly
- Include meaningful step titles and content — generic placeholders hurt conversion
- Respect lead data privacy — do not expose personal information in logs or decisions

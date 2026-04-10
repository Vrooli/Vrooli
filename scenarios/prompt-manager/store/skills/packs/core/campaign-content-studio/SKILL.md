## Tools focus: Campaign Content Studio

Create marketing content organized by campaign — generate blog posts, social media copy, email drafts, and ad copy using AI with document-grounded context.

---

### **1. When to Use**

| Use when | Don't use when |
|----------|----------------|
| Generating blog/social/email/ad copy for a campaign | Writing X/Twitter dev log threads (use x-dev-log) |
| Organizing content production around a campaign theme | Scheduling posts to platforms (use social-media-scheduler) |
| Grounding AI generation in uploaded reference documents | Doing SEO keyword research (use seo-optimizer) |
| Producing multiple content types from the same campaign brief | Creating one-off text that doesn't belong to a campaign |

```
            What content workflow do you need?
                        │
        ┌───────────────┼───────────────┐
        │               │               │
        ▼               ▼               ▼
  New campaign?   Generate content   Search campaign
        │         for existing       documents for
        ▼         campaign?          context?
  Create campaign       │               │
  + upload docs         ▼               ▼
        │         POST /generate    POST /search
        ▼         with campaign ID  with query
  POST /campaigns
```

---

### **2. Prerequisites**

Check scenario status before use:
```bash
vrooli scenario status campaign-content-studio
```

Required resources: PostgreSQL, Ollama, n8n, Qdrant, MinIO

---

### **3. Core Workflow**

#### Create a Campaign

```bash
curl -X POST http://localhost:${API_PORT}/campaigns \
  -H "Content-Type: application/json" \
  -d '{"name": "Q1 Launch", "description": "Feature launch campaign for funnel builder"}'
```

#### Upload Reference Documents

Upload PDFs, DOCX, TXT, or XLSX files that provide context for content generation. Documents are processed via unstructured-io and embedded in Qdrant for semantic retrieval.

```bash
curl -X POST http://localhost:${API_PORT}/campaigns/{campaignId}/documents \
  -F "file=@product-brief.pdf"
```

#### Generate Content

Generate content grounded in campaign documents. Specify the content type to get format-appropriate output.

```bash
curl -X POST http://localhost:${API_PORT}/generate \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "campaign-uuid",
    "content_type": "blog",
    "prompt": "Write a blog post about the new drag-and-drop funnel builder"
  }'
```

**Supported content types:**
| Type | Output |
|------|--------|
| `blog` | Long-form blog post (500-2000 words) |
| `social` | Platform-ready social media copy |
| `email` | Email marketing draft with subject line |
| `ads` | Ad copy variants for paid channels |

#### Search Campaign Documents

Find relevant context across uploaded documents using semantic search:

```bash
curl -X POST http://localhost:${API_PORT}/campaigns/{campaignId}/search \
  -H "Content-Type: application/json" \
  -d '{"query": "pricing strategy for enterprise customers"}'
```

---

### **4. API Reference**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/health` | Health check |
| `GET` | `/campaigns` | List all campaigns |
| `POST` | `/campaigns` | Create new campaign |
| `GET` | `/campaigns/{id}/documents` | List campaign documents |
| `POST` | `/campaigns/{id}/search` | Semantic search across documents |
| `POST` | `/generate` | Generate content using AI |

---

### **5. Verification**

After generating content:
1. **Review for accuracy** — AI-generated content must be factually verified against source documents
2. **Check tone** — Ensure output matches Vrooli's builder voice (authentic, technically credible, not corporate)
3. **Platform fit** — Verify length and format match the target platform's constraints
4. **Source attribution** — Generated content should reference the campaign documents it drew from

---

### **6. Planned Capabilities (Not Yet Available)**

These features are designed but not yet implemented:

- **Content calendar integration** — Schedule content production timelines within campaigns
- **Template gallery** — Pre-built content templates for common campaign types
- **Content regeneration** — Iterate on generated content with feedback
- **Multi-format export** — Export content in platform-ready formats

If you need any of these capabilities, log a decision or TODO noting what you needed, which feature would have helped, and why it matters for your current task.

---

### **7. Output Expectations**

You may:
- Create campaigns and generate content via the API
- Upload reference documents to ground generation in real context
- Generate multiple content types from a single campaign

You must:
- Verify the scenario is running before making API calls
- Review all generated content for accuracy before submitting for publication
- Maintain builder voice — reject and regenerate content that sounds corporate or generic
- Cite which campaign documents informed the generated content

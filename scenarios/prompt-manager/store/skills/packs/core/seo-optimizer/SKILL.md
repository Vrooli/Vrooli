## Tools focus: SEO Optimizer

Analyze and improve search engine visibility with site audits, keyword research, content optimization suggestions, and competitor analysis — all powered by AI.

---

### **1. When to Use**

| Use when | Don't use when |
|----------|----------------|
| Auditing a landing page or scenario UI for SEO issues | Writing content (use campaign-content-studio) |
| Researching keywords for content strategy | Scheduling social posts (use social-media-scheduler) |
| Optimizing existing content for search rankings | Analyzing non-web content (PDFs, docs) |
| Comparing SEO performance against a competitor | Need real-time search ranking monitoring |
| Preparing a content brief with target keywords | Building landing pages (use funnel-builder) |

```
        What SEO task do you need?
                    │
    ┌───────────────┼───────────────┐
    │               │               │
    ▼               ▼               ▼
  Audit a site? Optimize content? Research keywords
    │               │             or competitors?
    ▼               ▼               │
  POST /api/     POST /api/     ┌───┴───┐
  seo-audit      content-       ▼       ▼
                 optimize    keyword  competitor
                             research  analysis
```

---

### **2. Prerequisites**

Check scenario status before use:
```bash
vrooli scenario status seo-optimizer
```

Required resources: PostgreSQL, Ollama, browser-automation-studio
Optional: Redis (caching), Qdrant (semantic analysis)

---

### **3. Core Workflow**

#### Run an SEO Audit

Analyze a URL for technical SEO issues, content quality, and optimization opportunities:

```bash
curl -X POST http://localhost:${API_PORT}/api/seo-audit \
  -H "Content-Type: application/json" \
  -d '{"url": "https://vrooli.com", "depth": 3}'
```

The audit covers:
- Title tag and meta description optimization
- Content quality and keyword density
- Technical SEO factors (load speed, mobile-friendliness, structured data)
- Internal linking structure
- Backlink recommendations

#### Optimize Content

Submit existing content with target keywords for AI-powered optimization suggestions:

```bash
curl -X POST http://localhost:${API_PORT}/api/content-optimize \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Your existing blog post or page content here...",
    "target_keywords": "funnel builder, conversion optimization",
    "content_type": "blog"
  }'
```

#### Keyword Research

Discover keyword opportunities from a seed keyword:

```bash
curl -X POST http://localhost:${API_PORT}/api/keyword-research \
  -H "Content-Type: application/json" \
  -d '{
    "seed_keyword": "marketing automation",
    "target_location": "US",
    "language": "en"
  }'
```

#### Competitor Analysis

Compare your site's SEO performance against a competitor:

```bash
curl -X POST http://localhost:${API_PORT}/api/competitor-analysis \
  -H "Content-Type: application/json" \
  -d '{
    "your_url": "https://vrooli.com",
    "competitor_url": "https://competitor.com",
    "analysis_type": "full"
  }'
```

---

### **4. API Reference**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/health` | Health check |
| `POST` | `/api/seo-audit` | Run comprehensive SEO audit |
| `POST` | `/api/content-optimize` | Get content optimization suggestions |
| `POST` | `/api/keyword-research` | Research keyword opportunities |
| `POST` | `/api/competitor-analysis` | Compare against competitor |

---

### **5. Verification**

After running an audit or analysis:
1. **Check actionable items** — Focus on high-impact issues first (missing meta tags, broken links, slow load times)
2. **Validate recommendations** — AI suggestions should be cross-checked against current SEO best practices
3. **Prioritize fixes** — Technical SEO issues (broken, slow) before content optimization (keyword density, readability)
4. **Track changes** — Note which recommendations were implemented so follow-up audits can measure improvement

---

### **6. Planned Capabilities (Not Yet Available)**

These features are designed but not yet implemented:

- **Rank tracking** — Monitor keyword positions over time (P2)
- **Bulk URL auditing** — Audit multiple pages in one request (P2)
- **PDF report generation** — Export audit results as formatted reports (P2)
- **Redis caching** — Cache audit results to avoid re-crawling (P1)
- **Automated re-audit scheduling** — Periodic audits with change detection

If you need any of these capabilities, log a decision or TODO noting what you needed, which feature would have helped, and why it matters for your current task.

---

### **7. Output Expectations**

You may:
- Run audits on any public URL or locally-accessible scenario UI
- Use keyword research to inform content strategy
- Optimize content before publishing to improve search visibility
- Compare against competitors to identify gaps

You must:
- Verify the scenario is running before making API calls
- Treat audit results as recommendations, not mandates — context matters
- Focus content optimization on user value first, search rankings second
- Not expose competitor analysis results externally without review

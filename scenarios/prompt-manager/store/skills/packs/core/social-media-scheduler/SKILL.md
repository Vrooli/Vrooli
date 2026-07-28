## Tools focus: Social Media Scheduler

Schedule and manage content across multiple social media platforms with AI-powered optimization, calendar management, and performance analytics.

---

### **1. When to Use**

| Use when | Don't use when |
|----------|----------------|
| Scheduling a post to one or more platforms | Generating content from scratch (use the `x-<post-type>` skills) |
| Viewing the content calendar for upcoming posts | Writing X/Twitter dev log threads (use x-dev-log) |
| Optimizing content for a specific platform | Running SEO analysis (use seo-optimizer) |
| Checking post performance analytics | Managing email campaigns (no capability exists) |
| Bulk scheduling multiple posts from a content plan | Need real-time engagement/replies |

```
          What do you need to do?
                    │
    ┌───────────────┼───────────────┐
    │               │               │
    ▼               ▼               ▼
  Schedule      View/manage     Analyze
  new content?  existing posts? performance?
    │               │               │
    ▼               ▼               ▼
  POST /posts/    GET /posts/    GET /analytics/
  schedule        calendar       overview
```

---

### **2. Prerequisites**

Check scenario status before use:
```bash
vrooli scenario status social-media-scheduler
```

Required resources: PostgreSQL, Redis, MinIO, Ollama
Optional: browser-automation-studio (for social verification screenshots)

Platform accounts must be connected via OAuth before scheduling:
```bash
social-media-scheduler platforms    # See available platforms
social-media-scheduler accounts     # See connected accounts
```

---

### **3. Supported Platforms**

| Platform | Character Limit | Media | Hashtags | Notes |
|----------|----------------|-------|----------|-------|
| Twitter | 280 | Up to 4 images | 1-3 recommended | Short, punchy, hook-first |
| Instagram | 2,200 | Required (image/video) | 5-10 recommended | Visual-first, story-driven |
| LinkedIn | 3,000 | Optional | 1-3 professional | Professional tone, industry insights |
| Facebook | High | Optional | Minimal | Community-focused, conversational |
| TikTok | — | Required (video) | Trending tags | *P2 — not yet available* |

---

### **4. Core Workflow**

#### Schedule a Post (CLI)

```bash
social-media-scheduler schedule \
  "Feature Launch" \
  "Just shipped the drag-and-drop funnel builder. Build conversion funnels in minutes, not days." \
  "twitter,linkedin" \
  "2026-03-20T10:00:00Z"
```

#### Schedule a Post (API)

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/posts/schedule \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "title": "Feature Launch",
    "content": "Just shipped the drag-and-drop funnel builder...",
    "platforms": ["twitter", "linkedin"],
    "scheduled_at": "2026-03-20T10:00:00Z",
    "timezone": "America/New_York"
  }'
```

#### Optimize Content for a Platform

Let AI adapt content to platform-specific best practices:

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/posts/{id}/optimize \
  -H "Authorization: Bearer ${TOKEN}"
```

#### View Calendar

```bash
curl -X GET "http://localhost:${API_PORT}/api/v1/posts/calendar?start=2026-03-01&end=2026-03-31" \
  -H "Authorization: Bearer ${TOKEN}"
```

#### Bulk Schedule

```bash
curl -X POST http://localhost:${API_PORT}/api/v1/bulk/schedule \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"posts": [...]}'
```

---

### **5. CLI Reference**

```bash
social-media-scheduler login "<email>" "<password>"     # Authenticate
social-media-scheduler whoami                        # Current user info
social-media-scheduler platforms                     # List available platforms
social-media-scheduler accounts                      # Show connected accounts
social-media-scheduler schedule "<title>" "<content>" "<platforms>" "<datetime>"
social-media-scheduler list                          # List scheduled posts
social-media-scheduler status "<post_id>"              # Get post status
```

---

### **6. API Reference**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/api/v1/auth/login` | User authentication |
| `GET` | `/api/v1/user/accounts` | Connected social accounts |
| `POST` | `/api/v1/posts/schedule` | Schedule a new post |
| `GET` | `/api/v1/posts/calendar` | Calendar view of posts |
| `GET` | `/api/v1/posts/{id}` | Get post details |
| `PUT` | `/api/v1/posts/{id}` | Update a post |
| `DELETE` | `/api/v1/posts/{id}` | Delete a post |
| `POST` | `/api/v1/posts/{id}/optimize` | AI-optimize content for platform |
| `POST` | `/api/v1/posts/{id}/duplicate` | Duplicate a post |
| `GET` | `/api/v1/posts/{id}/preview` | Platform-specific preview |
| `POST` | `/api/v1/bulk/schedule` | Bulk schedule posts |
| `POST` | `/api/v1/media/upload` | Upload media files |
| `GET` | `/api/v1/analytics/overview` | Analytics overview |
| `GET` | `/api/v1/analytics/optimal-times` | Best posting times |
| `GET` | `/health` | Health check |

---

### **7. Verification**

Before scheduling:
1. **Platform connected** — Verify the target platform account is linked via `social-media-scheduler accounts`
2. **Character limits** — Ensure content fits the platform's constraints
3. **Media requirements** — Instagram requires media; verify images/videos are uploaded
4. **Scheduling time** — Confirm the datetime is in the future and timezone is correct
5. **Content quality** — Review AI-optimized content to ensure it hasn't lost the intended message

After posting:
- Check delivery status via `social-media-scheduler status <post_id>`
- Review analytics for engagement after 24-48 hours

---

### **8. Planned Capabilities (Not Yet Available)**

- **Bulk CSV import** — Import post schedules from spreadsheets (P1)
- **Optimal time recommendations** — AI-suggested posting times based on audience data (P1)
- **Content approval workflows** — Multi-step approval before auto-posting (P1)
- **TikTok integration** — Full TikTok posting support (P2)

If you need any of these capabilities, log a decision or TODO noting what you needed, which feature would have helped, and why it matters for your current task.

---

### **9. Output Expectations**

You may:
- Schedule posts to connected platforms
- Use AI optimization to adapt content per platform
- View calendar and analytics data
- Bulk schedule from content plans

You must:
- Verify the scenario is running and accounts are connected before scheduling
- Never auto-post without human approval — all content is draft until reviewed
- Respect platform character limits and media requirements
- Include timezone when scheduling to avoid posting at wrong times

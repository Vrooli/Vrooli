# Cloudflare Worker Excessive Requests Analysis

## Overview
The Cloudflare Worker `og-worker` is receiving 250k+ requests per day. This analysis investigates the potential causes.

## What the Worker Does
The `og-worker` is an Open Graph (OG) metadata worker that:
1. Intercepts requests to vrooli.com and *.vrooli.com domains
2. Checks if the request is from a crawler/bot (based on User-Agent)
3. For bot requests: Serves custom HTML with Open Graph/Twitter Card metadata
4. For human requests: Proxies to the origin server (do-origin.vrooli.com)

## Key Findings

### 1. Bot Detection Pattern
The worker detects these bots:
- Facebook (facebookexternalhit, facebot, facebookcatalog, meta-external)
- Twitter (twitterbot)
- Pinterest (pinterestbot)
- LinkedIn (linkedinbot)
- Slack (slackbot-linkexpanding)
- Discord (discordbot)
- Telegram (telegrambot)
- WhatsApp
- Skype (skypeuripreview)
- Various link preview services (embedly, opengraph, iframely, vkshare, bitlybot)

### 2. Route Coverage
The worker handles requests for:
- Static pages (/, /about, /calendar, /create, /login, /signup, etc.)
- Dynamic content (users, teams, APIs, notes, projects, chats, etc.)
- Every page request triggers the worker due to the catch-all pattern

### 3. Potential Sources of High Request Volume

#### A. Social Media Link Previews
- Every time a Vrooli link is shared on social media, multiple requests are made
- Facebook alone can make 3-5 requests per shared link
- Discord, Slack, and other platforms also fetch previews aggressively

#### B. SEO Crawlers Not in Bot List
- The bot detection doesn't include major search engines (Google, Bing, Yandex, Baidu)
- These crawlers would be proxied to origin, still counting as worker requests
- Search engines can generate thousands of requests daily

#### C. Health Checks and Monitoring
- Kubernetes health checks are configured:
  - UI: liveness/readiness checks on `/` path
  - Server: checks on `/healthcheck` path
  - Jobs: checks on `/healthcheck` path
- With multiple replicas and frequent checks (every 10-30 seconds), this adds up

#### D. PWA and Service Worker Activity
- The UI has a Progressive Web App setup with service workers
- Service worker checks for updates periodically
- Each user session could generate multiple background requests

#### E. Preconnect and Resource Hints
- The index.html includes preconnect hints to vrooli.com
- This can trigger DNS prefetching and connection warmup

#### F. No Request Deduplication
- The worker doesn't implement any caching for bot responses
- Each identical bot request generates a new API call to fetch metadata

### 4. Request Amplification Factors

1. **Multiple Domains**: Worker handles vrooli.com, www.vrooli.com, app.vrooli.com
2. **No Cache Headers**: Bot responses only cache for 1 hour (s-maxage=3600)
3. **API Calls**: Each dynamic route request triggers an API call to fetch metadata
4. **No Rate Limiting**: No protection against aggressive crawlers

## Recommendations

### 1. Implement Caching
- Add Cloudflare Cache API to cache bot responses
- Increase cache duration for static metadata
- Cache API responses for dynamic content

### 2. Add Major Search Engines to Bot List
- Include Googlebot, Bingbot, etc. to serve them static responses
- This prevents unnecessary origin requests

### 3. Rate Limiting
- Implement rate limiting for aggressive crawlers
- Use Cloudflare's built-in rate limiting features

### 4. Optimize Health Checks
- Consider using a dedicated health check endpoint that bypasses the worker
- Reduce health check frequency where possible

### 5. Add Crawl-Delay to robots.txt
- Add `Crawl-delay: 1` to slow down aggressive crawlers
- Consider blocking unnecessary bot user agents

### 6. Monitor and Analyze
- Use Cloudflare Analytics to identify top user agents
- Look for patterns in request URLs
- Consider implementing custom logging for better insights

### 7. Optimize Worker Code
- Implement early returns for known static assets
- Skip processing for certain paths (e.g., /api/*, /static/*)
- Use Cloudflare KV for storing pre-generated metadata

## Estimated Impact
With 250k requests/day:
- ~40% could be social media preview bots (100k)
- ~30% could be search engine crawlers (75k)
- ~20% could be health checks (50k)
- ~10% could be legitimate user traffic (25k)

Implementing caching alone could reduce origin requests by 70-80%, significantly reducing costs and improving performance.
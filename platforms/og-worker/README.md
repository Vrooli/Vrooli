# Open Graph Metadata Template for Cloudflare Workers

> **Template Purpose:** Enable ANY app to serve dynamic Open Graph/Twitter Card metadata for social media previews.
> This template provides a starting point for agents to add OG metadata capabilities to generated apps.

---

## 🤖 For AI Agents Using This Template

This template demonstrates how to create a Cloudflare Worker that:
1. Intercepts requests from social media crawlers (Facebook, Twitter, etc.)
2. Fetches dynamic data from your app's API
3. Generates appropriate meta tags for rich previews
4. Proxies regular users directly to your app

**Key Customization Points:**
- Route patterns in `src/index.ts` (match YOUR app's URL structure)
- API endpoints for fetching metadata
- Meta tag generation logic
- Domain and origin configuration

---

## 📋 Template Structure

```
og-worker/
├── src/
│   ├── index.ts          # Main worker logic (CUSTOMIZE THIS)
│   └── examples/         # Example implementations
│       ├── blog.ts       # Blog post OG tags
│       ├── ecommerce.ts  # Product OG tags
│       └── portfolio.ts  # Portfolio/profile OG tags
├── template.config.json  # Configuration hints for agents
├── wrangler.jsonc       # Cloudflare Worker config (UPDATE DOMAINS)
└── README.md            # This file
```

---

## 🚀 How to Adapt This Template

### Step 1: Identify Your App's Routes

Look at your app's URL structure and identify which routes need OG tags:
```
/blog/:slug        → Blog posts
/products/:id      → Product pages  
/profiles/:username → User profiles
/events/:eventId   → Event pages
```

### Step 2: Determine Data Sources

Find where your app stores the data needed for OG tags:
- REST API endpoints?
- GraphQL queries?
- Database direct access?
- Static JSON files?

### Step 3: Customize the Worker

Update `src/index.ts`:
1. Replace example routes with your app's routes
2. Update API calls to fetch from your data sources
3. Customize meta tag generation logic
4. Update domain references

### Step 4: Configure Cloudflare

Update `wrangler.jsonc`:
1. Set your app's domain
2. Configure origin host
3. Add any environment variables needed

---

## 🔧 Technical Requirements

| Requirement | Notes |
|------------|-------|
| Cloudflare account | Free tier works |
| Domain on Cloudflare | Must be proxied (orange cloud) |
| Origin server | Where your app is hosted |
| Node.js ≥ 18 | For local development |

---

## 📚 Implementation Examples

### Example 1: Blog App
```typescript
// Routes for a blog application
const routes = [
  { pattern: "/blog/:slug", api: getBlogPostMeta },
  { pattern: "/author/:id", api: getAuthorMeta },
  { pattern: "/category/:name", api: getCategoryMeta }
];

async function getBlogPostMeta({ slug }, env) {
  const post = await fetch(`${env.API_URL}/api/posts/${slug}`).then(r => r.json());
  return {
    title: post.title,
    description: post.excerpt,
    image: post.featuredImage || `${env.SITE_URL}/default-blog.jpg`,
    url: `${env.SITE_URL}/blog/${slug}`
  };
}
```

### Example 2: E-commerce App
```typescript
// Routes for an online store
const routes = [
  { pattern: "/product/:id", api: getProductMeta },
  { pattern: "/category/:slug", api: getCategoryMeta },
  { pattern: "/brand/:name", api: getBrandMeta }
];

async function getProductMeta({ id }, env) {
  const product = await fetch(`${env.API_URL}/api/products/${id}`).then(r => r.json());
  return {
    title: `${product.name} - ${product.price}`,
    description: product.description,
    image: product.images[0] || `${env.SITE_URL}/default-product.jpg`,
    url: `${env.SITE_URL}/product/${id}`,
    // Additional e-commerce specific tags
    extra: {
      "og:type": "product",
      "product:price:amount": product.price,
      "product:price:currency": "USD"
    }
  };
}
```

### Example 3: Social Platform
```typescript
// Routes for a social/community platform
const routes = [
  { pattern: "/u/:username", api: getUserMeta },
  { pattern: "/post/:id", api: getPostMeta },
  { pattern: "/group/:slug", api: getGroupMeta }
];

async function getUserMeta({ username }, env) {
  const user = await fetch(`${env.API_URL}/api/users/${username}`).then(r => r.json());
  return {
    title: `${user.displayName} (@${username})`,
    description: user.bio || `Check out ${user.displayName}'s profile`,
    image: user.avatar || `${env.SITE_URL}/default-avatar.png`,
    url: `${env.SITE_URL}/u/${username}`,
    extra: {
      "og:type": "profile",
      "profile:username": username
    }
  };
}
```

---

## 🔨 Development Workflow

```bash
# 1. Install dependencies
npm install wrangler --save-dev

# 2. Test locally
npx wrangler dev

# 3. Test with curl (simulate bot)
curl -H "User-Agent: Twitterbot" http://localhost:8787/your-route

# 4. Deploy to Cloudflare
npx wrangler deploy

# 5. Monitor logs
npx wrangler tail
```

---

## 🧪 Testing Your Implementation

### Local Testing
```bash
# Simulate Facebook crawler
curl -H "User-Agent: facebookexternalhit/1.1" http://localhost:8787/blog/my-post

# Simulate Twitter crawler  
curl -H "User-Agent: Twitterbot/1.0" http://localhost:8787/product/123
```

### Production Testing
- [Facebook Sharing Debugger](https://developers.facebook.com/tools/debug/)
- [Twitter Card Validator](https://cards-dev.twitter.com/validator)
- [LinkedIn Post Inspector](https://www.linkedin.com/post-inspector/)
- [Telegram Webpage Bot](https://t.me/WebpageBot)

---

## 🎯 Common Patterns to Implement

### Pattern 1: Fallback Chains
```typescript
// Try multiple data sources with fallbacks
async function getMeta(params, env) {
  try {
    // Try primary API
    return await fetchFromAPI(params, env);
  } catch {
    try {
      // Fallback to cache
      return await fetchFromCache(params, env);
    } catch {
      // Ultimate fallback
      return getDefaultMeta(params, env);
    }
  }
}
```

### Pattern 2: Dynamic Image Generation
```typescript
// Generate OG images on the fly
function getImageUrl(data, env) {
  // Option 1: Use a service like Cloudinary
  return `https://res.cloudinary.com/${env.CLOUD_NAME}/image/upload/` +
         `w_1200,h_630,c_fit,f_auto/` +
         `l_text:${encodeURIComponent(data.title)}/` +
         `${data.baseImage}`;
  
  // Option 2: Use another Worker for image generation
  return `${env.IMAGE_WORKER_URL}?title=${encodeURIComponent(data.title)}`;
}
```

### Pattern 3: Caching Strategy
```typescript
// Cache rendered HTML in KV store
async function getMetaWithCache(params, env) {
  const cacheKey = `og:${params.type}:${params.id}`;
  
  // Check cache
  const cached = await env.OG_CACHE.get(cacheKey);
  if (cached) return cached;
  
  // Generate fresh
  const fresh = await generateMeta(params, env);
  
  // Store with TTL
  await env.OG_CACHE.put(cacheKey, fresh, {
    expirationTtl: 3600 // 1 hour
  });
  
  return fresh;
}
```

---

## 📝 Configuration Reference

### Environment Variables (wrangler.jsonc)
```json
{
  "vars": {
    "BOT_UA": "Bot user agent regex pattern",
    "ORIGIN_HOST": "Your app's actual host",
    "API_URL": "Your API endpoint base URL",
    "SITE_URL": "Your public site URL",
    "DEFAULT_IMAGE": "Fallback OG image URL",
    "CACHE_TTL": "3600"
  }
}
```

### Route Configuration
```typescript
interface RouteConfig {
  pattern: string;        // URL pattern with params
  api: Function;          // Function to fetch metadata
  cache?: boolean;        // Enable caching for this route
  ttl?: number;          // Cache TTL in seconds
}
```

---

## 🚨 Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| Worker not triggering | Ensure domain is proxied (orange cloud) in Cloudflare |
| Infinite redirect loop | Use different hostname for origin (e.g., origin.domain.com) |
| API calls failing | Check CORS and add origin override in fetch |
| Meta tags not showing | Verify HTML structure and escape special characters |
| Cache not updating | Implement cache busting or reduce TTL |

---

## 🔗 Resources

- [Cloudflare Workers Docs](https://developers.cloudflare.com/workers/)
- [Open Graph Protocol](https://ogp.me/)
- [Twitter Cards Guide](https://developer.twitter.com/en/docs/twitter-for-websites/cards/overview/abouts-cards)
- [Schema.org Structured Data](https://schema.org/)

---

## 💡 Tips for AI Agents

1. **Start Simple**: Get basic OG tags working first, then add complexity
2. **Test Incrementally**: Test each route as you add it
3. **Use TypeScript**: Better type safety helps prevent runtime errors
4. **Log Extensively**: Add logging to debug issues in production
5. **Consider Performance**: Cache aggressively but invalidate smartly
6. **Handle Errors Gracefully**: Always have fallback values ready

Remember: This template is a starting point. Adapt it to fit the specific needs of the app you're enhancing!
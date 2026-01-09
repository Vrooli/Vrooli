/**
 * Production server for Landing Manager UI
 *
 * Uses @vrooli/api-base to automatically handle:
 * - /health endpoint with API connectivity checks
 * - /config endpoint with runtime configuration
 * - /api/* proxy to API server
 * - Static file serving from ./dist
 * - SPA fallback routing
 *
 * Additionally handles:
 * - Dynamic SEO meta tag injection based on site branding
 * - Favicon and OG image injection
 */

import { createScenarioServer, injectBaseTag } from '@vrooli/api-base/server';

// -----------------------------------------------------------------------------
// Branding Cache - fetches from API with TTL
// -----------------------------------------------------------------------------

let brandingCache = null;
let brandingCacheTime = 0;
let fetchInProgress = false;
const BRANDING_CACHE_TTL_MS = 60000; // 1 minute cache

/**
 * Fetches branding from API and updates cache.
 * Non-blocking - returns immediately with cached data if available.
 */
function refreshBrandingCache() {
  if (fetchInProgress) return;

  const apiPort = process.env.API_PORT || 3001;
  const apiUrl = `http://localhost:${apiPort}/api/v1/branding`;

  fetchInProgress = true;

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 2000);

  fetch(apiUrl, {
    signal: controller.signal,
    headers: { 'Accept': 'application/json' }
  })
    .then(response => {
      clearTimeout(timeoutId);
      if (response.ok) {
        return response.json();
      }
      throw new Error(`API returned ${response.status}`);
    })
    .then(data => {
      brandingCache = data;
      brandingCacheTime = Date.now();
    })
    .catch(err => {
      // Silently fail - we'll use cached data or defaults
      if (err.name !== 'AbortError') {
        console.warn('[SSR] Branding fetch failed:', err.message);
      }
    })
    .finally(() => {
      fetchInProgress = false;
    });
}

/**
 * Gets cached branding, triggering a background refresh if stale.
 * Always returns synchronously with whatever is in cache.
 */
function getBranding() {
  const now = Date.now();
  const isStale = !brandingCache || (now - brandingCacheTime) > BRANDING_CACHE_TTL_MS;

  if (isStale) {
    refreshBrandingCache();
  }

  return brandingCache;
}

// Start fetching branding on server startup
refreshBrandingCache();

// -----------------------------------------------------------------------------
// SEO Meta Tag Generation
// -----------------------------------------------------------------------------

function escapeHtml(str) {
  if (!str) return '';
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function generateMetaTags(branding) {
  if (!branding) return '';

  const tags = [];

  // Basic SEO
  if (branding.default_title || branding.site_name) {
    tags.push(`<title>${escapeHtml(branding.default_title || branding.site_name)}</title>`);
  }
  if (branding.default_description) {
    tags.push(`<meta name="description" content="${escapeHtml(branding.default_description)}" />`);
  }

  // Open Graph
  if (branding.site_name) {
    tags.push(`<meta property="og:site_name" content="${escapeHtml(branding.site_name)}" />`);
  }
  if (branding.default_title || branding.site_name) {
    tags.push(`<meta property="og:title" content="${escapeHtml(branding.default_title || branding.site_name)}" />`);
  }
  if (branding.default_description) {
    tags.push(`<meta property="og:description" content="${escapeHtml(branding.default_description)}" />`);
  }
  if (branding.default_og_image_url) {
    tags.push(`<meta property="og:image" content="${escapeHtml(branding.default_og_image_url)}" />`);
  }
  tags.push(`<meta property="og:type" content="website" />`);

  // Twitter Card
  tags.push(`<meta name="twitter:card" content="summary_large_image" />`);
  if (branding.default_title || branding.site_name) {
    tags.push(`<meta name="twitter:title" content="${escapeHtml(branding.default_title || branding.site_name)}" />`);
  }
  if (branding.default_description) {
    tags.push(`<meta name="twitter:description" content="${escapeHtml(branding.default_description)}" />`);
  }
  if (branding.default_og_image_url) {
    tags.push(`<meta name="twitter:image" content="${escapeHtml(branding.default_og_image_url)}" />`);
  }

  // Favicon
  if (branding.favicon_url) {
    tags.push(`<link rel="icon" type="image/x-icon" href="${escapeHtml(branding.favicon_url)}" />`);
    tags.push(`<link rel="shortcut icon" href="${escapeHtml(branding.favicon_url)}" />`);
  }
  if (branding.apple_touch_icon_url) {
    tags.push(`<link rel="apple-touch-icon" href="${escapeHtml(branding.apple_touch_icon_url)}" />`);
  }

  // Canonical URL
  if (branding.canonical_base_url) {
    tags.push(`<link rel="canonical" href="${escapeHtml(branding.canonical_base_url)}" />`);
  }

  // Google Site Verification
  if (branding.google_site_verification) {
    tags.push(`<meta name="google-site-verification" content="${escapeHtml(branding.google_site_verification)}" />`);
  }

  // Theme color for mobile browsers
  if (branding.theme_primary_color) {
    tags.push(`<meta name="theme-color" content="${escapeHtml(branding.theme_primary_color)}" />`);
  }

  return tags.join('\n    ');
}

function injectSeoTags(html, branding) {
  if (!branding) return html;

  const metaTags = generateMetaTags(branding);
  if (!metaTags) return html;

  // Remove existing static title and description
  let modified = html
    .replace(/<title>[^<]*<\/title>/i, '')
    .replace(/<meta\s+name=["']description["'][^>]*>/i, '');

  // Inject new meta tags after charset meta or at start of head
  const charsetMatch = modified.match(/<meta\s+charset=["'][^"']*["']\s*\/?>/i);
  if (charsetMatch) {
    const insertPoint = charsetMatch.index + charsetMatch[0].length;
    modified = modified.slice(0, insertPoint) + '\n    ' + metaTags + modified.slice(insertPoint);
  } else {
    // Fallback: insert after <head>
    modified = modified.replace(/<head>/i, '<head>\n    ' + metaTags);
  }

  return modified;
}

// -----------------------------------------------------------------------------
// Server Setup
// -----------------------------------------------------------------------------

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'landing-page-business-suite',
  version: '1.0.0',
  corsOrigins: '*',
  setupRoutes: (expressApp) => {
    // Middleware to inject base tag and SEO meta tags into HTML responses
    expressApp.use((req, res, next) => {
      const originalSend = res.send;

      res.send = function sendWithInjection(body) {
        const contentType = res.getHeader('content-type');
        const isHtml = contentType && typeof contentType === 'string' && contentType.includes('text/html');

        if (isHtml && typeof body === 'string') {
          // Inject base tag for SPA routing
          let modified = injectBaseTag(body, '/', { skipIfExists: true });

          // Get cached branding and inject SEO meta tags (synchronous)
          const branding = getBranding();
          modified = injectSeoTags(modified, branding);

          return originalSend.call(this, modified);
        }
        return originalSend.call(this, body);
      };

      next();
    });
  }
});

app.listen(process.env.UI_PORT);

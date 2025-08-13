/**
 * ============================================================================
 * CLOUDFLARE WORKER TEMPLATE FOR OPEN GRAPH METADATA
 * ============================================================================
 * 
 * This template provides a starting point for adding dynamic Open Graph (OG) 
 * metadata to ANY application. AI agents can customize this template to match
 * the specific needs of the app they're enhancing.
 * 
 * KEY CUSTOMIZATION POINTS:
 * 1. BASE_META - Update with your app's default metadata
 * 2. STATIC_ROUTES - Define static pages and their metadata
 * 3. DYNAMIC_ROUTES - Define dynamic routes that fetch data from APIs
 * 4. API Response Interfaces - Define the shape of your API responses
 * 5. Meta Fetcher Functions - Implement logic to fetch and transform data
 * 
 * ============================================================================
 */

// OPTIONAL: Install route parsing library if using dynamic routes
// npm install regexparam

/* ============================================================================
 * SECTION 1: CONFIGURATION & TYPES
 * Customize these to match your application
 * ============================================================================ */

interface Env {
    // CUSTOMIZE: Add your environment variables
    BOT_UA: string;          // Regex pattern for crawler user agents
    ORIGIN_HOST: string;     // Your app's actual host (e.g., origin.yourapp.com)
    API_URL?: string;        // Optional: Your API base URL if different from origin
    SITE_URL?: string;       // Optional: Your public site URL
    // Add more environment variables as needed...
}

interface Meta {
    title: string;
    description: string;
    image: string;
    url: string;
    // Optional extended properties
    siteName?: string;
    type?: string;
    locale?: string;
    imageWidth?: string;
    imageHeight?: string;
    twitterCard?: string;
    // Add custom properties as needed...
    extra?: Record<string, string>;  // For additional meta tags
}

/* ============================================================================
 * CUSTOMIZE: Update this regex to match the crawlers you want to serve
 * ============================================================================ */
const BOT_UA_PATTERN = 
    "(googlebot|bingbot|facebookexternalhit|twitterbot|linkedinbot|" +
    "slackbot|discordbot|telegrambot|whatsapp|pinterest|" +
    "applebot|chrome-lighthouse|pagespeed|gtmetrix)";

/* ============================================================================
 * CUSTOMIZE: Default metadata for your application
 * This is used as fallback and base for all pages
 * ============================================================================ */
const BASE_META: Omit<Meta, "url"> = {
    siteName: "Your App Name",  // CHANGE THIS
    title: "Your App Title",    // CHANGE THIS
    description: "Your app's description that appears in social media previews",  // CHANGE THIS
    image: "/default-og-image.jpg",  // CHANGE THIS - Path to your default OG image
    type: "website",
    locale: "en_US",
    imageWidth: "1200",
    imageHeight: "630",
    twitterCard: "summary_large_image",
};

// Cache configuration
const CACHE_TTL = 3600;           // 1 hour - Cloudflare edge cache
const CACHE_BROWSER_TTL = 300;   // 5 minutes - Browser cache

/* ============================================================================
 * SECTION 2: ROUTE DEFINITIONS
 * Define your app's routes and how to fetch their metadata
 * ============================================================================ */

/* ----------------------------------------------------------------------------
 * STATIC ROUTES
 * Define metadata for static pages (About, Contact, etc.)
 * ---------------------------------------------------------------------------- */
const STATIC_ROUTES: Record<string, Partial<Meta>> = {
    // CUSTOMIZE: Add your static pages here
    "/": {
        // Homepage uses BASE_META
    },
    "/about": {
        title: "About Us",
        description: "Learn more about our company and mission.",
    },
    "/contact": {
        title: "Contact Us",
        description: "Get in touch with our team.",
    },
    "/privacy": {
        title: "Privacy Policy",
        description: "Our commitment to protecting your privacy.",
    },
    // Add more static routes as needed...
};

/* ----------------------------------------------------------------------------
 * DYNAMIC ROUTES
 * Define patterns for dynamic content that needs API data
 * ---------------------------------------------------------------------------- */

// EXAMPLE: Blog post route
interface BlogPostResponse {
    title: string;
    excerpt: string;
    featuredImage?: string;
    author: {
        name: string;
        avatar?: string;
    };
    publishedAt: string;
}

// EXAMPLE: Product route
interface ProductResponse {
    name: string;
    description: string;
    price: number;
    currency: string;
    images: string[];
    inStock: boolean;
}

// EXAMPLE: User profile route
interface UserResponse {
    username: string;
    displayName: string;
    bio?: string;
    avatar?: string;
    joinedDate: string;
}

/* ============================================================================
 * SECTION 3: MAIN WORKER LOGIC
 * This is the core handler - customize the route matching logic
 * ============================================================================ */

export default {
    async fetch(req: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
        try {
            const url = new URL(req.url);
            const pathname = url.pathname;
            const ua = req.headers.get("user-agent") || "";
            
            // Skip health check endpoints
            if (["/health", "/healthz", "/api/health"].includes(pathname)) {
                return new Response("OK", { status: 200 });
            }
            
            // For non-bot traffic, proxy to origin
            const botRegex = new RegExp(env.BOT_UA || BOT_UA_PATTERN, "i");
            if (!botRegex.test(ua)) {
                return proxyToOrigin(req, env);
            }
            
            // For bots, generate OG metadata
            const baseUrl = `${url.protocol}//${url.host}`;
            
            // Try cache first
            const cache = caches.default;
            const cacheKey = new Request(url.toString());
            let response = await cache.match(cacheKey);
            if (response) {
                return response;
            }
            
            // Generate metadata based on route
            let meta: Meta | null = null;
            
            // Check static routes first
            if (pathname === "/" || STATIC_ROUTES[pathname]) {
                meta = {
                    ...BASE_META,
                    ...STATIC_ROUTES[pathname],
                    url: `${baseUrl}${pathname}`,
                };
            }
            
            // CUSTOMIZE: Add your dynamic route matching here
            // Example patterns shown below - replace with your actual routes
            
            // Example: Blog post route (/blog/my-post-slug)
            if (pathname.startsWith("/blog/")) {
                const slug = pathname.replace("/blog/", "");
                meta = await getBlogPostMeta(slug, env, baseUrl);
            }
            
            // Example: Product route (/product/123)
            else if (pathname.match(/^\/product\/\d+$/)) {
                const productId = pathname.replace("/product/", "");
                meta = await getProductMeta(productId, env, baseUrl);
            }
            
            // Example: User profile route (/user/username)
            else if (pathname.startsWith("/user/")) {
                const username = pathname.replace("/user/", "");
                meta = await getUserMeta(username, env, baseUrl);
            }
            
            // If no meta generated, use fallback
            if (!meta) {
                meta = {
                    ...BASE_META,
                    title: `Page Not Found - ${BASE_META.siteName}`,
                    url: `${baseUrl}${pathname}`,
                };
            }
            
            // Generate HTML response
            const html = renderHTML(meta, baseUrl);
            response = createCachedResponse(html);
            
            // Cache the response
            ctx.waitUntil(
                cache.put(cacheKey, response.clone()).catch(err => 
                    console.error("[OG-Worker] Cache error:", err)
                )
            );
            
            return response;
            
        } catch (error) {
            // On any error, proxy to origin (fail-open)
            console.error("[OG-Worker] Error:", error);
            return proxyToOrigin(req, env);
        }
    },
};

/* ============================================================================
 * SECTION 4: META FETCHER FUNCTIONS
 * Implement these to fetch data from your APIs
 * ============================================================================ */

/**
 * EXAMPLE: Fetch blog post metadata
 * CUSTOMIZE: Replace with your actual API logic
 */
async function getBlogPostMeta(slug: string, env: Env, baseUrl: string): Promise<Meta> {
    try {
        // CUSTOMIZE: Update API endpoint
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const res = await fetch(`${apiUrl}/api/posts/${slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!res.ok) throw new Error("Post not found");
        
        const post = await res.json() as BlogPostResponse;
        
        return {
            ...BASE_META,
            title: `${post.title} - ${BASE_META.siteName}`,
            description: post.excerpt,
            image: post.featuredImage || BASE_META.image,
            url: `${baseUrl}/blog/${slug}`,
            type: "article",
            extra: {
                "article:author": post.author.name,
                "article:published_time": post.publishedAt,
            }
        };
    } catch (error) {
        // Return fallback on error
        return {
            ...BASE_META,
            title: `Blog Post - ${BASE_META.siteName}`,
            url: `${baseUrl}/blog/${slug}`,
        };
    }
}

/**
 * EXAMPLE: Fetch product metadata
 * CUSTOMIZE: Replace with your actual API logic
 */
async function getProductMeta(productId: string, env: Env, baseUrl: string): Promise<Meta> {
    try {
        // CUSTOMIZE: Update API endpoint
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const res = await fetch(`${apiUrl}/api/products/${productId}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!res.ok) throw new Error("Product not found");
        
        const product = await res.json() as ProductResponse;
        
        return {
            ...BASE_META,
            title: `${product.name} - $${product.price} ${product.currency}`,
            description: product.description,
            image: product.images[0] || BASE_META.image,
            url: `${baseUrl}/product/${productId}`,
            type: "product",
            extra: {
                "product:price:amount": product.price.toString(),
                "product:price:currency": product.currency,
                "product:availability": product.inStock ? "in stock" : "out of stock",
            }
        };
    } catch (error) {
        return {
            ...BASE_META,
            title: `Product - ${BASE_META.siteName}`,
            url: `${baseUrl}/product/${productId}`,
        };
    }
}

/**
 * EXAMPLE: Fetch user profile metadata
 * CUSTOMIZE: Replace with your actual API logic
 */
async function getUserMeta(username: string, env: Env, baseUrl: string): Promise<Meta> {
    try {
        // CUSTOMIZE: Update API endpoint
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const res = await fetch(`${apiUrl}/api/users/${username}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!res.ok) throw new Error("User not found");
        
        const user = await res.json() as UserResponse;
        
        return {
            ...BASE_META,
            title: `${user.displayName} (@${user.username}) - ${BASE_META.siteName}`,
            description: user.bio || `Check out ${user.displayName}'s profile`,
            image: user.avatar || BASE_META.image,
            url: `${baseUrl}/user/${username}`,
            type: "profile",
            extra: {
                "profile:username": user.username,
            }
        };
    } catch (error) {
        return {
            ...BASE_META,
            title: `User Profile - ${BASE_META.siteName}`,
            url: `${baseUrl}/user/${username}`,
        };
    }
}

/* ============================================================================
 * SECTION 5: HELPER FUNCTIONS
 * These handle the technical details - usually don't need customization
 * ============================================================================ */

/**
 * Proxy requests to the origin server
 */
function proxyToOrigin(req: Request, env: Env): Promise<Response> {
    const url = new URL(req.url);
    url.hostname = env.ORIGIN_HOST;
    
    return fetch(url.toString(), {
        cf: { resolveOverride: env.ORIGIN_HOST },
        method: req.method,
        headers: req.headers,
        body: req.body,
        redirect: "manual",
    });
}

/**
 * Create a cached HTML response
 */
function createCachedResponse(html: string): Response {
    return new Response(html, {
        headers: {
            "Content-Type": "text/html;charset=utf-8",
            "Cache-Control": `public, s-maxage=${CACHE_TTL}, max-age=${CACHE_BROWSER_TTL}, stale-while-revalidate=86400`,
            "X-Robots-Tag": "index, follow",
        },
    });
}

/**
 * Escape HTML special characters
 */
function escapeHtml(str: string): string {
    const escapeMap: Record<string, string> = {
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "\"": "&quot;",
        "'": "&#39;",
    };
    return str ? str.replace(/[&<>"']/g, c => escapeMap[c] || c) : "";
}

/**
 * Render HTML with Open Graph metadata
 */
function renderHTML(meta: Meta, baseUrl: string): string {
    // Resolve image URL (prepend baseUrl if relative)
    const imageUrl = meta.image.startsWith("http") 
        ? meta.image 
        : `${baseUrl}${meta.image}`;
    
    // Build extra meta tags if provided
    const extraTags = meta.extra 
        ? Object.entries(meta.extra)
            .map(([key, value]) => `<meta property="${escapeHtml(key)}" content="${escapeHtml(value)}">`)
            .join("\n")
        : "";
    
    return `<!doctype html>
<html lang="${meta.locale?.split("_")[0] || "en"}">
<head>
    <meta charset="utf-8">
    <title>${escapeHtml(meta.title)}</title>
    <meta name="description" content="${escapeHtml(meta.description)}">
    
    <!-- Open Graph Tags -->
    <meta property="og:site_name" content="${escapeHtml(meta.siteName || BASE_META.siteName || "")}">
    <meta property="og:title" content="${escapeHtml(meta.title)}">
    <meta property="og:description" content="${escapeHtml(meta.description)}">
    <meta property="og:url" content="${meta.url}">
    <meta property="og:image" content="${imageUrl}">
    <meta property="og:type" content="${escapeHtml(meta.type || "website")}">
    ${meta.locale ? `<meta property="og:locale" content="${escapeHtml(meta.locale)}">` : ""}
    ${meta.imageWidth ? `<meta property="og:image:width" content="${escapeHtml(meta.imageWidth)}">` : ""}
    ${meta.imageHeight ? `<meta property="og:image:height" content="${escapeHtml(meta.imageHeight)}">` : ""}
    
    <!-- Twitter Card Tags -->
    <meta name="twitter:card" content="${escapeHtml(meta.twitterCard || "summary_large_image")}">
    <meta name="twitter:title" content="${escapeHtml(meta.title)}">
    <meta name="twitter:description" content="${escapeHtml(meta.description)}">
    <meta name="twitter:image" content="${imageUrl}">
    
    <!-- Additional Meta Tags -->
    ${extraTags}
</head>
<body>
    <!-- This body content is only shown to crawlers -->
    <h1>${escapeHtml(meta.title)}</h1>
    <p>${escapeHtml(meta.description)}</p>
    <img src="${imageUrl}" alt="${escapeHtml(meta.title)}">
    <p>Loading application...</p>
</body>
</html>`;
}

/* ============================================================================
 * ADDITIONAL PATTERNS & EXAMPLES
 * These show common patterns you might want to implement
 * ============================================================================ */

/**
 * PATTERN: Multi-language support
 * Extract language preference from request headers
 */
function getPreferredLanguage(req: Request): string {
    const acceptLanguage = req.headers.get("accept-language");
    if (!acceptLanguage) return "en";
    
    // Parse and return primary language code
    return acceptLanguage.split(",")[0].split("-")[0].toLowerCase();
}

/**
 * PATTERN: Dynamic image generation
 * Generate OG images on the fly using external services
 */
function generateDynamicImage(title: string, env: Env): string {
    // Example using Cloudinary
    // return `https://res.cloudinary.com/${env.CLOUDINARY_NAME}/image/upload/` +
    //        `w_1200,h_630,c_fit,l_text:Arial_60:${encodeURIComponent(title)}/og-base.jpg`;
    
    // Example using a separate Worker
    // return `${env.IMAGE_WORKER_URL}?title=${encodeURIComponent(title)}`;
    
    return BASE_META.image;  // Fallback to static image
}

/**
 * PATTERN: A/B testing different metadata
 * Randomly serve different versions for testing
 */
function getABTestMeta(baseMeta: Meta): Meta {
    const variant = Math.random() > 0.5 ? "A" : "B";
    
    if (variant === "B") {
        return {
            ...baseMeta,
            title: `🔥 ${baseMeta.title}`,  // Test emoji in title
            description: `Don't miss out! ${baseMeta.description}`,  // Test urgency
        };
    }
    
    return baseMeta;
}

/**
 * PATTERN: Structured data for rich snippets
 * Add JSON-LD structured data for better SEO
 */
function generateStructuredData(type: string, data: any): string {
    const structuredData = {
        "@context": "https://schema.org",
        "@type": type,
        ...data
    };
    
    return `<script type="application/ld+json">
        ${JSON.stringify(structuredData, null, 2)}
    </script>`;
}
/* Cloudflare Worker — metadata shim + transparent proxy  (dynamic domains) */

import { parse } from "regexparam";

/* -------- 0. Types & Constants ------------------------------------------ */

interface Env {
    BOT_UA: string;          // regex for crawler UAs (wrangler var)
    ORIGIN_HOST: string;     // do-origin.<domain> (wrangler var)
}

interface Meta {
    title: string;
    description: string;
    image: string;
    url: string;
    // Extended properties for base/defaults
    siteName?: string;
    type?: string;
    locale?: string;
    imageWidth?: string;
    imageHeight?: string;
    twitterCard?: string;
}

// User agent regex for crawlers
const BOT_UA_STRING =
    "(facebookexternalhit|facebot|facebookcatalog|meta-external(?:agent|fetcher)|" +
    "twitterbot|pinterest(?:bot)?|linkedinbot|slackbot-linkexpanding|discordbot|" +
    "telegrambot|whatsapp|skypeuripreview|embedly|opengraph|iframely|vkshare|bitlybot)";
const BOT_UA = new RegExp(BOT_UA_STRING, "i");


// Base metadata extracted from index.html
const BASE_META: Omit<Meta, 'url'> = {
    siteName: "Vrooli",
    title: "Vrooli",
    description: "A collaborative and self-improving automation platform",
    // Define image path relative to baseHost, will be prepended later
    image: "/og-image.webp",
    type: "website",
    locale: "en_US",
    imageWidth: "1200",
    imageHeight: "630",
    twitterCard: "summary_large_image",
};

// Base path for default image if specific images fail
const DEFAULT_IMAGE_PATH = "/default.png";

// Define API response interfaces
interface UserApiResponse { name: string; handle: string; bio?: string; avatar?: string; }
// Add other API response interfaces as needed
// interface TeamApiResponse { name: string; description?: string; logo?: string; }
// interface ProjectApiResponse { name: string; description?: string; /* ... */ }


/* -------- 1. Route table ------------------------------------------------ */

// Define static meta overrides (extend BASE_META)
const STATIC_META: Record<string, Partial<Meta>> = {
    "/about": {
        title: "About Vrooli",
        description: "Learn more about the Vrooli platform.",
    },
    "/search": {
        title: "Search • Vrooli",
        description: "Search across Vrooli.",
    },
    // Add other static pages from shared/src/consts/ui.ts as needed
};

const routes = [
    // Dynamic routes - fetch data from API
    {
        pattern: "/u/:username",
        api: getUserMeta,
        typeLabel: "User" // Label for "Not Found" messages
    },
    // Example for another dynamic route type
    // {
    //     pattern: "/team/:teamId",
    //     api: getTeamMeta,
    //     typeLabel: "Team"
    // },
    // { pattern: "/p/:projectId", api: getProjectMeta },
].map((r) => ({ ...r, routeInfo: parse(r.pattern) }));


/* -------- 2. Main handler ---------------------------------------------- */
export default {
    async fetch(req: Request, env: Env): Promise<Response> {
        const urlObj = new URL(req.url);
        const baseHost = `${urlObj.protocol}//${urlObj.host}`; // Dynamic base host (e.g., https://vrooli.com)
        const pathname = urlObj.pathname;
        console.log(`💡 Worker hit: ${req.url} (Base: ${baseHost})`);

        const ua = req.headers.get("user-agent") ?? "";

        /* Humans → proxy to origin ------------------------------------------ */
        if (!BOT_UA.test(ua)) {
            return proxyToOrigin(req, env);
        }

        /* Crawlers → OG/TC meta --------------------------------------------- */

        const cleanPath = pathname.replace(/\/+$/, "");   // drop trailing slash
        const barePath = cleanPath.split("?")[0];

        // Handle root path "/"
        if (barePath === "/") {
            console.log(` Mapped / to BASE_META`);
            const html = render(
                { ...BASE_META, url: baseHost + "/" },
                baseHost
            );
            return createHtmlResponse(html);
        }

        // Handle static routes
        if (STATIC_META[barePath]) {
            console.log(` Mapped ${barePath} to STATIC_META`);
            const staticMeta = {
                ...BASE_META, // Start with base
                ...STATIC_META[barePath], // Override with static specifics
                url: baseHost + barePath, // Set dynamic URL
            };
            const html = render(staticMeta, baseHost);
            return createHtmlResponse(html);
        }

        // Handle dynamic routes
        for (const { routeInfo, api, typeLabel } of routes) {
            const m = routeInfo.pattern.exec(barePath);
            if (!m) continue;

            console.log(` Mapped ${barePath} to dynamic route (Type: ${typeLabel})`);
            const params: Record<string, string> = {};
            if (Array.isArray(routeInfo.keys)) {
                routeInfo.keys.forEach((k, i) => (params[k] = m[i + 1]));
            }

            // Fetch meta, providing baseHost and typeLabel
            const meta = await api(params as any, env, baseHost, typeLabel);
            const html = render(meta, baseHost);
            return createHtmlResponse(html);
        }

        // If no route matched, proxy (could also return a generic 404 meta page)
        console.log(` No route matched for ${pathname}, proxying.`);
        return proxyToOrigin(req, env);
    },
};

/* -------- 3. Proxy & Response Helpers ---------------------------------- */

function proxyToOrigin(req: Request, env: Env): Promise<Response> {
    const origin = new URL(req.url);
    origin.hostname = env.ORIGIN_HOST; // e.g. do-origin.vrooli.com
    console.log(`↪️ Proxying to origin: ${origin.toString()}`);

    return fetch(origin.toString(), {
        cf: { resolveOverride: env.ORIGIN_HOST }, // Force resolution to origin IP
        method: req.method,
        headers: req.headers,
        body: req.body,
        redirect: "manual",
    });
}

function createHtmlResponse(html: string): Response {
    return new Response(html, {
        headers: {
            "Content-Type": "text/html;charset=utf-8",
            // Define cache settings (e.g., 1 hour public cache, revalidate after 1 day)
            "Cache-Control": "public, s-maxage=3600, stale-while-revalidate=86400",
        },
    });
}


/* -------- 4. Meta Fetchers --------------------------------------------- */

/**
 * Fetches metadata for a user profile.
 * @param params Route parameters (e.g., { username: '...' })
 * @param env Environment variables
 * @param baseHost The base URL of the current request (e.g., https://vrooli.com)
 * @param typeLabel Label for fallback messages (e.g., "User")
 * @returns Promise<Meta>
 */
async function getUserMeta(params: { username: string }, env: Env, baseHost: string, typeLabel: string): Promise<Meta> {
    const { username } = params;
    const apiUrl = `https://${env.ORIGIN_HOST}/api/v2/rest/user/${username}`;
    const profileUrl = `${baseHost}/u/${username}`; // Dynamic profile URL

    console.log(`  Fetching user meta from: ${apiUrl}`);
    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });

        if (!res.ok) {
            console.error(`  API Error: ${res.status} for ${apiUrl}`);
            return fallbackMeta(baseHost, profileUrl, typeLabel); // Use dynamic fallback
        }

        const u = await res.json() as UserApiResponse;
        console.log(`  User meta fetched successfully for ${username}`);
        return {
            ...BASE_META, // Start with base
            title: `${u.name} (${u.handle}) on Vrooli`,
            description: u.bio ?? BASE_META.description ?? "", // Fallback description if bio is empty
            image: u.avatar ?? baseHost + DEFAULT_IMAGE_PATH, // Use default image if avatar missing
            url: profileUrl,
        };
    } catch (error) {
        console.error(`  Fetch failed for ${apiUrl}:`, error);
        return fallbackMeta(baseHost, profileUrl, typeLabel);
    }
}

// Add other API fetchers here following the same pattern
// async function getTeamMeta(params: { teamId: string }, env: Env, baseHost: string, typeLabel: string): Promise<Meta> {
//     const { teamId } = params;
//     const apiUrl = `https://${env.ORIGIN_HOST}/api/v2/rest/team/${teamId}`;
//     const teamUrl = `${baseHost}/team/${teamId}`;
//     console.log(`  Fetching team meta from: ${apiUrl}`);
//     // ... implementation similar to getUserMeta ...
//     // return fallbackMeta(baseHost, teamUrl, typeLabel);
// }


/**
 * Generates fallback metadata for when a resource is not found or an error occurs.
 * @param baseHost The base URL of the current request (e.g., https://vrooli.com)
 * @param path The requested path (e.g., /u/nonexistentuser)
 * @param typeLabel The type of resource that wasn't found (e.g., "User", "Team")
 * @returns Meta object
 */
function fallbackMeta(baseHost: string, url: string, typeLabel: string = "Page"): Meta {
    console.warn(`  Generating fallback meta for ${url} (Type: ${typeLabel})`);
    return {
        ...BASE_META, // Start with base
        title: `${typeLabel} not found • Vrooli🐇`, // Custom not found title
        description: BASE_META.description ?? "", // Keep base description
        image: baseHost + DEFAULT_IMAGE_PATH, // Use default image
        url: url, // Reflect the requested URL
    };
}


/* -------- 5. HTML Renderer --------------------------------------------- */

const esc = (s: string) =>
    s ? s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]!)) : "";

/**
 * Renders the final HTML with Open Graph and Twitter Card metadata.
 * @param m Metadata object
 * @param baseHost The base URL (e.g., https://vrooli.com) needed for resolving relative image paths
 * @returns HTML string
 */
function render(m: Meta, baseHost: string): string {
    // Resolve image URL (prepend baseHost if it's a relative path)
    const imageUrl = m.image.startsWith('http') ? m.image : baseHost + m.image;

    return `<!doctype html><html lang="en"><head><meta charset="utf-8">
<title>${esc(m.title)}</title>
<meta name="description" content="${esc(m.description)}">
<meta property="og:site_name" content="${esc(m.siteName ?? BASE_META.siteName ?? 'Vrooli')}">
<meta property="og:title" content="${esc(m.title)}">
<meta property="og:description" content="${esc(m.description)}">
<meta property="og:url" content="${m.url}">
<meta property="og:image" content="${imageUrl}">
<meta property="og:type" content="${esc(m.type ?? BASE_META.type ?? 'website')}">
${m.locale ?? BASE_META.locale ? `<meta property="og:locale" content="${esc(m.locale ?? BASE_META.locale ?? 'en_US')}">` : ''}
${m.imageWidth ?? BASE_META.imageWidth ? `<meta property="og:image:width" content="${esc(m.imageWidth ?? BASE_META.imageWidth ?? '')}">` : ''}
${m.imageHeight ?? BASE_META.imageHeight ? `<meta property="og:image:height" content="${esc(m.imageHeight ?? BASE_META.imageHeight ?? '')}">` : ''}
<meta name="twitter:card" content="${esc(m.twitterCard ?? BASE_META.twitterCard ?? 'summary_large_image')}">
${m.twitterCard !== 'summary' ? `<meta name="twitter:image" content="${imageUrl}">` : ''}
${esc(m.title) ? `<meta name="twitter:title" content="${esc(m.title)}">` : ''}
${esc(m.description) ? `<meta name="twitter:description" content="${esc(m.description)}">` : ''}
</head><body><p>Loading...</p></body></html>`;
    // Note: Removed the script load, as the worker only serves metadata to bots.
    // The actual UI loading happens when a real user hits the page and gets proxied.
}

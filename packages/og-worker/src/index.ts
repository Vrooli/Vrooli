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

const TITLE_SUFFIX = " • Vrooli🐇";

// Define API response interfaces
interface ChatResponse {
    translations: {
        language: string;
        description?: string;
        name?: string;
    }
}
interface IssueResponse {
    translations: {
        language: string;
        description?: string;
        name: string;
    }
}
interface MeetingResponse {
    translations: {
        language: string;
        description?: string;
        link?: string; // e.g. zoom link
        name: string;
    }
}
// ResourceResponse used for Api, Note, Project, Data Structure, Prompt, Smart Contract, RoutineSingleStep, RoutineMultiStep
interface ResourceResponse {
    translations: {
        language: string;
        description?: string;
        details?: string;
        instructions?: string;
        name: string;
    }[];
    versionLabel: string;
}
interface RunResponse {
    name: string;
}
interface TeamResponse {
    handle?: string;
    profileImage?: string;
    translations: {
        language: string;
        bio?: string;
        name: string;
    }[];
}
interface UserResponse {
    handle?: string;
    name: string;
    profileImage?: string;
    translations: {
        language: string;
        bio?: string;
    }[];
}


/* -------- 1. Route table ------------------------------------------------ */

// Define static meta overrides (extend BASE_META)
const STATIC_META: Record<string, Partial<Meta>> = {
    "/about": {
        title: "About Vrooli",
        description: "Learn more about the Vrooli platform.",
    },
    "/calendar": {
        title: `Calendar${TITLE_SUFFIX}`,
        description: "Manage your schedule and events.",
    },
    "/create": {
        title: `Create${TITLE_SUFFIX}`,
        description: "Create new resources, teams, or projects.",
    },
    "/forgot-password": {
        title: `Forgot Password${TITLE_SUFFIX}`,
        description: "Reset your account password.",
    },
    "/login": {
        title: `Login${TITLE_SUFFIX}`,
        description: "Log in to your Vrooli account.",
    },
    "/privacy": {
        title: `Privacy Policy${TITLE_SUFFIX}`,
        description: "Read our Privacy Policy.",
    },
    "/reports": {
        title: `Reports${TITLE_SUFFIX}`,
        description: "View reports.",
    },
    "/reset-password": {
        title: `Reset Password${TITLE_SUFFIX}`,
        description: "Complete your password reset.",
    },
    "/search": {
        title: `Search${TITLE_SUFFIX}`,
        description: "Search across Vrooli.",
    },
    "/signup": {
        title: `Sign Up${TITLE_SUFFIX}`,
        description: "Create a new Vrooli account.",
    },
    "/stats": {
        title: `Stats${TITLE_SUFFIX}`,
        description: "View Vrooli platform statistics.",
    },
    "/terms": {
        title: `Terms of Service${TITLE_SUFFIX}`,
        description: "Read our Terms of Service.",
    },
    // Add other static pages from shared/src/consts/ui.ts as needed
};

const routes = [
    // Dynamic routes - fetch data from API
    {
        patterns: ["/u/:id"],
        api: getUserMeta,
        typeLabel: "User"
    },
    {
        patterns: ["/team/:id"],
        api: getTeamMeta,
        typeLabel: "Team"
    },
    {
        patterns: ["/api/:id", "/api/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "API"
    },
    {
        patterns: ["/note/:id", "/note/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Note"
    },
    {
        patterns: ["/project/:id", "/project/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Project"
    },
    {
        patterns: ["/code/:id", "/code/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Data Converter"
    },
    {
        patterns: ["/ds/:id", "/ds/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Data Structure"
    },
    {
        patterns: ["/prompt/:id", "/prompt/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Prompt"
    },
    {
        patterns: ["/flow/:id", "/flow/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Flow"
    },
    {
        patterns: ["/action/:id", "/action/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Action"
    },
    {
        patterns: ["/contract/:id", "/contract/:id/v/:version"],
        api: getResourceMeta,
        typeLabel: "Smart Contract"
    },
    {
        patterns: ["/chat/:id"],
        api: getChatMeta,
        typeLabel: "Chat"
    },
    {
        patterns: ["/issue/:id"],
        api: getIssueMeta,
        typeLabel: "Issue"
    },
    {
        patterns: ["/meeting/:id"],
        api: getMeetingMeta,
        typeLabel: "Meeting"
    },
    {
        patterns: ["/run/:id"],
        api: getRunMeta,
        typeLabel: "Run"
    },
];

// Flatten routes: Create an entry for each pattern
const flatRoutes = routes.flatMap((route) =>
    route.patterns.map((pattern) => ({
        pattern: pattern,
        routeInfo: parse(pattern),
        api: route.api,
        typeLabel: route.typeLabel,
    }))
);

// Helper to build API URLs
function buildApiUrl(originHost: string, requestPath: string): string {
    // Ensure path starts with a slash and remove trailing slashes
    const cleanPath = (requestPath.startsWith('/') ? requestPath : `/${requestPath}`).replace(/\/+$/, "");
    // Handle root explicitly if necessary, though unlikely for dynamic routes
    if (cleanPath === "") return `https://${originHost}/api/v2/rest/`;
    return `https://${originHost}/api/v2/rest${cleanPath}`;
}

// Helper to parse Accept-Language header (basic)
function getPreferredLanguages(req: Request): string[] {
    const acceptLanguage = req.headers.get("accept-language");
    if (!acceptLanguage) return ['en']; // Default to English if header is missing

    return acceptLanguage
        .split(',')
        .map(lang => lang.split(';')[0].trim().toLowerCase().split('-')[0]) // Get primary code (e.g., 'en' from 'en-US')
        .filter(Boolean); // Remove any empty strings
}

// Helper to find the best translation based on language preference
function getTranslation<T extends { language: string }>( // Ensure type T has language property for array logic
    translations: T[] | T | undefined | null, // Can be array, single object, or undefined
    preferredLanguages: string[]
): T | undefined {
    if (!translations) {
        return undefined;
    }

    // Handle if translations is potentially a single object (e.g., Chat, Issue, Meeting based on current types)
    // We assume if it's not an array, we just return it directly.
    // If these types change to arrays later, this check handles it.
    if (!Array.isArray(translations)) {
        // If the single object *doesn't* have a language property, this is fine.
        // If it *does*, we might want to add language checking here too.
        return translations;
    }

    // Handle Array case
    if (translations.length === 0) {
        return undefined;
    }

    // 1. Try preferred languages
    for (const lang of preferredLanguages) {
        // Ensure find callback type checks t.language
        const found = translations.find((t: T) => t.language === lang);
        if (found) return found;
    }

    // 2. Try English
    const english = translations.find((t: T) => t.language === 'en');
    if (english) return english;

    // 3. Fallback to the first one
    return translations[0];
}


/* -------- 2. Main handler ---------------------------------------------- */
export default {
    async fetch(req: Request, env: Env): Promise<Response> {
        const urlObj = new URL(req.url);
        const baseHost = `${urlObj.protocol}//${urlObj.host}`; // Dynamic base host (e.g., https://vrooli.com)
        const pathname = urlObj.pathname;

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
            const html = render(
                { ...BASE_META, url: baseHost + "/" },
                baseHost
            );
            return createHtmlResponse(html);
        }

        // Handle static routes
        if (STATIC_META[barePath]) {
            const staticMeta = {
                ...BASE_META, // Start with base
                ...STATIC_META[barePath], // Override with static specifics
                url: baseHost + barePath, // Set dynamic URL
            };
            const html = render(staticMeta, baseHost);
            return createHtmlResponse(html);
        }

        // Handle dynamic routes
        for (const { routeInfo, api, typeLabel } of flatRoutes) {
            const m = routeInfo.pattern.exec(barePath);
            if (!m) continue;

            // Fetch meta, providing baseHost and typeLabel
            const meta = await api(env, baseHost, typeLabel, barePath, req);
            const html = render(meta, baseHost);
            return createHtmlResponse(html);
        }

        // If no route matched, proxy (could also return a generic 404 meta page)
        return proxyToOrigin(req, env);
    },
};

/* -------- 3. Proxy & Response Helpers ---------------------------------- */

function proxyToOrigin(req: Request, env: Env): Promise<Response> {
    const origin = new URL(req.url);
    origin.hostname = env.ORIGIN_HOST; // e.g. do-origin.vrooli.com

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
 * @param env Environment variables
 * @param baseHost The base URL of the current request (e.g., https://vrooli.com)
 * @param typeLabel Label for fallback messages (e.g., "User")
 * @param pathname The original pathname of the request
 * @param req The request object
 * @returns Promise<Meta>
 */
async function getUserMeta(env: Env, baseHost: string, typeLabel: string, pathname: string, req: Request): Promise<Meta> {
    const apiUrl = buildApiUrl(env.ORIGIN_HOST, pathname);
    const profileUrl = `${baseHost}${pathname}`; // Use original path for user-facing URL

    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });

        if (!res.ok) {
            return fallbackMeta(baseHost, profileUrl, typeLabel);
        }

        const u = await res.json() as UserResponse;
        const preferredLanguages = getPreferredLanguages(req);
        const translation = getTranslation(u.translations, preferredLanguages);
        const bio = translation?.bio;
        return {
            ...BASE_META,
            title: `${u.name} (${u.handle})${TITLE_SUFFIX}`,
            description: bio ?? BASE_META.description ?? "",
            image: u.profileImage ?? BASE_META.image,
            url: profileUrl,
        };
    } catch (error) {
        return fallbackMeta(baseHost, profileUrl, typeLabel);
    }
}

async function getTeamMeta(env: Env, baseHost: string, typeLabel: string, pathname: string, req: Request): Promise<Meta> {
    const apiUrl = buildApiUrl(env.ORIGIN_HOST, pathname);
    const teamUrl = `${baseHost}${pathname}`;

    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });
        if (!res.ok) throw new Error(`API Error: ${res.status}`);
        const data = await res.json() as TeamResponse;
        const preferredLanguages = getPreferredLanguages(req);
        const translation = getTranslation(data.translations, preferredLanguages);
        const bio = translation?.bio;
        const name = translation?.name ?? data.handle ?? typeLabel;
        return {
            ...BASE_META,
            title: `${name}${data.handle ? ` (${data.handle})` : ''}${TITLE_SUFFIX}`,
            description: bio ?? BASE_META.description ?? "",
            image: data.profileImage ?? BASE_META.image,
            url: teamUrl,
        };
    } catch (error) {
        return fallbackMeta(baseHost, teamUrl, typeLabel);
    }
}

async function getResourceMeta(env: Env, baseHost: string, typeLabel: string, pathname: string, req: Request): Promise<Meta> {
    const apiUrl = buildApiUrl(env.ORIGIN_HOST, pathname);
    const resourceUrl = `${baseHost}${pathname}`;

    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });
        if (!res.ok) throw new Error(`API Error: ${res.status}`);
        const data = await res.json() as ResourceResponse;
        const preferredLanguages = getPreferredLanguages(req);
        const translation = getTranslation(data.translations, preferredLanguages);
        const name = translation?.name ?? typeLabel;
        const description = translation?.description ?? translation?.details ?? BASE_META.description ?? "";

        return {
            ...BASE_META,
            title: `${name}${TITLE_SUFFIX}`,
            description: description,
            url: resourceUrl,
        };
    } catch (error) {
        return fallbackMeta(baseHost, resourceUrl, typeLabel);
    }
}

async function getChatMeta(env: Env, baseHost: string, typeLabel: string, pathname: string, req: Request): Promise<Meta> {
    const apiUrl = buildApiUrl(env.ORIGIN_HOST, pathname);
    const chatUrl = `${baseHost}${pathname}`;
    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });
        if (!res.ok) throw new Error(`API Error: ${res.status}`);
        const data = await res.json() as ChatResponse;
        const preferredLanguages = getPreferredLanguages(req);
        const translation = getTranslation(data.translations, preferredLanguages);

        const name = translation?.name ?? typeLabel;
        const description = translation?.description ?? BASE_META.description ?? "";

        return {
            ...BASE_META,
            title: `${name}${TITLE_SUFFIX}`,
            description: description,
            url: chatUrl,
        };
    } catch (error) {
        return fallbackMeta(baseHost, chatUrl, typeLabel);
    }
}

async function getIssueMeta(env: Env, baseHost: string, typeLabel: string, pathname: string, req: Request): Promise<Meta> {
    const apiUrl = buildApiUrl(env.ORIGIN_HOST, pathname);
    const issueUrl = `${baseHost}${pathname}`;

    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });
        if (!res.ok) throw new Error(`API Error: ${res.status}`);
        const data = await res.json() as IssueResponse;
        const preferredLanguages = getPreferredLanguages(req);
        const translation = getTranslation(data.translations, preferredLanguages);

        const name = translation?.name ?? typeLabel;
        const description = translation?.description ?? BASE_META.description ?? "";
        return {
            ...BASE_META,
            title: `${name}${TITLE_SUFFIX}`,
            description: description,
            url: issueUrl,
        };
    } catch (error) {
        return fallbackMeta(baseHost, issueUrl, typeLabel);
    }
}

async function getMeetingMeta(env: Env, baseHost: string, typeLabel: string, pathname: string, req: Request): Promise<Meta> {
    const apiUrl = buildApiUrl(env.ORIGIN_HOST, pathname);
    const meetingUrl = `${baseHost}${pathname}`;

    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });
        if (!res.ok) throw new Error(`API Error: ${res.status}`);
        const data = await res.json() as MeetingResponse;
        const preferredLanguages = getPreferredLanguages(req);
        const translation = getTranslation(data.translations, preferredLanguages);

        const name = translation?.name ?? typeLabel;
        const description = translation?.description ?? BASE_META.description ?? "";
        return {
            ...BASE_META,
            title: `${name}${TITLE_SUFFIX}`,
            description: description,
            url: meetingUrl,
        };
    } catch (error) {
        return fallbackMeta(baseHost, meetingUrl, typeLabel);
    }
}

async function getRunMeta(env: Env, baseHost: string, typeLabel: string, pathname: string): Promise<Meta> {
    const apiUrl = buildApiUrl(env.ORIGIN_HOST, pathname);
    const runUrl = `${baseHost}${pathname}`;

    try {
        const res = await fetch(apiUrl, { cf: { resolveOverride: env.ORIGIN_HOST } });
        if (!res.ok) throw new Error(`API Error: ${res.status}`);
        const data = await res.json() as RunResponse;
        const name = data.name ?? typeLabel;
        return {
            ...BASE_META,
            title: `${name}${TITLE_SUFFIX}`,
            description: BASE_META.description ?? "",
            url: runUrl,
        };
    } catch (error) {
        return fallbackMeta(baseHost, runUrl, typeLabel);
    }
}

/**
 * Generates fallback metadata for when a resource is not found or an error occurs.
 * @param baseHost The base URL of the current request (e.g., https://vrooli.com)
 * @param path The requested path (e.g., /u/nonexistentuser)
 * @param typeLabel The type of resource that wasn't found (e.g., "User", "Team")
 * @returns Meta object
 */
function fallbackMeta(_baseHost: string, url: string, typeLabel: string = "Page"): Meta {
    return {
        ...BASE_META, // Start with base
        title: `${typeLabel} not found${TITLE_SUFFIX}`, // Custom not found title
        description: BASE_META.description ?? "", // Keep base description
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

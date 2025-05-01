/* Cloudflare Worker — metadata shim + transparent proxy  (dynamic domains) */

import { parse } from "regexparam";

/* -------- 0. Route table ------------------------------------------------ */
const routes = [
    { pattern: "/u/:username", api: getUserMeta },
    // { pattern: "/p/:projectId", api: getProjectMeta },
].map((r) => ({ ...r, routeInfo: parse(r.pattern) }));

/* -------- 1. Main handler ---------------------------------------------- */
export default {
    async fetch(req: Request, env: Env): Promise<Response> {
        console.log("💡 Worker hit", req.url);
        const ua = req.headers.get("user-agent") ?? "";
        const botRE = new RegExp(env.BOT_UA, "i");
        const urlObj = new URL(req.url);
        const base = `${urlObj.protocol}//${urlObj.host}`; // ← dynamic host

        /* Humans → origin ---------------------------------------------------- */
        if (!botRE.test(ua)) {
            return proxyToOrigin(req, env);
        }

        /* Crawlers → OG/TC meta --------------------------------------------- */
        for (const { routeInfo, api } of routes) {
            const m = routeInfo.pattern.exec(urlObj.pathname);
            if (!m) continue;

            const params: Record<string, string> = {};
            if (Array.isArray(routeInfo.keys)) {
                routeInfo.keys.forEach((k, i) => (params[k] = m[i + 1]));
            }

            const meta = await api(params, env);
            const html = render(meta);

            return new Response(html, {
                headers: {
                    "Content-Type": "text/html;charset=utf-8",
                    "Cache-Control":
                        "public, s-maxage=3600, stale-while-revalidate=86400",
                },
            });
        }

        return proxyToOrigin(req, env);
    },
};

/* -------- 2. Proxy helper (humans & assets) ---------------------------- */
function proxyToOrigin(req: Request, env: Env): Promise<Response> {
    const origin = new URL(req.url);
    origin.hostname = env.ORIGIN_HOST;               // e.g. do-origin.vrooli.com
    console.log("↪️  proxying to origin", origin.toString());

    return fetch(origin.toString(), {
        cf: { resolveOverride: env.ORIGIN_HOST },      // hit DO directly
        method: req.method,
        headers: req.headers,
        body: req.body,
        redirect: "manual",
    });
}

/* -------- 3. Meta fetchers --------------------------------------------- */
interface Meta { title: string; description: string; image: string; url: string; }
interface UserApiResponse { name: string; handle: string; bio?: string; avatar?: string; }

async function getUserMeta({ username }: any, env: any): Promise<Meta> {
    const url = `https://${env.ORIGIN_HOST}/api/v2/rest/user/${username}`;   // <-- change
    const res = await fetch(url, { cf: { resolveOverride: env.ORIGIN_HOST } });

    if (!res.ok) {
        console.log("API", res.status);
        return fallbackMeta(username, "User not found");
    }

    const u = await res.json() as UserApiResponse;
    return {
        title: `${u.name} (${u.handle}) on Vrooli`,
        description: u.bio ?? "",
        image: u.avatar ?? "https://vrooli.com/default.png",
        url: `https://vrooli.com/u/${username}`
    };
}

function fallbackMeta(base: string, path: string): Meta {
    return {
        title: "Not found • Vrooli🐇",
        description: "",
        image: `${base}/default.png`,
        url: `${base}${path}`,
    };
}

/* -------- 4. HTML renderer --------------------------------------------- */
const esc = (s: string) =>
    s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]!));

function render(m: Meta): string {
    return `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta property="og:title" content="${esc(m.title)}">
<meta property="og:description" content="${esc(m.description)}">
<meta property="og:image" content="${m.image}">
<meta property="og:url" content="${m.url}">
<meta name="twitter:card" content="summary_large_image">
</head><body><div id="root"></div>
<script type="module" src="/assets/index.js" defer></script>
</body></html>`;
}

/* -------- 5. Env bindings type ---------------------------------------- */
interface Env {
    BOT_UA: string;          // regex for crawler UAs (wrangler var)
    ORIGIN_HOST: string;     // do-origin.<domain> (wrangler var)
}

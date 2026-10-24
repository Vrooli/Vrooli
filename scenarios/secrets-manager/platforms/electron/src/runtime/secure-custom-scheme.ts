export const SECURE_DESKTOP_SCHEME = "vrooli-desktop";
export const SECURE_DESKTOP_HOST = "desktop";
export const SECURE_DESKTOP_URL = `${SECURE_DESKTOP_SCHEME}://${SECURE_DESKTOP_HOST}/`;

type SchemePrivilege = {
    scheme: string;
    privileges: { standard: boolean; secure: boolean; supportFetchAPI: boolean };
};

export interface SchemeRegistrar {
    registerSchemesAsPrivileged(schemes: SchemePrivilege[]): void;
}

export function registerSecureDesktopScheme(registrar: SchemeRegistrar, enabled: boolean): void {
    if (!enabled) return;
    registrar.registerSchemesAsPrivileged([{
        scheme: SECURE_DESKTOP_SCHEME,
        privileges: { standard: true, secure: true, supportFetchAPI: true },
    }]);
}

export type RequestFetcher = (request: Request) => Promise<Response>;

function localRuntimeOrigin(rawURL: string): URL {
    const target = new URL(rawURL);
    const loopbackHost = target.hostname === "localhost"
        || target.hostname === "127.0.0.1"
        || target.hostname === "[::1]";
    if (target.protocol !== "http:" || !loopbackHost || target.username || target.password) {
        throw new Error("secure custom-scheme UI requires an unauthenticated HTTP loopback runtime URL");
    }
    return target;
}

export function createSecureDesktopSchemeHandler(rawRuntimeURL: string, fetcher: RequestFetcher): (request: Request) => Promise<Response> {
    const runtime = localRuntimeOrigin(rawRuntimeURL);

    return async (request) => {
        let source: URL;
        try {
            source = new URL(request.url);
        } catch {
            return new Response("Invalid desktop URL", { status: 400 });
        }
        if (source.protocol !== `${SECURE_DESKTOP_SCHEME}:` || source.hostname !== SECURE_DESKTOP_HOST
            || source.username || source.password || source.port) {
            return new Response("Unknown desktop resource", { status: 404 });
        }

        const destination = new URL(`${source.pathname}${source.search}`, runtime.origin);
        const initial = new Request(destination, request);
        const headers = new Headers(initial.headers);
        // The main-process fetch is outside Chromium's renderer origin checks.
        // Present the runtime's own origin so its same-origin and CSRF checks
        // see the transport they already serve for the sidecar profile.
        headers.set("Origin", runtime.origin);
        try {
            const forwarded = new Request(initial, { headers, redirect: "manual" });
            return await fetcher(forwarded);
        } catch {
            return new Response("Bundled runtime request failed", { status: 502 });
        }
    };
}

import { isIP } from "node:net";

/**
 * Validation context propagation for generated Electron targets.
 *
 * The lease identifier never becomes a request header. The target launch
 * environment is the authority; requests carry only the compatibility marker
 * and non-secret context identity to app-owned origins.
 */

export interface DesktopValidationContext {
    contextId: string;
    scenarioName: string;
    artifactDigest: string;
    targetId: string;
    isolationLeaseId: string;
}

export function readDesktopValidationContext(env: Record<string, string | undefined>): DesktopValidationContext | null {
    const value = {
        contextId: env.VROOLI_VALIDATION_CONTEXT_ID?.trim() ?? "",
        scenarioName: env.VROOLI_VALIDATION_SCENARIO?.trim() ?? "",
        artifactDigest: env.VROOLI_VALIDATION_ARTIFACT_DIGEST?.trim() ?? "",
        targetId: env.VROOLI_VALIDATION_TARGET_ID?.trim() ?? "",
        isolationLeaseId: env.VROOLI_VALIDATION_ISOLATION_LEASE?.trim() ?? "",
    };
    if (Object.values(value).some((part) => part.length === 0)) return null;
    return value;
}

/**
 * Resolve the LPBS auth authority for a generated desktop app. Production
 * builds keep their generated authority. A loopback override is available
 * only to a fully bound desktop validation launch for the exact artifact.
 */
export function resolveValidationLpbsURL(env: Record<string, string | undefined>, productionURL: string): string {
    const candidate = env.VROOLI_VALIDATION_LPBS_URL?.trim();
    if (!candidate) return productionURL;

    const context = readDesktopValidationContext(env);
    if (!context || !/^sha256:[a-f0-9]{64}$/i.test(context.artifactDigest)) {
        throw new Error("validation LPBS origin requires a complete artifact validation context");
    }

    let parsed: URL;
    try {
        parsed = new URL(candidate);
    } catch {
        throw new Error("validation LPBS origin must be a numeric loopback HTTP origin");
    }
    const host = parsed.hostname.replace(/^\[|\]$/g, "");
    const ipVersion = isIP(host);
    const loopback = host === "127.0.0.1" || host === "::1" || (ipVersion === 4 && host.startsWith("127."));
    if (parsed.protocol !== "http:" || !loopback || !parsed.port || parsed.username || parsed.password ||
        parsed.pathname !== "/" || parsed.search || parsed.hash) {
        throw new Error("validation LPBS origin must be a numeric loopback HTTP origin with an explicit port");
    }
    return parsed.origin;
}

export function validationOrigins(urls: string[]): Set<string> {
    const origins = new Set<string>();
    for (const value of urls) {
        try {
            const origin = new URL(value).origin;
            if (origin !== "null") origins.add(origin);
        } catch {
            // Invalid optional template values are never eligible.
        }
    }
    return origins;
}

export function isOwnedValidationURL(rawURL: string, origins: Set<string>): boolean {
    try {
        return origins.has(new URL(rawURL).origin);
    } catch {
        return false;
    }
}

export function validationHeaders(context: DesktopValidationContext): Record<string, string> {
    return {
        "X-Vrooli-Test-Mode": "1",
        "X-Vrooli-Validation-Context": context.contextId,
    };
}

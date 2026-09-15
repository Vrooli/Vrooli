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

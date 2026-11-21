import { describe, expect, it } from "vitest";
import {
    isOwnedValidationURL,
    readDesktopValidationContext,
    resolveValidationLpbsURL,
    validationHeaders,
    validationOrigins,
} from "../validation-context";

describe("desktop validation context propagation", () => {
    const artifactDigest = `sha256:${"a".repeat(64)}`;
    const validationEnv = (extra: Record<string, string> = {}) => ({
        VROOLI_VALIDATION_CONTEXT_ID: "ctx-1",
        VROOLI_VALIDATION_SCENARIO: "browser-automation-studio",
        VROOLI_VALIDATION_ARTIFACT_DIGEST: artifactDigest,
        VROOLI_VALIDATION_TARGET_ID: "local-linux-amd64",
        VROOLI_VALIDATION_ISOLATION_LEASE: "lease-1",
        ...extra,
    });

    it("keeps the generated production LPBS authority when no override is supplied", () => {
        expect(resolveValidationLpbsURL({}, "https://vrooli.com")).toBe("https://vrooli.com");
    });

    it.each(["http://127.0.0.1:23456", "http://127.12.0.9:23456", "http://[::1]:23456"])(
        "allows validation-only numeric loopback LPBS origin %s for a digest-bound context",
        (origin) => {
            expect(resolveValidationLpbsURL(validationEnv({ VROOLI_VALIDATION_LPBS_URL: origin }), "https://vrooli.com")).toBe(origin);
        },
    );

    it.each([
        ["missing validation context", { VROOLI_VALIDATION_LPBS_URL: "http://127.0.0.1:23456" }],
        ["invalid artifact digest", validationEnv({ VROOLI_VALIDATION_ARTIFACT_DIGEST: "sha256:artifact", VROOLI_VALIDATION_LPBS_URL: "http://127.0.0.1:23456" })],
        ["localhost alias", validationEnv({ VROOLI_VALIDATION_LPBS_URL: "http://localhost:23456" })],
        ["remote host", validationEnv({ VROOLI_VALIDATION_LPBS_URL: "http://192.0.2.5:23456" })],
        ["HTTPS origin", validationEnv({ VROOLI_VALIDATION_LPBS_URL: "https://127.0.0.1:23456" })],
        ["missing port", validationEnv({ VROOLI_VALIDATION_LPBS_URL: "http://127.0.0.1" })],
        ["path", validationEnv({ VROOLI_VALIDATION_LPBS_URL: "http://127.0.0.1:23456/auth" })],
        ["query", validationEnv({ VROOLI_VALIDATION_LPBS_URL: "http://127.0.0.1:23456?next=/" })],
    ])("rejects %s validation LPBS override", (_name, env) => {
        expect(() => resolveValidationLpbsURL(env, "https://vrooli.com")).toThrow(/validation LPBS origin/);
    });

    it("requires the complete target binding and never exposes a partial lease", () => {
        expect(readDesktopValidationContext({ VROOLI_VALIDATION_CONTEXT_ID: "ctx-1" })).toBeNull();
        const context = readDesktopValidationContext({
            VROOLI_VALIDATION_CONTEXT_ID: "ctx-1",
            VROOLI_VALIDATION_SCENARIO: "demo",
            VROOLI_VALIDATION_ARTIFACT_DIGEST: "sha256:app",
            VROOLI_VALIDATION_TARGET_ID: "target-1",
            VROOLI_VALIDATION_ISOLATION_LEASE: "lease-1",
        });
        expect(context?.contextId).toBe("ctx-1");
        expect(validationHeaders(context!)).toEqual({
            "X-Vrooli-Test-Mode": "1",
            "X-Vrooli-Validation-Context": "ctx-1",
        });
        expect(validationHeaders(context!)["X-Vrooli-Isolation-Lease"]).toBeUndefined();
    });

    it("allows only exact app-owned origins", () => {
        const origins = validationOrigins(["http://127.0.0.1:4200", "https://example.test/app"]);
        expect(isOwnedValidationURL("http://127.0.0.1:4200/api/items", origins)).toBe(true);
        expect(isOwnedValidationURL("http://127.0.0.1:4201/api/items", origins)).toBe(false);
        expect(isOwnedValidationURL("https://attacker.test/api/items", origins)).toBe(false);
    });
});

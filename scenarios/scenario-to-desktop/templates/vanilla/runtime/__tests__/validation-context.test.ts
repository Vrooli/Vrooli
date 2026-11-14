import { describe, expect, it } from "vitest";
import {
    isOwnedValidationURL,
    readDesktopValidationContext,
    validationHeaders,
    validationOrigins,
} from "../validation-context";

describe("desktop validation context propagation", () => {
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

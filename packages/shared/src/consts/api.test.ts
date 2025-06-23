import { describe, it, expect } from "vitest";
import { 
    getOAuthInitRoute, 
    getOAuthCallbackRoute, 
    AUTH_ROUTE_PREFIX, 
    OAUTH_ROUTE_CALLBACK,
    OAUTH_PROVIDERS, 
} from "./api.js";

describe("OAuth Route Functions", () => {
    describe("getOAuthInitRoute", () => {
        it("should generate correct init route for valid providers", () => {
            expect(getOAuthInitRoute("google")).toBe("/auth/google");
            expect(getOAuthInitRoute("github")).toBe("/auth/github");
            expect(getOAuthInitRoute("facebook")).toBe("/auth/facebook");
            expect(getOAuthInitRoute("apple")).toBe("/auth/apple");
            expect(getOAuthInitRoute("x")).toBe("/auth/x");
        });

        it("should handle custom provider names", () => {
            expect(getOAuthInitRoute("custom-provider")).toBe("/auth/custom-provider");
            expect(getOAuthInitRoute("oauth2")).toBe("/auth/oauth2");
        });

        it("should handle empty provider name", () => {
            expect(getOAuthInitRoute("")).toBe("/auth/");
        });

        it("should use AUTH_ROUTE_PREFIX constant", () => {
            const testProvider = "test";
            const expected = `${AUTH_ROUTE_PREFIX}/${testProvider}`;
            expect(getOAuthInitRoute(testProvider)).toBe(expected);
        });
    });

    describe("getOAuthCallbackRoute", () => {
        it("should generate correct callback route for valid providers", () => {
            expect(getOAuthCallbackRoute("google")).toBe("/auth/google/callback");
            expect(getOAuthCallbackRoute("github")).toBe("/auth/github/callback");
            expect(getOAuthCallbackRoute("facebook")).toBe("/auth/facebook/callback");
            expect(getOAuthCallbackRoute("apple")).toBe("/auth/apple/callback");
            expect(getOAuthCallbackRoute("x")).toBe("/auth/x/callback");
        });

        it("should handle custom provider names", () => {
            expect(getOAuthCallbackRoute("custom-provider")).toBe("/auth/custom-provider/callback");
            expect(getOAuthCallbackRoute("oauth2")).toBe("/auth/oauth2/callback");
        });

        it("should handle empty provider name", () => {
            expect(getOAuthCallbackRoute("")).toBe("/auth//callback");
        });

        it("should use AUTH_ROUTE_PREFIX and OAUTH_ROUTE_CALLBACK constants", () => {
            const testProvider = "test";
            const expected = `${AUTH_ROUTE_PREFIX}/${testProvider}${OAUTH_ROUTE_CALLBACK}`;
            expect(getOAuthCallbackRoute(testProvider)).toBe(expected);
        });
    });

    describe("Integration with OAUTH_PROVIDERS", () => {
        it("should work correctly with all defined OAuth providers", () => {
            // Test init routes
            expect(getOAuthInitRoute(OAUTH_PROVIDERS.Google)).toBe("/auth/google");
            expect(getOAuthInitRoute(OAUTH_PROVIDERS.GitHub)).toBe("/auth/github");
            expect(getOAuthInitRoute(OAUTH_PROVIDERS.Facebook)).toBe("/auth/facebook");
            expect(getOAuthInitRoute(OAUTH_PROVIDERS.Apple)).toBe("/auth/apple");
            expect(getOAuthInitRoute(OAUTH_PROVIDERS.X)).toBe("/auth/x");

            // Test callback routes
            expect(getOAuthCallbackRoute(OAUTH_PROVIDERS.Google)).toBe("/auth/google/callback");
            expect(getOAuthCallbackRoute(OAUTH_PROVIDERS.GitHub)).toBe("/auth/github/callback");
            expect(getOAuthCallbackRoute(OAUTH_PROVIDERS.Facebook)).toBe("/auth/facebook/callback");
            expect(getOAuthCallbackRoute(OAUTH_PROVIDERS.Apple)).toBe("/auth/apple/callback");
            expect(getOAuthCallbackRoute(OAUTH_PROVIDERS.X)).toBe("/auth/x/callback");
        });
    });

    describe("Edge cases", () => {
        it("should handle providers with special characters", () => {
            expect(getOAuthInitRoute("provider-with-dashes")).toBe("/auth/provider-with-dashes");
            expect(getOAuthCallbackRoute("provider_with_underscores")).toBe("/auth/provider_with_underscores/callback");
        });

        it("should handle numeric provider names", () => {
            expect(getOAuthInitRoute("oauth2")).toBe("/auth/oauth2");
            expect(getOAuthCallbackRoute("provider123")).toBe("/auth/provider123/callback");
        });

        it("should maintain provider case sensitivity", () => {
            expect(getOAuthInitRoute("Google")).toBe("/auth/Google");
            expect(getOAuthCallbackRoute("GITHUB")).toBe("/auth/GITHUB/callback");
        });
    });
});

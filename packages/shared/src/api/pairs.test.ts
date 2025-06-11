import { describe, expect, it } from "vitest";
import {
    endpointsActions,
    endpointsApiKey,
    endpointsAuth,
    endpointsBookmark,
    endpointsChatInvite,
    endpointsResource,
    endpointsUser,
} from "./pairs.js";

describe("API Endpoint Pairs", () => {
    describe("HTTP Method Security and Semantics", () => {
        it("should use safe methods for read operations", () => {
            // GET should be used for read operations - they should be safe and idempotent
            expect(endpointsBookmark.findOne.method).toBe("GET");
            expect(endpointsBookmark.findMany.method).toBe("GET");
            expect(endpointsUser.profile.method).toBe("GET");
            expect(endpointsUser.exportData.method).toBe("GET");
            
            // Export operations should use GET since they're retrieving data
            expect(endpointsUser.exportCalendar.method).toBe("GET");
        });

        it("should use appropriate methods for data modification", () => {
            // POST should be used for non-idempotent operations (creation, actions)
            expect(endpointsBookmark.createOne.method).toBe("POST");
            expect(endpointsUser.importUserData.method).toBe("POST");
            expect(endpointsActions.copy.method).toBe("POST");
            
            // PUT should be used for idempotent updates
            expect(endpointsBookmark.updateOne.method).toBe("PUT");
            expect(endpointsUser.profileUpdate.method).toBe("PUT");
        });

        it("should use POST for destructive actions for CSRF protection", () => {
            // Destructive actions should use POST to prevent CSRF via GET requests
            expect(endpointsActions.deleteOne.method).toBe("POST");
            expect(endpointsActions.deleteMany.method).toBe("POST");
            expect(endpointsActions.deleteAll.method).toBe("POST");
            expect(endpointsActions.deleteAccount.method).toBe("POST");
        });
    });

    describe("CRUD Operation Patterns", () => {
        it("should use public identifiers for external-facing operations", () => {
            // Read operations should use publicId for external API access
            expect(endpointsBookmark.findOne.endpoint).toMatch(/:publicId$/);
            expect(endpointsResource.findOne.endpoint).toMatch(/:publicId$/);
            
            // This ensures URLs are not guessable and don't expose internal database IDs
        });

        it("should use internal IDs for authenticated update operations", () => {
            // Update operations can use internal IDs since they require authentication
            expect(endpointsBookmark.updateOne.endpoint).toMatch(/:id$/);
            expect(endpointsChatInvite.updateOne.endpoint).toMatch(/:id$/);
            
            // This allows for more efficient database operations while maintaining security
        });

        it("should follow REST resource naming conventions", () => {
            // Collection endpoints should use plural nouns
            expect(endpointsBookmark.findMany.endpoint).toBe("/bookmarks");
            expect(endpointsChatInvite.findMany.endpoint).toBe("/chatInvites");
            
            // Individual resource endpoints should use singular nouns
            expect(endpointsBookmark.createOne.endpoint).toBe("/bookmark");
            expect(endpointsChatInvite.createOne.endpoint).toBe("/chatInvite");
        });
    });

    describe("Authentication Endpoints", () => {
        it("should have secure authentication patterns", () => {
            // All auth endpoints should use POST for security
            Object.values(endpointsAuth).forEach(endpoint => {
                expect(endpoint.method).toBe("POST");
            });
            
            // Auth endpoints should be under /auth namespace
            Object.values(endpointsAuth).forEach(endpoint => {
                expect(endpoint.endpoint).toMatch(/^\/auth/);
            });
        });
    });

    describe("Action Endpoints", () => {
        it("should use POST for all destructive actions", () => {
            // All delete operations should use POST (not DELETE) for safety
            expect(endpointsActions.deleteOne.method).toBe("POST");
            expect(endpointsActions.deleteMany.method).toBe("POST");
            expect(endpointsActions.deleteAll.method).toBe("POST");
            expect(endpointsActions.deleteAccount.method).toBe("POST");
        });

        it("should use appropriate naming for actions", () => {
            // Action endpoints should have descriptive names
            expect(endpointsActions.copy.endpoint).toBe("/copy");
            expect(endpointsActions.deleteOne.endpoint).toBe("/deleteOne");
            expect(endpointsActions.deleteMany.endpoint).toBe("/deleteMany");
        });
    });

    describe("Batch Operations", () => {
        it("should support batch operations where appropriate", () => {
            // Chat invites support batch operations
            expect(endpointsChatInvite.createMany).toBeDefined();
            expect(endpointsChatInvite.updateMany).toBeDefined();
            
            // Batch endpoints should use plural paths
            expect(endpointsChatInvite.createMany.endpoint).toMatch(/s$/);
            expect(endpointsChatInvite.updateMany.endpoint).toMatch(/s$/);
            
            // Batch operations should use same HTTP method as single operations
            expect(endpointsChatInvite.createOne.method).toBe(endpointsChatInvite.createMany.method);
            expect(endpointsChatInvite.updateOne.method).toBe(endpointsChatInvite.updateMany.method);
        });
    });

    describe("Resource Versioning", () => {
        it("should have consistent version endpoint patterns", () => {
            // All version endpoints should follow pattern /:publicId/v/:versionLabel
            const versionEndpoints = [
                endpointsResource.findResourceVersion,
                endpointsResource.findApiVersion,
                endpointsResource.findNoteVersion,
                endpointsResource.findProjectVersion,
            ];
            
            versionEndpoints.forEach(endpoint => {
                expect(endpoint.endpoint).toMatch(/:publicId\/v\/:versionLabel$/);
                expect(endpoint.method).toBe("GET");
            });
        });

        it("should use appropriate resource type prefixes", () => {
            // Each resource type should have its own prefix
            expect(endpointsResource.findApiVersion.endpoint).toMatch(/^\/api\//);
            expect(endpointsResource.findDataConverterVersion.endpoint).toMatch(/^\/code\//);
            expect(endpointsResource.findNoteVersion.endpoint).toMatch(/^\/note\//);
            expect(endpointsResource.findProjectVersion.endpoint).toMatch(/^\/project\//);
        });
    });

    describe("User Profile Endpoints", () => {
        it("should distinguish between public and private user endpoints", () => {
            // Public user endpoint should use handle/id pattern
            expect(endpointsUser.findOne.endpoint).toMatch(/\/u\//);
            
            // Private profile endpoints should use /profile
            expect(endpointsUser.profile.endpoint).toBe("/profile");
            expect(endpointsUser.profileUpdate.endpoint).toBe("/profile");
            
            // Profile read should use GET, update should use PUT
            expect(endpointsUser.profile.method).toBe("GET");
            expect(endpointsUser.profileUpdate.method).toBe("PUT");
        });
    });

    describe("Import/Export Endpoints", () => {
        it("should use appropriate HTTP methods for import/export", () => {
            // Imports should use POST (sending data)
            expect(endpointsUser.importCalendar.method).toBe("POST");
            expect(endpointsUser.importUserData.method).toBe("POST");
            
            // Exports should use GET (retrieving data)
            expect(endpointsUser.exportCalendar.method).toBe("GET");
            expect(endpointsUser.exportData.method).toBe("GET");
        });
    });
});
// AI_CHECK: TEST_COVERAGE=6 | LAST: 2025-06-18
import { describe, it, expect } from "vitest";
import { 
    findOne, 
    findMany, 
    createOne, 
    createMany, 
    updateOne, 
    updateMany, 
    standardCRUD, 
    resourceVersion, 
} from "./pairs.js";

describe("API endpoint helper functions", () => {
    describe("findOne", () => {
        it("should generate correct findOne endpoint for single resource", () => {
            const result = findOne("user");
            expect(result).toEqual({
                findOne: {
                    endpoint: "/user/:publicId",
                    method: "GET",
                },
            });
        });

        it("should handle different resource names", () => {
            expect(findOne("post")).toEqual({
                findOne: {
                    endpoint: "/post/:publicId",
                    method: "GET",
                },
            });
            
            expect(findOne("comment")).toEqual({
                findOne: {
                    endpoint: "/comment/:publicId",
                    method: "GET",
                },
            });
        });

    });

    describe("findMany", () => {
        it("should generate correct findMany endpoint for multiple resources", () => {
            const result = findMany("users");
            expect(result).toEqual({
                findMany: {
                    endpoint: "/users",
                    method: "GET",
                },
            });
        });

        it("should handle different plural resource names", () => {
            expect(findMany("posts")).toEqual({
                findMany: {
                    endpoint: "/posts",
                    method: "GET",
                },
            });
            
            expect(findMany("comments")).toEqual({
                findMany: {
                    endpoint: "/comments",
                    method: "GET",
                },
            });
        });

    });

    describe("createOne", () => {
        it("should generate correct createOne endpoint", () => {
            const result = createOne("user");
            expect(result).toEqual({
                createOne: {
                    endpoint: "/user",
                    method: "POST",
                },
            });
        });

        it("should handle different resource names", () => {
            expect(createOne("post")).toEqual({
                createOne: {
                    endpoint: "/post",
                    method: "POST",
                },
            });
        });

    });

    describe("createMany", () => {
        it("should generate correct createMany endpoint", () => {
            const result = createMany("users");
            expect(result).toEqual({
                createMany: {
                    endpoint: "/users",
                    method: "POST",
                },
            });
        });

        it("should handle different plural resource names", () => {
            expect(createMany("posts")).toEqual({
                createMany: {
                    endpoint: "/posts",
                    method: "POST",
                },
            });
        });

    });

    describe("updateOne", () => {
        it("should generate correct updateOne endpoint", () => {
            const result = updateOne("user");
            expect(result).toEqual({
                updateOne: {
                    endpoint: "/user/:id",
                    method: "PUT",
                },
            });
        });

        it("should handle different resource names", () => {
            expect(updateOne("post")).toEqual({
                updateOne: {
                    endpoint: "/post/:id",
                    method: "PUT",
                },
            });
        });

    });

    describe("updateMany", () => {
        it("should generate correct updateMany endpoint", () => {
            const result = updateMany("users");
            expect(result).toEqual({
                updateMany: {
                    endpoint: "/users",
                    method: "PUT",
                },
            });
        });

        it("should handle different plural resource names", () => {
            expect(updateMany("posts")).toEqual({
                updateMany: {
                    endpoint: "/posts",
                    method: "PUT",
                },
            });
        });

    });

    describe("standardCRUD", () => {
        it("should combine findOne, findMany, createOne, and updateOne", () => {
            const result = standardCRUD("user", "users");
            expect(result).toEqual({
                findOne: {
                    endpoint: "/user/:publicId",
                    method: "GET",
                },
                findMany: {
                    endpoint: "/users",
                    method: "GET",
                },
                createOne: {
                    endpoint: "/user",
                    method: "POST",
                },
                updateOne: {
                    endpoint: "/user/:id",
                    method: "PUT",
                },
            });
        });

        it("should handle different resource combinations", () => {
            const result = standardCRUD("post", "posts");
            expect(result).toEqual({
                findOne: {
                    endpoint: "/post/:publicId",
                    method: "GET",
                },
                findMany: {
                    endpoint: "/posts",
                    method: "GET",
                },
                createOne: {
                    endpoint: "/post",
                    method: "POST",
                },
                updateOne: {
                    endpoint: "/post/:id",
                    method: "PUT",
                },
            });
        });

    });

    describe("resourceVersion", () => {
        it("should generate correct versioned resource endpoint", () => {
            const result = resourceVersion("api");
            expect(result).toEqual({
                endpoint: "/api/:publicId/v/:versionLabel",
                method: "GET",
            });
        });

        it("should handle different resource names", () => {
            expect(resourceVersion("routine")).toEqual({
                endpoint: "/routine/:publicId/v/:versionLabel",
                method: "GET",
            });

            expect(resourceVersion("project")).toEqual({
                endpoint: "/project/:publicId/v/:versionLabel",
                method: "GET",
            });
        });

    });
});

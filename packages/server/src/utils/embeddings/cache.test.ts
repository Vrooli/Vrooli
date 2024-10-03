import { VisibilityType } from "@local/shared";
import { RedisClientMock } from "../../__mocks__/redis";
import { SearchEmbeddingsCache } from "./cache";

const sampleKey = "search:type:query:option:visibility";
const sampleResults = [{ id: "1" }, { id: "2" }];

describe("SearchEmbeddingsCache", () => {

    beforeEach(() => {
        jest.clearAllMocks();
        RedisClientMock.resetMock();
        RedisClientMock.__setAllMockData({});
    });

    describe("createKey", () => {
        it("should include user ID for non-public visibility types", () => {
            const key = SearchEmbeddingsCache.createKey({
                objectType: "User",
                searchString: "query",
                sortOption: "EmbedTopDesc",
                visibility: VisibilityType.Own,
                userId: "123",
            });
            expect(key).toBe("search:User:query:EmbedTopDesc:Own:123");
        });

        it("should not include user ID for public visibility type", () => {
            const key = SearchEmbeddingsCache.createKey({
                objectType: "RoutineVersion",
                searchString: "query",
                sortOption: "EmbedTopAsc",
                visibility: VisibilityType.Public,
                userId: "123",
            });
            expect(key).toBe("search:RoutineVersion:query:EmbedTopAsc:Public");
        });

        it("should change to public if user ID is not provided", () => {
            const key = SearchEmbeddingsCache.createKey({
                objectType: "ProjectVersion",
                searchString: "query",
                sortOption: "EmbedDateCreatedDesc",
                visibility: VisibilityType.OwnOrPublic,
                userId: null,
            });
            expect(key).toBe("search:ProjectVersion:query:EmbedDateCreatedDesc:Public");
        });
    });

    describe("check", () => {
        it("should return null if there are no cached results", async () => {
            const result = await SearchEmbeddingsCache.check({ cacheKey: sampleKey, offset: 0, take: 2 });
            expect(result).toBeNull();
        });

        it("should return the correct slice of results if they exist", async () => {
            RedisClientMock.__setAllMockData({
                [sampleKey]: JSON.stringify(sampleResults),
            });
            const result = await SearchEmbeddingsCache.check({ cacheKey: sampleKey, offset: 0, take: 1 });
            expect(result).toEqual([{ id: "1" }]);
        });
    });

    describe("set", () => {
        it("should cache new results correctly", async () => {
            // No need to mock get since we're setting data
            await SearchEmbeddingsCache.set({ cacheKey: sampleKey, offset: 0, take: 2, results: sampleResults });
            expect(RedisClientMock.instance?.set).toHaveBeenCalledWith(sampleKey, JSON.stringify(sampleResults));
        });
    });
});

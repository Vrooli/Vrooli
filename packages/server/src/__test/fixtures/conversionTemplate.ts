/**
 * Template for converting endpoint tests to use the two-fixture pattern
 * 
 * CONVERSION CHECKLIST:
 * 
 * 1. Replace imports:
 *    - Remove: import { after, before, beforeEach, describe, it } from "mocha";
 *    - Remove: import sinon from "sinon";
 *    - Add: import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
 * 
 * 2. Add fixture imports:
 *    // Import database fixtures for seeding
 *    import { [Model]DbFactory, seed[Model]s } from "../../__test/fixtures/[model]Fixtures.js";
 *    import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/userFixtures.js";
 *    
 *    // Import validation fixtures for API input testing
 *    import { [model]TestDataFactory } from "@vrooli/shared/src/validation/models/__test__/fixtures/[model]Fixtures.js";
 * 
 * 3. Replace beforeAll logger stubs:
 *    OLD:
 *    let loggerErrorStub: sinon.SinonStub;
 *    let loggerInfoStub: sinon.SinonStub;
 *    beforeAll(() => {
 *        loggerErrorStub = sinon.stub(logger, "error");
 *        loggerInfoStub = sinon.stub(logger, "info");
 *    });
 *    
 *    NEW:
 *    beforeAll(() => {
 *        // Use Vitest spies to suppress logger output during tests
 *        vi.spyOn(logger, "error").mockImplementation(() => logger);
 *        vi.spyOn(logger, "info").mockImplementation(() => logger);
 *    });
 * 
 * 4. Replace beforeEach user creation:
 *    OLD:
 *    user1Id = generatePK();
 *    await DbProvider.get().user.create({
 *        data: {
 *            ...defaultPublicUserData(),
 *            id: user1Id,
 *            name: "Test User 1",
 *        },
 *    });
 *    
 *    NEW:
 *    // Seed test users using database fixtures
 *    testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
 * 
 * 5. Replace model creation in beforeEach:
 *    OLD:
 *    await DbProvider.get().model.create({
 *        data: {
 *            id: generatePK(),
 *            // ... manual construction
 *        }
 *    });
 *    
 *    NEW:
 *    // Use database fixtures for seeding
 *    modelData = await seedModels(DbProvider.get(), {
 *        // ... options
 *    });
 * 
 * 6. Replace afterAll cleanup:
 *    OLD:
 *    loggerErrorStub.restore();
 *    loggerInfoStub.restore();
 *    
 *    NEW:
 *    // Restore all mocks
 *    vi.restoreAllMocks();
 * 
 * 7. Update test assertions:
 *    OLD (Chai):
 *    expect(result).to.not.be.null;
 *    expect(result.id).to.equal(expectedId);
 *    expect(result.items).to.be.an("array");
 *    expect(result.flag).to.be.true;
 *    
 *    NEW (Vitest):
 *    expect(result).not.toBeNull();
 *    expect(result.id).toEqual(expectedId);
 *    expect(result.items).toBeInstanceOf(Array);
 *    expect(result.flag).toBe(true);
 * 
 * 8. Update API input creation:
 *    OLD:
 *    const input: ModelCreateInput = {
 *        id: uuid(),
 *        // ... manual construction
 *    };
 *    
 *    NEW:
 *    // Use validation fixtures for API input
 *    const input: ModelCreateInput = modelTestDataFactory.createMinimal({
 *        // ... overrides only
 *    });
 * 
 * 9. Update error expectations:
 *    OLD:
 *    try {
 *        await endpoint.method({ input }, { req, res });
 *        expect.fail("Expected an error");
 *    } catch (err) {
 *        // ...
 *    }
 *    
 *    NEW:
 *    await expect(async () => {
 *        await endpoint.method({ input }, { req, res });
 *    }).rejects.toThrow();
 * 
 * 10. Use dynamic test data:
 *     - Replace hardcoded IDs with fixture IDs or generated ones
 *     - Use testUsers[0].id instead of user1Id
 *     - Use modelData.items[0].id instead of hardcoded IDs
 */

// Example structure after conversion:
export const exampleConvertedTest = `
import { type ModelCreateInput, type ModelSearchInput, type ModelUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { model_createOne } from "../generated/model_createOne.js";
import { model_findMany } from "../generated/model_findMany.js";
import { model_findOne } from "../generated/model_findOne.js";
import { model_updateOne } from "../generated/model_updateOne.js";
import { model } from "./model.js";

// Import database fixtures for seeding
import { ModelDbFactory, seedModels } from "../../__test/fixtures/modelFixtures.js";
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/userFixtures.js";

// Import validation fixtures for API input testing
import { modelTestDataFactory } from "@vrooli/shared/src/validation/models/__test__/fixtures/modelFixtures.js";

describe("EndpointsModel", () => {
    let testUsers: any[];
    let modelData: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Seed models using database fixtures
        modelData = await seedModels(DbProvider.get(), {
            userId: testUsers[0].id,
            count: 2,
            // ... other options
        });
    });

    afterAll(async () => {
        // Clean up
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("createOne", () => {
        it("creates a model for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            
            // Use validation fixtures for API input
            const input: ModelCreateInput = modelTestDataFactory.createMinimal({
                // ... minimal overrides
            });
            
            const result = await model.createOne({ input }, { req, res }, model_createOne);
            expect(result).not.toBeNull();
            expect(result.id).toBeDefined();
        });
    });
});
`;
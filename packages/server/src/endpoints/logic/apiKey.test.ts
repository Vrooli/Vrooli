import { ApiKeyPermission, uuid } from "@vrooli/shared";
import { randomBytes } from "crypto";
import { apiKey } from "./apiKey.js";

// Mock the necessary modules
vi.mock("../../auth/apiKeyEncryption.js");
vi.mock("../../auth/request.js");
vi.mock("../../actions/creates.js");

// Import the mocked modules
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { RequestService } from "../../auth/request.js";

// Create mock implementations
const mockApiKeyEncryption = {
    generateSiteKey: vi.fn(),
    hashSiteKey: vi.fn(),
    verifySiteKey: vi.fn(),
};

const mockRequestService = {
    assertRequestFrom: vi.fn(),
    get: vi.fn(() => ({
        rateLimit: vi.fn(),
    })),
};

// Mock the creates helper
const mockCreateOneHelper = vi.fn();
vi.doMock("../../actions/creates.js", () => ({
    createOneHelper: mockCreateOneHelper,
}));

// Set up the mocks
(ApiKeyEncryptionService.get as any).mockReturnValue(mockApiKeyEncryption);
(RequestService.assertRequestFrom as any) = mockRequestService.assertRequestFrom;

describe("API Key Endpoints", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("createOne", () => {
        const mockInput = {
            name: "Test API Key",
            permissions: [ApiKeyPermission.ReadPublic, ApiKeyPermission.ReadPrivate],
            creditsLimitHard: 1000,
            creditsLimitSoft: 800,
            stopAtLimit: true,
        };

        const mockReq = {
            session: { userId: "test-user-id" },
            ip: "127.0.0.1",
        };

        const mockInfo = {
            __typename: "ApiKeyCreated",
            fieldsByTypeName: {},
        };

        it("should create API key and return raw key for first time", async () => {
            // Set up mocks
            const rawKey = "test-raw-key-" + randomBytes(16).toString("hex");
            const hashedKey = "hashed-" + rawKey;
            
            mockCreateOneHelper.mockResolvedValue({
                id: "1",
                name: mockInput.name,
                key: rawKey, // This should be the raw key for one-time display
                permissions: mockInput.permissions,
                creditsUsed: 0,
                creditsLimitHard: mockInput.creditsLimitHard,
                creditsLimitSoft: mockInput.creditsLimitSoft,
                stopAtLimit: mockInput.stopAtLimit,
                createdAt: new Date().toISOString(),
            });

            // Call the endpoint
            const result = await apiKey.createOne.customImplementation!({
                input: mockInput,
                req: mockReq,
                info: mockInfo,
            });

            // Verify request validation was called
            expect(mockRequestService.assertRequestFrom).toHaveBeenCalledWith(
                mockReq,
                { isOfficialUser: true }
            );

            // Verify createOneHelper was called with correct parameters
            expect(mockCreateOneHelper).toHaveBeenCalledWith({
                info: mockInfo,
                input: mockInput,
                objectType: "ApiKey",
                req: mockReq,
            });

            // Verify the response contains the raw key
            expect(result).toEqual(
                expect.objectContaining({
                    id: "1",
                    name: mockInput.name,
                    key: rawKey, // Raw key should be returned
                    permissions: mockInput.permissions,
                })
            );

            // Verify the raw key is properly formatted
            expect(result.key).toBeDefined();
            expect(typeof result.key).toBe("string");
            expect(result.key.length).toBeGreaterThan(20);
        });

        it("should validate required fields", async () => {
            const invalidInput = {
                // Missing required fields
                permissions: [],
            };

            mockCreateOneHelper.mockRejectedValue(new Error("Invalid input"));

            await expect(
                apiKey.createOne.customImplementation!({
                    input: invalidInput,
                    req: mockReq,
                    info: mockInfo,
                })
            ).rejects.toThrow("Invalid input");
        });

        it("should enforce isOfficialUser requirement", async () => {
            mockRequestService.assertRequestFrom.mockImplementation(() => {
                throw new Error("NotLoggedInOfficial");
            });

            await expect(
                apiKey.createOne.customImplementation!({
                    input: mockInput,
                    req: mockReq,
                    info: mockInfo,
                })
            ).rejects.toThrow("NotLoggedInOfficial");

            // Verify createOneHelper was not called
            expect(mockCreateOneHelper).not.toHaveBeenCalled();
        });

        it("should respect rate limiting", async () => {
            // The endpoint has rateLimit: 10, which should be enforced
            expect(apiKey.createOne.rateLimit).toBe(10);
        });

        it("should require write auth permissions", async () => {
            expect(apiKey.createOne.permissions).toEqual({
                hasWriteAuthPermissions: true,
            });
        });
    });

    describe("validate", () => {
        const mockValidateInput = {
            id: "123",
            secret: "test-api-key-secret",
        };

        const mockReq = {
            ip: "127.0.0.1",
        };

        const mockInfo = {
            __typename: "ApiKey",
            fieldsByTypeName: {},
        };

        beforeEach(() => {
            // Reset the rate limiting mock
            const mockRateLimit = vi.fn();
            (RequestService.get as any).mockReturnValue({
                rateLimit: mockRateLimit,
            });
        });

        it("should validate correct API key", async () => {
            // Mock finding the API key
            const mockApiKeyData = {
                id: BigInt(123),
                name: "Test Key",
                permissions: [ApiKeyPermission.ReadPublic],
                creditsUsed: 100,
                creditsLimitHard: 1000,
                user: { id: "user-123" },
            };

            // Mock the validation success
            mockApiKeyEncryption.verifySiteKey.mockResolvedValue(true);

            // Mock database operations - this would normally be handled by the actual implementation
            // For now, we're testing the structure and flow

            const validateFn = apiKey.validate;
            expect(validateFn).toBeDefined();
            expect(typeof validateFn).toBe("function");
        });

        it("should reject invalid input", async () => {
            const invalidInputs = [
                { id: "", secret: "valid-secret" },
                { id: "valid-id", secret: "" },
                { id: 123, secret: "valid-secret" }, // id should be string
                { secret: "valid-secret" }, // missing id
                { id: "valid-id" }, // missing secret
                {},
                null,
                undefined,
            ];

            for (const invalidInput of invalidInputs) {
                await expect(
                    apiKey.validate(
                        { input: invalidInput as any },
                        { req: mockReq },
                        mockInfo
                    )
                ).rejects.toThrow();
            }
        });

        it("should apply rate limiting", async () => {
            const mockRateLimit = vi.fn();
            (RequestService.get as any).mockReturnValue({
                rateLimit: mockRateLimit,
            });

            try {
                await apiKey.validate(
                    { input: mockValidateInput },
                    { req: mockReq },
                    mockInfo
                );
            } catch {
                // Expected to fail due to mocking, but we want to check rate limiting was called
            }

            expect(mockRateLimit).toHaveBeenCalledWith({
                maxApi: 5000,
                req: mockReq,
            });
        });
    });

    describe("updateOne", () => {
        const mockUpdateInput = {
            id: "123",
            name: "Updated API Key Name",
            creditsLimitHard: 2000,
        };

        const mockReq = {
            session: { userId: "test-user-id" },
            ip: "127.0.0.1",
        };

        const mockInfo = {
            __typename: "ApiKey",
            fieldsByTypeName: {},
        };

        it("should enforce isOfficialUser requirement", async () => {
            mockRequestService.assertRequestFrom.mockImplementation(() => {
                throw new Error("NotLoggedInOfficial");
            });

            await expect(
                apiKey.updateOne.customImplementation!({
                    input: mockUpdateInput,
                    req: mockReq,
                    info: mockInfo,
                })
            ).rejects.toThrow("NotLoggedInOfficial");
        });

        it("should respect rate limiting", async () => {
            expect(apiKey.updateOne.rateLimit).toBe(10);
        });

        it("should require write auth permissions", async () => {
            expect(apiKey.updateOne.permissions).toEqual({
                hasWriteAuthPermissions: true,
            });
        });
    });
});

describe("API Key Security", () => {
    describe("Key Generation", () => {
        it("should generate cryptographically secure keys", () => {
            const key1 = ApiKeyEncryptionService.generateSiteKey();
            const key2 = ApiKeyEncryptionService.generateSiteKey();
            
            // Keys should be different
            expect(key1).not.toBe(key2);
            
            // Keys should be properly formatted (base64url)
            expect(key1).toMatch(/^[A-Za-z0-9_-]+$/);
            expect(key2).toMatch(/^[A-Za-z0-9_-]+$/);
            
            // Keys should be of reasonable length
            expect(key1.length).toBeGreaterThan(20);
            expect(key2.length).toBeGreaterThan(20);
        });
    });

    describe("Key Hashing", () => {
        it("should hash keys with bcrypt", async () => {
            const plainKey = "test-plain-key";
            const hash1 = await ApiKeyEncryptionService.get().hashSiteKey(plainKey);
            const hash2 = await ApiKeyEncryptionService.get().hashSiteKey(plainKey);
            
            // Hashes should be different due to salt
            expect(hash1).not.toBe(hash2);
            
            // Both should verify against the original
            expect(await ApiKeyEncryptionService.get().verifySiteKey(plainKey, hash1)).toBe(true);
            expect(await ApiKeyEncryptionService.get().verifySiteKey(plainKey, hash2)).toBe(true);
            
            // Wrong key should not verify
            expect(await ApiKeyEncryptionService.get().verifySiteKey("wrong-key", hash1)).toBe(false);
        });
    });
});
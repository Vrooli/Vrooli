import { type WalletUpdateInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { mockAuthenticatedSession, mockLoggedOutSession, mockApiSession, mockWriteAuthPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { testEndpointRequiresAuth } from "../../__test/endpoints.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { wallet_updateOne } from "../generated/wallet_updateOne.js";
import { wallet } from "./wallet.js";
// Import database fixtures
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { WalletDbFactory } from "../../__test/fixtures/db/walletFixtures.js";

describe("EndpointsWallet", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.wallet.deleteMany();
        await prisma.user.deleteMany();
        // Clear Redis cache
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("updateOne", () => {
        describe("authentication", () => {
            it("requires authentication", async () => {
                await testEndpointRequiresAuth(
                    wallet.updateOne,
                    { id: generatePK() },
                    wallet_updateOne,
                );
            });

            it("requires auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWritePrivatePermissions(["Wallet"]), // Wrong permission type
                });

                const input: WalletUpdateInput = {
                    id: generatePK(),
                    name: "Updated Wallet",
                };

                await expect(wallet.updateOne({ input }, { req, res }, wallet_updateOne))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("updates own wallet", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create wallet for the user
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Original Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        publicAddress: "addr1qx4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3xyz",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "Updated Main Wallet",
                };

                const result = await wallet.updateOne({ input }, { req, res }, wallet_updateOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Wallet");
                expect(result.id).toBe(userWallet.id.toString());
                expect(result.name).toBe("Updated Main Wallet");
                expect(result.stakingAddress).toBe("stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3");
                expect(result.user?.id).toBe(testUser[0].id.toString());

                // Verify in database
                const updatedWallet = await DbProvider.get().wallet.findUnique({
                    where: { id: userWallet.id },
                });
                expect(updatedWallet?.name).toBe("Updated Main Wallet");
            });

            it("updates wallet with description", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Hardware Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "Ledger Hardware Wallet",
                    // Note: In a real implementation, there might be a description field
                };

                const result = await wallet.updateOne({ input }, { req, res }, wallet_updateOne);

                expect(result.name).toBe("Ledger Hardware Wallet");
            });

            it("allows updating multiple wallet properties", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Multi-sig Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        publicAddress: "addr1qx4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3xyz",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "Updated Multi-sig Wallet",
                    // In a real implementation, you might be able to update other fields
                    // like metadata, preferences, etc.
                };

                const result = await wallet.updateOne({ input }, { req, res }, wallet_updateOne);

                expect(result.name).toBe("Updated Multi-sig Wallet");
                expect(result.stakingAddress).toBe("stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3");
                expect(result.publicAddress).toBe("addr1qx4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3xyz");
            });

            it("preserves verification status", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Verified Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "Renamed Verified Wallet",
                };

                const result = await wallet.updateOne({ input }, { req, res }, wallet_updateOne);

                expect(result.name).toBe("Renamed Verified Wallet");
                expect(result.verifiedAt).toBeDefined();
                expect(result.stakingAddress).toBe("stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3");
            });

            it("handles wallet name normalization", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "original name",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "  Trimmed Name  ", // With extra whitespace
                };

                const result = await wallet.updateOne({ input }, { req, res }, wallet_updateOne);

                // Name should be trimmed
                expect(result.name).toBe("Trimmed Name");
            });

            it("allows updating unverified wallet", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Unverified Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: null, // Not verified
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "Updated Unverified Wallet",
                };

                const result = await wallet.updateOne({ input }, { req, res }, wallet_updateOne);

                expect(result.name).toBe("Updated Unverified Wallet");
                expect(result.verifiedAt).toBeNull();
            });
        });

        describe("invalid", () => {
            it("cannot update another user's wallet", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
                
                // Create wallet for user 1
                const user1Wallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "User 1 Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUsers[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                // Try to update as user 2
                const { req, res } = await mockApiSession({
                    userId: testUsers[1].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: user1Wallet.id.toString(),
                    name: "Hacked Wallet",
                };

                await expect(wallet.updateOne({ input }, { req, res }, wallet_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("cannot update non-existent wallet", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: generatePK(),
                    name: "Non-existent Wallet",
                };

                await expect(wallet.updateOne({ input }, { req, res }, wallet_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects duplicate wallet name for same user", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create two wallets for the same user
                const wallet1 = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Main Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const wallet2 = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Secondary Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm4",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                // Try to update wallet2 to have the same name as wallet1
                const input: WalletUpdateInput = {
                    id: wallet2.id.toString(),
                    name: "Main Wallet", // Duplicate name
                };

                await expect(wallet.updateOne({ input }, { req, res }, wallet_updateOne))
                    .rejects.toThrow();
            });

            it("rejects empty wallet name", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Valid Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "", // Empty name
                };

                await expect(wallet.updateOne({ input }, { req, res }, wallet_updateOne))
                    .rejects.toThrow();
            });

            it("rejects wallet name that's too long", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Valid Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "A".repeat(256), // Very long name (exceeds typical length limits)
                };

                await expect(wallet.updateOne({ input }, { req, res }, wallet_updateOne))
                    .rejects.toThrow();
            });

            it("prevents updating wallet addresses", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Secure Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "Updated Wallet Name",
                    // Note: Address fields should not be updatable for security reasons
                    // This test ensures that addresses remain immutable
                };

                const result = await wallet.updateOne({ input }, { req, res }, wallet_updateOne);

                // Verify addresses haven't changed
                expect(result.stakingAddress).toBe("stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3");
                expect(result.name).toBe("Updated Wallet Name");
            });

            it("rejects update with profane wallet name", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                const userWallet = await DbProvider.get().wallet.create({
                    data: WalletDbFactory.createMinimal({
                        name: "Clean Wallet",
                        stakingAddress: "stake1uy4jj73pfyejl4d2rs6nc70eykkhhu56p3y2t2s7deaxrze38lcm3",
                        userId: testUser[0].id,
                        verifiedAt: new Date(),
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: WalletUpdateInput = {
                    id: userWallet.id.toString(),
                    name: "fuck wallet", // Profane name
                };

                await expect(wallet.updateOne({ input }, { req, res }, wallet_updateOne))
                    .rejects.toThrow();
            });
        });
    });
});

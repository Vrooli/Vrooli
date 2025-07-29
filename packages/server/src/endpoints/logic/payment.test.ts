import { type FindByIdInput, type PaymentSearchInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertRequiresApiKeyReadPermissions, assertRequiresAuth } from "../../__test/authTestUtils.js";
import { mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { payment_findMany } from "../generated/payment_findMany.js";
import { payment_findOne } from "../generated/payment_findOne.js";
import { payment } from "./payment.js";
// Import database fixtures for seeding
import { PaymentDbFactory } from "../../__test/fixtures/db/paymentFixtures.js";
import { TeamDbFactory } from "../../__test/fixtures/db/teamFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsPayment", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["team", "member", "member_invite", "meeting", "user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.team(DbProvider.get());
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    async function createTestData() {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create test teams
        const testTeam1 = await DbProvider.get().team.create({
            data: TeamDbFactory.createMinimal({
                id: generatePK(),
                createdById: testUsers[0].id,
                members: {
                    create: [{
                        id: generatePK(),
                        userId: testUsers[0].id,
                        permissions: "[\"owner\"]",
                    }],
                },
            }),
        });

        const testTeam2 = await DbProvider.get().team.create({
            data: TeamDbFactory.createMinimal({
                id: generatePK(),
                createdById: testUsers[1].id,
                members: {
                    create: [{
                        id: generatePK(),
                        userId: testUsers[1].id,
                        permissions: "[\"owner\"]",
                    }],
                },
            }),
        });

        // Create various payments
        const payments = await Promise.all([
            // User 1's successful payment
            DbProvider.get().payment.create({
                data: PaymentDbFactory.createUserSubscription(testUsers[0].id, "PremiumMonthly", "Succeeded"),
            }),
            // User 1's failed payment
            DbProvider.get().payment.create({
                data: PaymentDbFactory.createFailed(testUsers[0].id, "user", "card_declined"),
            }),
            // User 2's pending payment
            DbProvider.get().payment.create({
                data: PaymentDbFactory.createUserSubscription(testUsers[1].id, "PremiumAnnual", "Pending"),
            }),
            // Team 1's successful payment
            DbProvider.get().payment.create({
                data: PaymentDbFactory.createTeamSubscription(testTeam1.id, "TeamMonthly", "Succeeded", 15),
            }),
            // Team 2's payment
            DbProvider.get().payment.create({
                data: PaymentDbFactory.createTeamSubscription(testTeam2.id, "TeamAnnual", "Succeeded", 50),
            }),
            // Credit purchase for user 1
            DbProvider.get().payment.create({
                data: PaymentDbFactory.createCreditPurchase(testUsers[0].id, "user", 25.00, 2500000, "Succeeded"),
            }),
        ]);

        return { testUsers, testTeam1, testTeam2, payments };
    }

    describe("findOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    payment.findOne,
                    { id: generatePK().toString() },
                    payment_findOne,
                );
            });

            it("API key - no read permissions", async () => {
                await assertRequiresApiKeyReadPermissions(
                    payment.findOne,
                    { id: generatePK().toString() },
                    payment_findOne,
                    ["Payment"],
                );
            });
        });

        describe("valid", () => {
            it("returns own user payment", async () => {
                const { testUsers, payments } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: payments[0].id.toString() };
                const result = await payment.findOne({ input }, { req, res }, payment_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(payments[0].id.toString());
                expect(result.amount).toBe(payments[0].amount);
                expect(result.status).toBe("Succeeded");
                expect(result.paymentType).toBe("PremiumMonthly");
                expect(result.user?.id).toBe(testUsers[0].id.toString());
            });

            it("returns own team payment as team owner", async () => {
                const { testUsers, testTeam1, payments } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const teamPayment = payments[3]; // Team 1's payment
                const input: FindByIdInput = { id: teamPayment.id.toString() };
                const result = await payment.findOne({ input }, { req, res }, payment_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(teamPayment.id.toString());
                expect(result.team?.id).toBe(testTeam1.id.toString());
            });

            it("returns payment via API key with read permissions", async () => {
                const { testUsers, payments } = await createTestData();
                const { req, res } = await mockApiSession({
                    userId: testUsers[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockReadPrivatePermissions(["Payment"]),
                });

                const input: FindByIdInput = { id: payments[0].id.toString() };
                const result = await payment.findOne({ input }, { req, res }, payment_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(payments[0].id.toString());
            });
        });

        describe("invalid", () => {
            it("cannot view other user's payment", async () => {
                const { testUsers, payments } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                // Try to access user 2's payment
                const input: FindByIdInput = { id: payments[2].id.toString() };
                const result = await payment.findOne({ input }, { req, res }, payment_findOne);

                expect(result).toBeNull();
            });

            it("cannot view payment from team not a member of", async () => {
                const { testUsers, payments } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[2].id.toString(), // User 3 is not in any team
                });

                // Try to access team 1's payment
                const input: FindByIdInput = { id: payments[3].id.toString() };
                const result = await payment.findOne({ input }, { req, res }, payment_findOne);

                expect(result).toBeNull();
            });

            it("returns null for non-existent payment", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: generatePK().toString() };
                const result = await payment.findOne({ input }, { req, res }, payment_findOne);

                expect(result).toBeNull();
            });
        });
    });

    describe("findMany", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    payment.findMany,
                    {},
                    payment_findMany,
                );
            });

            it("API key - no read permissions", async () => {
                await assertRequiresApiKeyReadPermissions(
                    payment.findMany,
                    {},
                    payment_findMany,
                    ["Payment"],
                );
            });
        });

        describe("valid", () => {
            it("returns only own payments", async () => {
                const { testUsers, payments } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: PaymentSearchInput = {};
                const result = await payment.findMany({ input }, { req, res }, payment_findMany);

                expect(result.results).toHaveLength(3); // User 1 has 3 payments (2 user + 1 team)
                expect(result.totalCount).toBe(3);

                // Verify all returned payments belong to user 1
                const paymentIds = result.results.map(p => p.id);
                expect(paymentIds).toContain(payments[0].id.toString()); // User 1's successful payment
                expect(paymentIds).toContain(payments[1].id.toString()); // User 1's failed payment
                expect(paymentIds).toContain(payments[3].id.toString()); // Team 1's payment
                expect(paymentIds).not.toContain(payments[2].id.toString()); // User 2's payment
            });

            it("filters by status", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: PaymentSearchInput = {
                    statuses: ["Succeeded"],
                };
                const result = await payment.findMany({ input }, { req, res }, payment_findMany);

                expect(result.results).toHaveLength(2); // 2 successful payments for user 1
                expect(result.results.every(p => p.status === "Succeeded")).toBe(true);
            });

            it("filters by payment type", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: PaymentSearchInput = {
                    paymentTypes: ["PremiumMonthly"],
                };
                const result = await payment.findMany({ input }, { req, res }, payment_findMany);

                expect(result.results).toHaveLength(2); // User 1 has 2 PremiumMonthly payments
                expect(result.results.every(p => p.paymentType === "PremiumMonthly")).toBe(true);
            });

            it("sorts by created date descending", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: PaymentSearchInput = {
                    sortBy: "DateCreatedDesc",
                };
                const result = await payment.findMany({ input }, { req, res }, payment_findMany);

                // Verify results are sorted by creation date (newest first)
                for (let i = 1; i < result.results.length; i++) {
                    const prevDate = new Date(result.results[i - 1].created_at);
                    const currDate = new Date(result.results[i].created_at);
                    expect(prevDate.getTime()).toBeGreaterThanOrEqual(currDate.getTime());
                }
            });

            it("paginates results", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                // First page
                const input1: PaymentSearchInput = {
                    take: 2,
                    skip: 0,
                };
                const result1 = await payment.findMany({ input: input1 }, { req, res }, payment_findMany);

                expect(result1.results).toHaveLength(2);
                expect(result1.totalCount).toBe(3);

                // Second page
                const input2: PaymentSearchInput = {
                    take: 2,
                    skip: 2,
                };
                const result2 = await payment.findMany({ input: input2 }, { req, res }, payment_findMany);

                expect(result2.results).toHaveLength(1);
                expect(result2.totalCount).toBe(3);

                // Ensure no overlap
                const ids1 = result1.results.map(p => p.id);
                const ids2 = result2.results.map(p => p.id);
                expect(ids1.some(id => ids2.includes(id))).toBe(false);
            });

            it("returns team payments for team members", async () => {
                const { testUsers, testTeam1, payments } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                // Team payments are automatically included for team members via visibility system
                const input: PaymentSearchInput = {};
                const result = await payment.findMany({ input }, { req, res }, payment_findMany);

                // User 1 should see their own payments plus team payments (3 total)
                expect(result.results).toHaveLength(3);

                // Check that team payment is included
                const teamPayment = result.results.find(p => p.team?.id === testTeam1.id.toString());
                expect(teamPayment).toBeDefined();
                expect(teamPayment?.id).toBe(payments[3].id.toString());
            });

            it("searches by amount range", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: PaymentSearchInput = {
                    minAmount: 1000, // $10.00
                    maxAmount: 3000, // $30.00
                };
                const result = await payment.findMany({ input }, { req, res }, payment_findMany);

                expect(result.results.length).toBeGreaterThan(0);
                expect(result.results.every(p => p.amount >= 1000 && p.amount <= 3000)).toBe(true);
            });
        });

        describe("invalid", () => {
            it("returns empty results when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: PaymentSearchInput = {};
                await expect(payment.findMany({ input }, { req, res }, payment_findMany))
                    .rejects.toThrow();
            });

            it("does not return other users' payments", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[2].id.toString(), // User 3 has no payments
                });

                const input: PaymentSearchInput = {};
                const result = await payment.findMany({ input }, { req, res }, payment_findMany);

                expect(result.results).toHaveLength(0);
                expect(result.totalCount).toBe(0);
            });
        });
    });
});

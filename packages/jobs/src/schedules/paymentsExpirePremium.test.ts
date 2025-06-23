import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mockQueueService, mockEmailAddTask, resetAllMocks } from "../__test/mocks/services.js";

// Mock services before imports
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        QueueService: mockQueueService,
        AUTH_EMAIL_TEMPLATES: {
            SubscriptionCanceled: vi.fn().mockReturnValue({
                subject: "Subscription Canceled",
                body: "Your subscription has been canceled",
            }),
        },
    };
});

import { paymentsExpirePlan } from "./paymentsExpirePremium.js";
import { DbProvider } from "@vrooli/server";

describe("paymentsExpirePlan integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testPlanIds: bigint[] = [];
    const testEmailIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testPlanIds.length = 0;
        testEmailIds.length = 0;

        // Reset mocks
        resetAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testEmailIds.length > 0) {
            await db.email.deleteMany({ where: { id: { in: testEmailIds } } });
        }
        if (testPlanIds.length > 0) {
            await db.plan.deleteMany({ where: { id: { in: testPlanIds } } });
        }
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should expire user plans that are past expiration date", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000); // Yesterday
        
        // Create user with expired plan
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Expired User",
                handle: "expireduser",
                stripeCustomerId: "cus_expired",
            },
        });
        testUserIds.push(user.id);

        const expiredPlan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: new Date(pastDate.getTime() - 86400000), // Enabled 2 days ago
                expiresAt: pastDate, // Expired yesterday
            },
        });
        testPlanIds.push(expiredPlan.id);

        const email = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                userId: user.id,
                emailAddress: "expired@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(email.id);

        await paymentsExpirePlan();

        // Check that plan was disabled
        const updatedPlan = await DbProvider.get().plan.findUnique({
            where: { id: expiredPlan.id },
        });
        expect(updatedPlan?.enabledAt).toBeNull();
        expect(updatedPlan?.expiresAt).toEqual(pastDate); // Keeps original expiry

        // Check that email was sent
        expect(mockEmailAddTask).toHaveBeenCalledWith(
            expect.objectContaining({
                to: ["expired@example.com"],
            }),
        );
    });

    it("should expire user plans with null expiration that are enabled", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000); // Yesterday
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Null Expiry User",
                handle: "nullexpiryuser",
                stripeCustomerId: "cus_null",
            },
        });
        testUserIds.push(user.id);

        const nullExpiryPlan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate, // Enabled yesterday, should be caught since null expiry
                expiresAt: null, // No expiration set
            },
        });
        testPlanIds.push(nullExpiryPlan.id);

        await paymentsExpirePlan();

        // Check that plan was disabled and given expiry date
        const updatedPlan = await DbProvider.get().plan.findUnique({
            where: { id: nullExpiryPlan.id },
        });
        expect(updatedPlan?.enabledAt).toBeNull();
        expect(updatedPlan?.expiresAt).not.toBeNull();
        expect(updatedPlan?.expiresAt?.getTime()).toBeGreaterThanOrEqual(now.getTime() - 1000);
    });

    it("should expire team plans that are past expiration date", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 172800000); // 2 days ago
        
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "expiredteam",
                stripeCustomerId: "cus_team",
            },
        });
        testTeamIds.push(team.id);

        const teamPlan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                teamId: team.id,
                enabledAt: new Date(pastDate.getTime() - 86400000), // Enabled 3 days ago
                expiresAt: pastDate, // Expired 2 days ago
            },
        });
        testPlanIds.push(teamPlan.id);

        const teamEmail1 = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                teamId: team.id,
                emailAddress: "team1@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(teamEmail1.id);

        const teamEmail2 = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                teamId: team.id,
                emailAddress: "team2@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(teamEmail2.id);

        await paymentsExpirePlan();

        // Check that plan was disabled
        const updatedPlan = await DbProvider.get().plan.findUnique({
            where: { id: teamPlan.id },
        });
        expect(updatedPlan?.enabledAt).toBeNull();

        // Check that emails were sent to all team emails
        expect(mockEmailAddTask).toHaveBeenCalledWith(
            expect.objectContaining({
                to: expect.arrayContaining(["team1@example.com", "team2@example.com"]),
            }),
        );
    });

    it("should not expire plans that are still valid", async () => {
        const now = new Date();
        const futureDate = new Date(now.getTime() + 86400000); // Tomorrow
        const pastDate = new Date(now.getTime() - 86400000); // Yesterday
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Valid User",
                handle: "validuser",
                stripeCustomerId: "cus_valid",
            },
        });
        testUserIds.push(user.id);

        const validPlan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate,
                expiresAt: futureDate, // Expires in the future
            },
        });
        testPlanIds.push(validPlan.id);

        await paymentsExpirePlan();

        // Check that plan was NOT changed
        const unchangedPlan = await DbProvider.get().plan.findUnique({
            where: { id: validPlan.id },
        });
        expect(unchangedPlan?.enabledAt).toEqual(pastDate);
        expect(unchangedPlan?.expiresAt).toEqual(futureDate);

        // Check that no email was sent
        expect(mockEmailAddTask).not.toHaveBeenCalled();
    });

    it("should not expire plans that haven't been enabled yet", async () => {
        const now = new Date();
        const futureDate = new Date(now.getTime() + 86400000); // Tomorrow
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Future User",
                handle: "futureuser",
                stripeCustomerId: "cus_future",
            },
        });
        testUserIds.push(user.id);

        const futurePlan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: futureDate, // Not enabled yet
                expiresAt: null,
            },
        });
        testPlanIds.push(futurePlan.id);

        await paymentsExpirePlan();

        // Check that plan was NOT changed
        const unchangedPlan = await DbProvider.get().plan.findUnique({
            where: { id: futurePlan.id },
        });
        expect(unchangedPlan?.enabledAt).toEqual(futureDate);
        expect(unchangedPlan?.expiresAt).toBeNull();
    });

    it("should handle users and teams without plans", async () => {
        const userWithoutPlan = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "No Plan User",
                handle: "noplanuser",
            },
        });
        testUserIds.push(userWithoutPlan.id);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner2",
            },
        });
        testUserIds.push(owner.id);

        const teamWithoutPlan = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "noplanteam",
            },
        });
        testTeamIds.push(teamWithoutPlan.id);

        // Should not throw
        await expect(paymentsExpirePlan()).resolves.not.toThrow();

        // No emails should be sent
        const { QueueService } = await import("@vrooli/server");
        const mockAddTask = QueueService.get().email.addTask as any;
        expect(mockAddTask).not.toHaveBeenCalled();
    });

    it("should handle multiple expired plans in batch", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000); // Yesterday
        
        // Create multiple users with expired plans
        const userPromises = [];
        for (let i = 0; i < 5; i++) {
            userPromises.push(
                DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        name: `Batch User ${i}`,
                        handle: `batchuser${i}`,
                        stripeCustomerId: `cus_batch${i}`,
                    },
                }),
            );
        }
        const users = await Promise.all(userPromises);
        users.forEach(u => testUserIds.push(u.id));

        // Create plans and emails for each user
        const planPromises = [];
        const emailPromises = [];
        for (let i = 0; i < users.length; i++) {
            planPromises.push(
                DbProvider.get().plan.create({
                    data: {
                        id: generatePK(),
                        userId: users[i].id,
                        enabledAt: new Date(pastDate.getTime() - 86400000), // Enabled 2 days ago
                        expiresAt: i % 2 === 0 ? pastDate : null, // Mix of expired dates and null
                    },
                }),
            );
            emailPromises.push(
                DbProvider.get().email.create({
                    data: {
                        id: generatePK(),
                        userId: users[i].id,
                        emailAddress: `batch${i}@example.com`,
                        verifiedAt: new Date(),
                    },
                }),
            );
        }
        const plans = await Promise.all(planPromises);
        plans.forEach(p => testPlanIds.push(p.id));
        const emails = await Promise.all(emailPromises);
        emails.forEach(e => testEmailIds.push(e.id));

        await paymentsExpirePlan();

        // Check that all plans were disabled
        const updatedPlans = await DbProvider.get().plan.findMany({
            where: { id: { in: plans.map(p => p.id) } },
        });
        updatedPlans.forEach(plan => {
            expect(plan.enabledAt).toBeNull();
        });

        // Check that emails were sent
        expect(mockEmailAddTask).toHaveBeenCalled();
    });

    it("should handle entities without email addresses", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000);
        
        const userNoEmail = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "No Email User",
                handle: "noemailuser",
                stripeCustomerId: "cus_noemail",
            },
        });
        testUserIds.push(userNoEmail.id);

        const planNoEmail = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: userNoEmail.id,
                enabledAt: pastDate,
                expiresAt: pastDate,
            },
        });
        testPlanIds.push(planNoEmail.id);

        await paymentsExpirePlan();

        // Plan should still be disabled
        const updatedPlan = await DbProvider.get().plan.findUnique({
            where: { id: planNoEmail.id },
        });
        expect(updatedPlan?.enabledAt).toBeNull();

        // No email should be sent (empty array)
        expect(mockEmailAddTask).not.toHaveBeenCalled();
    });
});

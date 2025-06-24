// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { PaymentStatus, PaymentType, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mockQueueService, mockEmailAddTask, resetAllMocks } from "../__test/mocks/services.js";

// Mock services before imports
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        QueueService: mockQueueService,
        AUTH_EMAIL_TEMPLATES: {
            PaymentFailed: vi.fn().mockReturnValue({
                subject: "Payment Failed",
                body: "Your payment has failed",
            }),
        },
    };
});

import { paymentsFail } from "./paymentsFail.js";
import { DbProvider } from "@vrooli/server";

describe("paymentsFail integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testPaymentIds: bigint[] = [];
    const testEmailIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testPaymentIds.length = 0;
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
        if (testPaymentIds.length > 0) {
            await db.payment.deleteMany({ where: { id: { in: testPaymentIds } } });
        }
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should mark old pending user payments as failed", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000); // 8 days ago
        
        // Create user
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "User with Failed Payment",
                handle: "failedpaymentuser",
            },
        });
        testUserIds.push(user.id);

        // Create old pending payment
        const oldPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: user.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 9999,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Premium Monthly Subscription",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(oldPayment.id);

        await paymentsFail();

        // Check that payment was marked as failed
        const updatedPayment = await DbProvider.get().payment.findUnique({
            where: { id: oldPayment.id },
        });
        expect(updatedPayment?.status).toBe(PaymentStatus.Failed);
    });

    it("should send failure notification email for non-donation payments", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000); // 8 days ago
        
        // Create user with email
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "User with Email",
                handle: "emailuser",
            },
        });
        testUserIds.push(user.id);

        const email = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                userId: user.id,
                emailAddress: "failed@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(email.id);

        // Create old pending payment
        const oldPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: user.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 9999,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Premium Monthly Subscription",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(oldPayment.id);

        await paymentsFail();

        // Check that email was sent
        expect(mockEmailAddTask).toHaveBeenCalledWith(
            expect.objectContaining({
                to: ["failed@example.com"],
            }),
        );
    });

    it("should mark old pending donation payments as failed", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 10 * 24 * 60 * 60 * 1000); // 10 days ago
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Donor User",
                handle: "donoruser",
            },
        });
        testUserIds.push(user.id);

        // Create old pending donation
        const donationPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: user.id,
                paymentType: PaymentType.Donation,
                amount: 5000,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Donation",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(donationPayment.id);

        await paymentsFail();

        // Check that payment was marked as failed
        const updatedPayment = await DbProvider.get().payment.findUnique({
            where: { id: donationPayment.id },
        });
        expect(updatedPayment?.status).toBe(PaymentStatus.Failed);
    });

    it("should mark old pending team payments as failed", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 9 * 24 * 60 * 60 * 1000); // 9 days ago
        
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
                handle: "failedteam",
            },
        });
        testTeamIds.push(team.id);

        // Create old pending team payment
        const teamPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                teamId: team.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 19999,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Team Premium Subscription",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(teamPayment.id);

        await paymentsFail();

        // Check that payment was marked as failed
        const updatedPayment = await DbProvider.get().payment.findUnique({
            where: { id: teamPayment.id },
        });
        expect(updatedPayment?.status).toBe(PaymentStatus.Failed);
    });

    it("should not fail recent pending payments", async () => {
        const now = new Date();
        const recentDate = new Date(now.getTime() - 3 * 24 * 60 * 60 * 1000); // 3 days ago
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Recent Payment User",
                handle: "recentuser",
            },
        });
        testUserIds.push(user.id);

        // Create recent pending payment
        const recentPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: user.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 9999,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Premium Monthly Subscription",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: recentDate,
                updatedAt: recentDate,
            },
        });
        testPaymentIds.push(recentPayment.id);

        await paymentsFail();

        // Check that payment was NOT changed
        const unchangedPayment = await DbProvider.get().payment.findUnique({
            where: { id: recentPayment.id },
        });
        expect(unchangedPayment?.status).toBe(PaymentStatus.Pending);

        // Check that no email was sent
        expect(mockEmailAddTask).not.toHaveBeenCalled();
    });

    it("should not affect non-pending payments", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000);
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Successful User",
                handle: "successuser",
            },
        });
        testUserIds.push(user.id);

        // Create various non-pending payments
        const successPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: user.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 9999,
                currency: "USD",
                checkoutId: `cs_test_${generatePK()}`,
                description: "Successful Premium Payment",
                status: PaymentStatus.Paid,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(successPayment.id);

        const failedPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: user.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 9999,
                currency: "USD",
                checkoutId: `cs_test_${generatePK()}`,
                description: "Failed Premium Payment",
                status: PaymentStatus.Failed,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(failedPayment.id);

        await paymentsFail();

        // Check that payments were NOT changed
        const unchangedSuccess = await DbProvider.get().payment.findUnique({
            where: { id: successPayment.id },
        });
        expect(unchangedSuccess?.status).toBe(PaymentStatus.Paid);

        const unchangedFailed = await DbProvider.get().payment.findUnique({
            where: { id: failedPayment.id },
        });
        expect(unchangedFailed?.status).toBe(PaymentStatus.Failed);
    });

    it("should handle payments without email addresses", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000);
        
        const userNoEmail = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "No Email User",
                handle: "noemailuser",
            },
        });
        testUserIds.push(userNoEmail.id);

        const paymentNoEmail = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: userNoEmail.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 9999,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Premium Monthly Subscription",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(paymentNoEmail.id);

        await paymentsFail();

        // Payment should still be failed
        const updatedPayment = await DbProvider.get().payment.findUnique({
            where: { id: paymentNoEmail.id },
        });
        expect(updatedPayment?.status).toBe(PaymentStatus.Failed);

        // No email should be sent
        expect(mockEmailAddTask).not.toHaveBeenCalled();
    });

    it("should send only one email per address for multiple failed payments", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000);
        
        // Create user with multiple payments
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Multi Payment User",
                handle: "multipaymentuser",
            },
        });
        testUserIds.push(user.id);

        const email = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                userId: user.id,
                emailAddress: "multi@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(email.id);

        // Create multiple old pending payments of same type
        for (let i = 0; i < 3; i++) {
            const payment = await DbProvider.get().payment.create({
                data: {
                    id: generatePK(),
                    userId: user.id,
                    paymentType: PaymentType.PremiumMonthly,
                    amount: 9999,
                    checkoutId: `cs_test_${generatePK()}`,
                    currency: "USD",
                    description: "Premium Monthly Subscription",
                    status: PaymentStatus.Pending,
                    paymentMethod: "card",
                    createdAt: oldDate,
                    updatedAt: oldDate,
                },
            });
            testPaymentIds.push(payment.id);
        }

        await paymentsFail();

        // Should send only one email to the address
        const emailCalls = mockEmailAddTask.mock.calls.filter((call: any) => 
            call[0].to.includes("multi@example.com"),
        );
        expect(emailCalls.length).toBe(1);
    });

    it("should fail both user and team payments in same batch", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000);
        
        // Create user payment
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Mixed User",
                handle: "mixeduser",
            },
        });
        testUserIds.push(user.id);

        const userPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                userId: user.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 9999,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Premium Monthly Subscription",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(userPayment.id);

        // Create team payment
        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "mixedteamowner",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "mixedteam",
            },
        });
        testTeamIds.push(team.id);

        const teamPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                teamId: team.id,
                paymentType: PaymentType.PremiumMonthly,
                amount: 19999,
                checkoutId: `cs_test_${generatePK()}`,
                currency: "USD",
                description: "Team Premium Subscription",
                status: PaymentStatus.Pending,
                paymentMethod: "card",
                createdAt: oldDate,
                updatedAt: oldDate,
            },
        });
        testPaymentIds.push(teamPayment.id);

        await paymentsFail();

        // Check that both payments were failed
        const updatedUserPayment = await DbProvider.get().payment.findUnique({
            where: { id: userPayment.id },
        });
        expect(updatedUserPayment?.status).toBe(PaymentStatus.Failed);

        const updatedTeamPayment = await DbProvider.get().payment.findUnique({
            where: { id: teamPayment.id },
        });
        expect(updatedTeamPayment?.status).toBe(PaymentStatus.Failed);
    });
});

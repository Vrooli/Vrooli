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

    it("should fail old pending payments for users", async () => {
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

        // Check that payment was marked as failed
        const updatedPayment = await DbProvider.get().payment.findUnique({
            where: { id: oldPayment.id },
        });
        expect(updatedPayment?.status).toBe(PaymentStatus.Failed);

        // Check that email was sent (non-donation)
        const { AUTH_EMAIL_TEMPLATES } = await import("@vrooli/server");
        const mockTemplate = AUTH_EMAIL_TEMPLATES.PaymentFailed as any;
        
        expect(mockEmailAddTask).toHaveBeenCalledWith(
            expect.objectContaining({
                to: ["failed@example.com"],
            }),
        );
        expect(mockTemplate).toHaveBeenCalledWith(expect.any(String), false); // false for non-donation
    });

    it("should fail old pending donation payments with proper email", async () => {
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

        const email = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                userId: user.id,
                emailAddress: "donor@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(email.id);

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

        // Check that email was sent (donation)
        const { AUTH_EMAIL_TEMPLATES } = await import("@vrooli/server");
        const mockTemplate = AUTH_EMAIL_TEMPLATES.PaymentFailed as any;
        
        expect(mockEmailAddTask).toHaveBeenCalledWith(
            expect.objectContaining({
                to: ["donor@example.com"],
            }),
        );
        expect(mockTemplate).toHaveBeenCalledWith(expect.any(String), true); // true for donation
    });

    it("should fail old pending team payments", async () => {
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

        // Check that emails were sent to all team emails
        expect(mockEmailAddTask).toHaveBeenCalledWith(
            expect.objectContaining({
                to: expect.arrayContaining(["team1@example.com", "team2@example.com"]),
            }),
        );
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

    it("should handle multiple payments in batch with unique email addresses", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000);
        
        // Create user with multiple payments (should only get one email)
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

        // Create multiple old pending payments
        const paymentPromises = [];
        for (let i = 0; i < 3; i++) {
            paymentPromises.push(
                DbProvider.get().payment.create({
                    data: {
                        id: generatePK(),
                        userId: user.id,
                        paymentType: i === 0 ? PaymentType.Donation : PaymentType.PremiumMonthly,
                        amount: (50 + i * 10) * 100,
                        checkoutId: `cs_test_${generatePK()}`,
                        currency: "USD",
                        description: i === 0 ? "Donation" : "Premium Monthly Subscription",
                        status: PaymentStatus.Pending,
                        paymentMethod: "card",
                        createdAt: oldDate,
                        updatedAt: oldDate,
                    },
                }),
            );
        }
        const payments = await Promise.all(paymentPromises);
        payments.forEach(p => testPaymentIds.push(p.id));

        await paymentsFail();

        // Check that all payments were failed
        const updatedPayments = await DbProvider.get().payment.findMany({
            where: { id: { in: payments.map(p => p.id) } },
        });
        updatedPayments.forEach(payment => {
            expect(payment.status).toBe(PaymentStatus.Failed);
        });

        // Check that emails were sent with unique addresses
        
        // Should have two calls - one for donation, one for other
        expect(mockEmailAddTask).toHaveBeenCalledTimes(2);
        
        // Each call should have unique email addresses
        mockEmailAddTask.mock.calls.forEach((call) => {
            const to = call[0].to;
            expect(new Set(to).size).toBe(to.length); // No duplicates
        });
    });

    it("should handle mixed user and team payments", async () => {
        const now = new Date();
        const oldDate = new Date(now.getTime() - 8 * 24 * 60 * 60 * 1000);
        
        // Create user
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Mixed User",
                handle: "mixeduser",
            },
        });
        testUserIds.push(user.id);

        const userEmail = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                userId: user.id,
                emailAddress: "user@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(userEmail.id);

        // Create team
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

        const teamEmail = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                teamId: team.id,
                emailAddress: "team@example.com",
                verifiedAt: new Date(),
            },
        });
        testEmailIds.push(teamEmail.id);

        // Create payments
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

        const teamPayment = await DbProvider.get().payment.create({
            data: {
                id: generatePK(),
                teamId: team.id,
                paymentType: PaymentType.Donation,
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

        // Check that emails were sent
        
        // Should have two calls - one for donation, one for premium
        expect(mockEmailAddTask).toHaveBeenCalledTimes(2);
        
        // Verify both emails are included
        const allEmails = mockEmailAddTask.mock.calls.flatMap((call: any) => call[0].to);
        expect(allEmails).toContain("user@example.com");
        expect(allEmails).toContain("team@example.com");
    });
});

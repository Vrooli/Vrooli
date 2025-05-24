import { AUTH_EMAIL_TEMPLATES, DbProvider, QueueService, batch, logger } from "@local/server";
import { PaymentStatus, PaymentType, WEEKS_1_MS, nanoid } from "@local/shared";
import { type Prisma } from "@prisma/client";

// Select shape for pending payments
const paymentSelect = {
    id: true,
    paymentType: true,
    user: { select: { id: true, emails: { select: { emailAddress: true } } } },
    team: { select: { id: true, emails: { select: { emailAddress: true } } } },
} as const;

// Payload type for pending payments
type PaymentPayload = Prisma.paymentGetPayload<{ select: typeof paymentSelect }>;

const PENDING_TIMEOUT = WEEKS_1_MS;

/**
 * Updates pending payments to failed if they have been stuck in pending for a long time
 */
export async function paymentsFail(): Promise<void> {
    try {
        await batch<Prisma.paymentFindManyArgs, PaymentPayload>({
            objectType: "Payment",
            processBatch: async (paymentBatch) => {
                const paymentIds = paymentBatch.map(payment => payment.id);
                if (paymentIds.length > 0) {
                    await DbProvider.get().payment.updateMany({
                        data: { status: PaymentStatus.Failed },
                        where: { id: { in: paymentIds } },
                    });
                }

                // Send notifications - batched by donation status
                const donationFailedEmailRecipients: string[] = [];
                const otherFailedEmailRecipients: string[] = [];

                for (const payment of paymentBatch) {
                    const recipientEmails = (payment.user?.emails ?? payment.team?.emails ?? [])
                        .map(email => email.emailAddress)
                        .filter(Boolean); // Ensure no null/undefined email strings

                    if (recipientEmails.length === 0) {
                        continue;
                    }

                    // Assuming payment.paymentType from Prisma is a string whose values
                    // are members of the shared PaymentType enum.
                    const sharedPaymentType = payment.paymentType as PaymentType;

                    if (sharedPaymentType === PaymentType.Donation) {
                        donationFailedEmailRecipients.push(...recipientEmails);
                    } else {
                        otherFailedEmailRecipients.push(...recipientEmails);
                    }
                }

                if (donationFailedEmailRecipients.length > 0) {
                    await QueueService.get().email.addTask({
                        to: [...new Set(donationFailedEmailRecipients)], // Ensure unique email addresses
                        ...AUTH_EMAIL_TEMPLATES.PaymentFailed(nanoid(), true), // true for isDonation
                    });
                }

                if (otherFailedEmailRecipients.length > 0) {
                    await QueueService.get().email.addTask({
                        to: [...new Set(otherFailedEmailRecipients)], // Ensure unique email addresses
                        ...AUTH_EMAIL_TEMPLATES.PaymentFailed(nanoid(), false), // false for isDonation
                    });
                }
            },
            select: paymentSelect,
            where: {
                status: PaymentStatus.Pending,
                updatedAt: { lte: new Date(Date.now() - PENDING_TIMEOUT) },
            },
        });
    } catch (error) {
        logger.error("paymentsFail caught error", { error, trace: "0222" });
    }
}

import { type Prisma } from "@prisma/client";
import { AUTH_EMAIL_TEMPLATES, DbProvider, QueueService, batch, logger } from "@vrooli/server";
import { PaymentStatus, PaymentType, WEEKS_1_MS, nanoid } from "@vrooli/shared";

/**
 * Type guard to check if a value is a valid PaymentType
 */
function isValidPaymentType(value: string): value is PaymentType {
    const paymentTypeValues: readonly string[] = Object.values(PaymentType);
    return paymentTypeValues.includes(value);
}

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

                    // Validate that payment.paymentType is a valid PaymentType
                    if (!isValidPaymentType(payment.paymentType)) {
                        logger.error("Invalid payment type encountered", {
                            paymentId: payment.id,
                            paymentType: payment.paymentType,
                        });
                        continue; // Skip this payment
                    }
                    
                    // payment.paymentType is now safely validated as PaymentType
                    if (payment.paymentType === PaymentType.Donation) {
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

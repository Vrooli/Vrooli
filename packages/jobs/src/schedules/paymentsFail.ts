import { DbProvider, batch, logger, sendPaymentFailed } from "@local/server";
import { PaymentStatus, type PaymentType, WEEKS_1_MS } from "@local/shared";
import { type Prisma } from "@prisma/client";

// Select shape for pending payments
const paymentSelect = {
    id: true,
    paymentType: true,
    team: { select: { emails: { select: { emailAddress: true } } } },
    user: { select: { emails: { select: { emailAddress: true } } } },
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
            processBatch: async (batch) => {
                // Set payments to failed
                const paymentIds = batch.map(payment => payment.id);
                await DbProvider.get().payment.updateMany({
                    data: { status: PaymentStatus.Failed },
                    where: { id: { in: paymentIds } },
                });
                // Send notifications
                const notifyData: { email: string, paymentType: PaymentType }[] = batch.map(payment => {
                    const emails = payment.team?.emails ?? payment.user?.emails ?? [];
                    return emails.map(email => ({ email: email.emailAddress, paymentType: payment.paymentType }));
                }).flat();
                for (const { email, paymentType } of notifyData) {
                    sendPaymentFailed(email, paymentType);
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

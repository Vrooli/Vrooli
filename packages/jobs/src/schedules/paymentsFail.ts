import { batch, logger, prismaInstance, sendPaymentFailed } from "@local/server";
import { PaymentStatus, PaymentType, WEEKS_1_MS } from "@local/shared";
import { Prisma } from "@prisma/client";

const PENDING_TIMEOUT = WEEKS_1_MS;

/**
 * Updates pending payments to failed if they have been stuck in pending for a long time
 */
export async function paymentsFail() {
    try {
        await batch<Prisma.paymentFindManyArgs>({
            objectType: "Payment",
            processBatch: async (batch) => {
                // Set payments to failed
                const paymentIds = batch.map(payment => payment.id);
                await prismaInstance.payment.updateMany({
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
            select: {
                id: true,
                paymentType: true,
                team: {
                    select: {
                        emails: { select: { emailAddress: true } },
                    },
                },
                user: {
                    select: {
                        emails: { select: { emailAddress: true } },
                    },
                },
            },
            where: {
                status: PaymentStatus.Pending,
                updated_at: { lte: new Date(Date.now() - PENDING_TIMEOUT) },
            },
        });
    } catch (error) {
        logger.error("paymentsFail caught error", { error, trace: "0222" });
    }
};

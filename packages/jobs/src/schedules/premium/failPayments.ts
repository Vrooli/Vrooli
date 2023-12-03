import { batch, sendPaymentFailed } from "@local/server";
import { PaymentStatus, PaymentType, Prisma } from "@prisma/client";

const PENDING_TIMEOUT = 7 * 24 * 60 * 60 * 1000; // 7 days

/**
 * Updates pending payments to failed if they have been stuck in pending for a long time
 */
export const failPayments = async () => {
    await batch<Prisma.paymentFindManyArgs>({
        objectType: "Payment",
        processBatch: async (batch, prisma) => {
            // Set payments to failed
            const paymentIds = batch.map(payment => payment.id);
            await prisma.payment.updateMany({
                data: { status: PaymentStatus.Failed },
                where: { id: { in: paymentIds } },
            });
            // Send notifications
            const notifyData: { email: string, paymentType: PaymentType }[] = batch.map(payment => {
                const emails = payment.organization?.emails ?? payment.user?.emails ?? [];
                return emails.map(email => ({ email: email.emailAddress, paymentType: payment.paymentType }));
            }).flat();
            for (const { email, paymentType } of notifyData) {
                sendPaymentFailed(email, paymentType);
            }
        },
        select: {
            id: true,
            paymentType: true,
            organization: {
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
        trace: "0222",
        where: {
            status: PaymentStatus.Pending,
            updated_at: { lte: new Date(Date.now() - PENDING_TIMEOUT) },
        },
    });
};

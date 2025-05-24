import { batch, BusService, logger, Notify, SocketService, type BillingEvent } from "@local/server";
import { API_CREDITS_PREMIUM } from "@local/shared";
import { CreditEntryType, CreditSourceSystem, type Prisma } from "@prisma/client";

const MAX_MONTHS_ACCRUED = 6;
/**
 * The max number of free credits to give to premium users.
 */
const MAX_FREE_CREDITS = BigInt(MAX_MONTHS_ACCRUED) * API_CREDITS_PREMIUM;

/**
 * Provides free credits to premium users by publishing BillingEvents.
 */
export async function paymentsCreditsFreePremium(): Promise<void> {
    const freeCreditsSelect = {
        id: true,
        languages: true,
        plan: {
            select: {
                id: true,
                enabledAt: true,
                expiresAt: true,
            },
        },
        creditAccount: {
            select: {
                id: true,
                currentBalance: true,
            },
        },
    } as const;
    type FreeCreditsPayload = Prisma.userGetPayload<{ select: typeof freeCreditsSelect }>;

    try {
        await batch<Prisma.userFindManyArgs, FreeCreditsPayload>({
            objectType: "User",
            processBatch: async (batch) => {
                for (const user of batch) {
                    if (!user.plan || !user.creditAccount) {
                        continue;
                    }

                    const isPremiumActive = user.plan.enabledAt &&
                        (!user.plan.expiresAt || user.plan.expiresAt > new Date());

                    if (!isPremiumActive) {
                        continue;
                    }

                    const currentCredits = user.creditAccount.currentBalance ?? BigInt(0);
                    let creditsToAdd = BigInt(0);
                    // finalCredits is now theoretical, actual balance managed by billing.ts
                    let theoreticalFinalCredits: bigint = currentCredits;

                    if (currentCredits < MAX_FREE_CREDITS) {
                        if (currentCredits < (MAX_FREE_CREDITS - API_CREDITS_PREMIUM)) {
                            creditsToAdd = API_CREDITS_PREMIUM;
                        } else {
                            creditsToAdd = MAX_FREE_CREDITS - currentCredits;
                        }
                        theoreticalFinalCredits = currentCredits + creditsToAdd;
                    }

                    if (creditsToAdd > BigInt(0)) {
                        const billingEventId = `freePremCredits-${user.id.toString()}-${Date.now()}`;
                        const billingEvent: BillingEvent = {
                            type: "billing:event",
                            id: billingEventId, // This will be the idempotencyKey for the ledger entry
                            accountId: user.creditAccount.id.toString(),
                            delta: creditsToAdd.toString(),
                            entryType: CreditEntryType.Bonus,
                            source: CreditSourceSystem.Scheduler,
                            meta: {
                                reason: "Monthly premium free credits",
                                jobName: "paymentsCreditsFreePremium",
                                originalUserId: user.id.toString(), // For tracing back to the user
                            },
                        };

                        try {
                            await BusService.get().getBus().publish(billingEvent);
                            logger.info(`Successfully published BillingEvent ${billingEventId} for user ${user.id} to award ${creditsToAdd} credits. New theoretical balance: ${theoreticalFinalCredits}.`);

                            // Notify user & emit socket event only after successful publish
                            if (API_CREDITS_PREMIUM > currentCredits) { // Notify if they were low and we added some
                                Notify(user.languages).pushFreeCreditsReceived().toUser(user.id.toString());
                            }
                            // Emit the theoretical new balance. UI might take a moment to reflect actual balance updated by billing worker.
                            SocketService.get().emitSocketEvent("apiCredits", user.id.toString(), { credits: theoreticalFinalCredits.toString() });

                        } catch (publishError) {
                            logger.error(`Failed to publish BillingEvent for user ${user.id}`, { error: publishError, eventId: billingEventId, userId: user.id, creditsToAdd, trace: "paymentsCreditsFreePremium_publishError" });
                            // Optionally, do not send notification or socket event if publish fails
                        }
                    } else {
                        // No credits to add, but still emit socket event with current (unchanged) balance for UI consistency if user is online
                        SocketService.get().emitSocketEvent("apiCredits", user.id.toString(), { credits: currentCredits.toString() });
                    }
                }
            },
            select: freeCreditsSelect,
            where: {
                isBot: false,
            },
        });
    } catch (error) {
        logger.error("Error in paymentsCreditsFreePremium job", { error, trace: "paymentsCreditsFreePremium_jobError" });
    }
}

// billing/handleBillingEvent.ts
import { generatePK } from "@local/shared";
import { CreditEntryType, Prisma } from "@prisma/client";
import { DbProvider } from "../db/provider.js";
import { type BillingEvent, BusService, BusWorker, type EventBus } from "./bus.js";

/**
 * Handles incoming billing events to update credit ledger entries and account balances.
 *
 * This function performs the following operations within a serializable database transaction:
 * 1. Validates the `accountId` from the message.
 * 2. Creates a new `credit_ledger_entry` record, using the event's `id` as an idempotency key.
 *    If an entry with the same idempotency key already exists (Prisma error P2002), it's treated as a successful duplicate and the function returns.
 * 3. Retrieves the corresponding `credit_account` and locks the row for update.
 * 4. Applies the `delta` from the billing event to the account's `currentBalance`.
 *    The balance is allowed to go negative.
 * 5. Updates the `credit_account` with the new balance.
 *
 * After the transaction commits:
 * - If the `finalBalance` of the account is negative, a `billing.negative_balance` event is published to the event bus.
 *
 * Error Handling:
 * - Invalid `accountId` format: Logs an error and returns, effectively acknowledging the message.
 * - Duplicate ledger entry (P2002): Returns silently, treating it as a success (idempotency).
 * - Other unknown errors: Propagated to the event bus, leaving the message pending for potential retries or dead-lettering.
 *
 * @param event The `BillingEvent` message to process.
 * @returns A Promise that resolves when processing is complete or an error is thrown.
 */
export async function handleBillingEvent(event: BillingEvent): Promise<void> {
    try {
        let accountIdAsBigInt: bigint;
        try {
            accountIdAsBigInt = BigInt(event.accountId);
        } catch (error) {
            if (error instanceof TypeError || error instanceof SyntaxError) {
                // Malformed accountId, non-retryable. Log and acknowledge the message.
                console.error(`Invalid accountId format: ${event.accountId}. Error: ${error}. Event ID: ${event.id}. This event will be acknowledged and not retried.`);
                return; // Acknowledge the message
            }
            // Log the error, and potentially send to a dead-letter queue or specific error event
            console.error(`Unexpected error parsing accountId: ${event.accountId}. Error: ${error}. Event ID: ${event.id}.`);
            // For other unexpected parsing errors, propagate to let the event bus handle it.
            throw error;
        }

        let deltaAsBigInt: bigint;
        try {
            // event.delta is typed as 'string' in the BillingEvent interface (for JSON safety)
            // and needs to be converted to a BigInt for calculations.
            // This block robustly converts event.delta to a BigInt, handling potential runtime type mismatches
            // (e.g. if event.delta is a non-numeric string, null, or undefined).
            // It will throw TypeError for null/undefined, or SyntaxError for unparseable strings.
            deltaAsBigInt = BigInt(event.delta);
        } catch (error) {
            if (error instanceof TypeError || error instanceof SyntaxError) {
                // Malformed delta, non-retryable. Log and acknowledge the message.
                console.error(`Invalid delta format: ${event.delta}. Error: ${error}. Event ID: ${event.id}. This event will be acknowledged and not retried.`);
                return; // Acknowledge the message
            }
            // Log the error, and potentially send to a dead-letter queue or specific error event
            console.error(`Unexpected error parsing delta: ${event.delta}. Error: ${error}. Event ID: ${event.id}.`);
            // For other unexpected parsing errors, propagate to let the event bus handle it.
            throw error;
        }

        // Ensure 'spend' types always result in a negative delta
        const outgoingEntryTypes: CreditEntryType[] = [
            CreditEntryType.Spend,
            CreditEntryType.TransferOut,
            CreditEntryType.DonationGiven,
            CreditEntryType.Expire,
            CreditEntryType.Chargeback,
            CreditEntryType.AdjustmentDecrease,
            CreditEntryType.Penalty,
        ];

        if (outgoingEntryTypes.includes(event.entryType) && deltaAsBigInt > BigInt(0)) {
            deltaAsBigInt = -deltaAsBigInt;
        }

        let finalBalance: bigint | null = null;

        await DbProvider.get().$transaction(async (tx) => {
            // Create the ledger entry first to throw DUP if it already exists

            // Sanitize meta to ensure it's valid JSON.
            // Prisma expects Prisma.InputJsonValue, which should be serializable.
            // event.meta is Record<string, unknown>. JSON.stringify will throw on BigInts by default
            // or on circular references. If event.meta comes from a well-behaved JSON source,
            // this should pass. If not, it's better to catch it here than in Prisma client.
            let sanitizedMeta: Prisma.InputJsonValue;
            try {
                // Note: JSON.stringify will throw for BigInts. If BigInts are expected in meta,
                // they must be pre-stringified by the event producer or handled with a replacer.
                // This also strips functions and undefined properties, and converts Dates to ISO strings.
                // Using a replacer to handle BigInts by converting them to strings.
                sanitizedMeta = JSON.parse(JSON.stringify(event.meta, (key, value) => {
                    if (typeof value === "bigint") {
                        return value.toString();
                    }
                    return value;
                }));
            } catch (stringifyError) {
                console.error(`Failed to serialize event.meta for event ID ${event.id}. Meta: ${JSON.stringify(event.meta, null, 2)}. Error: ${stringifyError}. Using empty object for meta.`);
                sanitizedMeta = {}; // Fallback to an empty object or handle as a more critical error
            }

            await tx.credit_ledger_entry.create({
                data: {
                    id: generatePK(),
                    idempotencyKey: event.id,
                    accountId: accountIdAsBigInt,
                    amount: deltaAsBigInt,
                    type: event.entryType,
                    source: event.source,
                    meta: sanitizedMeta, // Use the sanitized version
                },
            });
            // Lock the row
            const acct = await tx.credit_account.findUniqueOrThrow({
                where: { id: accountIdAsBigInt },
                select: { id: true, currentBalance: true },
            });
            await tx.$executeRaw`SELECT 1 FROM credit_account WHERE id = ${acct.id} FOR UPDATE`;

            /* 3 ─ apply delta */
            const newBal = acct.currentBalance + deltaAsBigInt;

            await tx.credit_account.update({
                where: { id: acct.id },
                data: { currentBalance: newBal },
            });
            finalBalance = newBal; // Store final balance to check after transaction
        }, { isolationLevel: "Serializable" });

        // After transaction, if balance is negative, publish an event
        if (finalBalance !== null && finalBalance < BigInt(0)) {
            const checkedFinalBalance: bigint = finalBalance;
            await BusService.get().getBus().publish({
                type: "billing.negative_balance",
                payload: {
                    accountId: event.accountId,
                    currentBalance: checkedFinalBalance.toString(),
                },
            });
        }

    } catch (error) {
        /* ---------- idempotency: duplicate key ---------- */
        if (error instanceof Prisma.PrismaClientKnownRequestError && error.code === "P2002") {
            return;                                // already booked – treat as success
        }

        /* ---------- unknown error → propagate so bus leaves msg pending ---------- */
        throw error;
    }
}

/**
 * A BusWorker responsible for subscribing to and processing billing-related events.
 *
 * On initialization (`init`), this worker subscribes to the event bus.
 * When a `billing:event` is received, it delegates the processing to the `handleBillingEvent` function.
 *
 * This worker does not require a custom `shutdown` method as Prisma client connections
 * are typically managed globally or by the `DbProvider`.
 */
export class BillingWorker extends BusWorker {
    /**
     * Initializes the BillingWorker by subscribing to relevant events on the event bus.
     *
     * Specifically, it subscribes to events and filters for those of type `billing:event`,
     * passing them to `handleBillingEvent` for processing.
     *
     * @param bus The `EventBus` instance to subscribe to.
     */
    protected static async init(bus: EventBus) {
        bus.subscribe(async (evt) => {
            if (evt.type !== "billing:event") return;
            await handleBillingEvent(evt);
        });
    }

    // no shutdown needed – Prisma handled globally
}

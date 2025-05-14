// billing/handleBillingEvent.ts
import { generatePK } from "@local/shared";
import { CreditEntryType, CreditSourceSystem, Prisma } from "@prisma/client";
import { DbProvider } from "../db/provider.js";
import { BusService, BusWorker, EventBus } from "./bus.js";

export interface BillingEvent {
    /** idempotency key (UUID v4 from emitter) */
    id: string;
    /** credit_account PK (user *or* team), NOT the teamId or userId */
    accountId: string;          // bigint → string for JSON safety
    /** signed amount.  + = top-up, – = spend */
    delta: bigint;
    /** typed reason */
    entryType: CreditEntryType;
    /** system that emitted it (Stripe, Scheduler, LLM, …) */
    source: CreditSourceSystem;
    /** any additional data you want to inspect later */
    meta: Record<string, unknown>;
}

export async function handleBillingEvent(msg: BillingEvent): Promise<void> {
    try {
        await DbProvider.get().$transaction(async (tx) => {
            // Create the ledger entry first to throw DUP if it already exists
            await tx.credit_ledger_entry.create({
                data: {
                    id: generatePK(),
                    idempotencyKey: msg.id,
                    accountId: BigInt(msg.accountId),
                    amount: msg.delta,
                    type: msg.entryType,
                    source: msg.source,
                    meta: msg.meta as Prisma.InputJsonValue,
                },
            });
            // Lock the row
            const acct = await tx.credit_account.findUniqueOrThrow({
                where: { id: BigInt(msg.accountId) },
                select: { id: true, currentBalance: true },
            });
            await tx.$executeRaw`SELECT 1 FROM credit_account WHERE id = ${acct.id} FOR UPDATE`;

            /* 3 ─ apply delta + overdraft check */
            const newBal = acct.currentBalance + msg.delta;
            if (newBal < BigInt(0)) {
                throw Object.assign(new Error("INSUFFICIENT_CREDITS"), { code: "INSUFFICIENT_CREDITS" });
            }

            await tx.credit_account.update({
                where: { id: acct.id },
                data: { currentBalance: newBal },
            });
        }, { isolationLevel: "Serializable" });

    } catch (err: any) {
        /* ---------- idempotency: duplicate key ---------- */
        if (err.code === "P2002") {
            return;                                // already booked – treat as success
        }

        /* ---------- not enough credits ---------- */
        if (err.code === "INSUFFICIENT_CREDITS") {
            await BusService.get().getBus().publish({
                type: "billing.insufficient",
                payload: { accountId: msg.accountId },
            } as any);
            return;                                // don't retry the same debit
        }

        /* ---------- unknown error → propagate so bus leaves msg pending ---------- */
        throw err;
    }
}

export class BillingWorker extends BusWorker {
    protected static async init(bus: EventBus) {
        bus.subscribe(async (evt) => {
            if (evt.type !== "billing") return;
            await handleBillingEvent(evt.payload as BillingEvent);   // let errors bubble
        });
    }

    // no shutdown needed – Prisma handled globally
}

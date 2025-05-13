// import { PrismaClient, CreditEntryType, CreditSourceSystem } from "@prisma/client";
// import { billingBus } from "./eventBus";        // same Redis/NATS instance
// const prisma = new PrismaClient({ log: ["error"] });

// billingBus.subscribe<BillingEvt>("billing", async (msg) => {
//   try {
//     await prisma.$transaction(async (tx) => {
//       /* 1 ── lock the account row */
//       const acct = await tx.credit_account.findUniqueOrThrow({
//         where: { id: BigInt(msg.accountId) },
//         select: { id: true, currentBalance: true },
//       });
//       // FOR UPDATE lock
//       await tx.$executeRaw`SELECT 1 FROM credit_account WHERE id = ${acct.id} FOR UPDATE`;

//       /* 2 ── attempt ledger insert (idempotent) */
//       const entry = await tx.credit_ledger_entry.create({
//         data: {
//           idempotencyKey : msg.id,                     // UNIQUE
//           accountId      : acct.id,
//           amount         : msg.delta,
//           type           : msg.entryType,
//           sourceSystem   : msg.sourceSystem,
//           meta           : msg.meta as any,
//         },
//       });

//       /* 3 ── apply delta – optional overdraft check */
//       const newBal = acct.currentBalance + msg.delta;
//       if (newBal < 0n) {
//         // OPTIONAL: reject, allow overdraft, or clamp to 0
//         throw new Error("INSUFFICIENT_CREDITS");
//       }

//       await tx.credit_account.update({
//         where: { id: acct.id },
//         data : {
//           currentBalance : newBal,
//           lastEntryId    : entry.id,
//         },
//       });
//     },
//     { isolationLevel: "Serializable" }     // ← safest; prevents phantom entry races
//     );

//     billingBus.ack(msg);                   // ACK queue message

//   } catch (err: any) {
//     if (err.code === "P2002") {            // Prisma unique constraint → duplicate
//       billingBus.ack(msg);                 // already processed – ACK and drop
//     } else if (err.message === "INSUFFICIENT_CREDITS") {
//       // NACK or publish "credit.insufficient" so ConversationLoop can halt
//       billingBus.nack(msg, { requeue: false });
//       publishInsufficientCredits(msg.accountId);
//     } else {
//       console.error("billing-worker error", err);
//       billingBus.nack(msg);                // retry later
//     }
//   }
// });
export { };


import { EventEmitter } from "node:events";
import { createClient, RedisClientType } from "redis";

import { getRedisUrl } from "../../redisConn.js";
import {
    ConversationEvent,
    EventCallback,
} from "./types.js";


/* ------------------------------------------------------------------
 * eventBus.ts  –  Unified publish/subscribe spine for Vrooli
 * ------------------------------------------------------------------
 * Why this exists
 * -------------------
 *  • ConversationLoop and other micro‑services interact *only* through the
 *    EventBus interface; they never call one another directly.
 *  • You can swap message back‑ends (in‑memory  → Redis  → JetStream)
 *    without touching domain code – just bind the desired concrete class.
 *
 *  **SINGLETON RULE**
 *  ------------------
 *  Create **exactly one EventBus instance per Node.js process / Kubernetes
 *  pod** and share it across *all* ConversationLoop objects in that process.
 *  Each EventBus implementation maintains its own network sockets
 *  (Redis: 2, NATS: 1–2). Spawning a bus per conversation would exhaust
 *  file‑descriptors and overwhelm your broker.
 *
 * Interface contract (recap)
 * -------------------------
 *  subscribe(cb) : fire callback for every new event **in order per channel**
 *  publish(evt)  : fire‑and‑forget, non‑blocking.
 *  close()       : flush & free resources on shutdown.
 *
 * Event shape (see ./types.ts)
 * ---------------------------
 *  {
 *    type:            "message.created" | "tool.result" | ...,
 *    conversationId:  "conv_…",
 *    turnId:          42,
 *    ...other fields
 *  }
 *
 * Implementations
 * ---------------
 *  1. InMemoryEventBus   – single‑process dev & unit tests. Zero deps.
 *  2. RedisPubSubBus     – same semantics as Node's EventEmitter but works
 *                          across processes / pods. At‑most‑once delivery.
 *  3. NatsJetStreamBus   – durable, at‑least‑once delivery with replay.
 *                          Light ops footprint (<30 MB binary).
 *                          Requires `nats-server -js`.
 *
 * ------------------------------------------------------------------ */
export abstract class EventBus {
    /**
     * Subscribe to *all* events.  Called on startup by ConversationLoop
     * and any other interested service.
     */
    abstract subscribe(cb: EventCallback): void;

    /** Publish a single event.  The call must resolve when the message is
     *  *accepted* by the back‑end (not necessarily delivered to all consumers).
     */
    abstract publish(evt: ConversationEvent): Promise<void>;

    /** Flush buffers and close connections.  Idempotent. */
    abstract close(): Promise<void>;
}

/* ------------------------------------------------------------------
 * 1) In‑memory, single‑process implementation ---------------------- */

export class InMemoryEventBus extends EventBus {
    private readonly ee = new EventEmitter();

    /** Register a listener. Duplicate registrations are allowed. */
    subscribe(cb: EventCallback) {
        this.ee.on("evt", cb as any);
    }

    async publish(evt: ConversationEvent) {
        this.ee.emit("evt", evt);
    }

    async close() {
        this.ee.removeAllListeners();
    }
}

/* ------------------------------------------------------------------
 * 2) Redis Pub/Sub (at‑most‑once) implementation ------------------- */

/**
 * Channel name used for *all* conversation events.  You can prefix this with
 * an environment ("dev:") or shard key if needed.
 */
const REDIS_CHANNEL = "vrooli.events";

export class RedisPubSubBus extends EventBus {
    private pub!: RedisClientType;   // dedicated publisher
    private sub!: RedisClientType;   // dedicated subscriber
    private ready = false;

    private async ensureConn() {
        if (this.ready) return;

        this.pub = createClient({ url: getRedisUrl() });
        this.sub = createClient({ url: getRedisUrl() });

        await Promise.all([this.pub.connect(), this.sub.connect()]);
        this.ready = true;
    }

    async publish(evt: ConversationEvent) {
        await this.ensureConn();
        await this.pub.publish(REDIS_CHANNEL, JSON.stringify(evt));
    }

    subscribe(cb: EventCallback) {
        (async () => {
            await this.ensureConn();
            await this.sub.subscribe(REDIS_CHANNEL, (raw) => {
                try { cb(JSON.parse(raw)); } catch (e) { /* log */ }
            });
        })().catch(console.error);
    }

    async close() {
        if (!this.ready) return;
        await Promise.allSettled([this.pub.disconnect(), this.sub.disconnect()]);
        this.ready = false;
    }
}

// /* ------------------------------------------------------------------
//  * 3) NATS JetStream (durable) implementation ----------------------- */

// /**
//  * NATS subject where we publish all events.  Subjects support wildcards so you
//  * could later evolve to "evt.*.message" etc. without breaking consumers.
//  */
// const NATS_SUBJECT = "vrooli.events";

// /** Stream + durable consumer names (convention). */
// const STREAM_NAME   = "VROOLI_EVT";
// const CONSUMER_NAME = "loop_worker"; // each worker uses a unique durable name

// export class NatsJetStreamBus extends EventBus {
//   private nc!: NatsConnection;
//   private js!: ReturnType<NatsConnection["jetstream"]>;
//   private jsConsumerClosed = false;
//   private readonly jc = JSONCodec<ConversationEvent>();

//   constructor(private readonly url = process.env.NATS_URL ?? "nats://127.0.0.1:4222") {
//     super();
//   }

//   /** Lazily connects & ensures stream exists. */
//   private async ensureConn() {
//     if (this.nc) return;
//     this.nc = await natsConnect({ servers: this.url });
//     this.js = this.nc.jetstream();
//     // Idempotent: add stream if missing
//     await this.js.addStream({ name: STREAM_NAME, subjects: [NATS_SUBJECT] }).catch(() => {});
//   }

//   async publish(evt: ConversationEvent) {
//     await this.ensureConn();
//     await this.js.publish(NATS_SUBJECT, this.jc.encode(evt));
//   }

//   /** Pull‑based durable consumer so we can back‑pressure ConversationLoop. */
//   subscribe(cb: EventCallback) {
//     (async () => {
//       await this.ensureConn();
//       const jsm = await this.nc.jetstreamManager();
//       // Ensure consumer
//       await jsm.consumers.add(STREAM_NAME, {
//         durable_name: CONSUMER_NAME,
//         ack_policy: "explicit",
//         deliver_policy: "all",
//       }).catch(() => {});

//       const consumer = this.js.pullSubscribe(NATS_SUBJECT, { durable: CONSUMER_NAME });

//       // Background pull loop
//       (async () => {
//         for await (const msgs of consumer) {
//           msgs.forEach((m: JsMsg) => {
//             try {
//               cb(this.jc.decode(m.data));
//               m.ack();
//             } catch (e) {
//               console.error("NATS handler error", e);
//               m.term(); // poison pill – skips msg, record for DLQ via JS config
//             }
//           });
//         }
//       })().catch(console.error);
//     })().catch(console.error);
//   }

//   async close() {
//     if (this.nc && !this.jsConsumerClosed) {
//       await this.nc.close();
//       this.jsConsumerClosed = true;
//     }
//   }
// }

/* ------------------------------------------------------------------
 * Usage hint (dependency‑injection)
 * ------------------------------------------------------------------
 *   const bus = process.env.NODE_ENV === "test"
 *                ? new InMemoryEventBus()
 *                : new NatsJetStreamBus();
 *   const loop = new ConversationLoop(bus, …);
 * ------------------------------------------------------------------ */

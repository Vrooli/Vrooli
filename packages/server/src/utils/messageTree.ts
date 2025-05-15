import { DbProvider } from "../db/provider.js";

/**
 * Chat-specific information that must be collected before shaping inputs 
 * for insertion into the database.
 */
export type ChatPreBranchInfo = {
    /** 
     * The message most recently created (i.e., has the highest Snowflake ID), or null if no messages.
     * 
     * This is used as a fallback for the `parent` field when creating new messages. 
     * Ideally the client will specify the parent ID for each new message, 
     * which is acceptable for single-user chats (i.e. talking to bots), 
     * since we have full control over bot response ordering.
     *  
     * It is worth revisiting this in the future if there are issues when 
     * multiple users are involved and messages are being sent in parallel.
     */
    lastMessageId: string | null,
    /** 
     * Map of message IDs to their parent ID and list of child IDs. 
     * This data should be provided for each message that's being deleted, so that we 
     * can heal the branch structure around the deleted messages.
     */
    messageTreePatchInfo: {
        [messageId: string]: { parentId: string | null; childIds: string[] } | undefined
    };
};

/**
 * Message-specific information that must be collected before shaping inputs 
 * for insertion into the database.
 */
export type ChatMessagePre = {
    /** Map of chat IDs to information about the chat */
    chatData: Record<string, PreMapChatData | undefined>;
    /** Map of message IDs to information about the message */
    messageData: Record<string, PreMapMessageData | undefined>;
};

/**
 * The shape that will be passed into the database to create a message.
 * 
 * This will be modified directly to ensure that the chat messages 
 * are properly linked together.
 */
type ChatMessageCreate = {
    id: string;
    // Every message is linked to a parent, all the way up to the root
    parent?: {
        connect: {
            id: string;
        };
    };
}

/**
 * The shape that will be passed into the database to update a message.
 * 
 * This will be modified directly to ensure that the chat messages 
 * are properly linked together.
 */
type ChatMessageUpdate = {
    id: string;
    // Parent cannot be changed directly, so it's omitted
}

/**
 * The shape that will be passed into the database to delete a message.
 */
type ChatMessageDelete = {
    id: string;
}

/**
 * All chat message operations that will be passed into the database in 
 * the next transaction. There is typically only one operation 
 * (i.e. you send a message directly), but certain situations 
 * may require multiple operations (e.g. initializing a chat with 
 * multiple messages).
 * 
 * It is crucial that we can gracefully handle these complex cases 
 * without breaking the chat message tree.
 */
export interface TreeOpsInput {
    /** Metadata about the current root/leaf of the chat - comes from DB. */
    branchInfo: ChatPreBranchInfo;
    /** Flat list of *new* messages (order does not matter). */
    create: readonly ChatMessageCreate[];
    /** Messages the client explicitly updated (parent will be overwritten). */
    update: readonly ChatMessageUpdate[];
    /** Messages to remove.  Children will be re‑parented automatically. */
    del: readonly ChatMessageDelete[];
}

/**
 * The shape that will be passed into the database to create new messages, 
 * from the `messages` relation in the `chat` table.
 * 
 * We do this because to connect a parent message using an ID, it must already 
 * exist in the database. To get around this, sequential new messages in the transaction 
 * can be connected together into one nested shape, and passed in as a single 
 * `create` operation.
 */
type ChatMessageCreateResult = Omit<ChatMessageCreate, "parent"> & {
    parent?: {
        create: ChatMessageCreateResult
    } | {
        connect: {
            id: string;
        }
    };
};

/**
 * When performing database operations from the chat table, this is the updated 
 * shape that will be passed in for the `messages` relation.
 */
type ChatMessageOperationsResult = {
    // The `create` field is a single object with all messages nested
    create?: ChatMessageCreateResult[];
    // Each update can now update its parent reference or remove it
    update?: (ChatMessageUpdate & { parent?: ({ connect: { id: string } } | { disconnect: true }) })[];
    // Deletes are unchanged
    delete?: ChatMessageDelete[];
};

type ChatMessageOperationsSummaryResult = {
    Create: { id: string, parentId: string | null }[];
    Update: { id: string, parentId: string | null }[];
}

type ChatMessageUpdateWithParent = ChatMessageUpdate & {
    parent?: { connect: { id: string } } | { disconnect: true };
};

export interface TreeOpsOutput {
    /** Nested Prisma operations or `undefined` if there is nothing to do. */
    prismaOps: ChatMessageOperationsResult | undefined;
    /**
     * Flat description of *every* parent change the function performed.
     * Used to keep `preMap.messageData` in sync.
     */
    summary: ChatMessageOperationsSummaryResult;
}

/* --------------------------------------------------------------------------
 * utils/messageTree.ts
 * --------------------------------------------------------------------------
 * A **single‑responsibility** utility that computes *tree‑safe* Prisma message
 * operations **and** produces a human‑readable summary of parent‑pointer
 * mutations.  It is the canonical place that understands how Vrooli chat
 * messages branch, heal after deletes, and nest for bulk‑create.
 *
 * Motivations
 * -----------
 * 1. **Consistency** – Every create / update / delete that hits the DB (or the
 *    in‑memory tests) passes through *exactly the same* rules.  No more ad‑hoc
 *    tree surgery sprinkled across the code‑base.
 * 2. **Auditability** – All decisions (e.g. "who becomes the new parent when a
 *    node is deleted?") are encoded in *pure* static functions that are trivial
 *    to unit‑test – no DB, no Redis.
 * 3. **Performance** – By returning a *single* Prisma‑ready object we let Prisma
 *    issue one nested query instead of N round‑trips.
 * 4. **Developer Ergonomics** - Callers only need to supply four flat arrays
 *    (`create`, `update`, `del`, `branchInfo`) and get back two objects:
 *      • `prismaOps` – the thing you hand to Prisma.
 *      • `summary` – feeds `preMap` so the trigger layer knows the new
 *        parent relationships.
 *
 * Public API
 * ----------
 * ```ts
 * import { MessageTree } from "../../utils/messageTree";
 *
 * const { prismaOps, summary } = MessageTree.buildOperations({
 *     branchInfo,
 *     create: newMessages,
 *     update: editedMessages,
 *     del:    deletedMessages,
 * });
 * ```
 *
 * The class is *stateless* – all methods are **static** and deterministic.
 * -------------------------------------------------------------------------- */
export class MessageTree {
    /** -------------------------------------------------------------------
     *  buildOperations                                                     
     * --------------------------------------------------------------------
     *  The one‑stop helper that callers should use.  It orchestrates:
     *    1. **Nesting** the create list into a valid Prisma tree.
     *    2. **Healing** orphaned children after deletes.
     *    3. **Merging** explicit client updates with the healed updates.
     *
     *  @param input  See {@link TreeOpsInput}.
     *  @returns      See {@link TreeOpsOutput}.
     */
    public static buildOperations(input: TreeOpsInput): TreeOpsOutput {
        this.#validateInput(input);

        const { branchInfo, create, update, del } = input;
        const summary: ChatMessageOperationsSummaryResult = { Create: [], Update: [] };

        /* 1️⃣  CREATE --------------------------------------------------------- */
        const idToCreate = new Map(create.map(m => [m.id, m]));
        const ordered = this.#topoSortCreates(create, idToCreate);
        const nested = this.#nestCreateForest(ordered, idToCreate, branchInfo, summary);

        /* 2️⃣  DELETE / HEAL -------------------------------------------------- */
        const { healedUpdates, deletedIds } = this.#healAfterDeletes(del, update, branchInfo, summary);

        /* 3️⃣  ASSEMBLE ------------------------------------------------------- */
        const prismaOps: ChatMessageOperationsResult | undefined = this.#compact({
            create: nested?.length ? nested : undefined,
            update: healedUpdates.length ? healedUpdates : undefined,
            delete: deletedIds.length ? [...del] : undefined,
        });

        return { prismaOps, summary };
    }

    /* ====================================================================== */
    /*  INTERNAL HELPERS (all prefixed with # to signal privacy)              */
    /* ====================================================================== */

    /**
     * Fast sanity checks so callers fail *early* (before Prisma I/O).
     */
    static #validateInput({ create, update, del }: TreeOpsInput): void {
        const ids = new Set<string>();
        // eslint-disable-next-line func-style
        const dup = (arr: readonly { id: string }[], name: string) =>
            arr.forEach(({ id }) => {
                if (ids.has(id)) throw new Error(`Duplicate message id '${id}' in ${name}`);
                ids.add(id);
            });

        dup(create, "`create`");
        dup(update, "`update`");
        dup(del, "`delete`");

        // ensure no create references a non-existent **new** peer
        for (const m of create) {
            const pid = m.parent?.connect?.id;
            if (pid && !ids.has(pid)) {
                throw new Error(`New message '${m.id}' references unknown parent '${pid}'`);
            }
            // It is fine if `pid` is an existing DB row – only forbid dangling
            // references to *other* new messages that are NOT in `create`.
            if (pid && !ids.has(pid) && create.findIndex(c => c.id === pid) === -1) {
                throw new Error(`Create '${m.id}' references parent '${pid}' that is neither in the DB nor in the current batch`);
            }
        }
    }

    /**
      * Deterministic, cycle-safe topological sort (leaf → root).
      * Throws on self-loops or larger cycles.
      */
    static #topoSortCreates(
        flat: readonly ChatMessageCreate[],
        map: ReadonlyMap<string, ChatMessageCreate>,
    ): ChatMessageCreate[] {
        const out: ChatMessageCreate[] = [];
        const state = new Map<string, "temp" | "perm">();   // DFS colours

        // eslint-disable-next-line func-style
        const visit = (node: ChatMessageCreate): void => {
            const id = node.id;
            const mark = state.get(id);
            if (mark === "perm") return;                      // already sorted
            if (mark === "temp") throw new Error(`Cycle detected at '${id}'`);
            state.set(id, "temp");

            const pid = node.parent?.connect?.id;
            if (pid && map.has(pid)) {
                const parentNode = map.get(pid);
                if (parentNode) { // Ensure parentNode is not undefined
                    visit(parentNode);
                }
            }    // recurse only into *new* parents

            state.set(id, "perm");
            out.push(node);                                   // leaf first
        };

        flat.forEach(visit);
        return out;
    }

    /**
     * Builds **leaf-rooted** nested create blobs:
     * the *leaf* becomes the outer‐most object so that every `.parent.create`
     * points "up" the tree – exactly how Prisma expects it.
     *
     * Implementation detail: we iterate the topo-sorted list **in reverse**
     * (leaf-first) and keep a `pending` map of not-yet-attached nodes.
     */
    static #nestCreateForest(
        ordered: readonly ChatMessageCreate[],
        idToCreate: ReadonlyMap<string, ChatMessageCreate>,
        branchInfo: ChatPreBranchInfo,
        summary: ChatMessageOperationsSummaryResult,
    ): ChatMessageCreateResult[] | undefined {
        if (!ordered.length) return undefined;
        const roots: ChatMessageCreateResult[] = [];
        const pending = new Map<string, ChatMessageCreateResult>(); // id → node awaiting attachment
        // Walk *leaf → root*
        for (const cur of [...ordered].reverse()) {
            const node: ChatMessageCreateResult = { ...cur };
            pending.set(cur.id, node);
            const parentId =
                cur.parent?.connect?.id ??
                branchInfo.lastMessageId ??
                null;
            summary.Create.push({ id: cur.id, parentId });
            /* Case 1 – parent created in same batch: nest it inside `node` */
            if (parentId && idToCreate.has(parentId)) {
                const pendingNode = pending.get(parentId);
                if (pendingNode) {
                    node.parent = { create: pendingNode };
                }
            }
            /* Case 2 – parent exists in DB: simple connect */
            else if (parentId) {
                node.parent = { connect: { id: parentId } };
            }
            /* The *outer-most* node of each chain ends up in `roots` */
            if (!parentId || !idToCreate.has(parentId)) {
                roots.push(node);
            }
        }
        return roots;
    }

    /**
     * Repairs the tree after deletes by re-parenting surviving children to the
     * nearest surviving ancestor.  Also merges client-supplied updates.
     */
    static #healAfterDeletes(
        del: readonly ChatMessageDelete[],
        update: readonly ChatMessageUpdate[],
        branchInfo: ChatPreBranchInfo,
        summary: ChatMessageOperationsSummaryResult,
    ) {
        const deleted = new Set<string>(del.map(d => d.id));
        const tree = branchInfo.messageTreePatchInfo;
        const healed: ChatMessageUpdateWithParent[] = [...update];

        // record explicit client updates
        for (const u of update) {
            summary.Update.push({ id: u.id, parentId: (u as any).parent?.connect?.id ?? null });
        }

        // eslint-disable-next-line func-style
        const nearestAncestor = (id: string | null): string | null => {
            let cur = id;
            while (cur && deleted.has(cur)) cur = tree[cur]?.parentId ?? null;
            return cur;
        };

        for (const { id } of del) {
            for (const child of tree[id]?.childIds ?? []) {
                if (deleted.has(child)) continue;               // child also deleted – cascade

                const parentId = nearestAncestor(tree[id]?.parentId ?? null);
                const op: ChatMessageUpdateWithParent = {
                    id: child,
                    parent: parentId ? { connect: { id: parentId } } : { disconnect: true },
                };

                const idx = healed.findIndex(u => u.id === child);
                idx >= 0 ? healed[idx] = { ...healed[idx], ...op }
                    : healed.push(op);

                summary.Update.push({ id: child, parentId });
            }
        }
        return { healedUpdates: healed, deletedIds: del.map(d => d.id) };
    }

    /** Removes `undefined` fields and empty arrays from an object. */
    static #compact<T extends Record<string, any>>(obj: T): T | undefined {
        const o = Object.fromEntries(
            Object.entries(obj).filter(([_, v]) => v !== undefined && (!(Array.isArray(v)) || v.length)),
        ) as T;
        return Object.keys(o).length ? o : undefined;
    }
}

/**
 * ---------------------------------------------------------------------------
 * BranchInfoLoader
 * ---------------------------------------------------------------------------
 * **Responsibility**  
 *   Convert a list of chat-ids (and the messages we intend to delete) into the
 *   exact `ChatPreBranchInfo` structure expected by `MessageTree.buildOperations`.
 *
 * **Why this lives in its own module**
 *   * **Single round-trip** – all data is fetched in parallel in one Promise.
 *   * **Pure & deterministic** – no side-effects, trivially unit-testable.
 *   * **Encapsulation** – callers never hand-roll _"get last message then load
 *     parent/children"_ queries; the rules stay consistent across the code-base.
 *
 * **What it returns**
 * ```ts
 * {
 *   "<chatId>": {
 *     lastSequenceId: "123" | null,               // the leaf
 *     messageTreePatchInfo: {
 *       "<msgId>": { parentId: "...", childIds: [...] }
 *       ...
 *     }
 *   }
 * }
 * ```
 *
 * **Typical usage**
 * ```ts
 * const branchInfo = await BranchInfoLoader.load(
 *   chatIds,            // every chat we mutate
 *   deletedIds,         // message ids about to be deleted
 * );
 * ```
 *
 * **Performance notes**
 *   * Two independent `findMany` calls wrapped in `Promise.all`.
 *   * Indexes required: `chat_message(chatId, sequence)` for the leaf lookup
 *     and `chat_message(id)` for parent/child expansion (already primary key).
 *
 * **Edge-cases handled**
 *   * Chats with **zero** messages – `lastSequenceId` is `null`.
 *   * Deleting a message that is **not** the leaf – parent/child info is still
 *     supplied so `MessageTree` can heal the gap.
 *
 * ---------------------------------------------------------------------------
 */
export class BranchInfoLoader {
    /**
     * Fetches branch metadata for every chat in `chatIds`.
     *
     * @param chatIds     Chats that will receive create/update/delete messages.
     * @param deletedIds  Message-ids scheduled for deletion (healing required).
     */
    static async load(
        chatIds: string[],
        deletedIds: string[],
    ): Promise<Record<string, ChatPreBranchInfo>> {
        if (!chatIds.length) return {};

        /* ------------------------------------------------------
         * 1. Fetch the *leaf* (highest `sequence`) message per chat
         * 2. Fetch parent/child links for any message we will delete
         * ---------------------------------------------------- */
        const [leaves, treeNodes] = await Promise.all([
            DbProvider.get().chat.findMany({
                where: { id: { in: chatIds.map(BigInt) } },
                select: {
                    id: true,
                    messages: {
                        orderBy: { id: "desc" },
                        take: 1,
                        select: { id: true },
                    },
                },
            }),
            deletedIds.length
                ? DbProvider.get().chat_message.findMany({
                    where: { id: { in: deletedIds.map(BigInt) } },
                    select: {
                        id: true,
                        chatId: true,
                        parentId: true,
                        children: { select: { id: true } },
                    },
                })
                : [],
        ]);

        /* ---------- Shape results into the output map ---------- */
        const out: Record<string, ChatPreBranchInfo> = {};

        // initialise per-chat containers with leaf info
        leaves.forEach(({ id, messages }) => {
            out[id.toString()] = {
                lastMessageId: messages[0]?.id?.toString() ?? null,
                messageTreePatchInfo: {},
            };
        });

        // fill in parent/child links for soon-to-be-deleted nodes
        treeNodes.forEach(({ id, chatId, parentId, children }) => {
            if (chatId === null) {
                // If chatId is null, we cannot associate this treeNode with a chat.
                // This case should ideally not happen if data integrity is maintained.
                console.warn(`Skipping treeNode with id ${id.toString()} due to null chatId.`);
                return; // Skip this iteration
            }
            const chatKey = chatId.toString();
            const tree = (out[chatKey] ??= {
                lastMessageId: null,
                messageTreePatchInfo: {},
            }).messageTreePatchInfo;

            tree[id.toString()] = {
                parentId: parentId?.toString() ?? null,
                childIds: children.map(c => c.id.toString()),
            };
        });

        return out;
    }
}

/**
 * ---------------------------------------------------------------------------
 * MessageInfoCollector
 * ---------------------------------------------------------------------------
 * **Responsibility**  
 *   Harvest the *minimal* per-chat and per-message facts needed by
 *   `ChatMessageModel` triggers (AI routing, websocket fan-out, etc.) in **one**
 *   database query.
 *
 * **Why this matters**
 *   * Keeps request latency predictable regardless of how many chats/messages
 *     the client mutates in a single batch.
 *   * Central place to change selection logic (e.g. "later we need reactions
 *     count" – add it once, everywhere benefits).
 *
 * **Returned object**
 * ```ts
 * {
 *   chatData: {
 *     "<chatId>": {
 *       hasBotParticipants: boolean,
 *       isNew: boolean,               // always false here (new chats handled elsewhere)
 *       lastMessageId?: string
 *     },
 *     ...
 *   },
 *   messageData: {
 *     "<msgId>": {
 *        __type : "Update",           // Create / Update / Delete filled by caller
 *        chatId : "...",
 *        messageId : "...",
 *        parentId? : "...",
 *        text    : string,
 *        userId? : string
 *     },
 *     ...
 *   }
 * }
 * ```
 *
 * **Usage contract**
 *   * The collector never mutates incoming arguments.
 *   * New messages are **not** in the database yet; caller appends its own
 *     `messageData` entries after calling `collect`.
 *
 * **Performance**
 *   * Uses `chat.findMany` with a composite OR-clause so PostgreSQL can satisfy
 *     everything from the covering index on `chat_message(chatId, id)`.
 *
 * ---------------------------------------------------------------------------
 */
export class MessageInfoCollector {
    /**
     * Fetches participant-/message-level meta for the supplied ids.
     *
     * @param chatIds  Chats that will receive brand-new messages.
     * @param msgIds   Messages that will be *updated* or *deleted*.
     */
    static async collect(
        chatIds: string[],
        msgIds: string[],
    ): Promise<ChatMessagePre> {
        if (!chatIds.length && !msgIds.length) {
            return { chatData: {}, messageData: {} };
        }

        const rows = await DbProvider.get().chat.findMany({
            where: {
                OR: [
                    ...(chatIds.length ? [{ id: { in: chatIds.map(BigInt) } }] : []),
                    ...(msgIds.length
                        ? [{ messages: { some: { id: { in: msgIds.map(BigInt) } } } }]
                        : []),
                ],
            },
            select: {
                id: true,
                participants: { select: { user: { select: { isBot: true } } } },
                messages: {
                    orderBy: { id: "desc" },
                    take: 1,
                    select: {
                        id: true,
                        parent: { select: { id: true } },
                        text: true,
                        user: { select: { id: true } },
                    },
                },
            },
        });

        /* ---------- Shape into ChatMessagePre ---------- */
        const chatData: Record<string, PreMapChatData> = {};
        const messageData: Record<string, PreMapMessageDataUpdate> = {};

        rows.forEach(chat => {
            const chatId = chat.id.toString();

            chatData[chatId] = {
                hasBotParticipants: chat.participants.some(p => p.user.isBot),
                isNew: false,
                lastMessageId: chat.messages[0]?.id?.toString(),
            };

            chat.messages.forEach(message => {
                const messageId = message.id.toString();
                const parentId = message.parent?.id?.toString() ?? undefined;
                const userId = message.user?.id?.toString() ?? undefined;
                messageData[messageId] = {
                    __type: "Update",
                    chatId,
                    messageId,
                    parentId,
                    text: message.text,
                    userId,
                };
            });
        });

        return { chatData, messageData };
    }
}


// TODO possibly simplify these types
type PreMapMessageDataCreate = {
    __type: "Create";
    chatId: string;
    messageId: string;
    parentId: string | null;
    text: string;
    userId: string;
}
type PreMapMessageDataUpdate = {
    __type: "Update";
    chatId: string;
    messageId: string;
    parentId?: string | null;
    text: string;
    userId?: string;
}
type PreMapMessageDataDelete = {
    __type: "Delete";
    chatId: string;
    messageId: string;
}

/** Information for a message, collected in mutate.shape.pre */
export type PreMapMessageData = PreMapMessageDataCreate | PreMapMessageDataUpdate | PreMapMessageDataDelete;

/** Information for a message's corresponding chat, collected in mutate.shape.pre */
export type PreMapChatData = {
    /** If there are bots in the chat */
    hasBotParticipants: boolean,
    /** If the chat is new */
    isNew: boolean,
    /** ID of the last message in the chat */
    lastMessageId?: string,
};

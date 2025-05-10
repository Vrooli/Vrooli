import { DEFAULT_LANGUAGE, generatePK } from "@local/shared";
import { chat_message, Prisma, PrismaClient, user } from "@prisma/client";
import { BotParticipant, Conversation, Message, MessageMetaPayload } from "./types.js";

const chatMessageSelect = {
    id: true,
    text: true,
    createdAt: true,
    config: true,
    user: {
        select: {
            id: true,
            handle: true,
            isBot: true,
            name: true,
        }
    }
} satisfies Prisma.chat_messageSelect;

const chatParticipantSelect = {
    id: true,
    user: {
        select: {
            id: true,
            handle: true,
            isBot: true,
            name: true,
        }
    }
} satisfies Prisma.chat_participantsSelect;

type ParticipantData = { user: Pick<user, "id" | "name" | "handle" | "isBot"> };
type MessageData = Pick<chat_message, "id" | "text" | "createdAt" | "config"> & { user?: Pick<user, "id" | "name" | "handle" | "isBot"> | null };

function mapParticipant(participant: ParticipantData): BotParticipant {
    return {
        id: participant.user.id.toString(),
        name: participant.user.name,
        meta: {}, //TODO need to get this somewhere. Isn't stored in the user because it's chat-specific
    };
}

function mapMessage(message: MessageData): Message {
    const config = (message.config ?? {}) as MessageMetaPayload;
    const role = config.role ?? (message.user?.isBot ? "assistant" : "user");
    const botId = message.user?.id.toString();

    return {
        id: message.id.toString(),
        botId,
        content: message.text,
        contextHints: config.contextHints,
        createdAt: message.createdAt,
        eventTopic: config.eventTopic,
        respondingBots: config.respondingBots,
        role,
        turnId: config.turnId,
    } satisfies Message;
}

/** Storage abstraction for conversations & messages. */
export abstract class ConversationRepository {
    /** Fetch a conversation by ID (participants & meta included). */
    abstract getConversation(id: string): Promise<Conversation | null>;

    /** Fetch a single message (raw).
     *  @throws if not found
     */
    abstract getMessage(id: string): Promise<Message>;

    /** Persist a new message and return the inserted row (incl. id). */
    abstract saveMessage(conversationId: string, msg: Message): Promise<Message>;
}

export class PrismaConversationRepository extends ConversationRepository {
    private prisma: PrismaClient;

    constructor(prisma: PrismaClient = new PrismaClient()) {
        super();
        this.prisma = prisma;
    }

    async getConversation(id: string): Promise<Conversation | null> {
        const cid = BigInt(id);

        const convo = await this.prisma.chat.findUnique({
            where: { id: cid },
            select: {
                config: true,
                participants: { select: chatParticipantSelect },
                turnCounter: true,
            },
        });

        if (!convo) return null;

        return {
            id,
            meta: (convo.config ?? {}) as Conversation["meta"],
            participants: convo.participants.map(mapParticipant),
            turnCounter: Number(convo.turnCounter),
        } satisfies Conversation;
    }

    async getMessage(id: string): Promise<Message> {
        const mid = BigInt(id);
        const row = await this.prisma.chat_message.findUnique({
            where: { id: mid },
            select: chatMessageSelect
        });
        if (!row) throw new Error(`Message ${id} not found`);
        return mapMessage(row);
    }

    //TODO messages have parentId that we need to handle
    async saveMessage(conversationId: string, msg: Message): Promise<Message> {
        const config: MessageMetaPayload = {
            contextHints: msg.contextHints,
            respondingBots: msg.respondingBots,
            eventTopic: msg.eventTopic,
            role: msg.role,
            turnId: msg.turnId,
        };

        const inserted = await this.prisma.chat_message.create({
            data: {
                id: generatePK(),
                config: config as Prisma.InputJsonValue,
                language: msg.language ?? DEFAULT_LANGUAGE,
                text: msg.content,
                chat: { connect: { id: BigInt(conversationId) } },
                ...(msg.botId ? { user: { connect: { id: BigInt(msg.botId) } } } : {}),
            },
            select: chatMessageSelect,
        });

        return mapMessage(inserted);
    }

    async updateConversationMeta(id: string, patch: Record<string, unknown>): Promise<void> {
        await this.prisma.chat.update({
            where: { id: BigInt(id) },
            data: {
                config: {
                    // Prisma Json merge semantics (Postgres jsonb ||)
                    path: [],
                    value: patch,
                } as any,
            },
        });
    }

    async incrementTurnCounter(conversationId: string): Promise<number> {
        const updated = await this.prisma.$queryRaw<{ turnCounter: bigint }[]>`
      UPDATE chat
         SET turnCounter = COALESCE(turnCounter,0) + 1
       WHERE id = ${BigInt(conversationId)}
   RETURNING turnCounter`;
        return Number(updated[0].turnCounter);
    }
}

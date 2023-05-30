import { ChatMessage, ChatMessageCreateInput, ChatMessageSearchInput, ChatMessageSortBy, ChatMessageUpdateInput, ChatMessageYou, MaxObjects, uuidValidate } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { ChatModel } from "./chat";
import { ReactionModel } from "./reaction";
import { ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "ChatMessage" as const;
type Permissions = Pick<ChatMessageYou, "canDelete" | "canUpdate" | "canReply" | "canReport" | "canReact">;
const suppFields = ["you"] as const;
export const ChatMessageModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ChatMessageCreateInput,
    GqlUpdate: ChatMessageUpdateInput,
    GqlModel: ChatMessage,
    GqlSearch: ChatMessageSearchInput,
    GqlSort: ChatMessageSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.chat_messageUpsertArgs["create"],
    PrismaUpdate: Prisma.chat_messageUpsertArgs["update"],
    PrismaModel: Prisma.chat_messageGetPayload<SelectWrap<Prisma.chat_messageSelect>>,
    PrismaSelect: Prisma.chat_messageSelect,
    PrismaWhere: Prisma.chat_messageWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat_message,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => {
                const best = bestTranslation(select.translations, languages)?.text ?? "";
                return best.length > 30 ? best.slice(0, 30) + "..." : best;
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            chat: "Chat",
            user: "User",
            // reactionSummaries: "ReactionSummary",
            reports: "Report",
        },
        prismaRelMap: {
            __typename,
            chat: "Chat",
            fork: "ChatMessage",
            children: "ChatMessage",
            user: "User",
            // reactionSummaries: "ReactionSummary",
            reports: "Report",
        },
        joinMap: {},
        countFields: {
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    mutate: {} as any, //TODO make sure that chat's updated_at is updated when a message is created, so that it shows up first in the list
    search: {
        defaultSort: ChatMessageSortBy.DateCreatedDesc,
        searchFields: {
            createdTimeFrame: true,
            minScore: true,
            chatId: true,
            userId: true,
            updatedTimeFrame: true,
            translationLanguages: true,
        },
        sortBy: ChatMessageSortBy,
        searchStringQuery: () => ({
            OR: [
                "transTextWrapped",
                { user: UserModel.search!.searchStringQuery() },
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: (data.chat as any).organization,
            User: (data.chat as any).user,
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isParticipant = uuidValidate(userId) && (data.chat as any).participants?.some((p) => p.userId === userId);
            return {
                ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
                canReply: () => isLoggedIn && !isDeleted && (isAdmin || isParticipant),
            };
        },
        permissionsSelect: () => ({
            id: true,
            chat: ["Chat", ["invites"]],
            user: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                chat: ChatModel.validate!.visibility.owner(userId),
            }),
        },
    },
});

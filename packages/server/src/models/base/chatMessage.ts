import { ChatMessageSortBy, MaxObjects, uuidValidate } from "@local/shared";
import { bestTranslation, defaultPermissions } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { ChatMessageFormat } from "../format/chatMessage";
import { ModelLogic } from "../types";
import { ChatModel } from "./chat";
import { ReactionModel } from "./reaction";
import { ChatMessageModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "ChatMessage" as const;
const suppFields = ["you"] as const;
export const ChatMessageModel: ModelLogic<ChatMessageModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.chat_message,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => {
                const best = bestTranslation(select.translations, languages)?.text ?? "";
                return best.length > 30 ? best.slice(0, 30) + "..." : best;
            },
        },
    },
    format: ChatMessageFormat,
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

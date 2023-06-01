import { ChatParticipant, ChatParticipantSearchInput, ChatParticipantSortBy, ChatParticipantUpdateInput, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ChatModel } from "./chat";
import { ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "ChatParticipant" as const;
const suppFields = [] as const;
export const ChatParticipantModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: ChatParticipantUpdateInput,
    GqlModel: ChatParticipant,
    GqlSearch: ChatParticipantSearchInput,
    GqlSort: ChatParticipantSortBy,
    GqlPermission: object,
    PrismaCreate: Prisma.chat_participantsUpsertArgs["create"],
    PrismaUpdate: Prisma.chat_participantsUpsertArgs["update"],
    PrismaModel: Prisma.chat_participantsGetPayload<SelectWrap<Prisma.chat_participantsSelect>>,
    PrismaSelect: Prisma.chat_participantsSelect,
    PrismaWhere: Prisma.chat_participantsWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat_participants,
    display: {
        // Label is the user's label
        label: {
            select: () => ({ id: true, user: { select: UserModel.display.label.select() } }),
            get: (select, languages) => UserModel.display.label.get(select.user as any, languages),
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            chat: "Chat",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            chat: "Chat",
            user: "User",
        },
        joinMap: {},
        countFields: {},
    },
    mutate: {} as any,
    search: {
        defaultSort: ChatParticipantSortBy.UserNameDesc,
        searchFields: {
            createdTimeFrame: true,
            chatId: true,
            userId: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatParticipantSortBy,
        searchStringQuery: () => ({
            OR: [
                { chat: ChatModel.search!.searchStringQuery() },
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
        permissionResolvers: defaultPermissions,
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

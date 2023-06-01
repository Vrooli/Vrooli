import { ChatParticipant, ChatParticipantSearchInput, ChatParticipantSortBy, ChatParticipantUpdateInput, MaxObjects } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { ChatModel } from "./chat";
import { ModelLogic } from "./types";
import { UserModel } from "./user";
import { Formatter } from "../types";

const __typename = "ChatParticipant" as const;
export const ChatParticipantFormat: Formatter<ModelChatParticipantLogic> = {
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
};

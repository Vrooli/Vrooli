import { ChatParticipantSortBy, chatParticipantValidation, MaxObjects } from "@local/shared";
import { ModelMap } from ".";
import { defaultPermissions } from "../../utils";
import { ChatParticipantFormat } from "../formats";
import { ChatModelInfo, ChatModelLogic, ChatParticipantModelLogic, UserModelInfo, UserModelLogic } from "./types";

const __typename = "ChatParticipant" as const;
export const ChatParticipantModel: ChatParticipantModelLogic = ({
    __typename,
    dbTable: "chat_participants",
    display: () => ({
        // Label is the user's label
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["PrismaModel"], languages),
        },
    }),
    format: ChatParticipantFormat,
    mutate: {
        shape: {
            update: async () => ({}),
        },
        yup: chatParticipantValidation,
    },
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
                { chat: ModelMap.get<ChatModelLogic>("Chat").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: (data?.chat as ChatModelInfo["PrismaModel"])?.organization,
            User: (data?.chat as ChatModelInfo["PrismaModel"])?.creator,
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
                chat: ModelMap.get<ChatModelLogic>("Chat").validate().visibility.owner(userId),
            }),
        },
    }),
});

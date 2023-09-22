import { ChatParticipantSortBy, chatParticipantValidation, MaxObjects } from "@local/shared";
import { defaultPermissions } from "../../utils";
import { ChatParticipantFormat } from "../formats";
import { ModelLogic } from "../types";
import { ChatModel } from "./chat";
import { ChatModelLogic, ChatParticipantModelLogic, UserModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "ChatParticipant" as const;
const suppFields = [] as const;
export const ChatParticipantModel: ModelLogic<ChatParticipantModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.chat_participants,
    display: {
        // Label is the user's label
        label: {
            select: () => ({ id: true, user: { select: UserModel.display.label.select() } }),
            get: (select, languages) => UserModel.display.label.get(select.user as UserModelLogic["PrismaModel"], languages),
        },
    },
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
                { chat: ChatModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: (data?.chat as ChatModelLogic["PrismaModel"])?.organization,
            User: (data?.chat as ChatModelLogic["PrismaModel"])?.creator,
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
                chat: ChatModel.validate.visibility.owner(userId),
            }),
        },
    },
});

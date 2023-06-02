import { ChatParticipantSortBy, MaxObjects } from "@local/shared";
import { defaultPermissions } from "../../utils";
import { ChatParticipantFormat } from "../format/chatParticipant";
import { ModelLogic } from "../types";
import { ChatModel } from "./chat";
import { ChatParticipantModelLogic } from "./types";
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
            get: (select, languages) => UserModel.display.label.get(select.user as any, languages),
        },
    },
    format: ChatParticipantFormat,
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

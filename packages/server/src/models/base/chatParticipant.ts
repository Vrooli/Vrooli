import { ChatParticipantSortBy, chatParticipantValidation, MaxObjects } from "@local/shared";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { ChatParticipantFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { ChatModelInfo, ChatModelLogic, ChatParticipantModelLogic, UserModelInfo, UserModelLogic } from "./types.js";

const __typename = "ChatParticipant" as const;
export const ChatParticipantModel: ChatParticipantModelLogic = ({
    __typename,
    dbTable: "chat_participants",
    display: () => ({
        // Label is the user's label
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["DbModel"], languages),
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
            Team: (data?.chat as ChatModelInfo["DbModel"])?.team,
            User: (data?.chat as ChatModelInfo["DbModel"])?.creator,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            chat: ["Chat", ["invites"]],
            user: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return { // If you created the chat or you are the participant
                    OR: [
                        { chat: useVisibility("Chat", "Own", data) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: null, // Search method disabled
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("ChatParticipant", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});

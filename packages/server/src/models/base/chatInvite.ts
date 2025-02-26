import { ChatInviteSortBy, chatInviteValidation, MaxObjects, uuidValidate } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { ChatInviteFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { ChatInviteModelInfo, ChatInviteModelLogic, ChatModelInfo, ChatModelLogic, UserModelInfo, UserModelLogic } from "./types.js";

const __typename = "ChatInvite" as const;
export const ChatInviteModel: ChatInviteModelLogic = ({
    __typename,
    dbTable: "chat_invite",
    display: () => ({
        // Label is the user label
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["DbModel"], languages),
        },
    }),
    format: ChatInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    message: noNull(data.message),
                    user: await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, objectType: "User", parentRelationshipName: "chatsInvited", data, ...rest }),
                    chat: await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, objectType: "Chat", parentRelationshipName: "invites", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => ({
                message: noNull(data.message),
            }),
        },
        trigger: {
            afterMutations: async ({ createdIds, userData }) => {
                //TODO Create invite notifications
            },
        },
        yup: chatInviteValidation,
    },
    search: {
        defaultSort: ChatInviteSortBy.DateCreatedDesc,
        searchFields: {
            createdTimeFrame: true,
            status: true,
            statuses: true,
            chatId: true,
            userId: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatInviteSortBy,
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { chat: ModelMap.get<ChatModelLogic>("Chat").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ChatInviteModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
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
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isYourInvite = uuidValidate(userId) && data.userId === userId;
            return {
                ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
                canDelete: () => isLoggedIn && !isDeleted && (isAdmin || isYourInvite),
            };
        },
        permissionsSelect: () => ({
            id: true,
            chat: ["Chat", ["invites"]],
            user: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return { // If you created the invite or were invited
                    OR: [
                        { chat: useVisibility("Chat", "Own", data) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: null, // Search method disabled
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("ChatInvite", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});

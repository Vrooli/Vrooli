import { ChatInviteSortBy, chatInviteValidation, MaxObjects, uuidValidate } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { ChatInviteFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ChatInviteModelLogic, ChatModelInfo, ChatModelLogic, UserModelInfo, UserModelLogic } from "./types";

const __typename = "ChatInvite" as const;
export const ChatInviteModel: ChatInviteModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.chat_invite,
    display: () => ({
        // Label is the user label
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["PrismaModel"], languages),
        },
    }),
    format: ChatInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    message: noNull(data.message),
                    ...(await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, objectType: "User", parentRelationshipName: "chatsInvited", data, ...rest })),
                    ...(await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, objectType: "Chat", parentRelationshipName: "invites", data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => ({
                message: noNull(data.message),
            }),
        },
        trigger: {
            afterMutations: async ({ createdIds, prisma, userData }) => {
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
            chatId: true,
            userId: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatInviteSortBy,
        searchStringQuery: () => ({
            OR: [
                "message",
                { chat: ModelMap.get<ChatModelLogic>("Chat").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
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
            Organization: (data?.chat as ChatModelInfo["PrismaModel"])?.organization,
            User: (data?.chat as ChatModelInfo["PrismaModel"])?.creator,
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
            private: {},
            public: {},
            owner: (userId) => ({
                chat: ModelMap.get<ChatModelLogic>("Chat").validate().visibility.owner(userId),
            }),
        },
    }),
});

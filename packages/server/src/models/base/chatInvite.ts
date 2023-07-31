import { ChatInviteSortBy, chatInviteValidation, MaxObjects, uuidValidate } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { ChatInviteFormat } from "../format/chatInvite";
import { ModelLogic } from "../types";
import { ChatModel } from "./chat";
import { ChatInviteModelLogic, ChatModelLogic, UserModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "ChatInvite" as const;
const suppFields = ["you"] as const;
export const ChatInviteModel: ModelLogic<ChatInviteModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.chat_invite,
    display: {
        // Label is the user label
        label: {
            select: () => ({ id: true, user: { select: UserModel.display.label.select() } }),
            get: (select, languages) => UserModel.display.label.get(select.user as UserModelLogic["PrismaModel"], languages),
        },
    },
    format: ChatInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    message: noNull(data.message),
                    ...(await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "User", parentRelationshipName: "chatsInvited", data, ...rest })),
                    ...(await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Chat", parentRelationshipName: "invites", data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => ({
                message: noNull(data.message),
            }),
        },
        trigger: {
            onCreated: async ({ created, prisma, userData }) => {
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
                { chat: ChatModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
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
            Organization: (data.chat as ChatModelLogic["PrismaModel"]).organization,
            User: (data.chat as ChatModelLogic["PrismaModel"]).creator,
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
                chat: ChatModel.validate.visibility.owner(userId),
            }),
        },
    },
});

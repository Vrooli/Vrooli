import { ChatInviteSortBy, ChatInviteYou, MaxObjects, uuidValidate } from "@local/shared";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { ChatModel } from "./chat";
import { ChatInviteModelLogic, ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "ChatInvite" as const;
type Permissions = Pick<ChatInviteYou, "canDelete" | "canUpdate">;
const suppFields = ["you"] as const;
export const ChatInviteModel: ModelLogic<ChatInviteModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat_invite,
    display: {
        // Label is the user label
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
    mutate: {} as any,
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
                chat: ChatModel.validate!.visibility.owner(userId),
            }),
        },
    },
});

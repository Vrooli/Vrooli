import { ChatSortBy, ChatYou, MaxObjects, uuidValidate } from "@local/shared";
import { ChatInviteStatus } from "@prisma/client";
import { PrismaType } from "../types";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { ChatModelLogic, ModelLogic } from "./types";

const __typename = "Chat" as const;
type Permissions = Pick<ChatYou, "canDelete" | "canInvite" | "canUpdate">;
const suppFields = ["you"] as const;
export const ChatModel: ModelLogic<ChatModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.chat,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: ({ translations }, languages) => bestTranslation(translations, languages).name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans.description,
                    name: trans.name,
                }, languages[0]);
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            organization: "Organization",
            restrictedToRoles: "Role",
            messages: "ChatMessage",
            participants: "ChatParticipant",
            invites: "ChatInvite",
            labels: "Label",
        },
        prismaRelMap: {
            __typename,
            creator: "User",
            organization: "Organization",
            restrictedToRoles: "Role",
            messages: "ChatMessage",
            participants: "ChatParticipant",
            invites: "ChatInvite",
            labels: "Label",
        },
        joinMap: { labels: "label", restrictedToRoles: "role" },
        countFields: {
            participantsCount: true,
            invitesCount: true,
            labelsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        // hasUnread: await ChatModel.query.getHasUnread(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: ChatSortBy.DateUpdatedDesc,
        searchFields: {
            createdTimeFrame: true,
            openToAnyoneWithInvite: true,
            labelsIds: true,
            organizationId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatSortBy,
        searchStringQuery: () => ({
            OR: [
                "labelsWrapped",
                "tagsWrapped",
                "transNameWrapped",
                "transDescriptionWrapped",
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Organization: data.organization,
            User: data.creator,
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isInvited = uuidValidate(userId) && data.invites?.some((i) => i.userId === userId && i.status === ChatInviteStatus.Pending);
            const isParticipant = uuidValidate(userId) && data.participants?.some((p) => p.userId === userId);
            return {
                ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
                canInvite: () => isLoggedIn && isAdmin,
                canRead: () => !isDeleted && (isPublic || isAdmin || isInvited || isParticipant),
            };
        },
        permissionsSelect: (userId) => ({
            id: true,
            creator: "User",
            ...(userId ? {
                participants: {
                    where: {
                        userId,
                    },
                    select: {
                        id: true,
                    },
                },
                invites: {
                    where: {
                        userId,
                    },
                    select: {
                        id: true,
                        status: true,
                    },
                },
            } : {}),
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                creator: { id: userId },
            }),
        },
    },
});

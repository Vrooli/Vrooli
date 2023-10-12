import { BotUpdateInput, MaxObjects, ProfileUpdateInput, UserSortBy, userValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { preShapeEmbeddableTranslatable, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, handlesCheck } from "../../validators";
import { UserFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { Mutater } from "../types";
import { BookmarkModelLogic, UserModelInfo, UserModelLogic, ViewModelLogic } from "./types";

const __typename = "User" as const;

const updateProfile: Mutater<UserModelInfo & { GqlUpdate: ProfileUpdateInput }>["shape"]["update"] = async ({ data, ...rest }) => ({
    bannerImage: data.bannerImage,
    handle: data.handle ?? null,
    name: noNull(data.name),
    profileImage: data.profileImage,
    theme: noNull(data.theme),
    isPrivate: noNull(data.isPrivate),
    isPrivateApis: noNull(data.isPrivateApis),
    isPrivateApisCreated: noNull(data.isPrivateApisCreated),
    isPrivateMemberships: noNull(data.isPrivateMemberships),
    isPrivateOrganizationsCreated: noNull(data.isPrivateOrganizationsCreated),
    isPrivateProjects: noNull(data.isPrivateProjects),
    isPrivateProjectsCreated: noNull(data.isPrivateProjectsCreated),
    isPrivatePullRequests: noNull(data.isPrivatePullRequests),
    isPrivateQuestionsAnswered: noNull(data.isPrivateQuestionsAnswered),
    isPrivateQuestionsAsked: noNull(data.isPrivateQuestionsAsked),
    isPrivateQuizzesCreated: noNull(data.isPrivateQuizzesCreated),
    isPrivateRoles: noNull(data.isPrivateRoles),
    isPrivateRoutines: noNull(data.isPrivateRoutines),
    isPrivateRoutinesCreated: noNull(data.isPrivateRoutinesCreated),
    isPrivateSmartContracts: noNull(data.isPrivateSmartContracts),
    isPrivateStandards: noNull(data.isPrivateStandards),
    isPrivateStandardsCreated: noNull(data.isPrivateStandardsCreated),
    isPrivateBookmarks: noNull(data.isPrivateBookmarks),
    isPrivateVotes: noNull(data.isPrivateVotes),
    notificationSettings: data.notificationSettings ?? null,
    // languages: TODO!!!
    ...(await shapeHelper({ relation: "focusModes", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "FocusMode", parentRelationshipName: "user", data, ...rest })),
    ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[rest.userData.id], data, ...rest })),
});

const updateBot: Mutater<UserModelInfo & { GqlUpdate: BotUpdateInput }>["shape"]["update"] = async ({ data, ...rest }) => ({
    bannerImage: data.bannerImage,
    botSettings: noNull(data.botSettings),
    handle: data.handle ?? null,
    isPrivate: noNull(data.isPrivate),
    name: noNull(data.name),
    profileImage: data.profileImage,
    ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[rest.userData.id], data, ...rest })),
});

export const UserModel: UserModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.user,
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name ?? "",
        },
        embed: {
            select: () => ({ id: true, name: true, handle: true, translations: { select: { id: true, bio: true, embeddingNeedsUpdate: true } } }),
            get: ({ name, handle, translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    bio: trans?.bio,
                    handle,
                    name,
                }, languages[0]);
            },
        },
    }),
    format: UserFormat,
    mutate: {
        shape: {
            pre: async ({ Update, prisma, userData }) => {
                await handlesCheck(prisma, __typename, [], Update, userData.languages);
                const maps = preShapeEmbeddableTranslatable<"id">({ Create: [], Update, objectType: __typename });
                return { ...maps };
            },
            /** Create only applies for bots */
            create: async ({ data, ...rest }) => ({
                id: data.id,
                bannerImage: noNull(data.bannerImage),
                botSettings: data.botSettings,
                handle: data.handle ?? null,
                isBot: true,
                isPrivate: data.isPrivate,
                name: data.name,
                profileImage: noNull(data.profileImage),
                invitedByUser: { connect: { id: rest.userData.id } },
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            /** Update can be either a bot or your profile */
            update: async ({ data, ...rest }) => {
                const isBot = (data as BotUpdateInput).id && (data as BotUpdateInput).id !== rest.userData.id;
                return isBot ? await updateBot({ data: data as BotUpdateInput, ...rest }) : await updateProfile({ data: data as ProfileUpdateInput, ...rest });
            },
        },
        yup: userValidation,
    },
    search: {
        defaultSort: UserSortBy.BookmarksDesc,
        sortBy: UserSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            isBot: true,
            maxBookmarks: true,
            maxViews: true,
            memberInOrganizationId: true,
            minBookmarks: true,
            minViews: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transBioWrapped",
                "nameWrapped",
                "handleWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            invitedByUser: ["User", ["invitedByUser"]], // If a bot, this is the user who created the bot
            isBot: true,
            isPrivate: true,
            languages: { select: { language: true } },
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({ User: data?.isBot ? data.invitedByUser : data }),
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        profanityFields: ["name", "handle"],
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { id: userId },
                    { isBot: true, invitedByUser: { id: userId } },
                ],
            }),
        },
        // createMany.forEach(input => lineBreaksCheck(input, ['bio'], 'LineBreaksBio'));
    }),
});

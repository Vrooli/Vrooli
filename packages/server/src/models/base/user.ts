import { BotCreateInput, BotUpdateInput, MaxObjects, ProfileUpdateInput, SEEDED_IDS, UserSortBy, getTranslation, userValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { withRedis } from "../../redisConn.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { handlesCheck } from "../../validators/handlesCheck.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { UserFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { BookmarkModelLogic, UserModelInfo, UserModelLogic, ViewModelLogic } from "./types.js";


type UserPre = PreShapeEmbeddableTranslatableResult;

const __typename = "User" as const;

/**
 * Length of random characters added to new user's name, 
 * if not provided by OAuth provider or other means.
 */
export const DEFAULT_USER_NAME_LENGTH = 8;

export const UserModel: UserModelLogic = ({
    __typename,
    dbTable: "user",
    dbTranslationTable: "user_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name ?? "",
        },
        embed: {
            select: () => ({ id: true, name: true, handle: true, translations: { select: { id: true, bio: true, embeddingNeedsUpdate: true } } }),
            get: ({ name, handle, translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    bio: trans.bio,
                    handle,
                    name,
                }, languages?.[0]);
            },
        },
    }),
    format: UserFormat,
    mutate: {
        shape: {
            pre: async ({ Update }): Promise<UserPre> => {
                await handlesCheck(__typename, [], Update);
                const maps = preShapeEmbeddableTranslatable<"id">({ Create: [], Update, objectType: __typename });
                return { ...maps };
            },
            // Create only applies for bots normally, but seeding and tests might create non-bot users
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as UserPre;
                const isUser = rest.userData.id === SEEDED_IDS.User.Admin && rest.additionalData?.isBot !== true;
                const commonData = {
                    id: data.id,
                    bannerImage: noNull(data.bannerImage),
                    handle: data.handle ?? null,
                    isPrivate: noNull(data.isPrivate),
                    name: data.name ?? "",
                    profileImage: noNull(data.profileImage),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
                if (!isUser) {
                    const botData = data as BotCreateInput;
                    return {
                        ...commonData,
                        botSettings: botData.botSettings,
                        isBot: true,
                        isBotDepictingPerson: botData.isBotDepictingPerson,
                        invitedByUser: { connect: { id: rest.userData.id } },
                    };
                }
                const profileData = data as (ProfileUpdateInput & { id: string });
                return {
                    ...commonData,
                    isBot: false,
                    theme: noNull(profileData.theme),
                    isPrivateApis: noNull(profileData.isPrivateApis),
                    isPrivateApisCreated: noNull(profileData.isPrivateApisCreated),
                    isPrivateCodes: noNull(profileData.isPrivateCodes),
                    isPrivateCodesCreated: noNull(profileData.isPrivateCodesCreated),
                    isPrivateMemberships: noNull(profileData.isPrivateMemberships),
                    isPrivateProjects: noNull(profileData.isPrivateProjects),
                    isPrivateProjectsCreated: noNull(profileData.isPrivateProjectsCreated),
                    isPrivatePullRequests: noNull(profileData.isPrivatePullRequests),
                    isPrivateQuestionsAnswered: noNull(profileData.isPrivateQuestionsAnswered),
                    isPrivateQuestionsAsked: noNull(profileData.isPrivateQuestionsAsked),
                    isPrivateQuizzesCreated: noNull(profileData.isPrivateQuizzesCreated),
                    isPrivateRoles: noNull(profileData.isPrivateRoles),
                    isPrivateRoutines: noNull(profileData.isPrivateRoutines),
                    isPrivateRoutinesCreated: noNull(profileData.isPrivateRoutinesCreated),
                    isPrivateStandards: noNull(profileData.isPrivateStandards),
                    isPrivateStandardsCreated: noNull(profileData.isPrivateStandardsCreated),
                    isPrivateTeamsCreated: noNull(profileData.isPrivateTeamsCreated),
                    isPrivateBookmarks: noNull(profileData.isPrivateBookmarks),
                    isPrivateVotes: noNull(profileData.isPrivateVotes),
                    notificationSettings: profileData.notificationSettings ?? null,
                    // languages: TODO!!!
                    focusModes: await shapeHelper({ relation: "focusModes", relTypes: ["Create"], isOneToOne: false, objectType: "FocusMode", parentRelationshipName: "user", data, ...rest }),
                }
            },
            /** Update can be either a bot or your profile */
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as UserPre;
                const isBot = rest.additionalData?.isBot ?? false;
                const commonData = {
                    bannerImage: data.bannerImage,
                    handle: data.handle ?? null,
                    id: data.id,
                    isPrivate: noNull(data.isPrivate),
                    name: noNull(data.name),
                    profileImage: data.profileImage,
                }
                if (isBot) {
                    const botData = data as BotUpdateInput;
                    return {
                        ...commonData,
                        botSettings: botData.botSettings,
                        isBotDepictingPerson: noNull(botData.isBotDepictingPerson),
                        translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                    }
                }
                const profileData = data as ProfileUpdateInput;
                return {
                    ...commonData,
                    theme: noNull(profileData.theme),
                    id: rest.userData.id,
                    isPrivateApis: noNull(profileData.isPrivateApis),
                    isPrivateApisCreated: noNull(profileData.isPrivateApisCreated),
                    isPrivateCodes: noNull(profileData.isPrivateCodes),
                    isPrivateCodesCreated: noNull(profileData.isPrivateCodesCreated),
                    isPrivateMemberships: noNull(profileData.isPrivateMemberships),
                    isPrivateProjects: noNull(profileData.isPrivateProjects),
                    isPrivateProjectsCreated: noNull(profileData.isPrivateProjectsCreated),
                    isPrivatePullRequests: noNull(profileData.isPrivatePullRequests),
                    isPrivateQuestionsAnswered: noNull(profileData.isPrivateQuestionsAnswered),
                    isPrivateQuestionsAsked: noNull(profileData.isPrivateQuestionsAsked),
                    isPrivateQuizzesCreated: noNull(profileData.isPrivateQuizzesCreated),
                    isPrivateRoles: noNull(profileData.isPrivateRoles),
                    isPrivateRoutines: noNull(profileData.isPrivateRoutines),
                    isPrivateRoutinesCreated: noNull(profileData.isPrivateRoutinesCreated),
                    isPrivateStandards: noNull(profileData.isPrivateStandards),
                    isPrivateStandardsCreated: noNull(profileData.isPrivateStandardsCreated),
                    isPrivateTeamsCreated: noNull(profileData.isPrivateTeamsCreated),
                    isPrivateBookmarks: noNull(profileData.isPrivateBookmarks),
                    isPrivateVotes: noNull(profileData.isPrivateVotes),
                    notificationSettings: profileData.notificationSettings ?? null,
                    // languages: TODO!!!
                    focusModes: await shapeHelper({ relation: "focusModes", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "FocusMode", parentRelationshipName: "user", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[rest.userData.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ deletedIds, updatedIds }) => {
                // Remove all updated and deleted users from botSettings cache
                if (deletedIds.length || updatedIds.length) {
                    await withRedis({
                        process: async (redisClient) => {
                            if (!redisClient) return;
                            const keys = [...deletedIds, ...updatedIds].map((id) => `bot:${id}`);
                            await redisClient.del(...keys as never); // Redis' types are being weird
                        },
                        trace: "0236",
                    });
                }
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
            isBotDepictingPerson: true,
            maxBookmarks: true,
            maxViews: true,
            memberInTeamId: true,
            minBookmarks: true,
            minViews: true,
            notInChatId: true,
            notInvitedToTeamId: true,
            notMemberInTeamId: true,
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<UserModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        isBookmarked: await ModelMap.get<BookmarkModelLogic>("Bookmark").query.getIsBookmarkeds(userData?.id, ids, __typename),
                        isViewed: await ModelMap.get<ViewModelLogic>("View").query.getIsVieweds(userData?.id, ids, __typename),
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
            own: function getOwn(data) {
                return {
                    OR: [ // Either yourself or a bot you created
                        { id: data.userId },
                        { isBot: true, invitedByUser: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        // Owned objects
                        useVisibility("User", "Own", data),
                        // Public objects
                        useVisibility("User", "Public", data),
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isPrivate: true,
                    OR: [ // Either yourself or a bot you created
                        { id: data.userId },
                        { isBot: true, invitedByUser: { id: data.userId } },
                    ],
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isPrivate: false,
                    OR: [ // Either yourself or a bot you created
                        { id: data.userId },
                        { isBot: true, invitedByUser: { id: data.userId } },
                    ],
                };
            },
            public: function getPublic() {
                return {
                    isPrivate: false,
                };
            },
        },
        // createMany.forEach(input => lineBreaksCheck(input, ['bio'], 'LineBreaksBio'));
    }),
});

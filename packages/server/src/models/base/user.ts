import { Prisma } from "@prisma/client";
import { type BotCreateInput, type BotUpdateInput, generatePublicId, getTranslation, MaxObjects, type ProfileUpdateInput, UserSortBy, userValidation } from "@vrooli/shared";
import { noNull } from "../../builders/noNull.js";
import { seedId } from "../../builders/seedIdHelper.js";
import { DbProvider } from "../../db/provider.js";
import { CacheService } from "../../redisConn.js";
import { EmbeddingService } from "../../services/embedding.js";
import { preShapeEmbeddableTranslatable, type PreShapeEmbeddableTranslatableResult } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { handlesCheck } from "../../validators/handlesCheck.js";
import { defaultPermissions, filterPermissions, getSingleTypePermissions } from "../../validators/permissions.js";
import { UserFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { type BookmarkModelLogic, type UserModelInfo, type UserModelLogic, type ViewModelLogic } from "./types.js";


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
            select: () => ({ id: true, name: true, handle: true, translations: { select: { id: true, bio: true, embeddingExpiredAt: true } } }),
            get: ({ name, handle, translations }, languages) => {
                const trans = getTranslation({ translations }, languages) as Partial<{
                    language: string;
                    bio: string;
                }>;
                return EmbeddingService.getEmbeddableString({
                    bio: trans?.bio || "",
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
            create: async ({ additionalData, adminFlags, data, idsCreateToConnect, preMap, userData }) => {
                const preData = preMap[__typename] as UserPre;
                const isSeeding = adminFlags?.isSeeding ?? false;
                const adminId = await DbProvider.getAdminId();
                const isUser = userData.id === adminId && additionalData?.isBot !== true;
                const commonDataBase = {
                    id: seedId(data.id, isSeeding),
                    bannerImage: typeof data.bannerImage === "string" ? data.bannerImage : null,
                    handle: data.handle ?? null,
                    isPrivate: noNull(data.isPrivate),
                    name: data.name ?? "",
                    profileImage: typeof data.profileImage === "string" ? data.profileImage : null,
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, additionalData, idsCreateToConnect, preMap, userData }),
                };
                if (!isUser) {
                    const botData = data as BotCreateInput;
                    const commonData = {
                        ...commonDataBase,
                        publicId: isSeeding ? (botData.publicId ?? generatePublicId()) : generatePublicId(),
                    };
                    return {
                        ...commonData,
                        // Cast to unknown first to satisfy Prisma's InputJsonValue type requirement
                        // BotConfigObject has specific properties but Prisma expects an index signature
                        botSettings: typeof botData.botSettings === "object" && botData.botSettings !== null ? botData.botSettings as unknown as Prisma.InputJsonValue : Prisma.DbNull,
                        isBot: true,
                        isBotDepictingPerson: botData.isBotDepictingPerson,
                        invitedByUser: { connect: { id: BigInt(userData.id) } },
                    };
                }
                const profileData = data as (ProfileUpdateInput & { id: string });
                const commonData = {
                    ...commonDataBase,
                    publicId: generatePublicId(), // Always generate new publicId for user profiles
                };
                return {
                    ...commonData,
                    isBot: false,
                    theme: noNull(profileData.theme),
                    isPrivate: noNull(profileData.isPrivate),
                    isPrivateMemberships: noNull(profileData.isPrivateMemberships),
                    isPrivatePullRequests: noNull(profileData.isPrivatePullRequests),
                    isPrivateResources: noNull(profileData.isPrivateResources),
                    isPrivateResourcesCreated: noNull(profileData.isPrivateResourcesCreated),
                    isPrivateTeamsCreated: noNull(profileData.isPrivateTeamsCreated),
                    isPrivateBookmarks: noNull(profileData.isPrivateBookmarks),
                    isPrivateVotes: noNull(profileData.isPrivateVotes),
                    notificationSettings: profileData.notificationSettings ? JSON.stringify(profileData.notificationSettings) : null,
                    // languages: TODO!!!
                };
            },
            /** Update can be either a bot or your profile */
            update: async ({ additionalData, data, idsCreateToConnect, preMap, userData }) => {
                const preData = preMap[__typename] as UserPre;
                const isBot = additionalData?.isBot ?? false;
                const commonData = {
                    bannerImage: typeof data.bannerImage === "string" ? data.bannerImage : (data.bannerImage === null ? null : undefined),
                    handle: data.handle ?? null,
                    isPrivate: noNull(data.isPrivate),
                    name: noNull(data.name),
                    profileImage: typeof data.profileImage === "string" ? data.profileImage : (data.profileImage === null ? null : undefined),
                };
                if (isBot) {
                    const botData = data as BotUpdateInput;
                    return {
                        ...commonData,
                        // Cast to unknown first to satisfy Prisma's InputJsonValue type requirement
                        // BotConfigObject has specific properties but Prisma expects an index signature
                        botSettings: typeof botData.botSettings === "object" && botData.botSettings !== null ? botData.botSettings as unknown as Prisma.InputJsonValue : (botData.botSettings === null ? Prisma.DbNull : undefined),
                        isBotDepictingPerson: noNull(botData.isBotDepictingPerson),
                        translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, additionalData, idsCreateToConnect, preMap, userData }),
                    };
                }
                const profileData = data as ProfileUpdateInput;

                // Security: Sanitize creditSettings to prevent manipulation of server-controlled fields
                let sanitizedCreditSettings = profileData.creditSettings;
                if (sanitizedCreditSettings) {
                    sanitizedCreditSettings = { ...sanitizedCreditSettings };

                    // Remove server-controlled tracking fields that users shouldn't be able to set
                    if (sanitizedCreditSettings.rollover) {
                        const { lastProcessedMonth: _, ...userRolloverSettings } = sanitizedCreditSettings.rollover;
                        sanitizedCreditSettings.rollover = userRolloverSettings;
                    }

                    if (sanitizedCreditSettings.donation) {
                        const { lastProcessedMonth: _, ...userDonationSettings } = sanitizedCreditSettings.donation;
                        sanitizedCreditSettings.donation = userDonationSettings;
                    }
                }

                return {
                    ...commonData,
                    creditSettings: sanitizedCreditSettings ? sanitizedCreditSettings as unknown as Prisma.InputJsonValue : undefined,
                    theme: noNull(profileData.theme),
                    isPrivate: noNull(profileData.isPrivate),
                    isPrivateMemberships: noNull(profileData.isPrivateMemberships),
                    isPrivatePullRequests: noNull(profileData.isPrivatePullRequests),
                    isPrivateResources: noNull(profileData.isPrivateResources),
                    isPrivateResourcesCreated: noNull(profileData.isPrivateResourcesCreated),
                    isPrivateTeamsCreated: noNull(profileData.isPrivateTeamsCreated),
                    isPrivateBookmarks: noNull(profileData.isPrivateBookmarks),
                    isPrivateVotes: noNull(profileData.isPrivateVotes),
                    notificationSettings: profileData.notificationSettings ? JSON.stringify(profileData.notificationSettings) : undefined,
                    // languages: TODO!!!
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[userData.id], data, additionalData, idsCreateToConnect, preMap, userData }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ deletedIds, updatedIds }) => {
                // Remove all updated and deleted users from botSettings cache
                if (deletedIds.length || updatedIds.length) {
                    const cacheService = CacheService.get();
                    const keys = [...deletedIds, ...updatedIds].map((id) => `bot:${id}`);
                    await Promise.all(keys.map(key => cacheService.del(key)));
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
            languages: true,
        }),
        permissionResolvers: ({ isAdmin, isDeleted, isLoggedIn, isPublic }) => {
            const base = defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic });
            return filterPermissions(base, [
                "canDelete", "canReport", "canUpdate",
                "canConnect", "canDisconnect", "canRead",
            ]);
        },
        owner: (data, _userId) => ({
            User: data?.isBot ? data.invitedByUser : data,
        }),
        isDeleted: () => false,
        isPublic: (data, _getParentInfo?) => data.isPrivate === false,
        profanityFields: ["name", "handle"],
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [ // Either yourself or a bot you created
                        { id: BigInt(data.userId) },
                        { isBot: true, invitedByUser: { id: BigInt(data.userId) } },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [ // Either yourself, or a bot you created, or a public user
                        { id: BigInt(data.userId) },
                        { isBot: true, invitedByUser: { id: BigInt(data.userId) } },
                        { isPrivate: false },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isPrivate: true,
                    OR: [ // Either yourself or a bot you created
                        { id: BigInt(data.userId) },
                        { isBot: true, invitedByUser: { id: BigInt(data.userId) } },
                    ],
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isPrivate: false,
                    OR: [ // Either yourself or a bot you created
                        { id: BigInt(data.userId) },
                        { isBot: true, invitedByUser: { id: BigInt(data.userId) } },
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

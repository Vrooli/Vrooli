import { BotUpdateInput, MaxObjects, ProfileUpdateInput, UserSortBy, getTranslation, userValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { withRedis } from "../../redisConn";
import { defaultPermissions, getEmbeddableString } from "../../utils";
import { PreShapeEmbeddableTranslatableResult, preShapeEmbeddableTranslatable, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, handlesCheck } from "../../validators";
import { UserFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { Mutater } from "../types";
import { BookmarkModelLogic, UserModelInfo, UserModelLogic, ViewModelLogic } from "./types";

type UserPre = PreShapeEmbeddableTranslatableResult;

const __typename = "User" as const;

type UpdateProfileType = Exclude<Mutater<UserModelInfo & { GqlUpdate: ProfileUpdateInput }>["shape"]["update"], undefined>;
async function updateProfile({ data, ...rest }: Parameters<UpdateProfileType>[0]): Promise<UserModelInfo["PrismaUpdate"]> {
    const preData = rest.preMap[__typename] as UserPre;
    return {
        bannerImage: data.bannerImage,
        handle: data.handle ?? null,
        name: noNull(data.name),
        profileImage: data.profileImage,
        theme: noNull(data.theme),
        isPrivate: noNull(data.isPrivate),
        isPrivateApis: noNull(data.isPrivateApis),
        isPrivateApisCreated: noNull(data.isPrivateApisCreated),
        isPrivateCodes: noNull(data.isPrivateCodes),
        isPrivateCodesCreated: noNull(data.isPrivateCodesCreated),
        isPrivateMemberships: noNull(data.isPrivateMemberships),
        isPrivateProjects: noNull(data.isPrivateProjects),
        isPrivateProjectsCreated: noNull(data.isPrivateProjectsCreated),
        isPrivatePullRequests: noNull(data.isPrivatePullRequests),
        isPrivateQuestionsAnswered: noNull(data.isPrivateQuestionsAnswered),
        isPrivateQuestionsAsked: noNull(data.isPrivateQuestionsAsked),
        isPrivateQuizzesCreated: noNull(data.isPrivateQuizzesCreated),
        isPrivateRoles: noNull(data.isPrivateRoles),
        isPrivateRoutines: noNull(data.isPrivateRoutines),
        isPrivateRoutinesCreated: noNull(data.isPrivateRoutinesCreated),
        isPrivateStandards: noNull(data.isPrivateStandards),
        isPrivateStandardsCreated: noNull(data.isPrivateStandardsCreated),
        isPrivateTeamsCreated: noNull(data.isPrivateTeamsCreated),
        isPrivateBookmarks: noNull(data.isPrivateBookmarks),
        isPrivateVotes: noNull(data.isPrivateVotes),
        notificationSettings: data.notificationSettings ?? null,
        // languages: TODO!!!
        focusModes: await shapeHelper({ relation: "focusModes", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "FocusMode", parentRelationshipName: "user", data, ...rest }),
        translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[rest.userData.id], data, ...rest }),
    };
}

type UpdateBotType = Exclude<Mutater<UserModelInfo & { GqlUpdate: BotUpdateInput }>["shape"]["update"], undefined>;
async function updateBot({ data, ...rest }: Parameters<UpdateBotType>[0]): Promise<UserModelInfo["PrismaUpdate"]> {
    const preData = rest.preMap[__typename] as UserPre;
    return {
        bannerImage: data.bannerImage,
        botSettings: noNull(data.botSettings),
        handle: data.handle ?? null,
        isBotDepictingPerson: noNull(data.isBotDepictingPerson),
        isPrivate: noNull(data.isPrivate),
        name: noNull(data.name),
        profileImage: data.profileImage,
        translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[rest.userData.id], data, ...rest }),
    };
}

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
                }, languages[0]);
            },
        },
    }),
    format: UserFormat,
    mutate: {
        shape: {
            pre: async ({ Update, userData }): Promise<UserPre> => {
                await handlesCheck(__typename, [], Update, userData.languages);
                const maps = preShapeEmbeddableTranslatable<"id">({ Create: [], Update, objectType: __typename });
                return { ...maps };
            },
            /** Create only applies for bots */
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as UserPre;
                return {
                    id: data.id,
                    bannerImage: noNull(data.bannerImage),
                    botSettings: data.botSettings,
                    handle: data.handle ?? null,
                    isBot: true,
                    isBotDepictingPerson: data.isBotDepictingPerson,
                    isPrivate: data.isPrivate,
                    name: data.name,
                    profileImage: noNull(data.profileImage),
                    invitedByUser: { connect: { id: rest.userData.id } },
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            /** Update can be either a bot or your profile */
            update: async ({ data, ...rest }) => {
                const isBot = Boolean((data as BotUpdateInput).id) && (data as BotUpdateInput).id !== rest.userData.id;
                return isBot ?
                    await updateBot({ data: data as BotUpdateInput, ...rest }) :
                    await updateProfile({ data: data as ProfileUpdateInput, ...rest });
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
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<UserModelInfo["GqlPermission"]>(__typename, ids, userData)),
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

import { MaxObjects, MemberInviteSortBy, MemberInviteStatus, memberInviteValidation, uuidValidate } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { MemberInviteFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { MemberInviteModelInfo, MemberInviteModelLogic, TeamModelLogic, UserModelInfo, UserModelLogic } from "./types.js";

const __typename = "MemberInvite" as const;
export const MemberInviteModel: MemberInviteModelLogic = ({
    __typename,
    dbTable: "member_invite",
    display: () => ({
        // Label is the member label
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["DbModel"], languages),
        },
    }),
    format: MemberInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: BigInt(data.id),
                message: noNull(data.message),
                status: MemberInviteStatus.Pending,
                willBeAdmin: noNull(data.willBeAdmin),
                willHavePermissions: noNull(data.willHavePermissions),
                team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "memberInvites", data, ...rest }),
                user: await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, objectType: "User", parentRelationshipName: "membershipsInvited", data, ...rest }),
            }),
            update: async ({ data }) => ({
                message: noNull(data.message),
                willBeAdmin: noNull(data.willBeAdmin),
                willHavePermissions: noNull(data.willHavePermissions),
            }),
        },
        yup: memberInviteValidation,
    },
    search: {
        defaultSort: MemberInviteSortBy.DateUpdatedDesc,
        sortBy: MemberInviteSortBy,
        searchFields: {
            createdTimeFrame: true,
            status: true,
            statuses: true,
            teamId: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { team: ModelMap.get<TeamModelLogic>("Team").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<MemberInviteModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
            status: true,
            team: "Team",
            user: "User",
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const inviteUserId = data.userId ?? data.user?.id;
            const isYourInvite = uuidValidate(userId) && inviteUserId === userId;
            const basePermissions = defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic });
            return {
                ...basePermissions,
                canAccept: () => isYourInvite,
                canDecline: () => isYourInvite,
                canRead: () => basePermissions.canRead() || isYourInvite,
            };
        },
        owner: (data) => ({
            Team: data?.team,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<MemberInviteModelInfo["DbSelect"]>([["team", "Team"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return { // If you created the invite or were invited
                    OR: [
                        { team: useVisibility("Team", "Own", data) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: function getOwnPrivate(data) {
                return useVisibility("MemberInvite", "Own", data);
            },
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("MemberInvite", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});

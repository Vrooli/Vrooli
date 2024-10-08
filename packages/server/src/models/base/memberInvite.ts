import { MaxObjects, MemberInviteSortBy, memberInviteValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MemberInviteFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { MemberInviteModelInfo, MemberInviteModelLogic, TeamModelLogic, UserModelInfo, UserModelLogic } from "./types";

const __typename = "MemberInvite" as const;
export const MemberInviteModel: MemberInviteModelLogic = ({
    __typename,
    dbTable: "member_invite",
    display: () => ({
        // Label is the member label
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["PrismaModel"], languages),
        },
    }),
    format: MemberInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                message: noNull(data.message),
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
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<MemberInviteModelInfo["GqlPermission"]>(__typename, ids, userData)),
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
            isAdmin: true,
            permissions: true,
            team: "Team",
            user: "User",
            roles: "Role",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Team: data?.team,
            User: data?.user,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<MemberInviteModelInfo["PrismaSelect"]>([["team", "Team"]], ...rest),
        // Not sure which search methods are needed, so we'll add them as needed
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        { team: useVisibility("Team", "OwnOrPublic", data) },
                        { user: { id: data.userId } },
                    ]
                };
            },
            ownOrPublic: null,
            ownPrivate: null,
            ownPublic: null,
            public: null,
        },
    }),
});

import { MaxObjects, MemberSortBy } from "@local/shared";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MemberFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { MemberModelInfo, MemberModelLogic, RoleModelLogic, TeamModelInfo, TeamModelLogic, UserModelInfo, UserModelLogic } from "./types";

const __typename = "Member" as const;
export const MemberModel: MemberModelLogic = ({
    __typename,
    dbTable: "member",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() },
            }),
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["PrismaModel"], languages),
        },
    }),
    format: MemberFormat,
    search: {
        defaultSort: MemberSortBy.DateCreatedDesc,
        sortBy: MemberSortBy,
        searchFields: {
            createdTimeFrame: true,
            isAdmin: true,
            roles: true,
            teamId: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                { role: ModelMap.get<RoleModelLogic>("Role").search.searchStringQuery() },
                { team: ModelMap.get<TeamModelLogic>("Team").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, team: "Team" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<TeamModelLogic>("Team").validate().owner(data?.team as TeamModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<TeamModelLogic>("Team").validate().isDeleted(data.team as TeamModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<MemberModelInfo["PrismaSelect"]>([["team", "Team"]], ...rest),
        visibility: {
            private: { team: ModelMap.get<TeamModelLogic>("Team").validate().visibility.private },
            public: { team: ModelMap.get<TeamModelLogic>("Team").validate().visibility.public },
            owner: (userId) => ({ team: ModelMap.get<TeamModelLogic>("Team").validate().visibility.owner(userId) }),
        },
    }),
});

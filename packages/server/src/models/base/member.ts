import { MaxObjects, MemberSortBy } from "@local/shared";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { MemberFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { type MemberModelInfo, type MemberModelLogic, type RoleModelLogic, type TeamModelInfo, type TeamModelLogic, type UserModelInfo, type UserModelLogic } from "./types.js";

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
            get: (select, languages) => ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["DbModel"], languages),
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<MemberModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
        owner: (data, userId) => ModelMap.get<TeamModelLogic>("Team").validate().owner(data?.team as TeamModelInfo["DbModel"], userId),
        isDeleted: (data) => ModelMap.get<TeamModelLogic>("Team").validate().isDeleted(data.team as TeamModelInfo["DbModel"]),
        isPublic: (...rest) => oneIsPublic<MemberModelInfo["DbSelect"]>([["team", "Team"]], ...rest),
        // Not sure which search methods are needed, so we'll add them as needed
        visibility: {
            own: null,
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    team: useVisibility("Team", "OwnOrPublic", data),
                };
            },
            ownPrivate: null,
            ownPublic: null,
            public: function getPublic(data) {
                return {
                    team: useVisibility("Team", "Public", data),
                };
            },
        },
    }),
});

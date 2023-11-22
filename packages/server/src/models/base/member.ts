import { MaxObjects, MemberSortBy } from "@local/shared";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MemberFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { MemberModelInfo, MemberModelLogic, OrganizationModelInfo, OrganizationModelLogic, RoleModelLogic, UserModelInfo, UserModelLogic } from "./types";

const __typename = "Member" as const;
export const MemberModel: MemberModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.member,
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
            organizationId: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                { organization: ModelMap.get<OrganizationModelLogic>("Organization").search.searchStringQuery() },
                { role: ModelMap.get<RoleModelLogic>("Role").search.searchStringQuery() },
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, organization: "Organization" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<OrganizationModelLogic>("Organization").validate().owner(data?.organization as OrganizationModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<OrganizationModelLogic>("Organization").validate().isDeleted(data.organization as OrganizationModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<MemberModelInfo["PrismaSelect"]>([["organization", "Organization"]], ...rest),
        visibility: {
            private: { organization: ModelMap.get<OrganizationModelLogic>("Organization").validate().visibility.private },
            public: { organization: ModelMap.get<OrganizationModelLogic>("Organization").validate().visibility.public },
            owner: (userId) => ({ organization: ModelMap.get<OrganizationModelLogic>("Organization").validate().visibility.owner(userId) }),
        },
    }),
});

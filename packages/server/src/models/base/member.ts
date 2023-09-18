import { MaxObjects, MemberSortBy } from "@local/shared";
import { defaultPermissions } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MemberFormat } from "../formats";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { RoleModel } from "./role";
import { MemberModelLogic, OrganizationModelLogic, UserModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "Member" as const;
const suppFields = ["you"] as const;
export const MemberModel: ModelLogic<MemberModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.member,
    display: {
        label: {
            select: () => ({
                id: true,
                user: { select: UserModel.display.label.select() },
            }),
            get: (select, languages) => UserModel.display.label.get(select.user as UserModelLogic["PrismaModel"], languages),
        },
    },
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
                { organization: OrganizationModel.search.searchStringQuery() },
                { role: RoleModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, organization: "Organization" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => OrganizationModel.validate.owner(data.organization as OrganizationModelLogic["PrismaModel"], userId),
        isDeleted: (data, languages) => OrganizationModel.validate.isDeleted(data.organization as OrganizationModelLogic["PrismaModel"], languages),
        isPublic: (data, languages) => OrganizationModel.validate.isPublic(data.organization as OrganizationModelLogic["PrismaModel"], languages),
        visibility: {
            private: { organization: OrganizationModel.validate.visibility.private },
            public: { organization: OrganizationModel.validate.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate.visibility.owner(userId) }),
        },
    },
});

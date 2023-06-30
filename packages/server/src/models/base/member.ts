import { MaxObjects, MemberSortBy } from "@local/shared";
import { selPad } from "../../builders";
import { defaultPermissions } from "../../utils";
import { MemberFormat } from "../format/member";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { RoleModel } from "./role";
import { MemberModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "Member" as const;
const suppFields = [] as const;
export const MemberModel: ModelLogic<MemberModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.member,
    display: {
        label: {
            select: () => ({
                id: true,
                user: selPad(UserModel.display.label.select),
            }),
            get: (select, languages) => UserModel.display.label.get(select.user as any, languages),
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
                { organization: OrganizationModel.search!.searchStringQuery() },
                { role: RoleModel.search!.searchStringQuery() },
                { user: UserModel.search!.searchStringQuery() },
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, organization: "Organization" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => OrganizationModel.validate.owner(data.organization as any, userId),
        isDeleted: (data, languages) => OrganizationModel.validate.isDeleted(data.organization as any, languages),
        isPublic: (data, languages) => OrganizationModel.validate.isPublic(data.organization as any, languages),
        visibility: {
            private: { organization: OrganizationModel.validate.visibility.private },
            public: { organization: OrganizationModel.validate.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate.visibility.owner(userId) }),
        },
    },
});

import { MemberSortBy } from "@local/consts";
import { selPad } from "../builders";
import { OrganizationModel } from "./organization";
import { RoleModel } from "./role";
import { UserModel } from "./user";
const __typename = "Member";
const suppFields = [];
export const MemberModel = ({
    __typename,
    delegate: (prisma) => prisma.member,
    display: {
        select: () => ({
            id: true,
            user: selPad(UserModel.display.select),
        }),
        label: (select, languages) => UserModel.display.label(select.user, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            organization: "Organization",
            roles: "Role",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
            roles: "Role",
            user: "User",
        },
        countFields: {},
    },
    search: {
        defaultSort: MemberSortBy.DateCreatedDesc,
        sortBy: MemberSortBy,
        searchFields: {
            createdTimeFrame: true,
            organizationId: true,
            updatedTimeFrame: true,
            userId: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                { organization: OrganizationModel.search.searchStringQuery() },
                { role: RoleModel.search.searchStringQuery() },
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
    },
});
//# sourceMappingURL=member.js.map
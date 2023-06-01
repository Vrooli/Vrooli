import { MemberSortBy } from "@local/shared";
import { selPad } from "../builders";
import { PrismaType } from "../types";
import { OrganizationModel } from "./organization";
import { RoleModel } from "./role";
import { MemberModelLogic, ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "Member" as const;
const suppFields = [] as const;
export const MemberModel: ModelLogic<MemberModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.member,
    display: {
        label: {
            select: () => ({
                id: true,
                user: selPad(UserModel.display.label.select),
            }),
            get: (select, languages) => UserModel.display.label.get(select.user as any, languages),
        },
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
        },
        searchStringQuery: () => ({
            OR: [
                { organization: OrganizationModel.search!.searchStringQuery() },
                { role: RoleModel.search!.searchStringQuery() },
                { user: UserModel.search!.searchStringQuery() },
            ],
        }),
    },
});

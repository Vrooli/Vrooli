import { Member, MemberSearchInput, MemberSortBy, MemberUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { OrganizationModel } from "./organization";
import { RoleModel } from "./role";
import { ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "Member" as const;
const suppFields = [] as const;
export const MemberModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: MemberUpdateInput,
    GqlModel: Member,
    GqlSearch: MemberSearchInput,
    GqlSort: MemberSortBy,
    GqlPermission: object,
    PrismaCreate: Prisma.memberUpsertArgs["create"],
    PrismaUpdate: Prisma.memberUpsertArgs["update"],
    PrismaModel: Prisma.memberGetPayload<SelectWrap<Prisma.memberSelect>>,
    PrismaSelect: Prisma.memberSelect,
    PrismaWhere: Prisma.memberWhereInput,
}, typeof suppFields> = ({
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

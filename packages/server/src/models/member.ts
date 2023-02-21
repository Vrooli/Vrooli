import { PrismaType } from "../types";
import { Member, MemberSearchInput, MemberSortBy, MemberUpdateInput } from '@shared/consts';
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { UserModel } from "./user";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { OrganizationModel } from "./organization";
import { RoleModel } from "./role";

const __typename = 'Member' as const;

const suppFields = [] as const;
export const MemberModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: MemberUpdateInput,
    GqlModel: Member,
    GqlSearch: MemberSearchInput,
    GqlSort: MemberSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.memberUpsertArgs['create'],
    PrismaUpdate: Prisma.memberUpsertArgs['update'],
    PrismaModel: Prisma.memberGetPayload<SelectWrap<Prisma.memberSelect>>,
    PrismaSelect: Prisma.memberSelect,
    PrismaWhere: Prisma.memberWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.member,
    display: {
        select: () => ({
            id: true,
            user: selPad(UserModel.display.select),
        }),
        label: (select, languages) => UserModel.display.label(select.user as any, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
            user: 'User',
            roles: 'Role',
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
                { organization: OrganizationModel.search!.searchStringQuery() },
                { role: RoleModel.search!.searchStringQuery() },
                { user: UserModel.search!.searchStringQuery() },
            ]
        })
    },
})
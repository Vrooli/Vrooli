import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Role, RoleCreateInput, RoleUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'Role' as const;
const suppFields = [] as const;
export const RoleModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoleCreateInput,
    GqlUpdate: RoleUpdateInput,
    GqlModel: Role,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.roleUpsertArgs['create'],
    PrismaUpdate: Prisma.roleUpsertArgs['update'],
    PrismaModel: Prisma.roleGetPayload<SelectWrap<Prisma.roleSelect>>,
    PrismaSelect: Prisma.roleSelect,
    PrismaWhere: Prisma.roleWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.role,
    display: {
        select: () => ({ 
            id: true, 
            name: true,
            translations: { select: { language: true, name: true } } 
        }),
        label: (select, languages) => {
            // Prefer translated name over default name
            const translated = bestLabel(select.translations, 'name', languages)
            if (translated.length > 0) return translated;
            return select.name;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            members: 'Member',
            organization: 'Organization',
        },
        prismaRelMap: {
            __typename,
            members: 'Member',
            meetings: 'Meeting',
            organization: 'Organization',
        },
        countFields: {
            membersCount: true,
        },
    },
})
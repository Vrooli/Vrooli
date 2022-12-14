import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Role, RoleCreateInput, RoleUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, Formatter } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RoleCreateInput,
    GqlUpdate: RoleUpdateInput,
    GqlModel: Role,
    GqlPermission: any,
    PrismaCreate: Prisma.roleUpsertArgs['create'],
    PrismaUpdate: Prisma.roleUpsertArgs['update'],
    PrismaModel: Prisma.roleGetPayload<SelectWrap<Prisma.roleSelect>>,
    PrismaSelect: Prisma.roleSelect,
    PrismaWhere: Prisma.roleWhereInput,
}

const __typename = 'Role' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        assignees: 'User',
        organization: 'Organization',
    },
    joinMap: { assignees: 'user' },
})

const displayer = (): Displayer<Model> => ({
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
})

export const RoleModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.role,
    display: displayer(),
    format: formatter(),
})
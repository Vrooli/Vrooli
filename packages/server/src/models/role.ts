import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Role } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, Formatter, GraphQLModelType } from "./types";

const formatter = (): Formatter<Role, any> => ({
    relationshipMap: {
        __typename: 'Role',
        assignees: 'User',
        organization: 'Organization',
    },
    joinMap: { assignees: 'user' },
})

const displayer = (): Displayer<
    Prisma.roleSelect,
    Prisma.roleGetPayload<SelectWrap<Prisma.roleSelect>>
> => ({
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
    delegate: (prisma: PrismaType) => prisma.role,
    display: displayer(),
    format: formatter(),
    type: 'Role' as GraphQLModelType,
})
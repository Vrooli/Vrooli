import { Prisma } from "@prisma/client";
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
    Prisma.roleGetPayload<{ select: { [K in keyof Required<Prisma.roleSelect>]: true } }>
> => ({
    select: () => ({ 
        id: true, 
        title: true,
        translations: { select: { language: true, title: true } } 
    }),
    label: (select, languages) => {
        // Prefer translated title over default title
        const translated = bestLabel(select.translations, 'title', languages)
        if (translated.length > 0) return translated;
        return select.title;
    },
})

export const RoleModel = ({
    delegate: (prisma: PrismaType) => prisma.role,
    display: displayer(),
    format: formatter(),
    type: 'Role' as GraphQLModelType,
})
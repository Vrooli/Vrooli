import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.labelSelect,
    Prisma.labelGetPayload<SelectWrap<Prisma.labelSelect>>
> => ({
    select: () => ({ id: true, label: true }),
    label: (select) => select.label,
})

export const LabelModel = ({
    delegate: (prisma: PrismaType) => prisma.label,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Label' as GraphQLModelType,
    validate: {} as any,
})
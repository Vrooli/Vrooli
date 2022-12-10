import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer } from "./types";

const __typename = 'RunProject' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.run_projectSelect,
    Prisma.run_projectGetPayload<SelectWrap<Prisma.run_projectSelect>>
> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name,
})

export const RunProjectModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})
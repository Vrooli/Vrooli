import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType } from "./types";

const __typename = 'RunProjectStep' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.run_project_stepSelect,
    Prisma.run_project_stepGetPayload<SelectWrap<Prisma.run_project_stepSelect>>
> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name,
})

export const RunProjectStepModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project_step,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})
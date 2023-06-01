import { RunRoutineSearchInput, RunRoutineSortBy, ScheduleException, ScheduleExceptionCreateInput, ScheduleExceptionUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { ScheduleModel } from "./schedule";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "ScheduleException" as const;
export const ScheduleExceptionFormat: Formatter<ModelScheduleExceptionLogic> = {
        gqlRelMap: {
            __typename,
            schedule: "Schedule",
        },
        prismaRelMap: {
            __typename,
            schedule: "Schedule",
        },
        countFields: {},
    },
    mutate: {} as any,
    validate: {} as any,
};

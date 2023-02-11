import { InputType, RunRoutineSortBy, RunStatus } from "@shared/consts";
import { runRoutineFindMany } from "api/generated/endpoints/runRoutine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchRunRoutines', lng),
    containers: [
        { totalItems: 1 },
    ],
    fields: [
        {
            fieldName: "status",
            label: "Status",
            type: InputType.Radio,
            props: {
                defaultValue: RunStatus.InProgress,
                row: true,
                options: [
                    { label: "In Progress", value: RunStatus.InProgress },
                    { label: "Completed", value: RunStatus.Completed },
                    { label: "Scheduled", value: RunStatus.Scheduled },
                    { label: "Failed", value: RunStatus.Failed },
                    { label: "Cancelled", value: RunStatus.Cancelled },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
    ]
})

export const runRoutineSearchParams = (lng: string) => toParams(runRoutineSearchSchema(lng), runRoutineFindMany, RunRoutineSortBy, RunRoutineSortBy.DateStartedAsc);
import { InputType, runRoutineFindMany, RunRoutineSortBy, RunStatus } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchRunRoutine"),
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
                    { label: "Don't Care", value: "undefined" },
                ],
            },
        },
    ],
});

export const runRoutineSearchParams = () => toParams(runRoutineSearchSchema(), runRoutineFindMany, RunRoutineSortBy, RunRoutineSortBy.DateStartedAsc);

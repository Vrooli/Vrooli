import { endpointGetRunRoutine, endpointGetRunRoutines, FormSchema, InputType, RunRoutineSortBy, RunStatus } from "@local/shared";
import { toParams } from "./base";
import { searchFormLayout } from "./common";

export const runRoutineSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchRunRoutine"),
    containers: [
        { direction: "column", totalItems: 1 },
    ],
    elements: [
        {
            fieldName: "status",
            id: "status",
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

export const runRoutineSearchParams = () => toParams(runRoutineSearchSchema(), endpointGetRunRoutines, endpointGetRunRoutine, RunRoutineSortBy, RunRoutineSortBy.DateStartedAsc);

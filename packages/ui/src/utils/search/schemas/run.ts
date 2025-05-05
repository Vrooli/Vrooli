import { endpointsRun, FormSchema, InputType, RunSortBy, RunStatus } from "@local/shared";
import { toParams } from "./base.js";
import { searchFormLayout } from "./common.js";

export function runSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchRun"),
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
    };
}

export function runSearchParams() {
    return toParams(runSearchSchema(), endpointsRun, RunSortBy, RunSortBy.DateStartedAsc);
}

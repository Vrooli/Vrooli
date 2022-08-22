import { InputType } from "@shared/consts";
import { FormSchema } from "forms/types";
import { RunStatus } from "graphql/generated/globalTypes";

export const runSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Runs",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
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
}
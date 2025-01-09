import { endpointsUnions, FormSchema, InputType, ProjectOrRoutineSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, complexityContainer, complexityFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, simplicityContainer, simplicityFields, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export function projectOrRoutineSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchProjectOrRoutine"),
        containers: [
            { direction: "column", totalItems: 1 },
            hasCompleteVersionContainer,
            votesContainer(),
            bookmarksContainer(),
            simplicityContainer(),
            complexityContainer(),
            languagesVersionContainer(),
            tagsContainer(),
        ],
        elements: [
            {
                fieldName: "objectType",
                id: "objectType",
                label: "Object Type",
                type: InputType.Radio,
                props: {
                    defaultValue: "undefined",
                    row: true,
                    options: [
                        { label: "Project", value: "Project" },
                        { label: "Routine", value: "Routine" },
                        { label: "Don't Care", value: "undefined" },
                    ],
                },
            },
            ...hasCompleteVersionFields(),
            ...votesFields(),
            ...bookmarksFields(),
            ...simplicityFields(),
            ...complexityFields(),
            ...languagesVersionFields(),
            ...tagsFields(),
        ],
    };
}

export function projectOrRoutineSearchParams() {
    return toParams(projectOrRoutineSearchSchema(), { findMany: endpointsUnions.projectOrRoutines }, ProjectOrRoutineSortBy, ProjectOrRoutineSortBy.BookmarksDesc);
}

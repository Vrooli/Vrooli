import { endpointsUnions, FormSchema, InputType, ProjectOrTeamSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields } from "./common.js";

export function projectOrTeamSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchProjectOrTeam"),
        containers: [
            { direction: "column", totalItems: 1 },
            bookmarksContainer(),
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
                        { label: "Team", value: "Team" },
                        { label: "Don't Care", value: "undefined" },
                    ],
                },
            },
            ...bookmarksFields(),
            ...languagesVersionFields(),
            ...tagsFields(),
        ],
    };
}

export function projectOrTeamSearchParams() {
    return toParams(projectOrTeamSearchSchema(), { findMany: endpointsUnions.projectOrTeams }, ProjectOrTeamSortBy, ProjectOrTeamSortBy.BookmarksDesc);
}

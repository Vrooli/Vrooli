import { endpointGetUnionsProjectOrTeams, InputType, ProjectOrTeamSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields } from "./common";

export const projectOrTeamSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchProjectOrTeam"),
    containers: [
        { totalItems: 1 },
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
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
});

export const projectOrTeamSearchParams = () => toParams(projectOrTeamSearchSchema(), endpointGetUnionsProjectOrTeams, undefined, ProjectOrTeamSortBy, ProjectOrTeamSortBy.BookmarksDesc);

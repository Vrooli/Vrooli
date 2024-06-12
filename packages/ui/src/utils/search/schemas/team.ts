import { endpointGetTeam, endpointGetTeams, TeamSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, tagsContainer, tagsFields, yesNoDontCare } from "./common";

export const teamSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchTeam"),
    containers: [
        { totalItems: 1 },
        bookmarksContainer(),
        languagesContainer(),
        tagsContainer(),
    ],
    fields: [
        {
            fieldName: "isOpenToNewMembers",
            label: "Accepting new members?",
            ...yesNoDontCare(),
        },
        ...bookmarksFields(),
        ...languagesFields(),
        ...tagsFields(),
    ],
});

export const teamSearchParams = () => toParams(teamSearchSchema(), endpointGetTeams, endpointGetTeam, TeamSortBy, TeamSortBy.BookmarksDesc);

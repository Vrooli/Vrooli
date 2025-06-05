import { TeamSortBy, endpointsTeam, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, tagsContainer, tagsFields, yesNoDontCare } from "./common.js";

export function teamSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchTeam"),
        containers: [
            { direction: "column", totalItems: 1 },
            bookmarksContainer(),
            languagesContainer(),
            tagsContainer(),
        ],
        elements: [
            {
                fieldName: "isOpenToNewMembers",
                id: "isOpenToNewMembers",
                label: "Accepting new members?",
                ...yesNoDontCare(),
            },
            ...bookmarksFields(),
            ...languagesFields(),
            ...tagsFields(),
        ],
    };
}

export function teamSearchParams() {
    return toParams(teamSearchSchema(), endpointsTeam, TeamSortBy, TeamSortBy.BookmarksDesc);
}


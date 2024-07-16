import { endpointGetProject, endpointGetProjects, ProjectSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const projectSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchProject"),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    elements: [
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ],
});

export const projectSearchParams = () => toParams(projectSearchSchema(), endpointGetProjects, endpointGetProject, ProjectSortBy, ProjectSortBy.ScoreDesc);

import { endpointGetStandard, endpointGetStandards, StandardSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const standardSearchSchema = (): FormSchema => ({
    layout: searchFormLayout("SearchStandard"),
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

export const standardSearchParams = () => toParams(standardSearchSchema(), endpointGetStandards, endpointGetStandard, StandardSortBy, StandardSortBy.ScoreDesc);

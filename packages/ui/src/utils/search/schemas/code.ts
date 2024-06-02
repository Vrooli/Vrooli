import { CodeSortBy, endpointGetCode, endpointGetCodes } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, hasCompleteVersionContainer, hasCompleteVersionFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const codeSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchCode"),
    containers: [
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ],
});

export const codeSearchParams = () => toParams(codeSearchSchema(), endpointGetCodes, endpointGetCode, CodeSortBy, CodeSortBy.ScoreDesc);

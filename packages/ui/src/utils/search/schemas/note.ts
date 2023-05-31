import { noteFindMany, NoteSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesVersionContainer, languagesVersionFields, searchFormLayout, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const noteSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchNote"),
    containers: [
        votesContainer(),
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
    ],
    fields: [
        ...votesFields(),
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ],
});

export const noteSearchParams = () => toParams(noteSearchSchema(), noteFindMany, NoteSortBy, NoteSortBy.ScoreDesc);

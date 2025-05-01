import { endpointsResourceVersion, FormSchema, ResourceVersionSortBy } from "@local/shared";
import { toParams } from "./base.js";
import { bookmarksRootContainer, bookmarksRootFields, complexityContainer, complexityFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common.js";

export function resourceVersionSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchResourceVersion"),
        containers: [
            isCompleteWithRootContainer,
            isLatestContainer,
            simplicityContainer(),
            complexityContainer(),
            votesRootContainer(),
            bookmarksRootContainer(),
            languagesContainer(),
            tagsRootContainer(),
        ],
        elements: [
            ...isCompleteWithRootFields(),
            ...isLatestFields(),
            ...simplicityFields(),
            ...complexityFields(),
            ...votesRootFields(),
            ...bookmarksRootFields(),
            ...languagesFields(),
            ...tagsRootFields(),
        ],
    };
}

export function resourceVersionSearchParams() { return toParams(resourceVersionSearchSchema(), endpointsResourceVersion, ResourceVersionSortBy, ResourceVersionSortBy.DateCreatedDesc); }

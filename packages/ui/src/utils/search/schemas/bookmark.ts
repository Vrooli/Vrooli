import { BookmarkSortBy, FormSchema, endpointGetBookmark, endpointGetBookmarks } from "@local/shared";
import i18next from "i18next";
import { toParams } from "./base";
import { searchFormLayout, yesNoDontCare } from "./common";

export function bookmarkSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchBookmark"),
        containers: [
            {
                title: i18next.t("ExcludeLinkedToTag"),
                description: i18next.t("ExcludeLinkedToTagHelp"),
                direction: "row",
                totalItems: 1,
            },
        ],
        elements: [
            {
                fieldName: "excludeLinkedToTag",
                id: "excludeLinkedToTag",
                label: i18next.t("ExcludeLinkedToTagLabel"),
                ...yesNoDontCare(),
            },
        ],
    };
}

export function bookmarkSearchParams() {
    return toParams(bookmarkSearchSchema(), endpointGetBookmarks, endpointGetBookmark, BookmarkSortBy, BookmarkSortBy.DateUpdatedDesc);
}

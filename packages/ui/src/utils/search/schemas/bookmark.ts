import { BookmarkSortBy, endpointGetBookmark, endpointGetBookmarks } from "@local/shared";
import { FormSchema } from "forms/types";
import i18next from "i18next";
import { toParams } from "./base";
import { searchFormLayout, yesNoDontCare } from "./common";

export const bookmarkSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchBookmark"),
    containers: [
        {
            title: i18next.t("ExcludeLinkedToTag"),
            description: i18next.t("ExcludeLinkedToTagHelp"),
            direction: "row",
            totalItems: 1,
        },
    ],
    fields: [
        {
            fieldName: "excludeLinkedToTag",
            id: "excludeLinkedToTag",
            label: i18next.t("ExcludeLinkedToTagLabel"),
            ...yesNoDontCare(),
        },
    ],
});

export const bookmarkSearchParams = () => toParams(bookmarkSearchSchema(), endpointGetBookmarks, endpointGetBookmark, BookmarkSortBy, BookmarkSortBy.DateUpdatedDesc);

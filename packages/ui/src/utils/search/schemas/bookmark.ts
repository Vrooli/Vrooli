import { BookmarkSortBy } from "@local/shared";
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
            totalItems: 1,
            spacing: 2,
        },
    ],
    fields: [
        {
            fieldName: "excludeLinkedToTag",
            label: i18next.t("ExcludeLinkedToTagLabel"),
            ...yesNoDontCare(),
        },
    ],
});

export const bookmarkSearchParams = () => toParams(bookmarkSearchSchema(), "/bookmarks", BookmarkSortBy, BookmarkSortBy.DateUpdatedDesc);

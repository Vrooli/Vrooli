import { BookmarkSortBy } from "@shared/consts";
import { bookmarkFindMany } from "api/generated/endpoints/bookmark";
import { FormSchema } from "forms/types";
import i18next from "i18next";
import { toParams } from "./base";
import { searchFormLayout, yesNoDontCare } from "./common";

export const bookmarkSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchBookmark', lng),
    containers: [
        {
            title: i18next.t(`common:ExcludeLinkedToTag`, { lng }),
            description: i18next.t(`common:ExcludeLinkedToTagHelp`, { lng }),
            totalItems: 1,
            spacing: 2,
        }
    ],
    fields: [
        {
            fieldName: "excludeLinkedToTag",
            label: i18next.t(`common:ExcludeLinkedToTagLabel`, { lng }),
            ...yesNoDontCare(lng),
        },
    ]
})

export const bookmarkSearchParams = (lng: string) => toParams(bookmarkSearchSchema(lng), bookmarkFindMany, BookmarkSortBy, BookmarkSortBy.DateUpdatedDesc);
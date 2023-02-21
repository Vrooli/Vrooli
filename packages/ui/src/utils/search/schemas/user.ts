import { UserSortBy } from "@shared/consts";
import { userFindMany } from "api/generated/endpoints/user";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, bookmarksContainer, bookmarksFields } from "./common";

export const userSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchUser', lng),
    containers: [
        bookmarksContainer(lng),
        languagesContainer(lng),
    ],
    fields: [
        ...bookmarksFields(lng),
        ...languagesFields(lng),
    ]
})

export const userSearchParams = (lng: string) => toParams(userSearchSchema(lng), userFindMany, UserSortBy, UserSortBy.BookmarksDesc)
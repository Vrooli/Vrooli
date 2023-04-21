import { UserSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { userFindMany } from "../../api/generated/endpoints/user_findMany";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout } from "./common";

export const userSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchUser'),
    containers: [
        bookmarksContainer(),
        languagesContainer(),
    ],
    fields: [
        ...bookmarksFields(),
        ...languagesFields(),
    ]
})

export const userSearchParams = () => toParams(userSearchSchema(), userFindMany, UserSortBy, UserSortBy.BookmarksDesc)
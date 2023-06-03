import { UserSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout } from "./common";

export const userSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchUser"),
    containers: [
        bookmarksContainer(),
        languagesContainer(),
    ],
    fields: [
        ...bookmarksFields(),
        ...languagesFields(),
    ],
});

export const userSearchParams = () => toParams(userSearchSchema(), "/users", UserSortBy, UserSortBy.BookmarksDesc);

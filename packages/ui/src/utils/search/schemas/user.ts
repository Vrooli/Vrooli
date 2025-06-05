import { InputType, UserSortBy, endpointsUser, type FormSchema } from "@vrooli/shared";
import { toParams } from "./base.js";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout } from "./common.js";

export function userSearchSchema(): FormSchema {
    return {
        layout: searchFormLayout("SearchUser"),
        containers: [
            {
                direction: "column",
                disableCollapse: true,
                totalItems: 1,
            },
            bookmarksContainer(),
            languagesContainer(),
        ],
        elements: [
            {
                fieldName: "isBot",
                id: "isBot",
                label: "Bot",
                type: InputType.Switch,
                props: {},
            },
            ...bookmarksFields(),
            ...languagesFields(),
        ],
    };
}

export function userSearchParams() {
    return toParams(userSearchSchema(), endpointsUser, UserSortBy, UserSortBy.BookmarksDesc);
}

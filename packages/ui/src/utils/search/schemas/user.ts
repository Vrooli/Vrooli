import { endpointGetUser, endpointGetUsers, FormSchema, InputType, UserSortBy } from "@local/shared";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout } from "./common";

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
    return toParams(userSearchSchema(), endpointGetUsers, endpointGetUser, UserSortBy, UserSortBy.BookmarksDesc);
}

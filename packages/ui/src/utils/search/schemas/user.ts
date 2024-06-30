import { endpointGetUser, endpointGetUsers, InputType, UserSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout } from "./common";

export function userSearchSchema(): FormSchema {
    return {
        formLayout: searchFormLayout("SearchUser"),
        containers: [
            {
                disableCollapse: true,
                totalItems: 1,
            },
            bookmarksContainer(),
            languagesContainer(),
        ],
        fields: [
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

export function userSearchParams() { return toParams(userSearchSchema(), endpointGetUsers, endpointGetUser, UserSortBy, UserSortBy.BookmarksDesc); }

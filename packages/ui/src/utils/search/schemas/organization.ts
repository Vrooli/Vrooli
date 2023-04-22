import { OrganizationSortBy } from "@shared/consts";
import { FormSchema } from "forms/types";
import { organizationFindMany } from "../../../api/generated/endpoints/organization_findMany";
import { toParams } from "./base";
import { bookmarksContainer, bookmarksFields, languagesContainer, languagesFields, searchFormLayout, tagsContainer, tagsFields, yesNoDontCare } from "./common";

export const organizationSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchOrganization"),
    containers: [
        { totalItems: 1 },
        bookmarksContainer(),
        languagesContainer(),
        tagsContainer(),
    ],
    fields: [
        {
            fieldName: "isOpenToNewMembers",
            label: "Accepting new members?",
            ...yesNoDontCare(),
        },
        ...bookmarksFields(),
        ...languagesFields(),
        ...tagsFields(),
    ]
})

export const organizationSearchParams = () => toParams(organizationSearchSchema(), organizationFindMany, OrganizationSortBy, OrganizationSortBy.BookmarksDesc);
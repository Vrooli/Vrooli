import { OrganizationSortBy } from "@shared/consts";
import { organizationFindMany } from "api/generated/endpoints/organization_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, yesNoDontCare } from "./common";

export const organizationSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchOrganization'),
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
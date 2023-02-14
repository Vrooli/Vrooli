import { InputType, OrganizationSortBy } from "@shared/consts";
import { organizationFindMany } from "api/generated/endpoints/organization";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, starsContainer, starsFields, tagsContainer, tagsFields } from "./common";

export const organizationSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchOrganization', lng),
    containers: [
        { totalItems: 1 },
        starsContainer,
        languagesContainer,
        tagsContainer,
    ],
    fields: [
        {
            fieldName: "isOpenToNewMembers",
            label: "Accepting new members?",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Yes", value: 'true' },
                    { label: "No", value: 'false' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        ...starsFields,
        ...languagesFields,
        ...tagsFields,
    ]
})

export const organizationSearchParams = (lng: string) => toParams(organizationSearchSchema(lng), organizationFindMany, OrganizationSortBy, OrganizationSortBy.BookmarksDesc);
import { InputType, ProjectOrOrganizationSortBy } from "@shared/consts";
import { projectOrOrganizationFindMany } from "api/generated/endpoints/projectOrOrganization";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const projectOrOrganizationSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchProjectOrOrganization', lng),
    containers: [
        { totalItems: 1 },
        bookmarksContainer(lng),
        languagesVersionContainer(lng),
        tagsContainer(lng),
    ],
    fields: [
        {
            fieldName: "objectType",
            label: "Object Type",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Project", value: 'Project' },
                    { label: "Organization", value: 'Organization' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        ...bookmarksFields(lng),
        ...languagesVersionFields(lng),
        ...tagsFields(lng),
    ]
})

export const projectOrOrganizationSearchParams = (lng: string) => toParams(projectOrOrganizationSearchSchema(lng), projectOrOrganizationFindMany, ProjectOrOrganizationSortBy, ProjectOrOrganizationSortBy.BookmarksDesc)
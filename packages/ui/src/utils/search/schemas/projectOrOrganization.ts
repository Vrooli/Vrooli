import { InputType, ProjectOrOrganizationSortBy } from "@shared/consts";
import { projectOrOrganizationFindMany } from "api/generated/endpoints/projectOrOrganization_findMany";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { searchFormLayout, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, languagesVersionContainer, languagesVersionFields } from "./common";

export const projectOrOrganizationSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchProjectOrOrganization'),
    containers: [
        { totalItems: 1 },
        bookmarksContainer(),
        languagesVersionContainer(),
        tagsContainer(),
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
        ...bookmarksFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ]
})

export const projectOrOrganizationSearchParams = () => toParams(projectOrOrganizationSearchSchema(), projectOrOrganizationFindMany, ProjectOrOrganizationSortBy, ProjectOrOrganizationSortBy.BookmarksDesc)
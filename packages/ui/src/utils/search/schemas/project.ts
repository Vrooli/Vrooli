import { InputType, ProjectSortBy } from "@shared/consts";
import { projectFindMany } from "api/generated/endpoints/project";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, starsContainer, starsFields, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const projectSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchProject', lng),
    containers: [
        { totalItems: 1 },
        votesContainer,
        starsContainer,
        languagesContainer,
        tagsContainer,
    ],
    fields: [
        {
            fieldName: "isComplete",
            label: "Is Complete?",
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
        ...votesFields,
        ...starsFields,
        ...languagesFields,
        ...tagsFields,
    ]
})

export const projectSearchParams = (lng: string) => toParams(projectSearchSchema(lng), projectFindMany, ProjectSortBy, ProjectSortBy.ScoreDesc)
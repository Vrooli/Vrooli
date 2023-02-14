import { InputType, ProjectOrRoutineSortBy } from "@shared/consts";
import { projectOrRoutineFindMany } from "api/generated/endpoints/projectOrRoutine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { complexityContainer, complexityFields, isCompleteContainer, isCompleteFields, languagesContainer, languagesFields, searchFormLayout, simplicityContainer, simplicityFields, starsContainer, starsFields, tagsContainer, tagsFields, votesContainer, votesFields } from "./common";

export const projectOrRoutineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchProjectOrRoutine', lng),
    containers: [
        { totalItems: 1 },
        isCompleteContainer,
        votesContainer,
        starsContainer,
        simplicityContainer,
        complexityContainer,
        languagesContainer,
        tagsContainer,
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
                    { label: "Routine", value: 'Routine' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        ...isCompleteFields,
        ...votesFields,
        ...starsFields,
        ...simplicityFields,
        ...complexityFields,
        ...languagesFields,
        ...tagsFields,
    ]
})

export const projectOrRoutineSearchParams = (lng: string) => toParams(projectOrRoutineSearchSchema(lng), projectOrRoutineFindMany, ProjectOrRoutineSortBy, ProjectOrRoutineSortBy.BookmarksDesc)
import { InputType, ProjectOrRoutineSortBy } from "@shared/consts";
import { projectOrRoutineFindMany } from "api/generated/endpoints/projectOrRoutine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { complexityContainer, complexityFields, searchFormLayout, simplicityContainer, simplicityFields, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, languagesVersionContainer, languagesVersionFields, hasCompleteVersionContainer, hasCompleteVersionFields } from "./common";

export const projectOrRoutineSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchProjectOrRoutine', lng),
    containers: [
        { totalItems: 1 },
        hasCompleteVersionContainer,
        votesContainer(lng),
        bookmarksContainer(lng),
        simplicityContainer(lng),
        complexityContainer(lng),
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
                    { label: "Routine", value: 'Routine' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        ...hasCompleteVersionFields(lng),
        ...votesFields(lng),
        ...bookmarksFields(lng),
        ...simplicityFields(lng),
        ...complexityFields(lng),
        ...languagesVersionFields(lng),
        ...tagsFields(lng),
    ]
})

export const projectOrRoutineSearchParams = (lng: string) => toParams(projectOrRoutineSearchSchema(lng), projectOrRoutineFindMany, ProjectOrRoutineSortBy, ProjectOrRoutineSortBy.BookmarksDesc)
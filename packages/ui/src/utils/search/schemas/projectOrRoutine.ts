import { InputType, ProjectOrRoutineSortBy } from "@shared/consts";
import { projectOrRoutineFindMany } from "api/generated/endpoints/projectOrRoutine";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { complexityContainer, complexityFields, searchFormLayout, simplicityContainer, simplicityFields, bookmarksContainer, bookmarksFields, tagsContainer, tagsFields, votesContainer, votesFields, languagesVersionContainer, languagesVersionFields, hasCompleteVersionContainer, hasCompleteVersionFields } from "./common";

export const projectOrRoutineSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout('SearchProjectOrRoutine'),
    containers: [
        { totalItems: 1 },
        hasCompleteVersionContainer,
        votesContainer(),
        bookmarksContainer(),
        simplicityContainer(),
        complexityContainer(),
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
                    { label: "Routine", value: 'Routine' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        ...hasCompleteVersionFields(),
        ...votesFields(),
        ...bookmarksFields(),
        ...simplicityFields(),
        ...complexityFields(),
        ...languagesVersionFields(),
        ...tagsFields(),
    ]
})

export const projectOrRoutineSearchParams = () => toParams(projectOrRoutineSearchSchema(), projectOrRoutineFindMany, ProjectOrRoutineSortBy, ProjectOrRoutineSortBy.BookmarksDesc)
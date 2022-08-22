/**
 * Advanced search form schema for routines. 
 * Can search by: 
 * - Type (Project or Routine) - Radio
 * - Is Complete? - Radio
 * - Min/max stars - QuantityBox
 * - Min/max score - QuantityBox
 * - Min/max simplicity (Routine only) - QuantityBox
 * - Min/max complexity (Routine only) - QuantityBox
 * - Languages - LanguageInput
 * - Tags - TagSelector
 */
import { InputType } from "@shared/consts";
import { FormSchema } from "forms/types";

export const projectOrRoutineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Projects/Routines",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            totalItems: 1,
        },
        {
            title: "Stars",
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Score",
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Simplicity (Routines only)",
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Complexity (Routines only)",
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Languages",
            totalItems: 1
        },
        {
            title: "Tags",
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "type",
            label: "Object Type",
            type: InputType.Radio,
            props: {
                defaultValue: 'dontCare',
                row: true,
                options: [
                    { label: "Project", value: 'project' },
                    { label: "Routine", value: 'routine' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "isComplete",
            label: "Is Complete?",
            type: InputType.Radio,
            props: {
                defaultValue: 'dontCare',
                row: true,
                options: [
                    { label: "Yes", value: 'true' },
                    { label: "No", value: 'false' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxStars",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minScore",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxScore",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minSimplicity",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxSimplicity",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minComplexity",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxComplexity",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
        {
            fieldName: "tags",
            label: "Tags",
            type: InputType.TagSelector,
            props: {}
        },
    ]
}
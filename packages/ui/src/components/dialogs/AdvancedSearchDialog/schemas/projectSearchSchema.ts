/**
 * Advanced search form schema for projects. 
 * Can search by: 
 * - Is Complete? - Radio
 * - Minimum stars - QuantityBox
 * - Minimum score - QuantityBox
 * - Langauges - LanguageInput
 * - Tags - TagSelector
 */
import { FormSchema, InputType } from "forms/types";

export const projectSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Projects",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            totalItems: 2,
            spacing: 2,
        },
        {
            totalItems: 1
        },
        {
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "isComplete",
            label: "Is Complete?",
            type: InputType.Radio,
            props: {
                defaultValue: 'dontCare',
                row: true,
                options: [
                    { label: "Yes", value: 'yes' },
                    { label: "No", value: 'no' },
                    { label: "Don't Care", value: 'dontCare' },
                ]
            }
        },
        {
            fieldName: "minStars",
            label: "Minimum Stars",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minScore",
            label: "Minimum Score",
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
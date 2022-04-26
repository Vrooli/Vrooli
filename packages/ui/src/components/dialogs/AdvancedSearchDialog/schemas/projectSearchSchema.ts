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
        rowSpacing: 5,
    },
    fields: [
        {
            fieldName: "isComplete",
            label: "Is Complete?",
            type: InputType.Radio,
            props: {
                defaultValue: null,
                row: true,
                options: [
                    { label: "Yes", value: true },
                    { label: "No", value: false },
                    { label: "Don't Care", value: null },
                ]
            }
        },
        {
            fieldName: "minimumStars",
            label: "Minimum Stars",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minimumScore",
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
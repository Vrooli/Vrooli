/**
 * Advanced search form schema for routines. 
 * Can search by: 
 * - Is Complete? - Radio
 * - Minimum stars - QuantityBox
 * - Minimum score - QuantityBox
 * - Minimum simplicity - QuantityBox
 * - Maximum simplicity - QuantityBox
 * - Mimimum complexity - QuantityBox
 * - Maximum complexity - QuantityBox
 * - Languages - LanguageInput
 * - Tags - TagSelector
 */
import { FormSchema, InputType } from "forms/types";

export const routineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Routines",
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
            fieldName: "minimumSimplicity",
            label: "Minimum Simplicity",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maximumSimplicity",
            label: "Maximum Simplicity",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minimumComplexity",
            label: "Minimum Complexity",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maximumComplexity",
            label: "Maximum Complexity",
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
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
import { InputType } from "@local/shared";
import { FormSchema } from "forms/types";

export const routineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Routines",
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
            totalItems: 2,
            spacing: 2,
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
            fieldName: "minSimplicity",
            label: "Minimum Simplicity",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxSimplicity",
            label: "Maximum Simplicity",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minComplexity",
            label: "Minimum Complexity",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxComplexity",
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
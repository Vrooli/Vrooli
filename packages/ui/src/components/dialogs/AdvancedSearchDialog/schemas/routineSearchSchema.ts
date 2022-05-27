/**
 * Advanced search form schema for routines. 
 * Can search by: 
 * - Is Complete? - Radio
 * - Min/max stars - QuantityBox
 * - Min/max score - QuantityBox
 * - Min/max simplicity - QuantityBox
 * - Min/max complexity - QuantityBox
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
            title: "Simplicity",
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Complexity",
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
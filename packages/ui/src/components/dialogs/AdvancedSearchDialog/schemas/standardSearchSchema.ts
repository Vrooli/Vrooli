/**
 * Advanced search form schema for standards. 
 * Can search by: 
 * - Minimum stars - QuantityBox
 * - Minimum score - QuantityBox
 * - Langauges - LanguageInput
 * - Tags - TagSelector
 */
import { InputType } from "@local/shared";
import { FormSchema } from "forms/types";

export const standardSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Standards",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            title: "Stars",
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Score",
            totalItems: 1,
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
            fieldName: "minStars",
            label: "Min",
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
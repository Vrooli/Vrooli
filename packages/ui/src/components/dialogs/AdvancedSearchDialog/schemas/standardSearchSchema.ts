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
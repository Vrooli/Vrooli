/**
 * Advanced search form schema for users. 
 * Can search by: 
 * - Minimum stars - QuantityBox
 * - Langauges - LanguageInput
 */
import { InputType } from "@local/shared";
import { FormSchema } from "forms/types";

export const userSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Users",
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
            title: "Languages",
            totalItems: 1
        },
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
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
    ]
}
/**
 * Advanced search form schema for organizations. 
 * Can search by: 
 * - Accepting new members? - Radio
 * - Minimum stars - QuantityBox
 * - Languages - LanguageInput
 * - Tags - TagSelector
 */
import { InputType } from "@shared/consts";
import { FormSchema } from "forms/types";

 export const organizationSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Organizations",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            title: "Stars",
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
            fieldName: "isOpenToNewMembers",
            label: "Accepting new members?",
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
/**
 * Advanced search form schema for organizations. 
 * Can search by: 
 * - Accepting new members? - Radio
 * - Minimum stars - QuantityBox
 * - Languages - LanguageInput
 * - Tags - TagSelector
 */
import { FormSchema, InputType } from "forms/types";

 export const organizationSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Organizations",
        direction: "column",
        rowSpacing: 5,
    },
    fields: [
        {
            fieldName: "isOpenToNewMembers",
            label: "Accepting new members?",
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
                defaultValue: 5,
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
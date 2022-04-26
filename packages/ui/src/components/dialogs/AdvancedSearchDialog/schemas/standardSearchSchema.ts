/**
 * Advanced search form schema for standards. 
 * Can search by: 
 * - Minimum stars - QuantityBox
 * - Minimum score - QuantityBox
 * - Langauges - LanguageInput
 * - Tags - TagSelector
 */
 import { FormSchema, InputType } from "forms/types";

 export const standardSearchSchema: FormSchema = {
     formLayout: {
         title: "Search Standards",
         direction: "column",
         rowSpacing: 5,
     },
     fields: [
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
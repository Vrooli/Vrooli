/**
 * Advanced search form schema for users. 
 * Can search by: 
 * - Minimum stars - QuantityBox
 * - Langauges - LanguageInput
 */
 import { FormSchema, InputType } from "forms/types";

 export const userSearchSchema: FormSchema = {
     formLayout: {
         title: "Search Users",
         direction: "column",
         rowSpacing: 5,
     },
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
             fieldName: "languages",
             label: "Languages",
             type: InputType.LanguageInput,
             props: {},
         },
     ]
 }
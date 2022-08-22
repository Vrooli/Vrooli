/**
 * Advanced search form schema for comments. 
 */
 import { InputType } from "@shared/consts";
 import { FormSchema } from "forms/types";
 
  export const commentSearchSchema: FormSchema = {
     formLayout: {
         title: "Search Comments",
         direction: "column",
         spacing: 4,
     },
     containers: [
        {
            title: "Votes",
            totalItems: 1,
            spacing: 2,
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
     ],
     fields: [
        {
            fieldName: "minVotes",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxVotes",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
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
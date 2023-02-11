import { InputType } from "@shared/consts";
import { FieldData, FormSchema, GridContainer } from "forms/types";
import i18next from "i18next";
import { CommonKey } from "types";

//TODO move to common.json
const starsDescription = `Stars are a way to bookmark an object. They don't affect the ranking of an object in default searches, but are still useful to get a feel for how popular an object is.`;
const votesDescription = `Votes are a way to show support for an object, which affect the ranking of an object in default searches.`;
const languagesDescription = `Filter results by the language(s) they are written in.`;
const tagsDescription = `Filter results by the tags they are associated with.`;
const simplicityDescription = `Simplicity is a mathematical measure of the shortest path to complete a routine. 

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;
const complexityDescription = `Complexity is a mathematical measure of the longest path to complete a routine.

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;


export const searchFormLayout = (title: CommonKey, lng: string): FormSchema['formLayout'] => ({
    title: i18next.t(`common:${title}`, { lng }),
    direction: "column",
    spacing: 4,
})

export const languagesContainer: GridContainer = {
    title: "Languages",
    description: languagesDescription,
    totalItems: 1
}

export const starsContainer: GridContainer = {
    title: "Stars",
    description: starsDescription,
    totalItems: 1,
    spacing: 2,
}

export const tagsContainer: GridContainer = {
    title: "Tags",
    description: tagsDescription,
    totalItems: 1
}

export const votesContainer: GridContainer = {
    title: "Votes",
    description: votesDescription,
    totalItems: 1,
    spacing: 2,
}

export const simplicityContainer: GridContainer = {
    title: "Simplicity",
    description: simplicityDescription,
    totalItems: 2,
    spacing: 2,
}

export const complexityContainer: GridContainer = {
    title: "Complexity",
    description: complexityDescription,
    totalItems: 2,
    spacing: 2,
}

export const isCompleteContainer: GridContainer = { totalItems: 1 }

export const languagesFields: FieldData[] = [
    {
        fieldName: "languages",
        label: "Languages",
        type: InputType.LanguageInput,
        props: {},
    },
]

export const starsFields: FieldData[] = [
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
]

export const tagsFields: FieldData[] = [
    {
        fieldName: "tags",
        label: "Tags",
        type: InputType.TagSelector,
        props: {}
    },
]

export const votesFields: FieldData[] = [
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
]

export const simplicityFields: FieldData[] = [
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
]

export const complexityFields: FieldData[] = [
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
]

export const isCompleteFields: FieldData[] = [
    {
        fieldName: "isComplete",
        label: "Is Complete?",
        type: InputType.Radio,
        props: {
            defaultValue: 'undefined',
            row: true,
            options: [
                { label: "Yes", value: 'true' },
                { label: "No", value: 'false' },
                { label: "Don't Care", value: 'undefined' },
            ]
        }
    },
]
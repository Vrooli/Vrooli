import { InputType } from "@shared/consts";
import { FieldData, FormSchema, GridContainer } from "forms/types";
import i18next from "i18next";
import { CommonKey } from "types";

//TODO move to common.json
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
export const languagesVersionContainer: GridContainer = languagesContainer;

export const bookmarksContainer: GridContainer = {
    title: "Bookmarks",
    totalItems: 1,
    spacing: 2,
}
export const bookmarksRootContainer: GridContainer = bookmarksContainer;

export const tagsContainer: GridContainer = {
    title: "Tags",
    description: tagsDescription,
    totalItems: 1
}
export const tagsRootContainer: GridContainer = tagsContainer;

export const votesContainer: GridContainer = {
    title: "Votes",
    description: votesDescription,
    totalItems: 1,
    spacing: 2,
}
export const votesRootContainer: GridContainer = votesContainer;

export const simplicityContainer: GridContainer = {
    title: "Simplicity",
    description: simplicityDescription,
    totalItems: 2,
    spacing: 2,
}
export const simplicityRootContainer: GridContainer = simplicityContainer;

export const complexityContainer: GridContainer = {
    title: "Complexity",
    description: complexityDescription,
    totalItems: 2,
    spacing: 2,
}
export const complexityRootContainer: GridContainer = complexityContainer;

export const hasCompleteVersionContainer: GridContainer = { totalItems: 1 }

export const isCompleteWithRootContainer: GridContainer = hasCompleteVersionContainer;

export const isLatestContainer: GridContainer = hasCompleteVersionContainer;

export const languagesFields: FieldData[] = [
    {
        fieldName: "translationLanguages",
        label: "Languages",
        type: InputType.LanguageInput,
        props: {},
    },
]
export const languagesVersionFields: FieldData[] = [
    {
        ...languagesFields[0],
        fieldName: "translationLanguagesLatestVersion",
    }
]

export const bookmarksFields: FieldData[] = [
    {
        fieldName: "minBookmarks",
        label: "Min",
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
    {
        fieldName: "maxBookmarks",
        label: "Min",
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
]
export const bookmarksRootFields: FieldData[] = [
    {
        ...bookmarksFields[0],
        fieldName: "minBookmarksRoot",
    },
    {
        ...bookmarksFields[1],
        fieldName: "maxBookmarksRoot",
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
export const tagsRootFields: FieldData[] = [
    {
        ...tagsFields[0],
        fieldName: "tagsRoot",
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
export const votesRootFields: FieldData[] = [
    {
        ...votesFields[0],
        fieldName: "minVotesRoot",
    },
    {
        ...votesFields[1],
        fieldName: "maxVotesRoot",
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
export const simplicityRootFields: FieldData[] = [
    {
        ...simplicityFields[0],
        fieldName: "minSimplicityRoot",
    },
    {
        ...simplicityFields[1],
        fieldName: "maxSimplicityRoot",
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
export const complexityRootFields: FieldData[] = [
    {
        ...complexityFields[0],
        fieldName: "minComplexityRoot",
    },
    {
        ...complexityFields[1],
        fieldName: "maxComplexityRoot",
    },
]

export const hasCompleteVersionFields: FieldData[] = [
    {
        fieldName: "hasCompleteVersion",
        label: "Has complete version?",
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

export const isCompleteWithRootFields: FieldData[] = [
    {
        ...hasCompleteVersionFields[0],
        fieldName: "isCompleteWithRoot",
    },
]

export const isLatestFields: FieldData[] = [
    {
        ...hasCompleteVersionFields[0],
        fieldName: "isLatest",
    },
]
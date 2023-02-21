import { InputType } from "@shared/consts";
import { FieldData, FormSchema, GridContainer } from "forms/types";
import i18next from "i18next";
import { CommonKey } from "types";

export const searchFormLayout = (title: CommonKey, lng: string): FormSchema['formLayout'] => ({
    title: i18next.t(`common:${title}`, { lng }),
    direction: "column",
    spacing: 4,
})

// Containers
export const languagesContainer = (lng: string): GridContainer => ({
    title: i18next.t(`common:Language`, { lng, count: 2 }),
    description: i18next.t(`common:LanguagesHelp`, { lng }),
    totalItems: 1
})
export const languagesVersionContainer = languagesContainer;
export const bookmarksContainer = (lng: string): GridContainer => ({
    title: i18next.t(`common:Bookmark`, { lng, count: 2 }),
    description: i18next.t(`common:BookmarksHelp`, { lng }),
    totalItems: 1,
    spacing: 2,
})
export const bookmarksRootContainer = bookmarksContainer;
export const tagsContainer = (lng: string): GridContainer => ({
    title: i18next.t(`common:Tag`, { lng, count: 2 }),
    description: i18next.t(`common:TagsHelp`, { lng }),
    totalItems: 1
})
export const tagsRootContainer = tagsContainer;
export const votesContainer = (lng: string): GridContainer => ({
    title: i18next.t(`common:Vote`, { lng, count: 2 }),
    description: i18next.t(`common:VotesHelp`, { lng }),
    totalItems: 1,
    spacing: 2,
})
export const votesRootContainer = votesContainer;
export const simplicityContainer = (lng: string): GridContainer => ({
    title: i18next.t(`common:Simplicity`, { lng }),
    description: i18next.t(`common:SimplicityHelp`, { lng }),
    totalItems: 2,
    spacing: 2,
})
export const simplicityRootContainer = simplicityContainer;
export const complexityContainer = (lng: string): GridContainer => ({
    title: i18next.t(`common:Complexity`, { lng }),
    description: i18next.t(`common:ComplexityHelp`, { lng }),
    totalItems: 2,
    spacing: 2,
})
export const complexityRootContainer = complexityContainer;
export const hasCompleteVersionContainer: GridContainer = { totalItems: 1 }
export const isCompleteWithRootContainer: GridContainer = hasCompleteVersionContainer;
export const isLatestContainer: GridContainer = hasCompleteVersionContainer;

// Partial fields
export const yesNoDontCare = (lng: string) => ({
    type: InputType.Radio as const,
    props: {
        defaultValue: 'undefined',
        row: true,
        options: [
            { label: i18next.t(`common:Yes`, { lng }), value: 'true' },
            { label: i18next.t(`common:No`, { lng }), value: 'false' },
            { label: i18next.t(`common:DontCare`, { lng }), value: 'undefined' },
        ]
    }
})

// Fields
export const languagesFields = (lng: string): FieldData[] => ([
    {
        fieldName: "translationLanguages",
        label: i18next.t(`common:Language`, { lng, count: 2 }),
        type: InputType.LanguageInput,
        props: {},
    },
])
export const languagesVersionFields = (lng: string): FieldData[] => ([
    {
        ...languagesFields(lng)[0],
        fieldName: "translationLanguagesLatestVersion",
    }
])
export const bookmarksFields = (lng: string): FieldData[] => ([
    {
        fieldName: "minBookmarks",
        label: i18next.t(`common:Min`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
    {
        fieldName: "maxBookmarks",
        label: i18next.t(`common:Max`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
])
export const bookmarksRootFields = (lng: string): FieldData[] => ([
    {
        ...bookmarksFields(lng)[0],
        fieldName: "minBookmarksRoot",
    },
    {
        ...bookmarksFields(lng)[1],
        fieldName: "maxBookmarksRoot",
    },
])
export const tagsFields = (lng: string): FieldData[] => ([
    {
        fieldName: "tags",
        label: i18next.t(`common:Tag`, { lng, count: 2 }),
        type: InputType.TagSelector,
        props: {}
    },
])
export const tagsRootFields = (lng: string): FieldData[] => ([
    {
        ...tagsFields(lng)[0],
        fieldName: "tagsRoot",
    },
])
export const votesFields = (lng: string): FieldData[] => ([
    {
        fieldName: "minVotes",
        label: i18next.t(`common:Min`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
    {
        fieldName: "maxVotes",
        label: i18next.t(`common:Max`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
])
export const votesRootFields = (lng: string): FieldData[] => ([
    {
        ...votesFields(lng)[0],
        fieldName: "minVotesRoot",
    },
    {
        ...votesFields(lng)[1],
        fieldName: "maxVotesRoot",
    },
])
export const simplicityFields = (lng: string): FieldData[] => ([
    {
        fieldName: "minSimplicity",
        label: i18next.t(`common:Min`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
    {
        fieldName: "maxSimplicity",
        label: i18next.t(`common:Max`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
])
export const simplicityRootFields = (lng: string): FieldData[] => ([
    {
        ...simplicityFields(lng)[0],
        fieldName: "minSimplicityRoot",
    },
    {
        ...simplicityFields(lng)[1],
        fieldName: "maxSimplicityRoot",
    },
])
export const complexityFields = (lng: string): FieldData[] => ([
    {
        fieldName: "minComplexity",
        label: i18next.t(`common:Min`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
    {
        fieldName: "maxComplexity",
        label: i18next.t(`common:Max`, { lng }),
        type: InputType.QuantityBox,
        props: {
            min: 0,
            defaultValue: 0,
        }
    },
])
export const complexityRootFields = (lng: string): FieldData[] => ([
    {
        ...complexityFields(lng)[0],
        fieldName: "minComplexityRoot",
    },
    {
        ...complexityFields(lng)[1],
        fieldName: "maxComplexityRoot",
    },
])
export const hasCompleteVersionFields = (lng: string): FieldData[] => ([
    {
        fieldName: "hasCompleteVersion",
        label: i18next.t(`common:HasCompleteVersion`, { lng }),
        ...yesNoDontCare(lng),
    },
])
export const isCompleteWithRootFields = (lng: string): FieldData[] => ([
    {
        fieldName: "isCompleteWithRoot",
        label: i18next.t(`common:VersionAndRootComplete`, { lng }),
        ...yesNoDontCare(lng),
    },
])
export const isLatestFields = (lng: string): FieldData[] => ([
    {
        fieldName: "isLatest",
        label: i18next.t(`common:IsLatestVersion`, { lng }),
        ...yesNoDontCare(lng),
    },
])
import { CommonKey, InputType } from "@local/shared";
import { FieldData, FormSchema, GridContainer } from "forms/types";
import i18next from "i18next";

export const searchFormLayout = (title: CommonKey): FormSchema["formLayout"] => ({
    title: i18next.t(title),
    direction: "column",
    spacing: 4,
});

// Containers
export const languagesContainer = (): GridContainer => ({
    title: i18next.t("Language", { count: 2 }),
    description: i18next.t("LanguagesHelp"),
    totalItems: 1,
});
export const languagesVersionContainer = languagesContainer;
export const bookmarksContainer = (): GridContainer => ({
    title: i18next.t("Bookmark", { count: 2 }),
    description: i18next.t("BookmarksHelp"),
    totalItems: 2,
    spacing: 2,
});
export const bookmarksRootContainer = bookmarksContainer;
export const tagsContainer = (): GridContainer => ({
    title: i18next.t("Tag", { count: 2 }),
    description: i18next.t("TagsHelp"),
    totalItems: 1,
});
export const tagsRootContainer = tagsContainer;
export const votesContainer = (): GridContainer => ({
    title: i18next.t("Vote", { count: 2 }),
    description: i18next.t("VotesHelp"),
    totalItems: 2,
    spacing: 2,
});
export const votesRootContainer = votesContainer;
export const simplicityContainer = (): GridContainer => ({
    title: i18next.t("Simplicity"),
    description: i18next.t("SimplicityHelp"),
    totalItems: 2,
    spacing: 2,
});
export const simplicityRootContainer = simplicityContainer;
export const complexityContainer = (): GridContainer => ({
    title: i18next.t("Complexity"),
    description: i18next.t("ComplexityHelp"),
    totalItems: 2,
    spacing: 2,
});
export const complexityRootContainer = complexityContainer;
export const hasCompleteVersionContainer: GridContainer = { totalItems: 1 };
export const isCompleteWithRootContainer: GridContainer = hasCompleteVersionContainer;
export const isLatestContainer: GridContainer = hasCompleteVersionContainer;

// Partial fields
export const yesNoDontCare = () => ({
    type: InputType.Radio as const,
    props: {
        defaultValue: "undefined",
        row: true,
        options: [
            { label: i18next.t("Yes"), value: "true" },
            { label: i18next.t("No"), value: "false" },
            { label: i18next.t("DontCare"), value: "undefined" },
        ],
    },
});

// Fields
export const languagesFields = (): FieldData[] => ([
    {
        fieldName: "translationLanguages",
        label: i18next.t("Language", { count: 2 }),
        type: InputType.LanguageInput,
        props: {},
    },
]);
export const languagesVersionFields = (): FieldData[] => ([
    {
        ...languagesFields()[0],
        fieldName: "translationLanguagesLatestVersion",
    },
]);
export const bookmarksFields = (): FieldData[] => ([
    {
        fieldName: "minBookmarks",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MinNo"),
        },
    },
    {
        fieldName: "maxBookmarks",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const bookmarksRootFields = (): FieldData[] => ([
    {
        ...bookmarksFields()[0],
        fieldName: "minBookmarksRoot",
    },
    {
        ...bookmarksFields()[1],
        fieldName: "maxBookmarksRoot",
    },
]);
export const tagsFields = (): FieldData[] => ([
    {
        fieldName: "tags",
        label: i18next.t("Tag", { count: 2 }),
        type: InputType.TagSelector,
        props: {},
    },
]);
export const tagsRootFields = (): FieldData[] => ([
    {
        ...tagsFields()[0],
        fieldName: "tagsRoot",
    },
]);
export const votesFields = (): FieldData[] => ([
    {
        fieldName: "minVotes",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MinNo"),
        },
    },
    {
        fieldName: "maxVotes",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const votesRootFields = (): FieldData[] => ([
    {
        ...votesFields()[0],
        fieldName: "minVotesRoot",
    },
    {
        ...votesFields()[1],
        fieldName: "maxVotesRoot",
    },
]);
export const simplicityFields = (): FieldData[] => ([
    {
        fieldName: "minSimplicity",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MinNo"),
        },
    },
    {
        fieldName: "maxSimplicity",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const simplicityRootFields = (): FieldData[] => ([
    {
        ...simplicityFields()[0],
        fieldName: "minSimplicityRoot",
    },
    {
        ...simplicityFields()[1],
        fieldName: "maxSimplicityRoot",
    },
]);
export const complexityFields = (): FieldData[] => ([
    {
        fieldName: "minComplexity",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MinNo"),
        },
    },
    {
        fieldName: "maxComplexity",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const complexityRootFields = (): FieldData[] => ([
    {
        ...complexityFields()[0],
        fieldName: "minComplexityRoot",
    },
    {
        ...complexityFields()[1],
        fieldName: "maxComplexityRoot",
    },
]);
export const hasCompleteVersionFields = (): FieldData[] => ([
    {
        fieldName: "hasCompleteVersion",
        label: i18next.t("HasCompleteVersion"),
        ...yesNoDontCare(),
    },
]);
export const isCompleteWithRootFields = (): FieldData[] => ([
    {
        fieldName: "isCompleteWithRoot",
        label: i18next.t("VersionAndRootComplete"),
        ...yesNoDontCare(),
    },
]);
export const isLatestFields = (): FieldData[] => ([
    {
        fieldName: "isLatest",
        label: i18next.t("IsLatestVersion"),
        ...yesNoDontCare(),
    },
]);

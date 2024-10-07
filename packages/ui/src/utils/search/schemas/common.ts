/* eslint-disable func-style */
import { FormInputType, FormSchema, GridContainer, InputType, TranslationKeyCommon } from "@local/shared";
import i18next from "i18next";

export const searchFormLayout = (title: TranslationKeyCommon): FormSchema["layout"] => ({
    title: i18next.t(title),
});

// Containers
export const languagesContainer = (): GridContainer => ({
    direction: "column",
    disableCollapse: true,
    totalItems: 1,
});
export const languagesVersionContainer = languagesContainer;
export const bookmarksContainer = (): GridContainer => ({
    title: i18next.t("Bookmark", { count: 2 }),
    description: i18next.t("BookmarksHelp"),
    direction: "row",
    disableCollapse: true,
    totalItems: 2,
});
export const bookmarksRootContainer = bookmarksContainer;
export const tagsContainer = (): GridContainer => ({
    direction: "column",
    disableCollapse: true,
    totalItems: 1,
});
export const tagsRootContainer = tagsContainer;
export const votesContainer = (): GridContainer => ({
    title: i18next.t("Vote", { count: 2 }),
    description: i18next.t("VotesHelp"),
    direction: "row",
    disableCollapse: true,
    totalItems: 2,
});
export const votesRootContainer = votesContainer;
export const simplicityContainer = (): GridContainer => ({
    title: i18next.t("Simplicity"),
    description: i18next.t("SimplicityHelp"),
    direction: "row",
    disableCollapse: true,
    totalItems: 2,
});
export const simplicityRootContainer = simplicityContainer;
export const complexityContainer = (): GridContainer => ({
    title: i18next.t("Complexity"),
    description: i18next.t("ComplexityHelp"),
    direction: "row",
    disableCollapse: true,
    totalItems: 2,
});
export const complexityRootContainer = complexityContainer;
export const hasCompleteVersionContainer: GridContainer = {
    direction: "column",
    disableCollapse: true,
    totalItems: 1,
};
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
export function languagesFields(): FormInputType[] {
    return [
        {
            fieldName: "translationLanguages",
            id: "translationLanguages",
            label: i18next.t("Language", { count: 2 }),
            type: InputType.LanguageInput,
            props: {},
        },
    ];
}
export const languagesVersionFields = (): FormInputType[] => ([
    {
        ...languagesFields()[0],
        fieldName: "translationLanguagesLatestVersion",
    },
]);
export const bookmarksFields = (): FormInputType[] => ([
    {
        fieldName: "minBookmarks",
        id: "minBookmarks",
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
        id: "maxBookmarks",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const bookmarksRootFields = (): FormInputType[] => ([
    {
        ...bookmarksFields()[0],
        fieldName: "minBookmarksRoot",
        id: "minBookmarksRoot",
    },
    {
        ...bookmarksFields()[1],
        fieldName: "maxBookmarksRoot",
        id: "maxBookmarksRoot",
    },
]);
export const tagsFields = (): FormInputType[] => ([
    {
        fieldName: "tags",
        id: "tags",
        label: i18next.t("Tag", { count: 2 }),
        type: InputType.TagSelector,
        props: {},
    },
]);
export const tagsRootFields = (): FormInputType[] => ([
    {
        ...tagsFields()[0],
        fieldName: "tagsRoot",
        id: "tagsRoot",
    },
]);
export const votesFields = (): FormInputType[] => ([
    {
        fieldName: "minVotes",
        id: "minVotes",
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
        id: "maxVotes",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const votesRootFields = (): FormInputType[] => ([
    {
        ...votesFields()[0],
        fieldName: "minVotesRoot",
        id: "minVotesRoot",
    },
    {
        ...votesFields()[1],
        fieldName: "maxVotesRoot",
        id: "maxVotesRoot",
    },
]);
export const simplicityFields = (): FormInputType[] => ([
    {
        fieldName: "minSimplicity",
        id: "minSimplicity",
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
        id: "maxSimplicity",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const simplicityRootFields = (): FormInputType[] => ([
    {
        ...simplicityFields()[0],
        fieldName: "minSimplicityRoot",
    },
    {
        ...simplicityFields()[1],
        fieldName: "maxSimplicityRoot",
    },
]);
export const complexityFields = (): FormInputType[] => ([
    {
        fieldName: "minComplexity",
        id: "minComplexity",
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
        id: "maxComplexity",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
            zeroText: i18next.t("MaxNo"),
        },
    },
]);
export const complexityRootFields = (): FormInputType[] => ([
    {
        ...complexityFields()[0],
        fieldName: "minComplexityRoot",
        id: "minComplexityRoot",
    },
    {
        ...complexityFields()[1],
        fieldName: "maxComplexityRoot",
        id: "maxComplexityRoot",
    },
]);
export const hasCompleteVersionFields = (): FormInputType[] => ([
    {
        fieldName: "hasCompleteVersion",
        id: "hasCompleteVersion",
        label: i18next.t("HasCompleteVersion"),
        ...yesNoDontCare(),
    },
]);
export const isCompleteWithRootFields = (): FormInputType[] => ([
    {
        fieldName: "isCompleteWithRoot",
        id: "isCompleteWithRoot",
        label: i18next.t("VersionAndRootComplete"),
        ...yesNoDontCare(),
    },
]);
export const isLatestFields = (): FormInputType[] => ([
    {
        fieldName: "isLatest",
        id: "isLatest",
        label: i18next.t("IsLatestVersion"),
        ...yesNoDontCare(),
    },
]);

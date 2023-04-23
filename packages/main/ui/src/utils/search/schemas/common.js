import { InputType } from "@local/consts";
import i18next from "i18next";
export const searchFormLayout = (title) => ({
    title: i18next.t(title),
    direction: "column",
    spacing: 4,
});
export const languagesContainer = () => ({
    title: i18next.t("Language", { count: 2 }),
    description: i18next.t("LanguagesHelp"),
    totalItems: 1,
});
export const languagesVersionContainer = languagesContainer;
export const bookmarksContainer = () => ({
    title: i18next.t("Bookmark", { count: 2 }),
    description: i18next.t("BookmarksHelp"),
    totalItems: 2,
    spacing: 2,
});
export const bookmarksRootContainer = bookmarksContainer;
export const tagsContainer = () => ({
    title: i18next.t("Tag", { count: 2 }),
    description: i18next.t("TagsHelp"),
    totalItems: 1,
});
export const tagsRootContainer = tagsContainer;
export const votesContainer = () => ({
    title: i18next.t("Vote", { count: 2 }),
    description: i18next.t("VotesHelp"),
    totalItems: 2,
    spacing: 2,
});
export const votesRootContainer = votesContainer;
export const simplicityContainer = () => ({
    title: i18next.t("Simplicity"),
    description: i18next.t("SimplicityHelp"),
    totalItems: 2,
    spacing: 2,
});
export const simplicityRootContainer = simplicityContainer;
export const complexityContainer = () => ({
    title: i18next.t("Complexity"),
    description: i18next.t("ComplexityHelp"),
    totalItems: 2,
    spacing: 2,
});
export const complexityRootContainer = complexityContainer;
export const hasCompleteVersionContainer = { totalItems: 1 };
export const isCompleteWithRootContainer = hasCompleteVersionContainer;
export const isLatestContainer = hasCompleteVersionContainer;
export const yesNoDontCare = () => ({
    type: InputType.Radio,
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
export const languagesFields = () => ([
    {
        fieldName: "translationLanguages",
        label: i18next.t("Language", { count: 2 }),
        type: InputType.LanguageInput,
        props: {},
    },
]);
export const languagesVersionFields = () => ([
    {
        ...languagesFields()[0],
        fieldName: "translationLanguagesLatestVersion",
    },
]);
export const bookmarksFields = () => ([
    {
        fieldName: "minBookmarks",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
    {
        fieldName: "maxBookmarks",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
]);
export const bookmarksRootFields = () => ([
    {
        ...bookmarksFields()[0],
        fieldName: "minBookmarksRoot",
    },
    {
        ...bookmarksFields()[1],
        fieldName: "maxBookmarksRoot",
    },
]);
export const tagsFields = () => ([
    {
        fieldName: "tags",
        label: i18next.t("Tag", { count: 2 }),
        type: InputType.TagSelector,
        props: {},
    },
]);
export const tagsRootFields = () => ([
    {
        ...tagsFields()[0],
        fieldName: "tagsRoot",
    },
]);
export const votesFields = () => ([
    {
        fieldName: "minVotes",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
    {
        fieldName: "maxVotes",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
]);
export const votesRootFields = () => ([
    {
        ...votesFields()[0],
        fieldName: "minVotesRoot",
    },
    {
        ...votesFields()[1],
        fieldName: "maxVotesRoot",
    },
]);
export const simplicityFields = () => ([
    {
        fieldName: "minSimplicity",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
    {
        fieldName: "maxSimplicity",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
]);
export const simplicityRootFields = () => ([
    {
        ...simplicityFields()[0],
        fieldName: "minSimplicityRoot",
    },
    {
        ...simplicityFields()[1],
        fieldName: "maxSimplicityRoot",
    },
]);
export const complexityFields = () => ([
    {
        fieldName: "minComplexity",
        label: i18next.t("Min"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
    {
        fieldName: "maxComplexity",
        label: i18next.t("Max"),
        type: InputType.IntegerInput,
        props: {
            min: 0,
            defaultValue: 0,
        },
    },
]);
export const complexityRootFields = () => ([
    {
        ...complexityFields()[0],
        fieldName: "minComplexityRoot",
    },
    {
        ...complexityFields()[1],
        fieldName: "maxComplexityRoot",
    },
]);
export const hasCompleteVersionFields = () => ([
    {
        fieldName: "hasCompleteVersion",
        label: i18next.t("HasCompleteVersion"),
        ...yesNoDontCare(),
    },
]);
export const isCompleteWithRootFields = () => ([
    {
        fieldName: "isCompleteWithRoot",
        label: i18next.t("VersionAndRootComplete"),
        ...yesNoDontCare(),
    },
]);
export const isLatestFields = () => ([
    {
        fieldName: "isLatest",
        label: i18next.t("IsLatestVersion"),
        ...yesNoDontCare(),
    },
]);
//# sourceMappingURL=common.js.map
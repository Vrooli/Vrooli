import { StandardVersion, StandardVersionTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";
import { versionYou } from "./root";

export const standardVersionTranslation: GqlPartial<StandardVersionTranslation> = {
    __typename: "StandardVersionTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        jsonVariable: true,
        name: true,
    },
    full: {},
    list: {},
};

export const standardVersion: GqlPartial<StandardVersion> = {
    __typename: "StandardVersion",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        codeLanguage: true,
        default: true,
        isComplete: true,
        isFile: true,
        isLatest: true,
        isPrivate: true,
        props: true,
        variant: true,
        versionIndex: true,
        versionLabel: true,
        yup: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        you: () => rel(versionYou, "full"),
    },
    full: {
        versionNotes: true,
        pullRequest: async () => rel((await import("./pullRequest")).pullRequest, "full", { omit: ["from", "to"] }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "common"),
        root: async () => rel((await import("./standard")).standard, "full", { omit: "versions" }),
        translations: () => rel(standardVersionTranslation, "full"),
    },
    list: {
        root: async () => rel((await import("./standard")).standard, "list", { omit: "versions" }),
        translations: () => rel(standardVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./standard")).standard, "nav", { omit: "versions" }),
        translations: () => rel(standardVersionTranslation, "list"),
    },
};

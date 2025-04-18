import { StandardVersion, StandardVersionTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";
import { versionYou } from "./root.js";

export const standardVersionTranslation: ApiPartial<StandardVersionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        jsonVariable: true,
        name: true,
    },
};

export const standardVersion: ApiPartial<StandardVersion> = {
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
        pullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "full", { omit: ["from", "to"] }),
        resourceList: async () => rel((await import("./resourceList.js")).resourceList, "common"),
        root: async () => rel((await import("./standard.js")).standard, "full", { omit: "versions" }),
        translations: () => rel(standardVersionTranslation, "full"),
    },
    list: {
        root: async () => rel((await import("./standard.js")).standard, "list", { omit: "versions" }),
        translations: () => rel(standardVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./standard.js")).standard, "nav", { omit: "versions" }),
        translations: () => rel(standardVersionTranslation, "list"),
    },
};

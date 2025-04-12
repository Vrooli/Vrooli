import { CodeVersion, CodeVersionTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";
import { versionYou } from "./root.js";

export const codeVersionTranslation: ApiPartial<CodeVersionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        jsonVariable: true,
        name: true,
    },
};

export const codeVersion: ApiPartial<CodeVersion> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        codeLanguage: true,
        codeType: true,
        default: true,
        versionIndex: true,
        versionLabel: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        you: () => rel(versionYou, "full"),
    },
    full: {
        calledByRoutineVersionsCount: true,
        content: true,
        versionNotes: true,
        pullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "full", { omit: ["from", "to"] }),
        resourceList: async () => rel((await import("./resourceList.js")).resourceList, "common"),
        root: async () => rel((await import("./code.js")).code, "full", { omit: "versions" }),
        translations: () => rel(codeVersionTranslation, "full"),
    },
    list: {
        root: async () => rel((await import("./code.js")).code, "list", { omit: "versions" }),
        translations: () => rel(codeVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./code.js")).code, "nav", { omit: "versions" }),
        translations: () => rel(codeVersionTranslation, "list"),
    },
};

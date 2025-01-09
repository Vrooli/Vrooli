import { CodeVersion, CodeVersionTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";
import { versionYou } from "./root";

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
        calledByRoutineVersionsCount: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        you: () => rel(versionYou, "full"),
    },
    full: {
        content: true,
        versionNotes: true,
        pullRequest: async () => rel((await import("./pullRequest")).pullRequest, "full", { omit: ["from", "to"] }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "common"),
        root: async () => rel((await import("./code")).code, "full", { omit: "versions" }),
        translations: () => rel(codeVersionTranslation, "full"),
    },
    list: {
        root: async () => rel((await import("./code")).code, "list", { omit: "versions" }),
        translations: () => rel(codeVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./code")).code, "nav", { omit: "versions" }),
        translations: () => rel(codeVersionTranslation, "list"),
    },
};

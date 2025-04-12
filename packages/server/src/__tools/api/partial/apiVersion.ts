import { ApiVersion, ApiVersionTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";
import { versionYou } from "./root.js";

export const apiVersionTranslation: ApiPartial<ApiVersionTranslation> = {
    common: {
        id: true,
        language: true,
        details: true,
        name: true,
        summary: true,
    },
};

export const apiVersion: ApiPartial<ApiVersion> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        callLink: true,
        commentsCount: true,
        documentationLink: true,
        forksCount: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        versionIndex: true,
        versionLabel: true,
        you: () => rel(versionYou, "full"),
    },
    full: {
        calledByRoutineVersionsCount: true,
        schemaLanguage: true,
        schemaText: true,
        versionNotes: true,
        pullRequest: async () => rel((await import("./pullRequest.js")).pullRequest, "full", { omit: ["from", "to"] }),
        root: async () => rel((await import("./api.js")).api, "full", { omit: "versions" }),
        translations: () => rel(apiVersionTranslation, "full"),
    },
    list: {
        schemaLanguage: true,
        root: async () => rel((await import("./api.js")).api, "list", { omit: "versions" }),
        translations: () => rel(apiVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./api.js")).api, "nav", { omit: "versions" }),
        translations: () => rel(apiVersionTranslation, "list"),
    },
};

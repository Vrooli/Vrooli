import { ApiVersion, ApiVersionTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";
import { versionYou } from "./root";

export const apiVersionTranslation: GqlPartial<ApiVersionTranslation> = {
    __typename: "ApiVersionTranslation",
    common: {
        id: true,
        language: true,
        details: true,
        name: true,
        summary: true,
    },
    full: {},
    list: {},
};

export const apiVersion: GqlPartial<ApiVersion> = {
    __typename: "ApiVersion",
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
        pullRequest: async () => rel((await import("./pullRequest")).pullRequest, "full", { omit: ["from", "to"] }),
        root: async () => rel((await import("./api")).api, "full", { omit: "versions" }),
        translations: () => rel(apiVersionTranslation, "full"),
        schemaText: true,
        versionNotes: true,
    },
    list: {
        root: async () => rel((await import("./api")).api, "list", { omit: "versions" }),
        translations: () => rel(apiVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./api")).api, "nav", { omit: "versions" }),
        translations: () => rel(apiVersionTranslation, "list"),
    },
};

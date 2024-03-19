import { SmartContractVersion, SmartContractVersionTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";
import { versionYou } from "./root";

export const smartContractVersionTranslation: GqlPartial<SmartContractVersionTranslation> = {
    __typename: "SmartContractVersionTranslation",
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

export const smartContractVersion: GqlPartial<SmartContractVersion> = {
    __typename: "SmartContractVersion",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        default: true,
        contractType: true,
        content: true,
        versionIndex: true,
        versionLabel: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        you: () => rel(versionYou, "full"),
    },
    full: {
        versionNotes: true,
        pullRequest: async () => rel((await import("./pullRequest")).pullRequest, "full", { omit: ["from", "to"] }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "full"),
        root: async () => rel((await import("./smartContract")).smartContract, "full", { omit: "versions" }),
        translations: () => rel(smartContractVersionTranslation, "full"),
    },
    list: {
        root: async () => rel((await import("./smartContract")).smartContract, "list", { omit: "versions" }),
        translations: () => rel(smartContractVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./smartContract")).smartContract, "nav", { omit: "versions" }),
        translations: () => rel(smartContractVersionTranslation, "list"),
    },
};

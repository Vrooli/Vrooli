import { rel } from "../utils";
import { versionYou } from "./root";
export const smartContractVersionTranslation = {
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
export const smartContractVersion = {
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
        translations: () => rel(smartContractVersionTranslation, "list"),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import("./smartContract")).smartContract, "nav"),
        translations: () => rel(smartContractVersionTranslation, "list"),
    },
};
//# sourceMappingURL=smartContractVersion.js.map
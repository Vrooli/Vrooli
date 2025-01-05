import { RoutineVersion, RoutineVersionTranslation, RoutineVersionYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const routineVersionTranslation: GqlPartial<RoutineVersionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        instructions: true,
        name: true,
    },
};

export const routineVersionYou: GqlPartial<RoutineVersionYou> = {
    common: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canBookmark: true,
        canReport: true,
        canRun: true,
        canUpdate: true,
        canRead: true,
        canReact: true,
    },
};

export const routineVersion: GqlPartial<RoutineVersion> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        completedAt: true,
        isAutomatable: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        routineType: true,
        simplicity: true,
        timesStarted: true,
        timesCompleted: true,
        versionIndex: true,
        versionLabel: true,
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        inputsCount: true,
        nodesCount: true,
        nodeLinksCount: true,
        outputsCount: true,
        reportsCount: true,
        you: () => rel(routineVersionYou, "common"),
    },
    full: {
        configCallData: true,
        configFormInput: true,
        configFormOutput: true,
        versionNotes: true,
        apiVersion: async () => rel((await import("./apiVersion")).apiVersion, "full"),
        codeVersion: async () => rel((await import("./codeVersion")).codeVersion, "full"),
        inputs: async () => rel((await import("./routineVersionInput")).routineVersionInput, "full"),
        nodes: async () => rel((await import("./node")).node, "full", { omit: "routineVersion" }),
        nodeLinks: async () => rel((await import("./nodeLink")).nodeLink, "full"),
        outputs: async () => rel((await import("./routineVersionOutput")).routineVersionOutput, "full"),
        pullRequest: async () => rel((await import("./pullRequest")).pullRequest, "full", { omit: ["from", "to"] }),
        resourceList: async () => rel((await import("./resourceList")).resourceList, "common"),
        root: async () => rel((await import("./routine")).routine, "full", { omit: "versions" }),
        suggestedNextByRoutineVersion: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
        translations: () => rel(routineVersionTranslation, "full"),
    },
    list: {
        root: async () => rel((await import("./routine")).routine, "list", { omit: "versions" }),
        translations: () => rel(routineVersionTranslation, "list"),
    },
    nav: {
        id: true,
        complexity: true, // Used by RunRoutine
        isAutomatable: true,
        isComplete: true,
        isDeleted: true,
        isLatest: true,
        isPrivate: true,
        root: async () => rel((await import("./routine")).routine, "nav", { omit: "versions" }),
        routineType: true,
        translations: () => rel(routineVersionTranslation, "list"),
        versionIndex: true,
        versionLabel: true,
    },
};

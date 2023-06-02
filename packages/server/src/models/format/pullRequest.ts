import { PullRequestModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "PullRequest" as const;
export const PullRequestFormat: Formatter<PullRequestModelLogic> = {
    gqlRelMap: {
        __typename,
        createdBy: "User",
        comments: "Comment",
        from: {
            fromApiVersion: "ApiVersion",
            fromNoteVersion: "NoteVersion",
            fromProjectVersion: "ProjectVersion",
            fromRoutineVersion: "RoutineVersion",
            fromSmartContractVersion: "SmartContractVersion",
            fromStandardVersion: "StandardVersion",
        },
        to: {
            toApi: "Api",
            toNote: "Note",
            toProject: "Project",
            toRoutine: "Routine",
            toSmartContract: "SmartContract",
            toStandard: "Standard",
        },
    },
    prismaRelMap: {
        __typename,
        fromApiVersion: "ApiVersion",
        fromNoteVersion: "NoteVersion",
        fromProjectVersion: "ProjectVersion",
        fromRoutineVersion: "RoutineVersion",
        fromSmartContractVersion: "SmartContractVersion",
        fromStandardVersion: "StandardVersion",
        toApi: "Api",
        toNote: "Note",
        toProject: "Project",
        toRoutine: "Routine",
        toSmartContract: "SmartContract",
        toStandard: "Standard",
        createdBy: "User",
        comments: "Comment",
    },
    countFields: {
        commentsCount: true,
        translationsCount: true,
    },
};

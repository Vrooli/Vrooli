import { noNull } from "../../builders";
import { translationShapeHelper } from "../../utils";
import { Formatter } from "../types";

const __typename = "PullRequest" as const;
export const PullRequestFormat: Formatter<ModelPullRequestLogic> = {
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
}'Canceled'.
    // Owner of object that pull request is on can set status to anything but 'Canceled'
    // TODO need to update params for shape to account for this (probably). Then need to update this function
    update: async ({ data, ...rest }) => ({
        status: noNull(data.status),
        ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
        if(data.createdBy?.id === userId) return({
            User: data.createdBy,
    };

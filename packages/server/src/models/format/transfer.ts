import { TransferModelLogic } from "../base/types";
import { Formatter } from "../types";

export const TransferFormat: Formatter<TransferModelLogic> = {
    gqlRelMap: {
        __typename: "Transfer",
        fromOwner: {
            fromUser: "User",
            fromOrganization: "Organization",
        },
        object: {
            api: "Api",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
        toOwner: {
            toUser: "User",
            toOrganization: "Organization",
        },
    },
    prismaRelMap: {
        __typename: "Transfer",
        fromUser: "User",
        fromOrganization: "Organization",
        toUser: "User",
        toOrganization: "Organization",
        api: "Api",
        note: "Note",
        project: "Project",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
    },
    countFields: {},
};

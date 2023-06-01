import { Formatter } from "../types";

const __typename = "Transfer" as const;
export const TransferFormat: Formatter<ModelTransferLogic> = {
    gqlRelMap: {
        __typename,
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
        __typename,
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

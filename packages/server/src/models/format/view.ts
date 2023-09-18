import { ViewModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ViewFormat: Formatter<ViewModelLogic> = {
    gqlRelMap: {
        __typename: "View",
        by: "User",
        to: {
            api: "Api",
            issue: "Issue",
            note: "Note",
            organization: "Organization",
            post: "Post",
            project: "Project",
            question: "Question",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            user: "User",
        },
    },
    prismaRelMap: {
        __typename: "View",
        by: "User",
        api: "Api",
        issue: "Issue",
        note: "Note",
        organization: "Organization",
        post: "Post",
        project: "Project",
        question: "Question",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        user: "User",
    },
    countFields: {},
};

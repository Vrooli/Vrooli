import { TagModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Tag" as const;
export const TagFormat: Formatter<TagModelLogic> = {
    gqlRelMap: {
        __typename,
        apis: "Api",
        notes: "Note",
        organizations: "Organization",
        posts: "Post",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        smartContracts: "SmartContract",
        standards: "Standard",
        bookmarkedBy: "User",
    },
    prismaRelMap: {
        __typename,
        createdBy: "User",
        apis: "Api",
        notes: "Note",
        organizations: "Organization",
        posts: "Post",
        projects: "Project",
        reports: "Report",
        routines: "Routine",
        smartContracts: "SmartContract",
        standards: "Standard",
        bookmarkedBy: "User",
        focusModeFilters: "FocusModeFilter",
    },
    joinMap: {
        apis: "tagged",
        notes: "tagged",
        organizations: "tagged",
        posts: "tagged",
        projects: "tagged",
        reports: "tagged",
        routines: "tagged",
        smartContracts: "tagged",
        standards: "tagged",
        bookmarkedBy: "user",
    },
    countFields: {},
};

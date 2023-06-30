import { LabelModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Label" as const;
export const LabelFormat: Formatter<LabelModelLogic> = {
    gqlRelMap: {
        __typename,
        apis: "Api",
        focusModes: "FocusMode",
        issues: "Issue",
        meetings: "Meeting",
        notes: "Note",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        projects: "Project",
        routines: "Routine",
        schedules: "Schedule",
    },
    prismaRelMap: {
        __typename,
        apis: "Api",
        focusModes: "FocusMode",
        issues: "Issue",
        meetings: "Meeting",
        notes: "Note",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        projects: "Project",
        routines: "Routine",
        schedules: "Schedule",
    },
    joinMap: {
        apis: "labelled",
        focusModes: "labelled",
        issues: "labelled",
        meetings: "labelled",
        notes: "labelled",
        projects: "labelled",
        routines: "labelled",
        schedules: "labelled",
    },
    countFields: {
        apisCount: true,
        focusModesCount: true,
        issuesCount: true,
        meetingsCount: true,
        notesCount: true,
        projectsCount: true,
        smartContractsCount: true,
        standardsCount: true,
        routinesCount: true,
        schedulesCount: true,
        translationsCount: true,
    },
};

import { rel } from "../utils";
export const labelYou = {
    __typename: "LabelYou",
    full: {
        canDelete: true,
        canUpdate: true,
    },
};
export const label = {
    __typename: "Label",
    common: {
        __define: {
            0: async () => rel((await import("./organization")).organization, "nav"),
            1: async () => rel((await import("./user")).user, "nav"),
        },
        id: true,
        created_at: true,
        updated_at: true,
        color: true,
        label: true,
        owner: {
            __union: {
                Organization: 0,
                User: 1,
            },
        },
        you: () => rel(labelYou, "full"),
    },
    full: {
        apisCount: true,
        focusModesCount: true,
        issuesCount: true,
        meetingsCount: true,
        notesCount: true,
        projectsCount: true,
        routinesCount: true,
        schedulesCount: true,
        smartContractsCount: true,
        standardsCount: true,
    },
    list: {},
};
//# sourceMappingURL=label.js.map
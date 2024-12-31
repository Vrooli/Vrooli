import { Label, LabelYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const labelYou: GqlPartial<LabelYou> = {
    __typename: "LabelYou",
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const label: GqlPartial<Label> = {
    __typename: "Label",
    common: {
        __define: {
            0: async () => rel((await import("./team")).team, "nav"),
            1: async () => rel((await import("./user")).user, "nav"),
        },
        id: true,
        created_at: true,
        updated_at: true,
        color: true,
        label: true,
        owner: {
            __union: {
                Team: 0,
                User: 1,
            },
        },
        you: () => rel(labelYou, "full"),
    },
    full: {
        apisCount: true,
        codesCount: true,
        focusModesCount: true,
        issuesCount: true,
        meetingsCount: true,
        notesCount: true,
        projectsCount: true,
        routinesCount: true,
        schedulesCount: true,
        standardsCount: true,
    },
    list: {},
};

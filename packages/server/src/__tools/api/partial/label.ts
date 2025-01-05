import { Label, LabelYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const labelYou: GqlPartial<LabelYou> = {
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const label: GqlPartial<Label> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        color: true,
        label: true,
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
};

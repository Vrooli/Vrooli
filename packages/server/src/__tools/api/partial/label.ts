import { Label, LabelYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const labelYou: ApiPartial<LabelYou> = {
    full: {
        canDelete: true,
        canUpdate: true,
    },
};

export const label: ApiPartial<Label> = {
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

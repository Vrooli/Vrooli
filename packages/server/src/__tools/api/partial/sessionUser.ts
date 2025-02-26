import { SessionUser } from "@local/shared";
import { ApiPartial } from "../types.js";

export const sessionUser: ApiPartial<SessionUser> = {
    full: {
        apisCount: true,
        codesCount: true,
        handle: true,
        hasPremium: true,
        id: true,
        languages: true,
        membershipsCount: true,
        name: true,
        notesCount: true,
        projectsCount: true,
        questionsAskedCount: true,
        routinesCount: true,
        standardsCount: true,
        theme: true,
    },
};

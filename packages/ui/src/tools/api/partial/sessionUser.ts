import { SessionUser } from "@local/shared";
import { GqlPartial } from "../types";

export const sessionUser: GqlPartial<SessionUser> = {
    __typename: "SessionUser",
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

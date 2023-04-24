import { SessionUser } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const sessionUser: GqlPartial<SessionUser> = {
    __typename: "SessionUser",
    full: {
        activeFocusMode: async () => rel((await import("./activeFocusMode")).activeFocusMode, "full"),
        apisCount: true,
        bookmarkLists: async () => rel((await import("./bookmarkList")).bookmarkList, "common"),
        focusModes: async () => rel((await import("./focusMode")).focusMode, "full"),
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
        smartContractsCount: true,
        standardsCount: true,
        theme: true,
    },
};

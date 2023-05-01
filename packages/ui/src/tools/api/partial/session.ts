import { Session } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const session: GqlPartial<Session> = {
    __typename: "Session",
    full: {
        isLoggedIn: true,
        timeZone: true,
        users: async () => rel((await import("./sessionUser")).sessionUser, "full"),
    },
};

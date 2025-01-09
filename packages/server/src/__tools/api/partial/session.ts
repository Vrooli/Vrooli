import { Session } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const session: ApiPartial<Session> = {
    full: {
        isLoggedIn: true,
        timeZone: true,
        users: async () => rel((await import("./sessionUser")).sessionUser, "full"),
    },
};

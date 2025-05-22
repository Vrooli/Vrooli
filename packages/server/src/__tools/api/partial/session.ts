import { type Session } from "@local/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const session: ApiPartial<Session> = {
    full: {
        isLoggedIn: true,
        timeZone: true,
        users: async () => rel((await import("./sessionUser.js")).sessionUser, "full"),
    },
};

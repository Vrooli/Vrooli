import { rel } from "../utils";
export const session = {
    __typename: "Session",
    full: {
        isLoggedIn: true,
        timeZone: true,
        users: async () => rel((await import("./sessionUser")).sessionUser, "full"),
    },
};
//# sourceMappingURL=session.js.map
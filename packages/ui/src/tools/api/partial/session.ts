import { Session } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const session: GqlPartial<Session> = {
    __typename: 'Session',
    full: {
        isLoggedIn: true,
        timeZone: true,
        users: async () => rel((await import('./sessionUser')).sessionUser, 'full'),
    }
}
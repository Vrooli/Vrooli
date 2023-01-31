import { Session } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const sessionPartial: GqlPartial<Session> = {
    __typename: 'Session',
    full: {
        isLoggedIn: true,
        timeZone: true,
        users: async () => relPartial((await import('./sessionUser')).sessionUserPartial, 'full'),
    }
}
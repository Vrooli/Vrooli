import { Session } from "@shared/consts";
import { GqlPartial } from "types";
import { sessionUserPartial } from "./sessionUser";

export const sessionPartial: GqlPartial<Session> = {
    __typename: 'Session',
    full: {
        isLoggedIn: true,
        timeZone: true,
        users: sessionUserPartial.full,
    }
}
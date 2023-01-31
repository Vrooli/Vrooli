import { SessionUser } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const sessionUser: GqlPartial<SessionUser> = {
    __typename: 'SessionUser',
    full: {
        handle: true,
        hasPremium: true,
        id: true,
        languages: true,
        name: true,
        schedules: async () => rel((await import('./userSchedule')).userSchedule, 'full'),
        theme: true,
    }
}
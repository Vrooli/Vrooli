import { SessionUser } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const sessionUser: GqlPartial<SessionUser> = {
    __typename: 'SessionUser',
    full: {
        focusModes: async () => rel((await import('./focusMode')).focusMode, 'full'),
        handle: true,
        hasPremium: true,
        id: true,
        languages: true,
        name: true,
        theme: true,
    }
}
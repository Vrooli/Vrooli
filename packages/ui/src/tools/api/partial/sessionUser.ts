import { SessionUser } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const sessionUser: GqlPartial<SessionUser> = {
    __typename: 'SessionUser',
    full: {
        activeFocusMode: async () => rel((await import('./activeFocusMode')).activeFocusMode, 'full'),
        focusModes: async () => rel((await import('./focusMode')).focusMode, 'full'),
        bookmarkLists: async () => rel((await import('./bookmarkList')).bookmarkList, 'common'),
        handle: true,
        hasPremium: true,
        id: true,
        languages: true,
        name: true,
        theme: true,
    }
}
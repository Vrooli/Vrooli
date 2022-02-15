import { ValueOf } from '.';

// The length of a user session
export const COOKIE = {
    Session: "session-f234y7fdiafhdja2",
} as const;
export type COOKIE = ValueOf<typeof COOKIE>;
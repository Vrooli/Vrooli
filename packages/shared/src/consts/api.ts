import { ValueOf } from ".";

// The length of a user session
export const COOKIE = {
    Jwt: "session-f234y7fdiafhdja2",
} as const;
export type COOKIE = ValueOf<typeof COOKIE>;

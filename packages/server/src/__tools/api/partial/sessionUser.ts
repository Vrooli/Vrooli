import { type SessionUser } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const sessionUser: ApiPartial<SessionUser> = {
    full: {
        handle: true,
        hasPremium: true,
        id: true,
        languages: true,
        name: true,
        theme: true,
    },
};

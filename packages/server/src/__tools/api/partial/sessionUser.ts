import { type SessionUser } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const sessionUser: ApiPartial<SessionUser> = {
    full: {
        handle: true,
        hasPremium: true,
        hasReceivedPhoneVerificationReward: true,
        id: true,
        languages: true,
        name: true,
        phoneNumberVerified: true,
        theme: true,
    },
};

import { type Email } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const email: ApiPartial<Email> = {
    full: {
        id: true,
        emailAddress: true,
        verifiedAt: true,
    },
};

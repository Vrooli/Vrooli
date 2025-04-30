import { Email } from "@local/shared";
import { ApiPartial } from "../types.js";

export const email: ApiPartial<Email> = {
    full: {
        id: true,
        emailAddress: true,
        verifiedAt: true,
    },
};

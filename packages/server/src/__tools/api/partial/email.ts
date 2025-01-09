import { Email } from "@local/shared";
import { ApiPartial } from "../types";

export const email: ApiPartial<Email> = {
    full: {
        id: true,
        emailAddress: true,
        verified: true,
    },
};

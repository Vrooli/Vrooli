import { Phone } from "@local/shared";
import { ApiPartial } from "../types.js";

export const phone: ApiPartial<Phone> = {
    full: {
        id: true,
        phoneNumber: true,
        verified: true,
    },
};

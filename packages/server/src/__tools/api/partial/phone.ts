import { Phone } from "@local/shared";
import { ApiPartial } from "../types";

export const phone: ApiPartial<Phone> = {
    full: {
        id: true,
        phoneNumber: true,
        verified: true,
    },
};

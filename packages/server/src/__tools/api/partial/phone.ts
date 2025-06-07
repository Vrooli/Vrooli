import { type Phone } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const phone: ApiPartial<Phone> = {
    full: {
        id: true,
        phoneNumber: true,
        verifiedAt: true,
    },
};

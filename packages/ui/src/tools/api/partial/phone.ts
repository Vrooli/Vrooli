import { Phone } from "@shared/consts";
import { GqlPartial } from "../types";

export const phonePartial: GqlPartial<Phone> = {
    __typename: 'Phone',
    full: {
        id: true,
        phoneNumber: true,
        verified: true,
    },
}
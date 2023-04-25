import { Email } from "@local/shared";
import { GqlPartial } from "../types";

export const email: GqlPartial<Email> = {
    __typename: "Email",
    full: {
        id: true,
        emailAddress: true,
        verified: true,
    },
};

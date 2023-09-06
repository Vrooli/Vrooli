import { Wallet } from "@local/shared";
import { GqlPartial } from "../types";

export const wallet: GqlPartial<Wallet> = {
    __typename: "Wallet",
    common: {
        id: true,
        name: true,
        publicAddress: true,
        stakingAddress: true,
        verified: true,
    },
    full: {},
    list: {},
};

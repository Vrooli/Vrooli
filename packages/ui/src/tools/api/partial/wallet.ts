import { Wallet } from "@shared/consts";
import { GqlPartial } from "../types";

export const wallet: GqlPartial<Wallet> = {
    __typename: "Wallet",
    common: {
        id: true,
        handles: {
            id: true,
            handle: true,
        },
        name: true,
        publicAddress: true,
        stakingAddress: true,
        verified: true,
    },
    full: {},
    list: {},
};

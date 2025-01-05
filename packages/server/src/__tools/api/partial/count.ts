import { Count } from "@local/shared";
import { GqlPartial } from "../types";

export const count: GqlPartial<Count> = {
    full: {
        count: true,
    },
};

import { Count } from ":local/consts";
import { GqlPartial } from "../types";

export const count: GqlPartial<Count> = {
    __typename: "Count",
    full: {
        count: true,
    },
};

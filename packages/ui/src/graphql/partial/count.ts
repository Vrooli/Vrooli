import { Count } from "@shared/consts";
import { GqlPartial } from "types";

export const countPartial: GqlPartial<Count> = {
    __typename: 'Count',
    full: {
        count: true,
    }
}
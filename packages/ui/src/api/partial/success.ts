import { Success } from "@shared/consts";
import { GqlPartial } from "types";

export const successPartial: GqlPartial<Success> = {
    __typename: 'Success',
    full: {
        success: true,
    }
}
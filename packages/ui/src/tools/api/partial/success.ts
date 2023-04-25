import { Success } from "@local/shared";
import { GqlPartial } from "../types";

export const success: GqlPartial<Success> = {
    __typename: 'Success',
    full: {
        success: true,
    }
}
import { Success } from "@shared/consts";
import { GqlPartial } from "../types";

export const success: GqlPartial<Success> = {
    __typename: 'Success',
    full: {
        success: true,
    }
}
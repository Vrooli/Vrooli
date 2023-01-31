import { PushDevice } from "@shared/consts";
import { GqlPartial } from "../types";

export const pushDevice: GqlPartial<PushDevice> = {
    __typename: 'PushDevice',
    full: {
        id: true,
        expires: true,
        name: true,
    },
}
import { PushDevice } from "@shared/consts";
import { GqlPartial } from "../types";

export const pushDevicePartial: GqlPartial<PushDevice> = {
    __typename: 'PushDevice',
    full: {
        id: true,
        expires: true,
        name: true,
    },
}
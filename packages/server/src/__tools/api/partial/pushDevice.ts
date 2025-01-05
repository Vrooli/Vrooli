import { PushDevice } from "@local/shared";
import { GqlPartial } from "../types";

export const pushDevice: GqlPartial<PushDevice> = {
    full: {
        id: true,
        expires: true,
        name: true,
    },
};

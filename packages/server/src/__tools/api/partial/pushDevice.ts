import { PushDevice } from "@local/shared";
import { ApiPartial } from "../types.js";

export const pushDevice: ApiPartial<PushDevice> = {
    full: {
        id: true,
        expires: true,
        name: true,
    },
};

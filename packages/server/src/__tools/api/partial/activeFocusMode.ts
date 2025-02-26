import { ActiveFocusMode } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const activeFocusMode: ApiPartial<ActiveFocusMode> = {
    full: {
        focusMode: async () => rel((await import("./focusMode.js")).focusMode, "full"),
        stopCondition: true,
        stopTime: true,
    },
};

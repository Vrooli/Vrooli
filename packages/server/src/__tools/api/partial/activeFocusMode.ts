import { ActiveFocusMode } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const activeFocusMode: ApiPartial<ActiveFocusMode> = {
    full: {
        focusMode: async () => rel((await import("./focusMode")).focusMode, "full"),
        stopCondition: true,
        stopTime: true,
    },
};

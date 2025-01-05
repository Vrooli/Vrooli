import { ActiveFocusMode } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const activeFocusMode: GqlPartial<ActiveFocusMode> = {
    full: {
        focusMode: async () => rel((await import("./focusMode")).focusMode, "full"),
        stopCondition: true,
        stopTime: true,
    },
};

import { ActiveFocusMode } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const activeFocusMode: GqlPartial<ActiveFocusMode> = {
    __typename: "ActiveFocusMode",
    full: {
        mode: async () => rel((await import("./focusMode")).focusMode, "full"),
        stopCondition: true,
        stopTime: true,
    },
};

import { rel } from "../utils";
export const activeFocusMode = {
    __typename: "ActiveFocusMode",
    full: {
        mode: async () => rel((await import("./focusMode")).focusMode, "full"),
        stopCondition: true,
        stopTime: true,
    },
};
//# sourceMappingURL=activeFocusMode.js.map
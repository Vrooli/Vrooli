import { FocusModeFilterType } from "@local/consts";
import { enumToYup, id, req, yupObj } from "../utils";
import { tagValidation } from "./tag";
const focusModeFilterType = enumToYup(FocusModeFilterType);
export const focusModeFilterValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        filterType: req(focusModeFilterType),
    }, [
        ["focusMode", ["Connect"], "one", "req"],
        ["tag", ["Create", "Connect"], "one", "opt", tagValidation],
    ], [], o),
};
//# sourceMappingURL=focusModeFilter.js.map
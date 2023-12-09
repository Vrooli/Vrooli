import { FocusModeFilterType } from "@local/shared";
import { enumToYup, id, req, YupModel, yupObj } from "../utils";
import { tagValidation } from "./tag";

const focusModeFilterType = enumToYup(FocusModeFilterType);

export const focusModeFilterValidation: YupModel<true, false> = {
    create: (d) => yupObj({
        id: req(id),
        filterType: req(focusModeFilterType),
    }, [
        ["focusMode", ["Connect"], "one", "req"],
        ["tag", ["Create", "Connect"], "one", "opt", tagValidation],
    ], [], d),
    // Can only create and delete
};

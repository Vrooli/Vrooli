import { FocusModeFilterType } from "../../api/types";
import { enumToYup, id, req, YupModel, yupObj } from "../utils";
import { tagValidation } from "./tag";

const focusModeFilterType = enumToYup(FocusModeFilterType);

export const focusModeFilterValidation: YupModel<["create"]> = {
    create: (d) => yupObj({
        id: req(id),
        filterType: req(focusModeFilterType),
    }, [
        ["focusMode", ["Connect"], "one", "req"],
        ["tag", ["Create", "Connect"], "one", "opt", tagValidation],
    ], [], d),
    // Can only create and delete
};

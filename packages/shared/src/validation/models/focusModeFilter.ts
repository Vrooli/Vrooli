import { FocusModeFilterType } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { tagValidation } from "./tag.js";

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

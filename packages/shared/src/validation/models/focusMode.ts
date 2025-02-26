import * as yup from "yup";
import { FocusModeStopCondition } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { description, id, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { focusModeFilterValidation } from "./focusModeFilter.js";
import { labelValidation } from "./label.js";
import { reminderListValidation } from "./reminderList.js";
import { resourceListValidation } from "./resourceList.js";
import { scheduleValidation } from "./schedule.js";

export const focusModeValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        name: req(name),
        description: opt(description),
    }, [
        ["filters", ["Create"], "many", "opt", focusModeFilterValidation],
        ["labels", ["Create", "Connect"], "many", "opt", labelValidation],
        ["reminderList", ["Create", "Connect"], "one", "opt", reminderListValidation],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["schedule", ["Create"], "one", "opt", scheduleValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        name: opt(name),
        description: opt(description),
    }, [
        ["filters", ["Create", "Delete"], "many", "opt", focusModeFilterValidation],
        ["labels", ["Create", "Connect", "Disconnect"], "many", "opt", labelValidation],
        ["reminderList", ["Connect", "Disconnect", "Create", "Update"], "one", "opt", reminderListValidation],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
    ], [], d),
};

const stopCondition = enumToYup(FocusModeStopCondition);

export const setActiveFocusModeValidation = yup.object().shape({
    id: opt(id),
    stopCondition: opt(stopCondition),
    stopTime: opt(yup.date()),
});

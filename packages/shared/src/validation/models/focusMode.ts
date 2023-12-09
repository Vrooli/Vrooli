import { description, id, name, opt, req, YupModel, yupObj } from "../utils";
import { focusModeFilterValidation } from "./focusModeFilter";
import { labelValidation } from "./label";
import { reminderListValidation } from "./reminderList";
import { resourceListValidation } from "./resourceList";
import { scheduleValidation } from "./schedule";

export const focusModeValidation: YupModel = {
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

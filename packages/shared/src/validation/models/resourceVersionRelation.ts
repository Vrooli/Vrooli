import { optArr, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const resourceVersionRelationValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        labels: optArr(label),
    }, [
        ["fromVersion", ["Connect"], "one", "req"],
        ["toVersion", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        labels: optArr(label),
    }, [], [], d),
};

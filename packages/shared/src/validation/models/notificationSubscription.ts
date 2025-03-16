import { SubscribableObject } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

const subscribableObject = enumToYup(SubscribableObject);

export const notificationSubscriptionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        silent: opt(bool),
        objectType: req(subscribableObject),
    }, [
        ["object", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        silent: opt(bool),
    }, [], [], d),
};

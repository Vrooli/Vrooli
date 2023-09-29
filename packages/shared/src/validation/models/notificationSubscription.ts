import { SubscribableObject } from "@local/shared";
import { bool, enumToYup, id, opt, req, YupModel, yupObj } from "../utils";

const subscribableObject = enumToYup(SubscribableObject);

export const notificationSubscriptionValidation: YupModel = {
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

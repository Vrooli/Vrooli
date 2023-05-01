import { SubscribableObject } from "@local/shared";
import { bool, enumToYup, id, opt, req, YupModel, yupObj } from "../utils";

const subscribableObject = enumToYup(SubscribableObject);

export const notificationSubscriptionValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        silent: opt(bool),
        objectType: req(subscribableObject),
    }, [
        ["object", ["Connect"], "one", "req"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        silent: opt(bool),
    }, [], [], o),
};

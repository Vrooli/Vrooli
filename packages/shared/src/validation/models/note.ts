import { bool, id, opt, req, YupModel, yupObj } from "../utils";
import { labelValidation } from "./label";
import { noteVersionValidation } from "./noteVersion";
import { tagValidation } from "./tag";

export const noteValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["user", ["Connect"], "one", "opt"],
        ["organization", ["Connect"], "one", "opt"],
        ["parent", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create"], "many", "opt", tagValidation],
        ["versions", ["Create"], "many", "opt", noteVersionValidation, ["root"]],
        ["labels", ["Connect", "Create"], "many", "opt", labelValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        isPrivate: opt(bool),
    }, [
        ["user", ["Connect"], "one", "opt"],
        ["organization", ["Connect"], "one", "opt"],
        ["tags", ["Connect", "Create", "Disconnect"], "one", "opt", tagValidation],
        ["versions", ["Create", "Update", "Delete"], "many", "opt", noteVersionValidation, ["root"]],
        ["labels", ["Connect", "Create", "Disconnect"], "many", "opt", labelValidation],
    ], [], o),
};

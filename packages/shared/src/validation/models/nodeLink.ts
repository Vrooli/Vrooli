import { id, nodeOperation, opt, req, YupModel, yupObj } from "../utils";
import { nodeLinkWhenValidation } from "./nodeLinkWhen";

export const nodeLinkValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        operation: opt(nodeOperation),
    }, [
        ["from", ["Connect"], "one", "req"],
        ["routineVersion", ["Connect"], "one", "req"],
        ["to", ["Connect"], "one", "req"],
        ["whens", ["Create"], "many", "opt", nodeLinkWhenValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        operation: opt(nodeOperation),
    }, [
        ["whens", ["Delete", "Create", "Update"], "many", "opt", nodeLinkWhenValidation],
        ["from", ["Connect"], "one", "opt"],
        ["to", ["Connect"], "one", "opt"],
    ], [], o),
};

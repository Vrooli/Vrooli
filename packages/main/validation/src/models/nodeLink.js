import { id, req, opt, nodeOperation, yupObj } from "../utils";
import { nodeLinkWhenValidation } from "./nodeLinkWhen";
export const nodeLinkValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        operation: opt(nodeOperation),
    }, [
        ["whens", ["Create"], "many", "opt", nodeLinkWhenValidation],
        ["from", ["Connect"], "one", "req"],
        ["to", ["Connect"], "one", "req"],
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
//# sourceMappingURL=nodeLink.js.map
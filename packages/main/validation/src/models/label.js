import * as yup from "yup";
import { blankToUndefined, description, hexColor, id, maxStrErr, opt, req, transRel, yupObj } from "../utils";
const label = yup.string().transform(blankToUndefined).max(128, maxStrErr);
export const labelTranslationValidation = transRel({
    create: {
        description: req(description),
    },
    update: {
        description: opt(description),
    },
});
export const labelValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        label: req(label),
        color: opt(hexColor),
    }, [
        ["organization", ["Connect"], "one", "opt"],
        ["apis", ["Connect"], "many", "opt"],
        ["focusModes", ["Connect"], "many", "opt"],
        ["issues", ["Connect"], "many", "opt"],
        ["notes", ["Connect"], "many", "opt"],
        ["projects", ["Connect"], "many", "opt"],
        ["routines", ["Connect"], "many", "opt"],
        ["smartContracts", ["Connect"], "many", "opt"],
        ["standards", ["Connect"], "many", "opt"],
        ["meetings", ["Connect"], "many", "opt"],
        ["schedules", ["Connect"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", labelTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        label: req(label),
        color: opt(hexColor),
    }, [
        ["apis", ["Connect", "Disconnect"], "many", "opt"],
        ["focusModes", ["Connect", "Disconnect"], "many", "opt"],
        ["issues", ["Connect", "Disconnect"], "many", "opt"],
        ["notes", ["Connect", "Disconnect"], "many", "opt"],
        ["projects", ["Connect", "Disconnect"], "many", "opt"],
        ["routines", ["Connect", "Disconnect"], "many", "opt"],
        ["smartContracts", ["Connect", "Disconnect"], "many", "opt"],
        ["standards", ["Connect", "Disconnect"], "many", "opt"],
        ["meetings", ["Connect", "Disconnect"], "many", "opt"],
        ["schedules", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", labelTranslationValidation],
    ], [], o),
};
//# sourceMappingURL=label.js.map
import { id, req, opt, yupObj, bool } from "../utils";
export const nodeEndValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        wasSuccessful: opt(bool),
    }, [
        ["node", ["Connect"], "one", "req"],
        ["suggestedNextRoutineVersions", ["Connect"], "many", "opt"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        wasSuccessful: opt(bool),
    }, [
        ["suggestedNextRoutineVersions", ["Connect", "Disconnect"], "many", "opt"],
    ], [], o),
};
//# sourceMappingURL=nodeEnd.js.map
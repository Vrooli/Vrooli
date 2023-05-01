import { bool, id, opt, req, YupModel, yupObj } from "../utils";

export const nodeEndValidation: YupModel = {
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

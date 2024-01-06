import { bool, id, opt, req, YupModel, yupObj } from "../utils";

export const nodeEndValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        wasSuccessful: opt(bool),
    }, [
        ["node", ["Connect"], "one", "req"],
        ["suggestedNextRoutineVersions", ["Connect"], "many", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        wasSuccessful: opt(bool),
    }, [
        ["suggestedNextRoutineVersions", ["Connect", "Disconnect"], "many", "opt"],
    ], [], d),
};

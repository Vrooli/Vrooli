import { bool, description, id, index, instructions, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { standardVersionValidation } from "./standardVersion";

export const routineVersionInputTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        helpText: opt(instructions),
    }),
    update: () => ({
        description: opt(description),
        helpText: opt(instructions),
    }),
});

export const routineVersionInputValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        index: opt(index),
        isRequired: opt(bool),
        name: opt(name),
    }, [
        ["routineVersion", ["Connect"], "one", "req"],
        ["standardVersion", ["Connect", "Create"], "one", "opt", standardVersionValidation],
        ["translations", ["Create"], "many", "opt", routineVersionInputTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        index: opt(index),
        isRequired: opt(bool),
        name: opt(name),
    }, [
        ["standardVersion", ["Connect", "Create", "Disconnect"], "one", "opt", standardVersionValidation],
        ["translations", ["Create"], "many", "opt", routineVersionInputTranslationValidation],
    ], [], d),
};

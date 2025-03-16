import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, id, index, instructions, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { standardVersionValidation } from "./standardVersion.js";

export const routineVersionOutputTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        helpText: opt(instructions),
    }),
    update: () => ({
        description: opt(description),
        helpText: opt(instructions),
    }),
});

export const routineVersionOutputValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        index: opt(index),
        isRequired: opt(bool),
        name: opt(name),
    }, [
        ["routineVersion", ["Connect"], "one", "req"],
        ["standardVersion", ["Connect", "Create"], "one", "req", standardVersionValidation],
        ["translations", ["Create"], "many", "opt", routineVersionOutputTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        index: opt(index),
        isRequired: opt(bool),
        name: opt(name),
    }, [
        ["standardVersion", ["Connect", "Create", "Disconnect"], "one", "req", standardVersionValidation],
        ["translations", ["Create"], "many", "opt", routineVersionOutputTranslationValidation],
    ], [], d),
};

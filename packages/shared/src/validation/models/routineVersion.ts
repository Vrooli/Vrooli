import { RoutineType } from "../../api/types.js";
import { enumToYup } from "../utils/builders/convert.js";
import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, configCallData, configFormInput, configFormOutput, description, id, instructions, name, versionLabel, versionNotes } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { resourceListValidation } from "./resourceList.js";
import { routineValidation } from "./routine.js";
import { routineVersionInputValidation } from "./routineVersionInput.js";
import { routineVersionOutputValidation } from "./routineVersionOutput.js";

const routineType = enumToYup(RoutineType);

export const routineVersionTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        instructions: opt(instructions),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        instructions: opt(instructions),
        name: opt(name),
    }),
});

export const routineVersionValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        configCallData: opt(configCallData),
        configFormInput: opt(configFormInput),
        configFormOutput: opt(configFormOutput),
        isComplete: opt(bool),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        routineType: req(routineType),
        versionLabel: req(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "opt", routineValidation, ["versions"]],
        ["apiVersion", ["Connect"], "one", "opt"],
        ["codeVersion", ["Connect"], "one", "opt"],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["inputs", ["Create"], "many", "opt", routineVersionInputValidation],
        ["outputs", ["Create"], "many", "opt", routineVersionOutputValidation],
        ["translations", ["Create"], "many", "opt", routineVersionTranslationValidation],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["subroutineLinks", ["Connect"], "many", "opt"],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        configCallData: opt(configCallData),
        configFormInput: opt(configFormInput),
        configFormOutput: opt(configFormOutput),
        isComplete: opt(bool),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(d)),
        versionNotes: opt(versionNotes),
    }, [
        ["apiVersion", ["Connect", "Disconnect"], "one", "opt"],
        ["codeVersion", ["Connect", "Disconnect"], "one", "opt"],
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["inputs", ["Create", "Update", "Delete"], "many", "opt", routineVersionInputValidation],
        ["outputs", ["Create", "Update", "Delete"], "many", "opt", routineVersionOutputValidation],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["root", ["Update"], "one", "opt", routineValidation, ["versions"]],
        ["subroutineLinks", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", routineVersionTranslationValidation],
    ], [], d),
};

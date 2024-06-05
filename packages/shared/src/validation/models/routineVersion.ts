import { RoutineType } from "../../api/generated";
import { YupModel, bool, configCallData, description, enumToYup, id, instructions, name, opt, req, transRel, versionLabel, versionNotes, yupObj } from "../utils";
import { nodeValidation } from "./node";
import { nodeLinkValidation } from "./nodeLink";
import { resourceListValidation } from "./resourceList";
import { routineValidation } from "./routine";
import { routineVersionInputValidation } from "./routineVersionInput";
import { routineVersionOutputValidation } from "./routineVersionOutput";

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
        ["nodes", ["Create"], "many", "req", nodeValidation],
        ["nodeLinks", ["Create"], "many", "req", nodeLinkValidation],
        ["inputs", ["Create"], "many", "req", routineVersionInputValidation],
        ["outputs", ["Create"], "many", "req", routineVersionOutputValidation],
        ["translations", ["Create"], "many", "opt", routineVersionTranslationValidation],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["suggestedNextByProject", ["Connect"], "many", "opt"],
    ], [["rootConnect", "rootCreate", true]], d),
    update: (d) => yupObj({
        id: req(id),
        configCallData: opt(configCallData),
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
        ["nodeLinks", ["Create", "Update", "Delete"], "many", "opt", nodeLinkValidation],
        ["nodes", ["Create", "Update", "Delete"], "many", "opt", nodeValidation],
        ["outputs", ["Create", "Update", "Delete"], "many", "opt", routineVersionOutputValidation],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["root", ["Update"], "one", "opt", routineValidation, ["versions"]],
        ["suggestedNextByProject", ["Connect", "Disconnect"], "many", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", routineVersionTranslationValidation],
    ], [], d),
};

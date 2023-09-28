import { apiCallData, bool, description, id, instructions, name, opt, req, smartContractCallData, transRel, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { nodeValidation } from "./node";
import { nodeLinkValidation } from "./nodeLink";
import { resourceListValidation } from "./resourceList";
import { routineValidation } from "./routine";
import { routineVersionInputValidation } from "./routineVersionInput";
import { routineVersionOutputValidation } from "./routineVersionOutput";

export const routineVersionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        instructions: req(instructions),
        name: req(name),
    },
    update: {
        description: opt(description),
        instructions: opt(instructions),
        name: opt(name),
    },
});

export const routineVersionValidation: YupModel = {
    create: ({ o, minVersion = "0.0.1" }) => yupObj({
        id: req(id),
        apiCallData: opt(apiCallData),
        smartContractCallData: opt(smartContractCallData),
        isComplete: opt(bool),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        versionLabel: req(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ["root", ["Connect", "Create"], "one", "req", routineValidation, ["versions"]],
        ["apiVersion", ["Connect"], "one", "opt"],
        ["smartContractVersion", ["Connect"], "one", "opt"],
        ["resourceList", ["Create"], "one", "opt", resourceListValidation],
        ["nodes", ["Create"], "many", "req", nodeValidation],
        ["nodeLinks", ["Create"], "many", "req", nodeLinkValidation],
        ["inputs", ["Create"], "many", "req", routineVersionInputValidation],
        ["outputs", ["Create"], "many", "req", routineVersionOutputValidation],
        ["translations", ["Create"], "many", "opt", routineVersionTranslationValidation],
        ["directoryListings", ["Connect"], "many", "opt"],
        ["suggestedNextByProject", ["Connect"], "many", "opt"],
    ], [["rootConnect", "rootCreate"]], o),
    update: ({ o, minVersion = "0.0.1" }) => yupObj({
        id: req(id),
        apiCallData: opt(apiCallData),
        smartContractCallData: opt(smartContractCallData),
        isComplete: opt(bool),
        isInternal: opt(bool),
        isPrivate: opt(bool),
        versionLabel: opt(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ["apiVersion", ["Connect", "Disconnect"], "one", "opt"],
        ["smartContractVersion", ["Connect", "Disconnect"], "one", "opt"],
        ["resourceList", ["Create", "Update"], "one", "opt", resourceListValidation],
        ["nodes", ["Create", "Update", "Delete"], "many", "opt", nodeValidation],
        ["nodeLinks", ["Create", "Update", "Delete"], "many", "opt", nodeLinkValidation],
        ["inputs", ["Create", "Update", "Delete"], "many", "opt", routineVersionInputValidation],
        ["outputs", ["Create", "Update", "Delete"], "many", "opt", routineVersionOutputValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", routineVersionTranslationValidation],
        ["directoryListings", ["Connect", "Disconnect"], "many", "opt"],
        ["suggestedNextByProject", ["Connect", "Disconnect"], "many", "opt"],
    ], [], o),
};

import * as yup from "yup";
import { NodeType } from "../../api/generated";
import { description, enumToYup, id, maxStrErr, minNumErr, minStrErr, opt, req, transRel, YupModel, yupObj } from "../utils";
import { nodeEndValidation } from "./nodeEnd";
import { nodeLoopValidation } from "./nodeLoop";
import { nodeRoutineListValidation } from "./nodeRoutineList";

const columnIndex = yup.number().integer().min(0, minNumErr).nullable();
const rowIndex = yup.number().integer().min(0, minNumErr).nullable();
// Node name is less strict than other object names
// eslint-disable-next-line no-magic-numbers
const name = yup.string().trim().removeEmptyString().min(1, minStrErr).max(50, maxStrErr);
const nodeType = enumToYup(NodeType);

export const nodeTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: () => ({
        description: opt(description),
        name: req(name),
    }),
    update: () => ({
        description: opt(description),
        name: opt(name),
    }),
});

export const nodeValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        columnIndex: opt(columnIndex),
        nodeType: req(nodeType),
        rowIndex: opt(rowIndex),
    }, [
        ["end", ["Create"], "one", "opt", nodeEndValidation],
        ["loop", ["Create"], "one", "opt", nodeLoopValidation],
        ["routineList", ["Create"], "one", "opt", nodeRoutineListValidation],
        ["routineVersion", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", nodeTranslationValidation],
    ], [["endCreate", "routineListCreate", false]], d),
    update: (d) => yupObj({
        id: req(id),
        columnIndex: opt(columnIndex),
        nodeType: opt(nodeType),
        rowIndex: opt(rowIndex),
    }, [
        ["end", ["Create", "Update"], "one", "opt", nodeEndValidation, ["node"]],
        ["loop", ["Create", "Update", "Delete"], "one", "opt", nodeLoopValidation, ["node"]],
        ["routineList", ["Create", "Update"], "one", "opt", nodeRoutineListValidation, ["node"]],
        ["routineVersion", ["Connect"], "one", "opt"],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", nodeTranslationValidation],
    ], [], d),
};

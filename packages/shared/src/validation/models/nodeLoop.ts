import * as yup from "yup";
import { id, maxStrErr, minStrErr, nodeOperation, opt, req, YupModel, yupObj } from "../utils";
import { nodeLoopWhileValidation } from "./nodeLoopWhile";

const loops = yup.number().integer().min(0, minStrErr).max(100, maxStrErr);
const maxLoops = yup.number().integer().min(1, minStrErr).max(100, maxStrErr);

export const nodeLoopValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        loops: opt(loops),
        maxLoops: opt(maxLoops),
        operation: opt(nodeOperation),
    }, [
        ["node", ["Connect"], "one", "req"],
        ["whiles", ["Create"], "many", "req", nodeLoopWhileValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        loops: opt(loops),
        maxLoops: opt(maxLoops),
        operation: opt(nodeOperation),
    }, [
        ["whiles", ["Create", "Update", "Delete"], "many", "req", nodeLoopWhileValidation],
    ], [], d),
};

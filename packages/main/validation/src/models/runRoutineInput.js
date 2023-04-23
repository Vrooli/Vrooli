import * as yup from "yup";
import { blankToUndefined, id, maxStrErr, req, yupObj } from "../utils";
const data = yup.string().transform(blankToUndefined).max(8192, maxStrErr);
export const runRoutineInputValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        data: req(data),
    }, [
        ["input", ["Connect"], "one", "req"],
        ["runRoutine", ["Connect"], "one", "req"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        data: req(data),
    }, [], [], o),
};
//# sourceMappingURL=runRoutineInput.js.map
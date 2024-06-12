import { Job } from "bull";
import { CustomError } from "../../events";
import { RunPayload, RunProjectPayload, RunRoutinePayload } from "./queue";

export const doRunProject = async (data: RunProjectPayload) => {
    //TODO
};

export const doRunRoutine = async (data: RunRoutinePayload) => {
    //TODO
};

export const llmProcess = async ({ data }: Job<RunPayload>) => {
    switch (data.__process) {
        case "Project":
            return doRunProject(data);
        case "Routine":
            return doRunRoutine(data);
        default:
            throw new CustomError("0568", "InternalError", ["en"], { process: (data as { __process?: unknown }).__process });
    }
};

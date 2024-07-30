import { Job } from "bull";
import { CustomError } from "../../events";
import { RunProjectPayload, RunRequestPayload, RunRoutinePayload } from "./queue";

export async function doRunProject(data: RunProjectPayload) {
    //TODO
}

export async function doRunRoutine(data: RunRoutinePayload) {
    //TODO
}

export async function runProcess({ data }: Job<RunRequestPayload>) {
    switch (data.__process) {
        case "Project":
            return doRunProject(data);
        case "Routine":
            return doRunRoutine(data);
        default:
            throw new CustomError("0568", "InternalError", ["en"], { process: (data as { __process?: unknown }).__process });
    }
}

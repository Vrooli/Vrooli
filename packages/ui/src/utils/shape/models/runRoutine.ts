import { RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { OrganizationShape } from "./organization";
import { RoutineVersionShape } from "./routineVersion";
import { RunProjectShape } from "./runProject";
import { RunRoutineInputShape, shapeRunRoutineInput } from "./runRoutineInput";
import { RunRoutineStepShape, shapeRunRoutineStep } from "./runRoutineStep";
import { ScheduleShape, shapeSchedule } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type RunRoutineShape = Pick<RunRoutine, "id" | "isPrivate" | "completedComplexity" | "contextSwitches" | "name" | "status" | "timeElapsed"> & {
    __typename?: "RunRoutine";
    steps?: RunRoutineStepShape[] | null;
    inputs?: RunRoutineInputShape[] | null;
    schedule?: ScheduleShape | null;
    routineVersion: { id: string } | RoutineVersionShape;
    runProject: { id: string } | RunProjectShape;
    organization?: { id: string } | OrganizationShape | null;
}

export const shapeRunRoutine: ShapeModel<RunRoutineShape, RunRoutineCreateInput, RunRoutineUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate", "completedComplexity", "contextSwitches", "name", "status"),
        ...createRel(d, "steps", ["Create"], "many", shapeRunRoutineStep),
        ...createRel(d, "inputs", ["Create"], "many", shapeRunRoutineInput),
        ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "runProject", ["Connect"], "one"),
        ...createRel(d, "organization", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "completedComplexity", "contextSwitches", "status", "timeElapsed"),
        ...updateRel(o, u, "inputs", ["Create", "Update", "Delete"], "many", shapeRunRoutineInput),
        ...updateRel(o, u, "steps", ["Create", "Update", "Delete"], "many", shapeRunRoutineStep),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
    }, a),
};

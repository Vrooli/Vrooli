import { RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { NodeShape } from "./node";
import { ProjectVersionDirectoryShape } from "./projectVersionDirectory";
import { RunProjectShape } from "./runProject";
import { createPrims, createRel, shapeUpdate, updatePrims } from "./tools";

export type RunProjectStepShape = Pick<RunProjectStep, "id" | "contextSwitches" | "name" | "order" | "status" | "step" | "timeElapsed"> & {
    __typename: "RunProjectStep";
    directory?: CanConnect<ProjectVersionDirectoryShape> | null;
    node?: CanConnect<NodeShape> | null;
    runProject: CanConnect<RunProjectShape>;
}

export const shapeRunProjectStep: ShapeModel<RunProjectStepShape, RunProjectStepCreateInput, RunProjectStepUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "contextSwitches", "name", "order", "status", "step", "timeElapsed"),
        ...createRel(d, "directory", ["Connect"], "one"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "runProject", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "contextSwitches", "status", "timeElapsed"),
    }, a),
};

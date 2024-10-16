import { RunProject, RunProjectCreateInput, RunProjectUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { ProjectVersionShape } from "./projectVersion";
import { RunProjectStepShape, shapeRunProjectStep } from "./runProjectStep";
import { ScheduleShape, shapeSchedule } from "./schedule";
import { TeamShape } from "./team";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type RunProjectShape = Pick<RunProject, "id" | "isPrivate" | "completedComplexity" | "contextSwitches" | "name" | "status" | "timeElapsed"> & {
    __typename: "RunProject";
    steps?: RunProjectStepShape[] | null;
    schedule?: ScheduleShape | null;
    projectVersion?: CanConnect<ProjectVersionShape> | null;
    team?: CanConnect<TeamShape> | null;
}

export const shapeRunProject: ShapeModel<RunProjectShape, RunProjectCreateInput, RunProjectUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate", "completedComplexity", "contextSwitches", "name", "status", "timeElapsed"),
        ...createRel(d, "steps", ["Create"], "many", shapeRunProjectStep),
        ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        ...createRel(d, "projectVersion", ["Connect"], "one"),
        ...createRel(d, "team", ["Connect"], "one"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "completedComplexity", "contextSwitches", "status", "timeElapsed"),
        ...updateRel(o, u, "steps", ["Create", "Update", "Delete"], "many", shapeRunProjectStep),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
    }),
};

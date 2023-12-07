import { Routine, RoutineCreateInput, RoutineUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { RoutineVersionShape, shapeRoutineVersion } from "./routineVersion";
import { TagShape, shapeTag } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";


export type RoutineShape = Pick<Routine, "id" | "isInternal" | "isPrivate" | "permissions"> & {
    __typename: "Routine";
    labels?: CanConnect<LabelShape>[] | null;
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<RoutineVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<RoutineVersionShape, "root">[] | null;
}

export const shapeRoutine: ShapeModel<RoutineShape, RoutineCreateInput, RoutineUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isInternal", "isPrivate", "permissions"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeRoutineVersion),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isInternal", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeRoutineVersion),
    }, a),
};

import { Standard, StandardCreateInput, StandardUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { StandardVersionShape, shapeStandardVersion } from "./standardVersion";
import { TagShape, shapeTag } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";


export type StandardShape = Pick<Standard, "id" | "isInternal" | "isPrivate" | "permissions"> & {
    __typename: "Standard";
    parent?: CanConnect<StandardVersionShape> | null;
    owner?: OwnerShape | null;
    labels?: CanConnect<LabelShape>[] | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<StandardVersionShape, "root">[] | null;
}

export type StandardShapeUpdate = Omit<StandardShape, "default" | "isInternal" | "name" | "props" | "yup" | "type" | "version" | "creator">;

export const shapeStandard: ShapeModel<StandardShape, StandardCreateInput, StandardUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isInternal", "isPrivate", "permissions"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeStandardVersion),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isInternal", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeStandardVersion),
    }, a),
};

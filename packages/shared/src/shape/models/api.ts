import { Api, ApiCreateInput, ApiUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { ApiVersionShape, shapeApiVersion } from "./apiVersion";
import { LabelShape, shapeLabel } from "./label";
import { TagShape, shapeTag } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";

export type ApiShape = Pick<Api, "id" | "isPrivate"> & {
    __typename: "Api";
    labels?: CanConnect<LabelShape>[] | null;
    owner: CanConnect<OwnerShape> | null | undefined;
    parent?: CanConnect<ApiVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    versions?: Omit<ApiVersionShape, "root">[] | null;
}

export const shapeApi: ShapeModel<ApiShape, ApiCreateInput, ApiUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeApiVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeApiVersion),
    }),
};

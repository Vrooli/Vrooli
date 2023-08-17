import { SmartContract, SmartContractCreateInput, SmartContractUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { shapeSmartContractVersion, SmartContractVersionShape } from "./smartContractVersion";
import { shapeTag, TagShape } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";

export type SmartContractShape = Pick<SmartContract, "id" | "isPrivate"> & {
    __typename?: "SmartContract";
    labels?: ({ id: string } | LabelShape)[];
    owner: OwnerShape | null | undefined;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    versions?: Omit<SmartContractVersionShape, "root">[] | null;
}

export const shapeSmartContract: ShapeModel<SmartContractShape, SmartContractCreateInput, SmartContractUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeSmartContractVersion),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeSmartContractVersion),
    }, a),
};

import { SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type SmartContractVersionShape = Pick<SmartContractVersion, 'id'>

export const shapeSmartContractVersion: ShapeModel<SmartContractVersionShape, SmartContractVersionCreateInput, SmartContractVersionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}
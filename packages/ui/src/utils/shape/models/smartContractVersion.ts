import { SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type SmartContractVersionShape = Pick<SmartContractVersion, 'id'>

export const shapeSmartContractVersion: ShapeModel<SmartContractVersionShape, SmartContractVersionCreateInput, SmartContractVersionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}
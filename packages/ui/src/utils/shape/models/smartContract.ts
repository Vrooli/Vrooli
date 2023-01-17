import { SmartContract, SmartContractCreateInput, SmartContractUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type SmartContractShape = Pick<SmartContract, 'id'>

export const shapeSmartContract: ShapeModel<SmartContractShape, SmartContractCreateInput, SmartContractUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}
import { SmartContract, SmartContractCreateInput, SmartContractUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type SmartContractShape = Pick<SmartContract, 'id'>

export const shapeSmartContract: ShapeModel<SmartContractShape, SmartContractCreateInput, SmartContractUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}
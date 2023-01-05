import { Wallet, WalletUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type WalletShape = Pick<Wallet, 'id'>

export const shapeWallet: ShapeModel<WalletShape, null, WalletUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {}) as any
}
import { Wallet, WalletUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";

export type WalletShape = Pick<Wallet, 'id'>

export const shapeWallet: ShapeModel<WalletShape, null, WalletUpdateInput> = {
    update: (o, u) => ({}) as any
}
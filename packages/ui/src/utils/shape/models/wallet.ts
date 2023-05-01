import { Wallet, WalletUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { shapeUpdate } from "./tools";

export type WalletShape = Pick<Wallet, 'id'> & {
    __typename?: 'Wallet';
}

export const shapeWallet: ShapeModel<WalletShape, null, WalletUpdateInput> = {
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}
import { Wallet, WalletUpdateInput } from "../../api/types";
import { ShapeModel } from "../../consts/commonTypes";
import { shapeUpdate } from "./tools";

export type WalletShape = Pick<Wallet, "id"> & {
    __typename: "Wallet";
}

export const shapeWallet: ShapeModel<WalletShape, null, WalletUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {}) as any,
};

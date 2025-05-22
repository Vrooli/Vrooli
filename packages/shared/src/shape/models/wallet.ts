import { type Wallet, type WalletUpdateInput } from "../../api/types.js";
import { type ShapeModel } from "../../consts/commonTypes.js";
import { shapeUpdate } from "./tools.js";

export type WalletShape = Pick<Wallet, "id"> & {
    __typename: "Wallet";
}

export const shapeWallet: ShapeModel<WalletShape, null, WalletUpdateInput> = {
    update: (o, u) => shapeUpdate(u, {}) as any,
};

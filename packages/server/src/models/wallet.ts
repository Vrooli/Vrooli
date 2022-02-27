import { PrismaType } from "types";
import { Wallet } from "../schema/types";
import { FormatConverter, GraphQLModelType } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const walletFormatter = (): FormatConverter<Wallet> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Wallet,
        'user': GraphQLModelType.User,
        'organization': GraphQLModelType.Organization,
        'project': GraphQLModelType.Project,
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function WalletModel(prisma: PrismaType) {
    const prismaObject = prisma.wallet;
    const format = walletFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================
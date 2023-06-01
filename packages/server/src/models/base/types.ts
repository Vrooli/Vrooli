import { Api, ApiCreateInput, ApiSearchInput, ApiSortBy, ApiUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";


export type ModelApiLogic = {
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: ApiCreateInput,
    GqlUpdate: ApiUpdateInput,
    GqlModel: Api,
    GqlPermission: Permissions,
    GqlSearch: ApiSearchInput,
    GqlSort: ApiSortBy,
    PrismaCreate: Prisma.apiUpsertArgs["create"],
    PrismaUpdate: Prisma.apiUpsertArgs["update"],
    PrismaModel: Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>,
    PrismaSelect: Prisma.apiSelect,
    PrismaWhere: Prisma.apiWhereInput,
}

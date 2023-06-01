import { Member, MemberSearchInput, MemberSortBy, MemberUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { OrganizationModel } from "./organization";
import { RoleModel } from "./role";
import { ModelLogic } from "./types";
import { UserModel } from "./user";
import { Formatter } from "../types";

const __typename = "Member" as const;
export const MemberFormat: Formatter<ModelMemberLogic> = {
        gqlRelMap: {
            __typename,
            organization: "Organization",
            roles: "Role",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
            roles: "Role",
            user: "User",
        },
        countFields: {},
    },
    search: {
        defaultSort: MemberSortBy.DateCreatedDesc,
        sortBy: MemberSortBy,
        searchFields: {
            createdTimeFrame: true,
            organizationId: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                { organization: OrganizationModel.search!.searchStringQuery() },
                { role: RoleModel.search!.searchStringQuery() },
                { user: UserModel.search!.searchStringQuery() },
            ],
        }),
    },
};

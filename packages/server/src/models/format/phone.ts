import { MaxObjects, Phone, PhoneCreateInput, phoneValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../../builders/types";
import { Trigger } from "../../events";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Phone" as const;
export const PhoneFormat: Formatter<ModelPhoneLogic> = {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            user: "User",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, userData }) => ({
                phoneNumber: data.phoneNumber,
                user: { connect: { id: userData.id } },
            }),
        },
        trigger: {
            onCreated: async ({ created, prisma, userData }) => {
                for (const { id: objectId } of created) {
                    await Trigger(prisma, userData.languages).objectCreated({
                        createdById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                    });
                }
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
};

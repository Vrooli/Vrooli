import { MaxObjects, PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput, pushDeviceValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "PushDevice" as const;
export const PushDeviceFormat: Formatter<ModelPushDeviceLogic> = {
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
                endpoint: data.endpoint,
                expires: noNull(data.expires),
                auth: data.keys.auth,
                p256dh: data.keys.p256dh,
                name: noNull(data.name),
                user: { connect: { id: userData.id } },
            }),
            update: async ({ data }) => ({
                name: noNull(data.name),
            }),
        owner: (data) => ({
            User: data.user,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
};

import { MaxObjects, Resource, ResourceCreateInput, ResourceSearchInput, ResourceSortBy, ResourceUpdateInput, resourceValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { bestTranslation, translationShapeHelper } from "../../utils";
import { ResourceListModel } from "./resourceList";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Resource" as const;
export const ResourceFormat: Formatter<ModelResourceLogic> = {
        gqlRelMap: {
            __typename,
            list: "ResourceList",
        },
        prismaRelMap: {
            __typename,
            list: "ResourceList",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                index: noNull(data.index),
                link: data.link,
                usedFor: data.usedFor,
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "ResourceList", parentRelationshipName: "resources", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                index: noNull(data.index),
                link: noNull(data.link),
                usedFor: noNull(data.usedFor),
                ...(await shapeHelper({ relation: "list", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "ResourceList", parentRelationshipName: "resources", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        searchFields: {
            createdTimeFrame: true,
            resourceListId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        searchStringQuery: () => ({
            OR: [
                "transDescriptionWrapped",
                "transNameWrapped",
                "linkWrapped",
            ],
        permissionsSelect: () => ({
            id: true,
            list: "ResourceList",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ list: ResourceListModel.validate!.visibility.owner(userId) }),
};

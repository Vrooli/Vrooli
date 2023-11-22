import { OrganizationCreateInput, OrganizationUpdateInput } from "@local/shared";
import { bannerImageConfig, profileImageConfig } from "../../utils";
import { organization_create, organization_findMany, organization_findOne, organization_update } from "../generated";
import { OrganizationEndpoints } from "../logic/organization";
import { setupRoutes, UploadConfig } from "./base";

const imagesConfig: UploadConfig<OrganizationCreateInput | OrganizationUpdateInput> = {
    acceptsFiles: true,
    fields: [{
        ...profileImageConfig,
        fieldName: "profileImage",
        fileNameBase: (input) => `${input.id}-profile`,
    }, {
        ...bannerImageConfig,
        fieldName: "bannerImage",
        fileNameBase: (input) => `${input.id}-banner`,
    }],
};

export const OrganizationRest = setupRoutes({
    "/organization/:id": {
        get: [OrganizationEndpoints.Query.organization, organization_findOne],
        put: [OrganizationEndpoints.Mutation.organizationUpdate, organization_update, imagesConfig],
    },
    "/organizations": {
        get: [OrganizationEndpoints.Query.organizations, organization_findMany],
    },
    "/organization": {
        post: [OrganizationEndpoints.Mutation.organizationCreate, organization_create, imagesConfig],
    },
});

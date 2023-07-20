import { organization_create, organization_findMany, organization_findOne, organization_update } from "../generated";
import { OrganizationEndpoints } from "../logic";
import { UploadConfig, setupRoutes } from "./base";

const imagesConfig: UploadConfig = {
    acceptsFiles: true,
    imageSizes: [
        { width: 1024, height: 1024 },
        { width: 512, height: 512 },
        { width: 256, height: 256 },
        { width: 128, height: 128 },
        { width: 64, height: 64 },
    ],
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

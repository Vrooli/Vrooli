import { organization_create, organization_findMany, organization_findOne, organization_update } from "../generated";
import { OrganizationEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const OrganizationRest = setupRoutes({
    "/organization/:id": {
        get: [OrganizationEndpoints.Query.organization, organization_findOne],
        put: [OrganizationEndpoints.Mutation.organizationUpdate, organization_update],
    },
    "/organizations": {
        get: [OrganizationEndpoints.Query.organizations, organization_findMany],
    },
    "/organization": {
        post: [OrganizationEndpoints.Mutation.organizationCreate, organization_create],
    },
});

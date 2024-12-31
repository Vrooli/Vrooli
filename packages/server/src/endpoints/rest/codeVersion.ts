import { endpointsCodeVersion } from "@local/shared";
import { codeVersion_create, codeVersion_findMany, codeVersion_findOne, codeVersion_update } from "../generated";
import { CodeVersionEndpoints } from "../logic/codeVersion";
import { setupRoutes } from "./base";

export const CodeVersionRest = setupRoutes([
    [endpointsCodeVersion.findOne, CodeVersionEndpoints.Query.codeVersion, codeVersion_findOne],
    [endpointsCodeVersion.findMany, CodeVersionEndpoints.Query.codeVersions, codeVersion_findMany],
    [endpointsCodeVersion.createOne, CodeVersionEndpoints.Mutation.codeVersionCreate, codeVersion_create],
    [endpointsCodeVersion.updateOne, CodeVersionEndpoints.Mutation.codeVersionUpdate, codeVersion_update],
]);

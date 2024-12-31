import { endpointsCode } from "@local/shared";
import { code_create, code_findMany, code_findOne, code_update } from "../generated";
import { CodeEndpoints } from "../logic/code";
import { setupRoutes } from "./base";

export const CodeRest = setupRoutes([
    [endpointsCode.findOne, CodeEndpoints.Query.code, code_findOne],
    [endpointsCode.findMany, CodeEndpoints.Query.codes, code_findMany],
    [endpointsCode.createOne, CodeEndpoints.Mutation.codeCreate, code_create],
    [endpointsCode.updateOne, CodeEndpoints.Mutation.codeUpdate, code_update],
]);

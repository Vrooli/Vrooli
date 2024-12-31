import { endpointsEmail } from "@local/shared";
import { email_create, email_verify } from "../generated";
import { EmailEndpoints } from "../logic/email";
import { setupRoutes } from "./base";

export const EmailRest = setupRoutes([
    [endpointsEmail.createOne, EmailEndpoints.Mutation.emailCreate, email_create],
    [endpointsEmail.verify, EmailEndpoints.Mutation.sendVerificationEmail, email_verify],
]);

import { email_create, email_verify } from "../generated";
import { EmailEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const EmailRest = setupRoutes({
    "/email": {
        post: [EmailEndpoints.Mutation.emailCreate, email_create],
    },
    "/email/verification": {
        post: [EmailEndpoints.Mutation.sendVerificationEmail, email_verify],
    },
});

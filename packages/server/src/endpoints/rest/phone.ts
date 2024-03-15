import { phone_create, phone_validate, phone_verify } from "../generated";
import { PhoneEndpoints } from "../logic/phone";
import { setupRoutes } from "./base";

export const PhoneRest = setupRoutes({
    "/phone": {
        post: [PhoneEndpoints.Mutation.phoneCreate, phone_create],
    },
    "/phone/verificationText": {
        post: [PhoneEndpoints.Mutation.sendVerificationText, phone_verify],
    },
    "/phone/validateText": {
        post: [PhoneEndpoints.Mutation.validateVerificationText, phone_validate],
    },
});

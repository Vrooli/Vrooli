import { phone_create } from "../generated";
import { PhoneEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const PhoneRest = setupRoutes({
    "/phone": {
        post: [PhoneEndpoints.Mutation.phoneCreate, phone_create],
    },
    // "/phone/verificationText": {
    //     post: [PhoneEndpoints.Mutation.sendVerificationText, phone_verify],
    // },
});

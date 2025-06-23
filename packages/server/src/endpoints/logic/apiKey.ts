import { type ApiKey, type ApiKeyCreateInput, type ApiKeyCreated, type ApiKeyUpdateInput, type ApiKeyValidateInput } from "@vrooli/shared";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { RequestService } from "../../auth/request.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints } from "../helpers/endpointFactory.js";

export type EndpointsApiKey = {
    createOne: ApiEndpoint<ApiKeyCreateInput, ApiKeyCreated>;
    updateOne: ApiEndpoint<ApiKeyUpdateInput, ApiKey>;
    validate: ApiEndpoint<ApiKeyValidateInput, ApiKey>;
}

const objectType = "ApiKey";
export const apiKey: EndpointsApiKey = createStandardCrudEndpoints({
    objectType,
    endpoints: {
        createOne: {
            rateLimit: { maxUser: 25 },
            permissions: { hasWriteAuthPermissions: true },
            customImplementation: async ({ input, req, info }) => {
                RequestService.assertRequestFrom(req, { isOfficialUser: true });
                const { createOneHelper } = await import("../../actions/creates.js");
                return createOneHelper({ info, input, objectType, req });
            },
        },
        updateOne: {
            rateLimit: { maxUser: 25 },
            permissions: { hasWriteAuthPermissions: true },
            customImplementation: async ({ input, req, info }) => {
                RequestService.assertRequestFrom(req, { isOfficialUser: true });
                const { updateOneHelper } = await import("../../actions/updates.js");
                return updateOneHelper({ info, input, objectType, req });
            },
        },
    },
    customEndpoints: {
        validate: async (wrapped, { req }, info) => {
            const validateInput = wrapped?.input;
            await RequestService.get().rateLimit({ maxApi: 5000, req });

            if (!validateInput || typeof validateInput.id !== "string" || typeof validateInput.secret !== "string") {
                throw new CustomError("0900", "InvalidArgs", { message: "API Key ID and secret are required and must be strings." });
            }

            let apiKeyId: bigint;
            try {
                apiKeyId = BigInt(validateInput.id);
            } catch (error) {
                throw new CustomError("0901", "InvalidArgs", { message: "Invalid API Key ID format." });
            }

            const prisma = DbProvider.get();
            const apiKeyRecord = await prisma.api_key.findUnique({
                where: { id: apiKeyId },
            });

            if (!apiKeyRecord) {
                throw new CustomError("0902", "NotFound", { message: "API Key not found." });
            }

            let decryptedStoredKey: string;
            try {
                decryptedStoredKey = ApiKeyEncryptionService.get().decryptExternal(apiKeyRecord.key);
            } catch (error) {
                console.error("Error decrypting API key:", { trace: "0903-decrypt", error });
                throw new CustomError("0903", "InternalError", { message: "Failed to process API Key." });
            }

            if (decryptedStoredKey !== validateInput.secret) {
                throw new CustomError("0904", "InvalidCredentials", { message: "Invalid API Key secret." });
            }

            if (apiKeyRecord.disabledAt) {
                throw new CustomError("0905", "Unauthorized", { message: "API Key is disabled." });
            }

            if (apiKeyRecord.stopAtLimit && apiKeyRecord.creditsUsed >= apiKeyRecord.limitHard) {
                throw new CustomError("0906", "CostLimitExceeded", { message: "API Key usage limit exceeded." });
            }

            const converter = InfoConverter.get();
            return converter.fromDbToApi<ApiKey>(apiKeyRecord, info);
        },
    },
});

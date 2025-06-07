import { type ApiKey, type ApiKeyCreateInput, type ApiKeyCreated, type ApiKeyUpdateInput, type ApiKeyValidateInput } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { updateOneHelper } from "../../actions/updates.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { RequestService } from "../../auth/request.js";
import { InfoConverter } from "../../builders/infoConverter.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsApiKey = {
    createOne: ApiEndpoint<ApiKeyCreateInput, ApiKeyCreated>;
    updateOne: ApiEndpoint<ApiKeyUpdateInput, ApiKey>;
    validate: ApiEndpoint<ApiKeyValidateInput, ApiKey>;
}

const objectType = "ApiKey";
export const apiKey: EndpointsApiKey = {
    createOne: async ({ input }, { req }, info) => {
        RequestService.assertRequestFrom(req, { isOfficialUser: true });
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        return createOneHelper({ info, input, objectType, req });
        //TODO need to make sure non-encrypted key is stored during key generation and added to the response
    },
    updateOne: async ({ input }, { req }, info) => {
        RequestService.assertRequestFrom(req, { isOfficialUser: true });
        await RequestService.get().rateLimit({ maxUser: 10, req });
        RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
        return updateOneHelper({ info, input, objectType, req });
    },
    validate: async ({ input: validateInput }, { req }, info) => {
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
};

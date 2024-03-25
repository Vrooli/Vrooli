// import GoogleClient, { ChatCompletionResponse } from "@mistralai/mistralai";
// import { logger } from "../../../events/logger";
// import { ChatContextCollector } from "../context";
// import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

// type GoogleGenerateModel = "open-mistral-7b" | "open-mixtral-8x7b";
// type GoogleTokenModel = "default";

// export class GoogleService implements LanguageModelService<GoogleGenerateModel, GoogleTokenModel> {
//     public __id = "google";
//     private client: GoogleClient;
//     private defaultModel: GoogleGenerateModel = "open-mistral-7b";

//     constructor() {
//         this.client = new GoogleClient(process.env.GOOGLE_API_KEY);
//     }

//     estimateTokens(params: EstimateTokensParams) {
//         return tokenEstimationDefault(params);
//     }

//     async getConfigObject(params: GetConfigObjectParams) {
//         return getDefaultConfigObject(params);
//     }

//     async generateContext(params: GenerateContextParams): Promise<LanguageModelContext> {
//         return generateDefaultContext({
//             ...params,
//             service: this,
//         });
//     }

//     async generateResponse({
//         contextInfo,
//         force,
//         model,
//         participantsData,
//         respondingBotConfig,
//         respondingBotId,
//         respondingToMessage,
//         task,
//         userData,
//     }: GenerateResponseParams) {
// // Check if service is active
//         const { messages, systemMessage } = await this.generateContext({
//             contextInfo,
//             force,
//             model,
//             participantsData,
//             respondingBotConfig,
//             respondingBotId,
//             task,
//             userData,
//         });

//         // Ensure roles alternate between "user" and "assistant". This is a requirement of the Google API.
//         const alternatingMessages: LanguageModelMessage[] = [];
//         const messagesWithResponding = respondingToMessage ? [...messages, { role: "user" as const, content: respondingToMessage.text }] : messages;
//         let lastRole: LanguageModelMessage["role"] = "assistant";
//         for (const { role, content } of messagesWithResponding) {
//             // Skip empty messages. This is another requirement of the Google API.
//             if (content.trim() === "") {
//                 continue;
//             }
//             if (role !== lastRole) {
//                 alternatingMessages.push({ role, content });
//                 lastRole = role;
//             } else {
//                 // Merge consecutive messages with the same role
//                 if (alternatingMessages.length > 0) {
//                     alternatingMessages[alternatingMessages.length - 1].content += "\n" + content;
//                 } else {
//                     alternatingMessages.push({ role, content });
//                 }
//             }
//         }

//         // Ensure first message is from the user. This is another requirement of the Google API.
//         if (alternatingMessages.length && alternatingMessages[0].role === "assistant") {
//             alternatingMessages.shift();
//         }

//         const params = {
//             messages: [
//                 // Add system message first
//                 { role: "system" as const, content: systemMessage },
//                 // Add other messages
//                 ...alternatingMessages.map(({ role, content }) => ({ role, content })),
//             ],
//             model,
//             max_tokens: 1024, // Adjust as needed
//         };

//         const completion: ChatCompletionResponse = await this.client
//             .chat(params)
//             .catch((error) => {
// const trace = "0011";
// const errorType = this.getErrorType(error);
// LlmServiceRegistry.get().updateServiceState(this.__id, errorType)
// logger.error("Failed to call Google", { trace, error, errorType });
// throw new CustomError(trace, "InternalError", userData.languages, { error, errorType });
//             });
//         return completion.choices[0].message.content ?? "";
//     }

//     // eslint-disable-next-line @typescript-eslint/no-unused-vars
//     getContextSize(_requestedModel?: string | null) {
//         return 32000;
//     }

//     // eslint-disable-next-line @typescript-eslint/no-unused-vars
//     getEstimationMethod(_requestedModel?: string | null | undefined): "default" {
//         return "default";
//     }

//     getEstimationTypes() {
//         return ["default"] as const;
//     }

//     getModel(requestedModel?: string | null) {
//         if (typeof requestedModel !== "string") return this.defaultModel;
//         if (requestedModel.includes("8x7b")) return "open-mixtral-8x7b";
//         if (requestedModel.includes("mistral-7b")) return "open-mistral-7b";
//         return this.defaultModel;
//     }

// getErrorType(error: unknown) {
//     return LlmServiceErrorType.Authentication; //TODO
// }
// }

export { };


import { MistralModel, mistralServiceInfo } from "@local/shared";
import MistralClient, { ChatCompletionResponse } from "@mistralai/mistralai";
import { CustomError } from "../../../events/error";
import { logger } from "../../../events/logger";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry } from "../registry";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetOutputTokenLimitParams, GetOutputTokenLimitResult, GetResponseCostParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, getDefaultMaxOutputTokensRestrained, getDefaultResponseCost, tokenEstimationDefault } from "../service";

type MistralTokenModel = "default";

export class MistralService implements LanguageModelService<MistralModel, MistralTokenModel> {
    public __id = LlmServiceId.Mistral;
    private client: MistralClient;
    private defaultModel: MistralModel = MistralModel.Nemo;

    constructor() {
        this.client = new MistralClient(process.env.MISTRAL_API_KEY);
    }

    estimateTokens(params: EstimateTokensParams) {
        return tokenEstimationDefault(params);
    }

    async getConfigObject(params: GetConfigObjectParams) {
        return getDefaultConfigObject(params);
    }

    async generateContext(params: GenerateContextParams): Promise<LanguageModelContext> {
        const { messages, systemMessage } = await generateDefaultContext({
            ...params,
            service: this,
        });

        // Ensure roles alternate between "user" and "assistant". This is a requirement of the Mistral API.
        const alternatingMessages: LanguageModelMessage[] = [];
        let lastRole: LanguageModelMessage["role"] = "assistant";
        for (const { role, content } of messages) {
            // Skip empty messages. This is another requirement of the Mistral API.
            if (content.trim() === "") {
                continue;
            }
            if (role !== lastRole) {
                alternatingMessages.push({ role, content });
                lastRole = role;
            } else {
                // Merge consecutive messages with the same role
                if (alternatingMessages.length > 0) {
                    alternatingMessages[alternatingMessages.length - 1].content += "\n" + content;
                } else {
                    alternatingMessages.push({ role, content });
                }
            }
        }

        // Ensure first message is from the user. This is another requirement of the Mistral API.
        if (alternatingMessages.length && alternatingMessages[0].role === "assistant") {
            alternatingMessages.shift();
        }

        const messagesWithContext = [
            // Add system message first
            { role: "system" as const, content: systemMessage },
            // Add other messages
            ...alternatingMessages.map(({ role, content }) => ({ role, content })),
        ] as LanguageModelMessage[];

        return { messages: messagesWithContext, systemMessage };
    }

    async generateResponse({
        maxTokens,
        messages,
        model,
        userData,
    }: GenerateResponseParams) {
        // Generate response
        const params = {
            messages,
            model,
            max_tokens: maxTokens ?? mistralServiceInfo.fallbackMaxTokens,
        } as const;
        const completion: ChatCompletionResponse = await this.client
            .chat(params)
            .catch((error) => {
                const trace = "0249";
                const errorType = this.getErrorType(error);
                LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
                logger.error("Failed to call Mistral", { trace, error, errorType });
                throw new CustomError(trace, "InternalError", { error, errorType });
            });
        const message = completion.choices[0].message.content ?? "";
        const cost = this.getResponseCost({
            model,
            usage: {
                input: completion.usage.prompt_tokens,
                output: completion.usage?.completion_tokens,
            },
        });
        return { attempts: 1, cost, message };
    }

    async *generateResponseStreaming({
        maxTokens,
        messages,
        model,
    }: GenerateResponseParams) {
        const params = {
            max_tokens: maxTokens ?? mistralServiceInfo.fallbackMaxTokens,
            model,
            messages,
        } as const;

        let accumulatedMessage = "";
        // NOTE: Mistral's API might not provide token usage when streaming. You'll need to estimate it yourself.
        const inputTokens = this.estimateTokens({ model, text: messages.map(m => m.content).join("\n") }).tokens;
        let accumulatedOutputTokens = 0;

        try {
            // Create the stream
            const chatStreamResponse = this.client.chatStream(params);
            for await (const chunk of chatStreamResponse) {
                const content = chunk.choices[0]?.delta?.content || "";
                accumulatedMessage += content;
                const outputTokens = this.estimateTokens({ model, text: content }).tokens;
                accumulatedOutputTokens += outputTokens;
                const cost = this.getResponseCost({
                    model,
                    usage: {
                        input: inputTokens,
                        output: accumulatedOutputTokens, // Update this as you go
                    },
                });
                yield { __type: "stream" as const, message: content, cost };
            }
            // Return the final message
            const cost = this.getResponseCost({
                model,
                usage: {
                    input: inputTokens,
                    output: this.estimateTokens({ model, text: accumulatedMessage }).tokens,
                },
            });
            yield { __type: "end" as const, message: accumulatedMessage, cost };
        } catch (error) {
            const trace = "0324";
            const errorType = this.getErrorType(error);
            LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to stream from Mistral", { trace, error, errorType });

            const cost = this.getResponseCost({
                model,
                usage: {
                    input: inputTokens,
                    output: this.estimateTokens({ model, text: accumulatedMessage }).tokens,
                },
            });
            yield { __type: "error" as const, message: accumulatedMessage, cost };
        }
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        return mistralServiceInfo.models[model].contextWindow;
    }

    getModelInfo() {
        return mistralServiceInfo.models;
    }

    getMaxOutputTokens(requestedModel?: string | null | undefined): number {
        const model = this.getModel(requestedModel);
        return mistralServiceInfo.models[model].maxOutputTokens;
    }

    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number {
        return getDefaultMaxOutputTokensRestrained(params, this);
    }

    getResponseCost(params: GetResponseCostParams) {
        return getDefaultResponseCost(params, this);
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    getEstimationMethod(_model?: string | null | undefined): "default" {
        return "default";
    }

    getEstimationTypes() {
        return ["default"] as const;
    }

    getModel(model?: string | null) {
        if (typeof model !== "string") return this.defaultModel;
        if (model.includes("codestral")) return MistralModel.Codestral;
        if (model.includes("large")) return MistralModel.Large2;
        if (model.includes("nemo")) return MistralModel.Nemo;
        return this.defaultModel;
    }

    getErrorType(error: unknown) {
        return LlmServiceErrorType.Authentication; //TODO can't find error codes
    }

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        const moderationPrompt = `
    You're given a list of moderation categories as below:
    
    - illegal: Illegal activity.
    - child abuse: child sexual abuse material or any content that exploits or harms children.
    - hate violence harassment: Generation of hateful, harassing, or violent content: content that expresses, incites, or promotes hate based on identity, content that intends to harass, threaten, or bully an individual, content that promotes or glorifies violence or celebrates the suffering or humiliation of others.
    - malware: Generation of malware: content that attempts to generate code that is designed to disrupt, damage, or gain unauthorized access to a computer system.
    - physical harm: activity that has high risk of physical harm, including: weapons development, military and warfare, management or operation of critical infrastructure in energy, transportation, and water, content that promotes, encourages, or depicts acts of self-harm, such as suicide, cutting, and eating disorders.
    - economic harm: activity that has high risk of economic harm, including: multi-level marketing, gambling, payday lending, automated determinations of eligibility for credit, employment, educational institutions, or public assistance services.
    - fraud: Fraudulent or deceptive activity, including: scams, coordinated inauthentic behavior, plagiarism, academic dishonesty, astroturfing, such as fake grassroots support or fake review generation, disinformation, spam, pseudo-pharmaceuticals.
    - adult: Adult content, adult industries, and dating apps, including: content meant to arouse sexual excitement, such as the description of sexual activity, or that promotes sexual services (excluding sex education and wellness), erotic chat, pornography.
    - political: Political campaigning or lobbying, by: generating high volumes of campaign materials, generating campaign materials personalized to or targeted at specific demographics, building conversational or interactive systems such as chatbots that provide information about campaigns or engage in political advocacy or lobbying, building products for political campaigning or lobbying purposes.
    - privacy: Activity that violates people's privacy, including: tracking or monitoring an individual without their consent, facial recognition of private individuals, classifying individuals based on protected characteristics, using biometrics for identification or assessment, unlawful collection or disclosure of personal identifiable information or educational, financial, or other protected records.
    - unqualified law: Engaging in the unauthorized practice of law, or offering tailored legal advice without a qualified person reviewing the information.
    - unqualified financial: Offering tailored financial advice without a qualified person reviewing the information.
    - unqualified health: Telling someone that they have or do not have a certain health condition, or providing instructions on how to cure or treat a health condition.
    
    Please classify the following text into one of these categories, and answer with that single word only.
    
    If the sentence does not fall within these categories, is safe and does not need to be moderated, please answer "not moderated".
    
    Text to classify: "${input}"
    `;
        const moderationModel = MistralModel.Nemo;

        try {
            const params = {
                model: moderationModel,
                messages: [{ role: "user", content: moderationPrompt }],
                max_tokens: 10,
                temperature: 0,
                safe_prompt: true, // Enable Mistral's safety prompt
            };

            const completion: ChatCompletionResponse = await this.client
                .chat(params)
                .catch((error) => {
                    const trace = "0423";
                    const errorType = this.getErrorType(error);
                    LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
                    logger.error("Failed to perform content moderation", { trace, error, errorType });
                    throw new CustomError(trace, "InternalError", { error, errorType });
                });

            const moderationResult = completion.choices[0].message.content.trim().toLowerCase();

            console.log("Mistral moderation result:", moderationResult);

            // If the result is "not moderated", the content is safe
            const isSafe = moderationResult.trim().toLocaleLowerCase() === "not moderated";
            const cost = this.getResponseCost({
                model: moderationModel,
                usage: {
                    input: completion.usage.prompt_tokens,
                    output: completion.usage?.completion_tokens,
                },
            });
            return { cost, isSafe };
        } catch (error) {
            const trace = "0423";
            const errorType = this.getErrorType(error);
            LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to perform content moderation", { trace, error, errorType });

            // In case of an error, we assume the input is not safe
            return { cost: 0, isSafe: false };
        }
    }
}

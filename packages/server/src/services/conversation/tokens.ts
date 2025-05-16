import { type Tiktoken, type TiktokenEncoding, type TiktokenModel } from "tiktoken";
import { logger } from "../../events/logger.js";
import { TokenEstimatorType, type EstimateTokensParams, type EstimateTokensResult } from "./services.js";

/** Interface for token estimators */
interface TokenEstimator {
    /** Whether the estimator is initialized */
    isInitialized: boolean;
    /**
     * Estimates the number of tokens for the given text and model.
     * @param params - The parameters for token estimation.
     * @returns The estimation result.
     */
    estimateTokens(params: EstimateTokensParams): EstimateTokensResult;
    /**
     * @returns the estimation model and encoding for the model
     */
    getEstimationInfo(model?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding">;
    /**
     * Optional initialization method for estimators that need asynchronous setup.
     * @returns A promise that resolves when initialization is complete.
     */
    init?(): Promise<void>;
}

/**
 * Default token estimator that estimates tokens based on half the UTF-8 byte length of the text.
 * 
 * WARNING: This is a fallback method and may overestimate or underestimate the true token count. 
 * Only use this if you don't have any other options.
 */
class DefaultTokenEstimator implements TokenEstimator {
    isInitialized = true;
    /**
     * Basic token estimation used as a fallback when an actual LLM tokenizer is unavailable.
     * 
     * This implementation estimates tokens based on the UTF-8 byte length of the input text,
     * assigning approximately one token per 2 bytes (rounded up). This approach:
     * - **Handles Unicode Robustly**: Unlike character-count methods, it accounts for complex
     *   Unicode sequences (e.g., combining characters, emoji), where a single visual character
     *   may consist of multiple code points or bytes. This prevents underestimation in cases
     *   like Unicode selector attacks, where attackers pack data into dense characters.
     * - **Language-Agnostic**: Avoids word-splitting assumptions, making it suitable for all
     *   scripts, including those without spaces (e.g., Chinese, Japanese).
     * - **Conservative Estimation**: Overestimates tokens for typical text (e.g., English at
     *   ~2-3 bytes/token) to ensure safety in resource allocation, while scaling appropriately
     *   for text with many combining marks (closer to 1 token per byte).
     * 
     * The heuristic of `Math.ceil(byteLength / 2)` is chosen because:
     * - Modern LLM tokenizers (e.g., BPE on UTF-8) often process text at the byte level,
     *   where common tokens span multiple bytes, and complex characters may split into
     *   multiple tokens.
     * - It balances simplicity and robustness without requiring external dependencies beyond
     *   standard JavaScript APIs (`TextEncoder`).
     * 
     * Note: This is a fallback method and may overestimate tokens (e.g., "Hello world" might
     * estimate 6 tokens vs. ~3 actual). For precise counts, use the target LLM’s tokenizer.
     * 
     * @param params Parameters for token estimation
     * @returns An object containing the model name ("default") and the estimated token count
     */
    public estimateTokens(params: EstimateTokensParams): EstimateTokensResult {
        const encoder = new TextEncoder();
        const byteLength = encoder.encode(params.text).length;
        // Estimate tokens as half the byte length, rounded up
        const tokens = Math.ceil(byteLength / 2);
        return {
            ...this.getEstimationInfo(),
            tokens,
        };
    }

    getEstimationInfo(): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        return {
            estimationModel: TokenEstimatorType.Default,
            encoding: "default",
        };
    }
}

class TiktokenWasmEstimator implements TokenEstimator {
    isInitialized = false;

    private defaultEncodingName = "cl100k_base";
    private encodingForModel?: (model: TiktokenModel) => Tiktoken;
    private getEncoding?: (encoding: TiktokenEncoding) => Tiktoken;

    /**
     * Initializes the estimator by dynamically importing the tiktoken module.
     * This sets up the encoding_for_model and get_encoding functions for later use.
     * @returns A promise that resolves when initialization is complete.
     */
    public async init(): Promise<void> {
        const { encoding_for_model, get_encoding } = await import("tiktoken");
        this.encodingForModel = encoding_for_model;
        this.getEncoding = get_encoding;
        this.isInitialized = true;
    }

    /**
     * Estimates the number of tokens in the provided text using the tiktoken library.
     * Attempts to use the encoding specific to the given model; falls back to a default
     * encoding if the model is not recognized.
     * @param params - Parameters containing the model name and text to estimate.
     * @returns An object with the model name "tiktoken" and the token count.
     */
    public estimateTokens(params: EstimateTokensParams): EstimateTokensResult {
        const { aiModel, text } = params;
        let encoding: Tiktoken;

        if (!this.encodingForModel || !this.getEncoding) {
            throw new Error("TiktokenWasmEstimator not initialized");
        }

        try {
            // Attempt to get the encoding for the specified model
            encoding = this.encodingForModel(aiModel as TiktokenModel);
        } catch (error) {
            // Fall back to default encoding if model-specific encoding fails
            encoding = this.getEncoding(this.defaultEncodingName as TiktokenEncoding);
        }
        const encodingName = encoding.name ?? this.defaultEncodingName;

        // Encode the text and calculate token count
        const tokens = encoding.encode(text);
        const tokenCount = tokens.length;

        // Free the encoding object to release resources
        encoding.free();

        return {
            encoding: encodingName,
            estimationModel: TokenEstimatorType.Tiktoken,
            tokens: tokenCount,
        };
    }

    getEstimationInfo(aiModel?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        let encoding: Tiktoken;

        if (!this.encodingForModel || !this.getEncoding) {
            throw new Error("TiktokenWasmEstimator not initialized");
        }

        try {
            encoding = this.encodingForModel(aiModel as TiktokenModel);
        } catch (error) {
            // Fall back to default encoding if model-specific encoding fails
            encoding = this.getEncoding(this.defaultEncodingName as TiktokenEncoding);
        }
        const encodingName = encoding.name ?? this.defaultEncodingName;

        return {
            estimationModel: TokenEstimatorType.Tiktoken,
            encoding: encodingName,
        };
    }
}

const estimators: Record<TokenEstimatorType, TokenEstimator> = {
    [TokenEstimatorType.Default]: new DefaultTokenEstimator(),
    [TokenEstimatorType.Tiktoken]: new TiktokenWasmEstimator(),
};

/**
 * Singleton class for managing token estimators.
 * 
 * These are used to estimate the number of tokens (and thus the cost) of passing
 * a given string to an LLM.
 */
export class TokenEstimationRegistry {
    private static instance: TokenEstimationRegistry | undefined;
    private defaultEstimator: TokenEstimator = estimators[TokenEstimatorType.Default];

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Gets the singleton instance of the registry.
     * 
     * @returns The TokenEstimationRegistry instance.
     */
    public static get(): TokenEstimationRegistry {
        if (!TokenEstimationRegistry.instance) {
            TokenEstimationRegistry.instance = new TokenEstimationRegistry();
        }
        return TokenEstimationRegistry.instance;
    }

    /**
     * Initializes all registered estimators that have an init method.
     * 
     * @returns A promise that resolves when all initializations are complete.
     */
    static async init() {
        for (const estimator of Object.values(estimators)) {
            if (estimator.init) {
                try {
                    await estimator.init();
                } catch (error) {
                    logger.error(`Error initializing estimator ${estimator.constructor.name}: ${error}`);
                }
            }
        }
    }

    /**
     * Estimates the number of tokens for the given text, model, and desired estimator type.
     * 
     * @param requestedEstimator The desired estimator type.
     * @param estimatorParams The parameters for token estimation.
     * @returns The estimation result.
     */
    public estimateTokens(requestedEstimator: TokenEstimatorType, estimatorParams: EstimateTokensParams): EstimateTokensResult {
        let estimator = estimators[requestedEstimator];
        if (!estimator || !estimator.isInitialized) {
            estimator = this.defaultEstimator;
        }
        return estimator.estimateTokens(estimatorParams);
    }

    /**
     * Gets the estimation info for the given estimator type and parameters.
     * 
     * @param requestedEstimator The desired estimator type.
     * @param aiModel The AI model to get the estimation info for.
     * @returns The estimation info.
     */
    public getEstimationInfo(requestedEstimator: TokenEstimatorType, aiModel?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        let estimator = estimators[requestedEstimator];
        if (!estimator || !estimator.isInitialized) {
            estimator = this.defaultEstimator;
        }
        return estimator.getEstimationInfo(aiModel);
    }
}

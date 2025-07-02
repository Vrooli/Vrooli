/**
 * Method for token estimation
 */
export enum TokenEstimatorType {
    Default = "Default",
    Tiktoken = "Tiktoken",
}

export type EstimateTokensParams = {
    /** The requested model to base token logic on */
    aiModel: string;
    /** The text to estimate tokens for */
    text: string;
};

export type EstimateTokensResult = {
    /** The encoding used for the token estimation */
    encoding: string;
    /** The name of the token estimation model used (if requested one was invalid/incomplete) */
    estimationModel: TokenEstimatorType;
    /** The estimated amount of tokens calculated by this method/encoding pair */
    tokens: number;
};

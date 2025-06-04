


/**
 * Default cost calculator for language model input and output tokens.
 * 
 * NOTE: input and output costs should be in cents per API_CREDITS_MULTIPLIER tokens. 
 * Since credits are stored with this multiplier already, it should cancel out (meaning 
 * we won't be using API_CREDITS_MULTIPLIER in the calculation).
 */
function getDefaultResponseCost<GenerateNameType extends string>(
    { model, usage }: GetResponseCostParams,
    service: LanguageModelService<GenerateNameType>,
) {
    const { input, output } = usage;
    const modelToUse = service.getModel(model);
    const modelInfo = service.getModelInfo()[modelToUse];

    if (!modelInfo || !modelInfo.inputCost || !modelInfo.outputCost) {
        throw new Error(`Model "${model}" (converted to ${modelToUse}) not found in cost records`);
    }

    return Math.max((modelInfo.inputCost * input), 0) + Math.max((modelInfo.outputCost * output), 0);
}

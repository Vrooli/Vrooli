export * from "./TailwindTextInput.js";
export * from "./types.js";
export * from "./textInputStyles.js";

// Export TailwindTextInput components as the default TextInput
export { TailwindTextInput as TextInput, TranslatedTailwindTextInput as TranslatedTextInput } from "./TailwindTextInput.js";

// Keep the original TextInput available under different names if needed
export { TextInput as MuiTextInput, TranslatedTextInput as TranslatedMuiTextInput } from "./TextInput.js";
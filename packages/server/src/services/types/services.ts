import type { Responses } from "openai/resources/responses/responses.js";

type OpenAITool = Responses.Tool;

export type CodeInterpreterSpec = Extract<OpenAITool, { type: "code_interpreter" }>;
export type ImageGenerationSpec = Extract<OpenAITool, { type: "image_generation" }>;
export type MCPSpec = Extract<OpenAITool, { type: "mcp" }>;
export type FileSearchSpec = Responses.FileSearchTool;
export type WebSearchPreviewSpec = Responses.WebSearchTool;
export type ComputerUsePreviewSpec = Responses.ComputerTool;

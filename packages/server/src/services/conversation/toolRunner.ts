import { OkErr, ToolMeta } from "./types.js";

export abstract class ToolRunner {
    /**
     * Dispatch a tool call –  must validate args against schema, call MCP
     * server (or perform side‑effects) and return a structured result.
     */
    abstract run(
        name: string,
        args: unknown,
        meta: ToolMeta
    ): Promise<OkErr<unknown>>;
}

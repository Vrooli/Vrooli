/** Byte bounds cover retained encoded audio; browser decoder/heap costs are additional. */
export async function readAudio(response: Response, limit: number): Promise<ArrayBuffer> {
    const budgetError = () => new Error("Audition audio exceeds the browser budget. Unload a candidate before loading another.");
    if (Number(response.headers?.get("Content-Length")) > limit) {
        await response.body?.cancel();
        throw budgetError();
    }
    const reader = response.body?.getReader();
    if (!reader) {
        const bytes = await response.arrayBuffer();
        if (bytes.byteLength > limit)
            throw budgetError();
        return bytes;
    }
    const chunks: Uint8Array[] = [];
    let total = 0;
    try {
        while (true) {
            const { value, done } = await reader.read();
            if (done)
                break;
            total += value.byteLength;
            if (total > limit) {
                await reader.cancel();
                throw budgetError();
            }
            chunks.push(value);
        }
    }
    finally {
        reader.releaseLock();
    }
    const bytes = new Uint8Array(total);
    let offset = 0;
    for (const chunk of chunks) {
        bytes.set(chunk, offset);
        offset += chunk.byteLength;
    }
    return bytes.buffer;
}

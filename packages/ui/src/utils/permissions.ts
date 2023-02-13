/**
 * Requests microphone or camera permission from the user.
 * @returns The stream if permission was granted, undefined otherwise.
 */
export async function requestStream(type: 'audio' | 'video'): Promise<MediaStream | undefined> {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ [type]: true });
        console.info("Microphone permission granted!");
        return stream;
    } catch (err) {
        console.error("Microphone permission denied: ", err);
    }
}

/**
 * Stops a stream.
 */
export function stopStream(stream: MediaStream) {
    stream.getTracks().forEach(track => track.stop());
}
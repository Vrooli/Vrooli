export async function requestStream(type) {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ [type]: true });
        console.info("Microphone permission granted!");
        return stream;
    }
    catch (err) {
        console.error("Microphone permission denied: ", err);
    }
}
export function stopStream(stream) {
    stream.getTracks().forEach(track => track.stop());
}
//# sourceMappingURL=permissions.js.map
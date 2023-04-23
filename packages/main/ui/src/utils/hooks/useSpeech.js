import { useEffect, useState } from "react";
import { PubSub } from "../pubsub";
export const useSpeech = () => {
    const [isSpeechSupported, setIsSpeechSupported] = useState(false);
    useEffect(() => {
        if ("SpeechRecognition" in window || "webkitSpeechRecognition" in window) {
            setIsSpeechSupported(true);
        }
    }, []);
    const [speechRecognition, setSpeechRecognition] = useState(null);
    const [transcript, setTranscript] = useState("");
    const [isListening, setIsListening] = useState(false);
    const startListening = () => {
        if (isSpeechSupported) {
            const sr = new (window.SpeechRecognition || window.webkitSpeechRecognition)();
            sr.lang = navigator.language;
            sr.interimResults = true;
            sr.start();
            sr.addEventListener("result", (event) => {
                const currentTranscript = event.results[0][0].transcript;
                setTranscript(currentTranscript);
            });
            sr.addEventListener("end", () => {
                setIsListening(false);
            });
            sr.addEventListener("error", (event) => {
                console.error("Speech recognition error detected: ", event.error);
                if (event.error === "not-allowed") {
                    PubSub.get().publishSnack({ messageKey: "MicrophonePermissionDenied", severity: "Error" });
                }
            });
            setIsListening(true);
            setSpeechRecognition(sr);
        }
    };
    const stopListening = () => {
        if (isSpeechSupported) {
            speechRecognition.stop();
            setIsListening(false);
        }
    };
    const resetTranscript = () => {
        setTranscript("");
    };
    return {
        transcript,
        isListening,
        isSpeechSupported,
        startListening,
        stopListening,
        resetTranscript,
    };
};
//# sourceMappingURL=useSpeech.js.map
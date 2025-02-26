import { useEffect, useState } from "react";
import { DeviceOS, getDeviceInfo } from "utils/display/device";
import { PubSub } from "utils/pubsub.js";

/**
 * A hook for converting speech to a transcript
 *
 * @returns {Object} An object containing the following properties:
 *    - transcript: the text of the speech that has been recorded
 *    - isListening: a boolean indicating whether speech is being recorded
 *    - isSpeechSupported: a boolean indicating whether the browser supports speech recognition
 *    - startListening: a function to start recording speech
 *    - stopListening: a function to stop recording speech
 *    - resetTranscript: a function to reset the transcript to an empty string
 */
export function useSpeech() {
    // Check if speech recognition is supported in the browser
    const [isSpeechSupported, setIsSpeechSupported] = useState(false);
    useEffect(() => {
        // Doesn't work on iOS right now
        const { deviceOS } = getDeviceInfo();
        if (deviceOS === DeviceOS.IOS || deviceOS === DeviceOS.MacOS) { setIsSpeechSupported(false); }
        else if ("SpeechRecognition" in window || "webkitSpeechRecognition" in window) { setIsSpeechSupported(true); }
    }, []);

    // State for the speech recognition object
    const [speechRecognition, setSpeechRecognition] = useState<any>(null);

    // State for the transcript of the speech
    const [transcript, setTranscript] = useState("");

    // State for whether the speech recognition is listening
    const [isListening, setIsListening] = useState(false);

    // Function to start listening to speech
    function startListening() {
        if (isSpeechSupported) {
            // Create a new instance of SpeechRecognition
            const sr: any = new ((window as any).SpeechRecognition || (window as any).webkitSpeechRecognition)();
            // Set the language for speech recognition
            sr.lang = navigator.language;
            // Set the interim results
            sr.interimResults = true;
            // Start recognition
            sr.start();
            // Add event listener for when a speech result is received
            sr.addEventListener("result", (event) => {
                // Get the transcript from the event
                const currentTranscript = event.results[0][0].transcript;
                // Set the transcript state
                setTranscript(currentTranscript);
            });
            // Add event listener for when speech recognition ends
            sr.addEventListener("end", () => {
                setIsListening(false);
            });
            // Add event listener to detect errors
            sr.addEventListener("error", (event) => {
                console.error("Speech recognition error detected: ", event.error);
                // If error is not-allowed, then show a message to the user
                if (event.error === "not-allowed") {
                    PubSub.get().publish("snack", { messageKey: "MicrophonePermissionDenied", severity: "Error" });
                }
            });
            setIsListening(true);
            setSpeechRecognition(sr);
        }
    }

    // Function to stop listening to speech
    function stopListening() {
        if (isSpeechSupported) {
            speechRecognition.stop();
            setIsListening(false);
        }
    }

    // Function to reset the transcript
    function resetTranscript() {
        setTranscript("");
    }

    return {
        transcript,
        isListening,
        isSpeechSupported,
        startListening,
        stopListening,
        resetTranscript,
    };
}

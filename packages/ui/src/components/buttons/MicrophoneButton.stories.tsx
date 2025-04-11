// eslint-disable-next-line import/extensions
import { noop } from "@local/shared";
import { action } from "@storybook/addon-actions";
import React from "react";
import { MicrophoneButton, TranscriptDialog } from "./MicrophoneButton.js";

export default {
    title: "Components/Buttons/MicrophoneButton",
    component: MicrophoneButton,
};

const outerStyle = {
    width: "100px",
    padding: "20px",
    border: "1px solid #ccc",
} as const;
function Outer({ children }: { children: React.ReactNode }) {
    return (
        <div style={outerStyle}>
            {children}
        </div>
    );
}

export function Default() {
    return (
        <Outer>
            <MicrophoneButton
                disabled={false}
                onTranscriptChange={action("onTranscriptChange")}
                showWhenUnavailable={true}
            />
        </Outer>
    );
}
Default.parameters = {
    docs: {
        description: {
            story: "Displays the microphone button.",
        },
    },
};

export function Disabled() {
    return (
        <Outer>
            <MicrophoneButton
                disabled={true}
                onTranscriptChange={action("onTranscriptChange")}
                showWhenUnavailable={true}
            />
        </Outer>
    );
}
Disabled.parameters = {
    docs: {
        description: {
            story: "Displays the microphone button in a disabled state.",
        },
    },
};

export function CustomSize() {
    return (
        <Outer>
            <MicrophoneButton
                height={30}
                onTranscriptChange={action("onTranscriptChange")}
                showWhenUnavailable={true}
                width={30}
            />
            <MicrophoneButton
                height={100}
                onTranscriptChange={action("onTranscriptChange")}
                showWhenUnavailable={true}
                width={100}
            />
        </Outer>
    );
}
CustomSize.parameters = {
    docs: {
        description: {
            story: "Displays the microphone button with a custom size.",
        },
    },
};

export function DialogNoTranscript() {
    return (
        <TranscriptDialog
            handleClose={noop}
            isListening={true}
            showHint={true}
            transcript=""
        />
    );
}
DialogNoTranscript.parameters = {
    docs: {
        description: {
            story: "Displays the transcript dialog with no transcript.",
        },
    },
};

export function DialogWithTranscript() {
    return (
        <TranscriptDialog
            handleClose={noop}
            isListening={true}
            showHint={true}
            transcript="Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
        />
    );
}
DialogWithTranscript.parameters = {
    docs: {
        description: {
            story: "Displays the transcript dialog with a transcript.",
        },
    },
};

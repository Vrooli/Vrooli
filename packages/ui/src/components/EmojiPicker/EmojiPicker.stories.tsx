import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { useState } from "react";
import { EmojiPicker } from "./EmojiPicker.js";

function Outer({ children }: { children: React.ReactNode }) {
    return (
        <Box
            height="500px"
            maxWidth="800px"
            padding="20px"
            border="1px solid #ccc"
            borderColor="divider"
            display="flex"
            alignItems="center"
            gap={2}
            p={2}
        >
            {children}
        </Box>
    );
}

/**
 * Basic usage of the EmojiPicker component.
 */
export default {
    title: "Components/EmojiPicker",
    component: EmojiPicker,
    decorators: [
        (Story) => (
            <Outer>
                <Story />
            </Outer>
        ),
    ],
};

export function Native() {
    const [selectedEmoji, setSelectedEmoji] = useState<string | null>(null);

    function handleEmojiSelect(emoji: string) {
        setSelectedEmoji(emoji);
    }

    return (
        <>
            <EmojiPicker onSelect={handleEmojiSelect} />
            <Typography>Selected emoji: {selectedEmoji}</Typography>
        </>
    );
}

export function Fallback() {
    const [selectedEmoji, setSelectedEmoji] = useState<string | null>(null);

    function handleEmojiSelect(emoji: string) {
        setSelectedEmoji(emoji);
    }

    return (
        <>
            <EmojiPicker disableNative onSelect={handleEmojiSelect} />
            <Typography>Selected emoji: {selectedEmoji}</Typography>
        </>
    );
}

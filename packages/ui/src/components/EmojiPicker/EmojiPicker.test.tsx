import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { render } from "../../__test/testUtils.js";
import { EmojiPicker } from "./EmojiPicker.js";

// Mock the emoji data hook
vi.mock("./useEmojiData.js", () => ({
    useEmojiData: () => ({
        emojiData: {},
    }),
}));

describe("EmojiPicker", () => {
    it("renders the emoji picker button", () => {
        const onSelect = vi.fn();
        render(<EmojiPicker onSelect={onSelect} />);

        const button = screen.getByRole("button");
        expect(button).toBeInTheDocument();
    });

    it("opens the fallback picker when clicked", async () => {
        const onSelect = vi.fn();
        render(<EmojiPicker onSelect={onSelect} />);

        const button = screen.getByRole("button");
        fireEvent.click(button);

        await waitFor(() => {
            expect(screen.getByRole("dialog")).toBeInTheDocument();
        });
    });

    it("has a working search input", async () => {
        const onSelect = vi.fn();
        render(<EmojiPicker onSelect={onSelect} />);

        // Open the picker
        const button = screen.getByRole("button");
        fireEvent.click(button);

        await waitFor(() => {
            const searchInput = screen.getByPlaceholderText("Search...");
            expect(searchInput).toBeInTheDocument();
        });
    });

    it("closes the picker when clicking outside", async () => {
        const onSelect = vi.fn();
        render(<EmojiPicker onSelect={onSelect} />);

        // Open the picker
        const button = screen.getByRole("button");
        fireEvent.click(button);

        // Wait for the picker to open
        await waitFor(() => {
            expect(screen.getByRole("dialog")).toBeInTheDocument();
        });

        // Click outside
        fireEvent.mouseDown(document.body);

        // Wait for the picker to close
        await waitFor(() => {
            expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
        });
    });

    it("calls onSelect when an emoji is selected", async () => {
        const onSelect = vi.fn();
        render(<EmojiPicker onSelect={onSelect} />);

        // Open the picker
        const button = screen.getByRole("button");
        fireEvent.click(button);

        // Wait for the picker to open and emojis to load
        await waitFor(() => {
            const emojiButtons = screen.getAllByRole("button");
            // Click the first emoji button (excluding the opener button)
            if (emojiButtons.length > 1) {
                fireEvent.click(emojiButtons[1]);
            }
        });

        expect(onSelect).toHaveBeenCalled();
    });
}); 

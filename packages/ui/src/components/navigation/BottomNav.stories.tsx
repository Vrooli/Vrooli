import { LINKS } from "@local/shared";
import { Meta, StoryObj } from "@storybook/react";
import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { clearMockedLocationForStorybook, mockLocationForStorybook } from "../../route/useLocation.js";
import { BottomNav } from "./BottomNav.js";

/**
 * The BottomNav component provides navigation options at the bottom of the screen on mobile devices.
 * It adapts its visibility based on the user's session and current location.
 */
const meta = {
    title: "Components/Navigation/BottomNav",
    component: BottomNav,
    parameters: {
        layout: "fullscreen",
    },
} satisfies Meta<typeof BottomNav>;

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default state with a regular user.
 */
export const LoggedOut: Story = {
    parameters: {
        session: loggedOutSession,
    },
    play: async () => {
        // Clear any mocked path to show normal behavior
        clearMockedLocationForStorybook();
    },
};

/**
 * State for a premium user.
 */
export const SignedIn: Story = {
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
    play: async () => {
        // Clear any mocked path to show normal behavior
        clearMockedLocationForStorybook();
    },
};

/**
 * Hidden when on chat page.
 */
export const HiddenOnChat: Story = {
    parameters: {
        session: signedInNoPremiumNoCreditsSession,
    },
    play: async () => {
        // Mock navigation to chat page which should hide the bottom nav
        mockLocationForStorybook(LINKS.Chat);
    },
};

/**
 * Hidden when at home page.
 */
export const HiddenOnHome: Story = {
    parameters: {
        session: signedInNoPremiumNoCreditsSession,
    },
    play: async () => {
        // Mock navigation to home page which should hide the bottom nav when signed in
        mockLocationForStorybook(LINKS.Home);
    },
};

const buttonRowStyle = {
    display: "flex",
    gap: "0.5rem",
} as const;

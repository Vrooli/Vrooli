import type { Meta, StoryObj } from "@storybook/react";
import { PrivacyPolicyView, TermsView } from "./PolicyView.js";

const meta = {
    title: "Views/PolicyView",
    component: PrivacyPolicyView,
    parameters: {
        layout: "fullscreen",
    },
    tags: ["autodocs"],
} satisfies Meta<typeof PrivacyPolicyView>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Privacy: Story = {
    args: {
        display: "page",
        onClose: () => console.log("Close clicked"),
    },
};

export const Terms: Story = {
    render: (args) => <TermsView {...args} />,
    args: {
        display: "page",
        onClose: () => console.log("Close clicked"),
    },
};

// Dialog view stories
export const PrivacyDialog: Story = {
    args: {
        display: "dialog",
        onClose: () => console.log("Close clicked"),
    },
};

export const TermsDialog: Story = {
    render: (args) => <TermsView {...args} />,
    args: {
        display: "dialog",
        onClose: () => console.log("Close clicked"),
    },
}; 

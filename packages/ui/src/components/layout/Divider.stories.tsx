import type { Meta, StoryObj } from "@storybook/react";
import { Divider, DividerFactory } from "./Divider.js";

const meta: Meta<typeof Divider> = {
    title: "Components/Layout/Divider",
    component: Divider,
    tags: ["autodocs"],
    argTypes: {
        orientation: {
            control: { type: "select" },
            options: ["horizontal", "vertical"],
            description: "Orientation of the divider",
        },
        textAlign: {
            control: { type: "select" },
            options: ["left", "center", "right"],
            description: "Position of text when children is provided",
        },
        children: {
            control: { type: "text" },
            description: "Text content to display in the middle of the divider",
        },
    },
    parameters: {
        layout: "padded",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default horizontal divider
 */
export const Default: Story = {
    args: {},
};

/**
 * Divider with text content
 */
export const WithText: Story = {
    args: {
        children: "OR",
    },
};

/**
 * Vertical orientation example
 */
export const VerticalOrientation: Story = {
    render: () => (
        <div className="tw-flex tw-h-32 tw-items-center tw-justify-center tw-space-x-4">
            <div className="tw-text-center">
                <p>Content on the left</p>
            </div>
            <Divider orientation="vertical" />
            <div className="tw-text-center">
                <p>Content on the right</p>
            </div>
        </div>
    ),
};

/**
 * Dividers with text in different positions
 */
export const TextAlignment: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <div>
                <h3 className="tw-text-sm tw-font-medium tw-mb-2">Left aligned</h3>
                <Divider textAlign="left">Section Start</Divider>
            </div>
            <div>
                <h3 className="tw-text-sm tw-font-medium tw-mb-2">Center aligned</h3>
                <Divider textAlign="center">OR</Divider>
            </div>
            <div>
                <h3 className="tw-text-sm tw-font-medium tw-mb-2">Right aligned</h3>
                <Divider textAlign="right">Section End</Divider>
            </div>
        </div>
    ),
};

/**
 * Complex example with icons in text
 */
export const WithIcons: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <Divider>
                <span className="tw-flex tw-items-center tw-gap-2">
                    <svg className="tw-w-4 tw-h-4" fill="currentColor" viewBox="0 0 20 20">
                        <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                        <path fillRule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clipRule="evenodd" />
                    </svg>
                    Continue with
                    <svg className="tw-w-4 tw-h-4" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                    </svg>
                </span>
            </Divider>
        </div>
    ),
};

/**
 * Pre-configured factory components
 */
export const FactoryComponents: Story = {
    render: () => (
        <div className="tw-space-y-8">
            <div>
                <h3 className="tw-text-lg tw-font-semibold tw-mb-2">Factory Components</h3>
                <div className="tw-space-y-4">
                    <div>
                        <p className="tw-text-sm tw-mb-1">DividerFactory.Horizontal</p>
                        <DividerFactory.Horizontal />
                    </div>
                    <div className="tw-flex tw-h-20">
                        <p className="tw-text-sm tw-mr-2">DividerFactory.Vertical</p>
                        <DividerFactory.Vertical />
                    </div>
                </div>
            </div>
        </div>
    ),
};

/**
 * Form section dividers
 */
export const FormSections: Story = {
    render: () => (
        <div className="tw-max-w-md tw-mx-auto tw-space-y-6">
            <div>
                <h2 className="tw-text-xl tw-font-bold tw-mb-4">User Registration</h2>
                
                <Divider>Personal Information</Divider>
                
                <div className="tw-space-y-4 tw-my-4">
                    <input type="text" placeholder="First Name" className="tw-w-full tw-p-2 tw-border tw-rounded" />
                    <input type="text" placeholder="Last Name" className="tw-w-full tw-p-2 tw-border tw-rounded" />
                </div>
                
                <Divider>Account Details</Divider>
                
                <div className="tw-space-y-4 tw-my-4">
                    <input type="email" placeholder="Email" className="tw-w-full tw-p-2 tw-border tw-rounded" />
                    <input type="password" placeholder="Password" className="tw-w-full tw-p-2 tw-border tw-rounded" />
                </div>
                
                <Divider>Optional</Divider>
                
                <div className="tw-space-y-4 tw-my-4">
                    <input type="tel" placeholder="Phone Number" className="tw-w-full tw-p-2 tw-border tw-rounded" />
                </div>
            </div>
        </div>
    ),
};

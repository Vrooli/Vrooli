import type { Meta, StoryObj } from "@storybook/react";
import { MyStuffView } from "./MyStuffView.js";
import { projectVersionMSWHandlers } from "../../__test/fixtures/index.js";

const meta: Meta<typeof MyStuffView> = {
    title: "Views/Main/MyStuffView",
    component: MyStuffView,
    parameters: {
        layout: "fullscreen",
        msw: {
            handlers: [
                ...projectVersionMSWHandlers.createSuccessHandlers(),
            ],
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
    args: {
        display: "Page",
    },
};

export const WithProjects: Story = {
    args: {
        display: "Page",
    },
    parameters: {
        msw: {
            handlers: [
                ...projectVersionMSWHandlers.createSuccessHandlers(),
            ],
        },
    },
};

export const Loading: Story = {
    args: {
        display: "Page",
    },
    parameters: {
        msw: {
            handlers: [
                ...projectVersionMSWHandlers.createLoadingHandlers(2000),
            ],
        },
    },
};

export const EmptyState: Story = {
    args: {
        display: "Page",
    },
    parameters: {
        msw: {
            handlers: [
                // Return empty projects list
                {
                    rest: {
                        post: (path: string, resolver: any) => {
                            if (path.includes("projectVersion/findMany")) {
                                return (req: any, res: any, ctx: any) => {
                                    return res(
                                        ctx.status(200),
                                        ctx.json({
                                            edges: [],
                                            pageInfo: {
                                                hasNextPage: false,
                                                hasPreviousPage: false,
                                            },
                                        }),
                                    );
                                };
                            }
                        },
                    },
                },
            ],
        },
    },
};
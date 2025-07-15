import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { Switch } from "../Switch/Switch.js";
import { FormDivider } from "./FormDivider.js";

const meta: Meta<typeof FormDivider> = {
    title: "Components/Inputs/Form/FormDivider",
    component: FormDivider,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Showcase: Story = {
    render: () => {
        const [isEditing, setIsEditing] = useState(false);

        const handleDelete = () => {
            console.log("FormDivider deleted");
        };

        return (
            <div className="tw-p-8 tw-h-screen tw-overflow-auto tw-bg-background-default">
                <div className="tw-flex tw-flex-col tw-gap-8 tw-max-w-3xl tw-mx-auto">
                    {/* Controls Section */}
                    <div className="tw-p-6 tw-bg-background-paper tw-rounded-lg tw-shadow-sm tw-w-full">
                        <h1 className="tw-text-2xl tw-font-bold tw-mb-6 tw-text-text-primary">
                            FormDivider Controls
                        </h1>
                        
                        <div className="tw-grid tw-grid-cols-1 sm:tw-grid-cols-2 tw-gap-6">
                            <Switch
                                checked={isEditing}
                                onChange={(checked) => setIsEditing(checked)}
                                size="sm"
                                label="Editing Mode"
                                labelPosition="right"
                            />
                        </div>
                    </div>

                    {/* Component Display */}
                    <div className="tw-p-6 tw-bg-background-paper tw-rounded-lg tw-shadow-sm tw-w-full">
                        <h1 className="tw-text-2xl tw-font-bold tw-mb-6 tw-text-text-primary">
                            FormDivider Component
                        </h1>
                        
                        <div className="tw-p-4 tw-border-2 tw-border-dashed tw-border-text-secondary tw-rounded">
                            <FormDivider
                                isEditing={isEditing}
                                onDelete={handleDelete}
                            />
                        </div>
                    </div>
                </div>
            </div>
        );
    },
};

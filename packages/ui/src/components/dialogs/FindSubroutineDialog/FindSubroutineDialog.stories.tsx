import type { Meta, StoryObj } from "@storybook/react";
import { generatePK } from "@vrooli/shared";
import { useState } from "react";
import { signedInNoPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { Button } from "../../buttons/Button.js";
import { FindSubroutineDialog } from "./FindSubroutineDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof FindSubroutineDialog> = {
    title: "Components/Dialogs/FindSubroutineDialog",
    component: FindSubroutineDialog,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
        session: signedInNoPremiumWithCreditsSession,
    },
    tags: ["autodocs"],
    argTypes: {
        isOpen: {
            control: { type: "boolean" },
            description: "Whether the dialog is open",
        },
        nodeId: {
            control: { type: "text" },
            description: "ID of the node requesting subroutine",
        },
        routineVersionId: {
            control: { type: "text" },
            description: "ID of the current routine version (for filtering)",
        },
        handleCancel: {
            action: "dialog-cancelled",
            description: "Callback when dialog is cancelled",
        },
        handleComplete: {
            action: "subroutine-selected",
            description: "Callback when subroutine is selected",
        },
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase with controls
export const Showcase: Story = {
    render: (args) => {
        const [isOpen, setIsOpen] = useState(args.isOpen ?? false);

        const handleCancel = () => {
            setIsOpen(false);
            args.handleCancel?.();
        };

        const handleComplete = (nodeId: string, subroutine: any) => {
            console.log("Selected subroutine for node:", nodeId, subroutine);
            setIsOpen(false);
            args.handleComplete?.(nodeId, subroutine);
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button onClick={() => setIsOpen(true)} variant="outlined">
                    Open Find Subroutine Dialog
                </Button>
                <FindSubroutineDialog
                    {...args}
                    isOpen={isOpen}
                    handleCancel={handleCancel}
                    handleComplete={handleComplete}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Dialog State: {isOpen ? "Open" : "Closed"}</p>
                    <p>Node ID: {args.nodeId}</p>
                    <p>Routine Version ID: {args.routineVersionId || "None"}</p>
                </div>
            </div>
        );
    },
    args: {
        isOpen: false,
        nodeId: generatePK().toString(),
        routineVersionId: generatePK().toString(),
    },
};

// New routine (no routine version ID)
export const NewRoutine: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        const handleCancel = () => setIsOpen(false);
        const handleComplete = (nodeId: string, subroutine: any) => {
            console.log("Selected subroutine for new routine node:", nodeId, subroutine);
            setIsOpen(false);
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button onClick={() => setIsOpen(true)} variant="outlined">
                    Find Subroutine for New Routine
                </Button>
                <FindSubroutineDialog
                    isOpen={isOpen}
                    nodeId={generatePK().toString()}
                    routineVersionId={null}
                    handleCancel={handleCancel}
                    handleComplete={handleComplete}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Scenario: Creating a new routine</p>
                    <p>Only public subroutines will be shown</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Find subroutine dialog when creating a new routine - shows only public subroutines.",
            },
        },
    },
};

// Existing routine (with routine version ID)
export const ExistingRoutine: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const routineVersionId = generatePK().toString();

        const handleCancel = () => setIsOpen(false);
        const handleComplete = (nodeId: string, subroutine: any) => {
            console.log("Selected subroutine for existing routine node:", nodeId, subroutine);
            setIsOpen(false);
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button onClick={() => setIsOpen(true)} variant="outlined">
                    Find Subroutine for Existing Routine
                </Button>
                <FindSubroutineDialog
                    isOpen={isOpen}
                    nodeId={generatePK().toString()}
                    routineVersionId={routineVersionId}
                    handleCancel={handleCancel}
                    handleComplete={handleComplete}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Scenario: Editing an existing routine</p>
                    <p>Current routine will be excluded from results</p>
                    <p>Own incomplete/internal routines will be included</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Find subroutine dialog when editing an existing routine - excludes current routine and includes own incomplete routines.",
            },
        },
    },
};

// Multiple node IDs demonstration
export const MultipleNodes: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [currentNodeId, setCurrentNodeId] = useState(generatePK().toString());
        const routineVersionId = generatePK().toString();

        const nodeIds = [
            generatePK().toString(),
            generatePK().toString(),
            generatePK().toString(),
        ];

        const handleCancel = () => setIsOpen(false);
        const handleComplete = (nodeId: string, subroutine: any) => {
            console.log(`Selected subroutine for node ${nodeId}:`, subroutine);
            setIsOpen(false);
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ display: "flex", gap: "10px", flexWrap: "wrap" }}>
                    {nodeIds.map((nodeId, index) => (
                        <Button
                            key={nodeId}
                            onClick={() => {
                                setCurrentNodeId(nodeId);
                                setIsOpen(true);
                            }}
                            variant="outlined"
                            size="sm"
                        >
                            Find for Node {index + 1}
                        </Button>
                    ))}
                </div>
                <FindSubroutineDialog
                    isOpen={isOpen}
                    nodeId={currentNodeId}
                    routineVersionId={routineVersionId}
                    handleCancel={handleCancel}
                    handleComplete={handleComplete}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Multiple nodes in same routine can find subroutines</p>
                    <p>Current Node: {currentNodeId}</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Demonstration of finding subroutines for different nodes within the same routine.",
            },
        },
    },
};

// Always open for design testing
export const AlwaysOpen: Story = {
    render: () => {
        const handleCancel = () => console.log("Dialog cancelled");
        const handleComplete = (nodeId: string, subroutine: any) => {
            console.log("Subroutine selected:", { nodeId, subroutine });
        };

        return (
            <FindSubroutineDialog
                isOpen={true}
                nodeId={generatePK().toString()}
                routineVersionId={generatePK().toString()}
                handleCancel={handleCancel}
                handleComplete={handleComplete}
            />
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Dialog that stays open for design and interaction testing.",
            },
        },
    },
};

import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { FormErrorBoundary } from "./FormErrorBoundary";

const meta: Meta<typeof FormErrorBoundary> = {
    title: "Forms/FormErrorBoundary",
    component: FormErrorBoundary,
    parameters: {
        layout: "centered",
        docs: {
            description: {
                component: "A React Error Boundary specifically designed for form components. Catches JavaScript errors in child components and displays a user-friendly error message with debugging details.",
            },
        },
    },
    decorators: [
        (Story) => (
            <div style={{ maxWidth: "600px", width: "100%", padding: "20px" }}>
                <Story />
            </div>
        ),
    ],
    argTypes: {
        children: {
            control: false,
            description: "Child components to wrap with error boundary protection",
        },
        onError: {
            action: "error-occurred",
            description: "Optional callback function called when an error occurs",
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

const NormalComponent = () => (
    <div className="tw-p-5 tw-border tw-border-gray-300 tw-rounded-lg tw-bg-background-paper tw-shadow-sm tw-max-w-md">
        <h3 className="tw-text-lg tw-font-medium tw-mb-3 tw-text-text-primary">Normal Form Component</h3>
        <p className="tw-text-text-secondary tw-mb-4">This component renders normally without any errors.</p>
        <input 
            type="text" 
            placeholder="Sample input field" 
            className="tw-w-full tw-p-2 tw-border tw-border-gray-400 tw-rounded tw-bg-background-paper tw-text-text-primary"
        />
    </div>
);

const ErrorThrowingComponent = ({ shouldThrow }: { shouldThrow: boolean }) => {
    if (shouldThrow) {
        throw new Error("This is a simulated form validation error!");
    }
    return (
        <div className="tw-p-5 tw-border tw-border-green-400 tw-rounded-lg tw-bg-green-50 tw-max-w-md">
            <h3 className="tw-text-lg tw-font-medium tw-mb-2 tw-text-green-800">Component Working Normally</h3>
            <p className="tw-text-green-700">No errors thrown when shouldThrow is false.</p>
        </div>
    );
};

// Component that always throws an error for static stories
const AlwaysErrorComponent = () => {
    const [shouldError, setShouldError] = React.useState(false);
    
    React.useEffect(() => {
        // Delay the error slightly to ensure proper rendering
        const timer = setTimeout(() => setShouldError(true), 100);
        return () => clearTimeout(timer);
    }, []);
    
    if (shouldError) {
        throw new Error("This component always throws an error to demonstrate the error boundary");
    }
    
    return <div>Loading component that will error...</div>;
};

// Custom error component for different error types
const CustomErrorComponent = ({ errorMessage }: { errorMessage: string }) => {
    const [shouldError, setShouldError] = React.useState(false);
    
    React.useEffect(() => {
        const timer = setTimeout(() => setShouldError(true), 100);
        return () => clearTimeout(timer);
    }, []);
    
    if (shouldError) {
        throw new Error(errorMessage);
    }
    
    return <div>Component processing... (will error)</div>;
};

// Form validation error component
const FormValidationErrorComponent = () => {
    const [shouldError, setShouldError] = React.useState(false);
    
    React.useEffect(() => {
        const timer = setTimeout(() => setShouldError(true), 100);
        return () => clearTimeout(timer);
    }, []);
    
    if (shouldError) {
        throw new Error("Validation failed: Email address is required and must be in a valid format");
    }
    
    return <div>Validating form... (will error)</div>;
};

const InteractiveErrorComponent = () => {
    const [shouldThrow, setShouldThrow] = useState(false);

    return (
        <div className="tw-p-5 tw-space-y-4">
            <button 
                onClick={() => setShouldThrow(!shouldThrow)}
                className={`tw-px-4 tw-py-2 tw-rounded tw-font-medium tw-transition-colors tw-duration-200 ${
                    shouldThrow 
                        ? "tw-bg-danger-main tw-text-white hover:tw-bg-red-600" 
                        : "tw-bg-primary-main tw-text-white hover:tw-bg-blue-600"
                }`}
            >
                {shouldThrow ? "Fix Error" : "Trigger Error"}
            </button>
            <ErrorThrowingComponent shouldThrow={shouldThrow} />
        </div>
    );
};

export const Default: Story = {
    args: {
        children: <NormalComponent />,
    },
};

export const WithError: Story = {
    render: (args) => (
        <FormErrorBoundary {...args}>
            <AlwaysErrorComponent />
        </FormErrorBoundary>
    ),
    args: {
        onError: () => console.log("Error boundary caught an error!"),
    },
    parameters: {
        docs: {
            description: {
                story: "Shows how the error boundary displays when a child component throws an error. The error is caught and a user-friendly fallback UI is shown.",
            },
        },
    },
};

export const Interactive: Story = {
    args: {
        children: <InteractiveErrorComponent />,
        onError: () => console.log("Interactive error boundary triggered!"),
    },
    parameters: {
        docs: {
            description: {
                story: "Interactive example where you can trigger and resolve errors to see the error boundary in action. Click the button to toggle between error and normal states.",
            },
        },
    },
};

export const WithCallback: Story = {
    render: (args) => (
        <FormErrorBoundary {...args}>
            <AlwaysErrorComponent />
        </FormErrorBoundary>
    ),
    args: {
        onError: () => {
            console.log("Error callback triggered!");
            alert("Error boundary callback was called!");
        },
    },
    parameters: {
        docs: {
            description: {
                story: "Demonstrates the onError callback functionality. When an error occurs, the callback is executed (check console and alert).",
            },
        },
    },
};

export const MultipleChildren: Story = {
    render: (args) => (
        <FormErrorBoundary {...args}>
            <div>
                <NormalComponent />
                <div style={{ margin: "20px 0" }}>
                    <AlwaysErrorComponent />
                </div>
                <NormalComponent />
            </div>
        </FormErrorBoundary>
    ),
    args: {},
    parameters: {
        docs: {
            description: {
                story: "Shows how the error boundary handles multiple child components, where one throws an error. The entire boundary content is replaced with the error display.",
            },
        },
    },
};

export const DifferentErrorTypes: Story = {
    render: (args) => (
        <FormErrorBoundary {...args}>
            <CustomErrorComponent errorMessage="TypeError: Cannot read property 'x' of undefined" />
        </FormErrorBoundary>
    ),
    args: {},
    parameters: {
        docs: {
            description: {
                story: "Shows how the error boundary displays different types of errors with their specific error messages.",
            },
        },
    },
};

export const FormContextExample: Story = {
    render: (args) => (
        <FormErrorBoundary {...args}>
            <FormValidationErrorComponent />
        </FormErrorBoundary>
    ),
    args: {},
    parameters: {
        docs: {
            description: {
                story: "Example showing how the error boundary would appear in a real form context with validation errors.",
            },
        },
    },
};

export const SimpleError: Story = {
    render: () => {
        const SimpleErrorComponent = () => {
            throw new Error("Simple immediate error for testing");
        };
        
        return (
            <div style={{ padding: "20px" }}>
                <h3>Error Boundary Test</h3>
                <p>The component below will throw an error that should be caught by the error boundary:</p>
                <FormErrorBoundary>
                    <SimpleErrorComponent />
                </FormErrorBoundary>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Simple test showing immediate error throwing. If you see this story content, the error boundary is working correctly.",
            },
        },
    },
};


// Complex form with multiple fields that throws an error
const ComplexFormComponent = () => (
    <div className="tw-space-y-4 tw-p-6 tw-bg-background-paper tw-rounded-lg tw-shadow-sm tw-border tw-border-gray-200">
        <div>
            <label className="tw-block tw-text-sm tw-font-medium tw-text-text-primary tw-mb-1">
                Email Address
            </label>
            <input 
                type="email" 
                className="tw-w-full tw-p-2 tw-border tw-border-gray-400 tw-rounded tw-bg-background-paper"
                placeholder="Enter your email"
            />
        </div>
        <div>
            <label className="tw-block tw-text-sm tw-font-medium tw-text-text-primary tw-mb-1">
                Password
            </label>
            <input 
                type="password" 
                className="tw-w-full tw-p-2 tw-border tw-border-gray-400 tw-rounded tw-bg-background-paper"
                placeholder="Enter your password"
            />
        </div>
        <AlwaysErrorComponent />
    </div>
);

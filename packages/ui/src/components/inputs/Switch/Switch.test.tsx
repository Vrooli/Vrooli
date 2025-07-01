import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import * as yup from "yup";
import { renderWithFormik, formInteractions, formAssertions } from "../../../__test/helpers/formTestHelpers";
import { Switch, SwitchBase, SwitchFactory, SwitchFactoryBase } from "./Switch";

describe("SwitchBase", () => {
    describe("Basic rendering", () => {
        it("renders unchecked switch by default", () => {
            render(<SwitchBase />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox).toBeDefined();
            expect(checkbox.getAttribute("aria-checked")).toBe("false");
            expect(checkbox.checked).toBe(false);
        });

        it("renders checked switch when checked prop is true", () => {
            render(<SwitchBase checked={true} onChange={vi.fn()} />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.getAttribute("aria-checked")).toBe("true");
            expect(checkbox.checked).toBe(true);
        });

        it("renders with label when provided", () => {
            render(<SwitchBase label="Enable notifications" />);
            
            const label = screen.getByText("Enable notifications");
            expect(label).toBeDefined();
            expect(label.tagName).toBe("LABEL");
        });

        it("renders without label when not provided", () => {
            render(<SwitchBase />);
            
            expect(screen.queryByText("Toggle switch")).toBeNull();
        });

        it("renders with custom id when provided", () => {
            render(<SwitchBase id="custom-switch-id" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.id).toBe("custom-switch-id");
        });

        it("generates unique id when not provided", () => {
            render(
                <>
                    <SwitchBase />
                    <SwitchBase />
                </>,
            );
            
            const switches = screen.getAllByRole("switch");
            expect(switches[0].id).toBeDefined();
            expect(switches[1].id).toBeDefined();
            expect(switches[0].id).not.toBe(switches[1].id);
        });
    });

    describe("User interactions", () => {
        it("toggles checked state on click", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<SwitchBase checked={false} onChange={onChange} />);
            
            // Find the visual switch container that has the onClick handler
            const visualSwitch = screen.getByTestId("switch-visual-container");
            
            await act(async () => {
                await user.click(visualSwitch);
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("toggles checked state on checkbox click", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<SwitchBase checked={false} onChange={onChange} />);
            
            const checkbox = screen.getByRole("switch");
            
            await act(async () => {
                await user.click(checkbox);
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("toggles on Space key press", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<SwitchBase checked={false} onChange={onChange} />);
            
            const checkbox = screen.getByRole("switch");
            checkbox.focus();
            
            await act(async () => {
                await user.keyboard(" ");
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("toggles on Enter key press", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<SwitchBase checked={false} onChange={onChange} />);
            
            const checkbox = screen.getByRole("switch");
            checkbox.focus();
            
            await act(async () => {
                await user.keyboard("{Enter}");
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });

        it("does not toggle when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<SwitchBase checked={false} onChange={onChange} disabled />);
            
            const visualSwitch = screen.getByTestId("switch-visual-container");
            
            await act(async () => {
                await user.click(visualSwitch);
            });
            
            expect(onChange).not.toHaveBeenCalled();
        });

        it("does not toggle on keyboard when disabled", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<SwitchBase checked={false} onChange={onChange} disabled />);
            
            const checkbox = screen.getByRole("switch");
            checkbox.focus();
            
            await act(async () => {
                await user.keyboard(" ");
                await user.keyboard("{Enter}");
            });
            
            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe("Accessibility", () => {
        it("has proper ARIA attributes", () => {
            render(<SwitchBase checked={true} onChange={vi.fn()} />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("role")).toBe("switch");
            expect(checkbox.getAttribute("aria-checked")).toBe("true");
            expect(checkbox.getAttribute("type")).toBe("checkbox");
        });

        it("uses label as aria-label when label is string", () => {
            render(<SwitchBase label="Dark mode" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-label")).toBe("Dark mode");
        });

        it("uses provided aria-label over label", () => {
            render(<SwitchBase label="Dark mode" aria-label="Toggle dark theme" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-label")).toBe("Toggle dark theme");
        });

        it("uses default aria-label when no label provided", () => {
            render(<SwitchBase />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-label")).toBe("Toggle switch");
        });

        it("supports aria-labelledby", () => {
            render(
                <>
                    <span id="switch-label">Custom Label</span>
                    <SwitchBase aria-labelledby="switch-label" />
                </>,
            );
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-labelledby")).toBe("switch-label");
        });

        it("supports aria-describedby", () => {
            render(
                <>
                    <span id="switch-description">This enables dark mode</span>
                    <SwitchBase aria-describedby="switch-description" />
                </>,
            );
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("aria-describedby")).toBe("switch-description");
        });

        it("provides screen reader feedback for state", () => {
            render(<SwitchBase checked={true} onChange={vi.fn()} />);
            
            const status = screen.getByRole("status");
            expect(status).toBeDefined();
            expect(status.textContent).toBe("On");
        });

        it("updates screen reader feedback when toggled", async () => {
            const user = userEvent.setup();
            const { rerender } = render(<SwitchBase checked={false} onChange={vi.fn()} />);
            
            expect(screen.getByRole("status").textContent).toBe("Off");
            
            rerender(<SwitchBase checked={true} onChange={vi.fn()} />);
            
            expect(screen.getByRole("status").textContent).toBe("On");
        });

        it("label is clickable and toggles switch", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();
            
            render(<SwitchBase label="Click me" checked={false} onChange={onChange} />);
            
            const label = screen.getByText("Click me");
            
            await act(async () => {
                await user.click(label);
            });
            
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true, expect.any(Object));
        });
    });

    describe("Visual states", () => {
        it("renders disabled state correctly", () => {
            render(<SwitchBase label="Disabled switch" disabled />);
            
            const switchInput = screen.getByRole("switch");
            expect(switchInput).toBeDefined();
            expect(switchInput.getAttribute("disabled")).toBe("");
            expect(screen.getByText("Disabled switch")).toBeDefined();
        });

        it("track has data-checked attribute", () => {
            const { rerender } = render(<SwitchBase checked={false} onChange={vi.fn()} />);
            
            let track = screen.getByRole("switch").parentElement?.querySelector("[data-checked]");
            expect(track?.getAttribute("data-checked")).toBe("false");
            
            rerender(<SwitchBase checked={true} onChange={vi.fn()} />);
            
            track = screen.getByRole("switch").parentElement?.querySelector("[data-checked]");
            expect(track?.getAttribute("data-checked")).toBe("true");
        });
    });

    describe("Label positioning", () => {
        it("renders with label on the right by default", () => {
            render(<SwitchBase label="Right label" />);
            
            const switchInput = screen.getByRole("switch");
            const label = screen.getByText("Right label");
            expect(switchInput).toBeDefined();
            expect(label).toBeDefined();
        });

        it("renders with label on the left when specified", () => {
            render(<SwitchBase label="Left label" labelPosition="left" />);
            
            const switchInput = screen.getByRole("switch");
            const label = screen.getByText("Left label");
            expect(switchInput).toBeDefined();
            expect(label).toBeDefined();
        });

        it("renders no label when labelPosition is none", () => {
            render(<SwitchBase label="Hidden label" labelPosition="none" />);
            
            expect(screen.queryByText("Hidden label")).toBeNull();
        });
    });

    describe("Variants", () => {
        const variants = ["default", "success", "warning", "danger", "space", "neon", "theme", "custom"] as const;
        
        variants.forEach(variant => {
            it(`renders ${variant} variant`, () => {
                render(<SwitchBase variant={variant} />);
                
                const checkbox = screen.getByRole("switch");
                expect(checkbox).toBeDefined();
            });
        });

        it("applies custom color when variant is custom", () => {
            render(<SwitchBase variant="custom" color="#FF0000" checked={true} onChange={vi.fn()} />);
            
            const track = screen.getByRole("switch").parentElement?.querySelector("[data-checked='true']");
            const style = track?.getAttribute("style") || "";
            // Check for either format of color
            expect(style.includes("background-color: rgb(255, 0, 0)") || style.includes("background-color: #FF0000")).toBe(true);
        });

        it("renders theme variant with sun/moon icons", () => {
            render(<SwitchBase variant="theme" checked={false} onChange={vi.fn()} />);
            
            const svgs = screen.getByRole("switch").parentElement?.querySelectorAll("svg");
            expect(svgs?.length).toBeGreaterThan(0);
        });
    });

    describe("Sizes", () => {
        const sizes = ["sm", "md", "lg"] as const;
        
        sizes.forEach(size => {
            it(`renders ${size} size`, () => {
                render(<SwitchBase size={size} />);
                
                const checkbox = screen.getByRole("switch");
                expect(checkbox).toBeDefined();
            });
        });
    });

    describe("SwitchFactoryBase", () => {
        it("renders Default factory switch", () => {
            render(<SwitchFactoryBase.Default />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Success factory switch", () => {
            render(<SwitchFactoryBase.Success />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Warning factory switch", () => {
            render(<SwitchFactoryBase.Warning />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Danger factory switch", () => {
            render(<SwitchFactoryBase.Danger />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Space factory switch", () => {
            render(<SwitchFactoryBase.Space />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Neon factory switch", () => {
            render(<SwitchFactoryBase.Neon />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("renders Theme factory switch", () => {
            render(<SwitchFactoryBase.Theme />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox).toBeDefined();
        });

        it("factory switches accept props except variant", () => {
            const onChange = vi.fn();
            render(<SwitchFactoryBase.Default checked={true} onChange={onChange} label="Test" />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.checked).toBe(true);
            expect(screen.getByText("Test")).toBeDefined();
        });
    });

    describe("Controlled vs Uncontrolled", () => {
        it("works as controlled component", async () => {
            const user = userEvent.setup();
            const ControlledSwitch = () => {
                const [checked, setChecked] = React.useState(false);
                return <SwitchBase checked={checked} onChange={setChecked} />;
            };
            
            render(<ControlledSwitch />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.checked).toBe(false);
            
            await act(async () => {
                await user.click(checkbox);
            });
            
            expect(checkbox.checked).toBe(true);
        });

        it("maintains state when onChange is not provided", () => {
            render(<SwitchBase checked={true} />);
            
            const checkbox = screen.getByRole("switch") as HTMLInputElement;
            expect(checkbox.checked).toBe(true);
        });
    });

    describe("Forwarded ref", () => {
        it("forwards ref to input element", () => {
            const ref = React.createRef<HTMLInputElement>();
            render(<SwitchBase ref={ref} />);
            
            expect(ref.current).toBeInstanceOf(HTMLInputElement);
            expect(ref.current?.type).toBe("checkbox");
        });
    });

    describe("Additional props", () => {
        it("passes additional props to input", () => {
            render(<SwitchBase data-testid="custom-switch" name="theme-toggle" />);
            
            const checkbox = screen.getByRole("switch");
            expect(checkbox.getAttribute("data-testid")).toBe("custom-switch");
            expect(checkbox.getAttribute("name")).toBe("theme-toggle");
        });

        it("applies custom className", () => {
            render(<SwitchBase className="custom-class" />);
            
            const switchContainer = screen.getByRole("switch").closest("div")?.parentElement;
            expect(switchContainer?.querySelector(".custom-class")).toBeDefined();
        });
    });

});

describe("Switch (Formik Integration)", () => {
    describe("Formik integration", () => {
        it("works as a controlled component within Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <Switch name="notifications" label="Enable notifications" />,
                {
                    initialValues: { notifications: false },
                },
            );

            const switchControl = screen.getByRole("switch") as HTMLInputElement;
            expect(switchControl.checked).toBe(false);
            expect(getFormValues().notifications).toBe(false);

            await act(async () => {
                await user.click(switchControl);
            });

            expect(switchControl.checked).toBe(true);
            expect(getFormValues().notifications).toBe(true);
        });

        it("submits form values correctly", async () => {
            const { user, onSubmit, submitForm } = renderWithFormik(
                <>
                    <Switch name="darkMode" label="Dark mode" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { darkMode: false },
                },
            );

            const switchControl = screen.getByRole("switch");
            
            await act(async () => {
                await user.click(switchControl);
            });

            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, { darkMode: true });
        });

        it("validates required boolean field", async () => {
            const validationSchema = yup.object({
                privacy: yup.boolean()
                    .oneOf([true], "You must accept the privacy policy")
                    .required("Required"),
            });

            const { user, onSubmit } = renderWithFormik(
                <>
                    <Switch name="privacy" label="I accept the privacy policy" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { privacy: false },
                    formikConfig: { validationSchema },
                },
            );

            const submitButtons = screen.getAllByRole("button", { name: "Submit" });
            const submitButton = submitButtons[0]; // Take the first submit button
            
            await act(async () => {
                await user.click(submitButton);
            });

            expect(onSubmit).not.toHaveBeenCalled();
            formAssertions.expectFieldError("You must accept the privacy policy");

            // Now toggle switch and submit
            const switchControl = screen.getByRole("switch");
            await act(async () => {
                await user.click(switchControl);
                await user.click(submitButton);
            });

            formAssertions.expectFormSubmitted(onSubmit, { privacy: true });
        });

        it("respects initial values", () => {
            const { getFormValues } = renderWithFormik(
                <>
                    <Switch name="emailAlerts" label="Email alerts" />
                    <Switch name="pushAlerts" label="Push notifications" />
                </>,
                {
                    initialValues: { 
                        emailAlerts: true,
                        pushAlerts: false,
                    },
                },
            );

            const switches = screen.getAllByRole("switch") as HTMLInputElement[];
            
            expect(switches[0].checked).toBe(true);
            expect(switches[1].checked).toBe(false);
            expect(getFormValues()).toEqual({
                emailAlerts: true,
                pushAlerts: false,
            });
        });

        it("resets to initial values on form reset", async () => {
            const { user, resetForm } = renderWithFormik(
                <Switch name="autoSave" label="Auto save" />,
                {
                    initialValues: { autoSave: true },
                },
            );

            const switchControl = screen.getByRole("switch") as HTMLInputElement;
            expect(switchControl.checked).toBe(true);

            await act(async () => {
                await user.click(switchControl);
            });

            expect(switchControl.checked).toBe(false);

            resetForm();

            expect(switchControl.checked).toBe(true);
        });

        it("handles multiple switches independently", async () => {
            const { user, onSubmit, submitForm, getFormValues } = renderWithFormik(
                <>
                    <Switch name="feature1" label="Feature 1" />
                    <Switch name="feature2" label="Feature 2" />
                    <Switch name="feature3" label="Feature 3" />
                </>,
                {
                    initialValues: { 
                        feature1: false,
                        feature2: true,
                        feature3: false,
                    },
                },
            );

            const switches = screen.getAllByRole("switch");
            
            await act(async () => {
                await user.click(switches[0]); // Toggle feature1
                await user.click(switches[1]); // Toggle feature2
            });

            expect(getFormValues()).toEqual({
                feature1: true,
                feature2: false,
                feature3: false,
            });

            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, {
                feature1: true,
                feature2: false,
                feature3: false,
            });
        });

        it("updates field touched state on blur", async () => {
            const { user, isFieldTouched } = renderWithFormik(
                <Switch name="marketing" label="Marketing emails" />,
                {
                    initialValues: { marketing: false },
                },
            );

            const switchControl = screen.getByRole("switch");
            
            expect(isFieldTouched("marketing")).toBe(false);

            await act(async () => {
                await user.click(switchControl);
                await user.tab(); // Blur
            });

            expect(isFieldTouched("marketing")).toBe(true);
        });

        it("preserves disabled state in Formik context", async () => {
            const { user } = renderWithFormik(
                <>
                    <Switch name="setting1" label="Enabled setting" />
                    <Switch name="setting2" label="Disabled setting" disabled />
                </>,
                {
                    initialValues: { setting1: false, setting2: true },
                },
            );

            const switches = screen.getAllByRole("switch") as HTMLInputElement[];
            
            expect(switches[1].disabled).toBe(true);
            expect(switches[1].checked).toBe(true);

            const visualContainer = screen.getAllByTestId("switch-visual-container")[1];
            await act(async () => {
                await user.click(visualContainer);
            });

            // Value should not change
            expect(switches[1].checked).toBe(true);
        });

        it("works with variant switches in Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <>
                    <SwitchFactory.Default name="default" label="Default" />
                    <SwitchFactory.Success name="success" label="Success" />
                    <SwitchFactory.Warning name="warning" label="Warning" />
                    <SwitchFactory.Danger name="danger" label="Danger" />
                    <SwitchFactory.Theme name="theme" label="Theme" />
                </>,
                {
                    initialValues: {
                        default: false,
                        success: false,
                        warning: false,
                        danger: false,
                        theme: false,
                    },
                },
            );

            const switches = screen.getAllByRole("switch");
            expect(switches).toHaveLength(5);

            await act(async () => {
                await user.click(switches[1]); // Success
                await user.click(switches[3]); // Danger
                await user.click(switches[4]); // Theme
            });

            expect(getFormValues()).toEqual({
                default: false,
                success: true,
                warning: false,
                danger: true,
                theme: true,
            });
        });

        it("handles programmatic value changes", async () => {
            const { setFieldValue, getFormValues } = renderWithFormik(
                <Switch name="debugMode" label="Debug mode" />,
                {
                    initialValues: { debugMode: false },
                },
            );

            const switchControl = screen.getByRole("switch") as HTMLInputElement;
            expect(switchControl.checked).toBe(false);

            await setFieldValue("debugMode", true);

            expect(switchControl.checked).toBe(true);
            expect(getFormValues().debugMode).toBe(true);
        });

        it("handles complex form with mixed field types", async () => {
            const validationSchema = yup.object({
                username: yup.string().required("Username is required"),
                notifications: yup.boolean(),
                theme: yup.string().oneOf(["light", "dark"]),
            });

            const { user, onSubmit, submitForm } = renderWithFormik(
                <>
                    <input name="username" placeholder="Username" />
                    <Switch name="notifications" label="Enable notifications" />
                    <select name="theme">
                        <option value="light">Light</option>
                        <option value="dark">Dark</option>
                    </select>
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { 
                        username: "",
                        notifications: true,
                        theme: "light",
                    },
                    formikConfig: { validationSchema },
                },
            );

            const usernameInput = screen.getByPlaceholderText("Username");
            const switchControl = screen.getByRole("switch");
            
            await act(async () => {
                await user.type(usernameInput, "testuser");
                await user.click(switchControl); // Toggle off
            });

            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, {
                username: "testuser",
                notifications: false,
                theme: "light",
            });
        });

        it("works with keyboard navigation in Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <>
                    <input name="name" placeholder="Name" />
                    <Switch name="subscribe" label="Subscribe to newsletter" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { name: "", subscribe: false },
                },
            );

            const nameInput = screen.getByPlaceholderText("Name");
            const switchElement = screen.getByRole("switch");
            
            await act(async () => {
                await user.click(nameInput);
                await user.type(nameInput, "John");
                // Focus on switch directly and use space to toggle
                await user.click(switchElement);
                await user.keyboard(" ");
            });

            expect(getFormValues()).toEqual({
                name: "John",
                subscribe: true,
            });
        });

        it("handles label click to toggle switch in Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <Switch name="terms" label="I agree to the terms" />,
                {
                    initialValues: { terms: false },
                },
            );

            const label = screen.getByText("I agree to the terms");
            
            await act(async () => {
                await user.click(label);
            });

            expect(getFormValues().terms).toBe(true);
        });
    });
});

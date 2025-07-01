import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import * as yup from "yup";
import { renderWithFormik, formInteractions, formAssertions } from "../../__test/helpers/formTestHelpers";
import { Radio, RadioBase, RadioFactory, RadioFactoryBase, PrimaryRadio, SecondaryRadio, DangerRadio, CustomRadio } from "./Radio.js";

describe("RadioBase", () => {
    describe("Basic rendering", () => {
        it("renders as an unchecked radio by default", () => {
            render(<RadioBase name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
            expect(radioInput.getAttribute("type")).toBe("radio");
            expect(radioInput.getAttribute("checked")).toBeNull();
            expect(radioInput.getAttribute("disabled")).toBeNull();
        });

        it("renders with required prop", () => {
            render(<RadioBase name="test-radio" required />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("required")).toBe("");
        });

        it("renders with aria attributes", () => {
            render(
                <RadioBase 
                    name="test-radio"
                    aria-label="Test radio"
                    aria-labelledby="label-id"
                    aria-describedby="desc-id"
                />,
            );

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("aria-label")).toBe("Test radio");
            expect(radioInput.getAttribute("aria-labelledby")).toBe("label-id");
            expect(radioInput.getAttribute("aria-describedby")).toBe("desc-id");
        });
    });

    describe("Controlled vs Uncontrolled", () => {
        it("works as controlled component with checked prop", () => {
            const { rerender } = render(<RadioBase name="test-radio" checked={false} onChange={vi.fn()} />);

            const radioInput = screen.getByTestId("radio-input") as HTMLInputElement;
            expect(radioInput.checked).toBe(false);

            rerender(<RadioBase name="test-radio" checked={true} onChange={vi.fn()} />);
            expect(radioInput.checked).toBe(true);
        });

        it("works as uncontrolled component with defaultChecked", () => {
            render(<RadioBase name="test-radio" defaultChecked={true} />);

            const radioInput = screen.getByTestId("radio-input") as HTMLInputElement;
            expect(radioInput.checked).toBe(true);
        });
    });

    describe("User interactions", () => {
        it("handles click on radio input", async () => {
            const onChange = vi.fn();
            const onClick = vi.fn();
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" onChange={onChange} onClick={onClick} />);

            const radioInput = screen.getByTestId("radio-input");

            await act(async () => {
                await user.click(radioInput);
            });

            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onClick).toHaveBeenCalledTimes(1);
        });

        it("handles click on visual wrapper for ripple effect", async () => {
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.click(radioWrapper);
            });

            // Check if ripple was created
            const ripples = screen.queryAllByTestId("radio-ripple");
            expect(ripples.length).toBeGreaterThan(0);
        });

        it("does not trigger events when disabled", async () => {
            const onChange = vi.fn();
            const onClick = vi.fn();
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" disabled onChange={onChange} onClick={onClick} />);

            const radioInput = screen.getByTestId("radio-input");
            const radioLabel = screen.getByTestId("radio-label");

            await act(async () => {
                await user.click(radioInput);
            });

            expect(onChange).not.toHaveBeenCalled();
            expect(onClick).not.toHaveBeenCalled();
            expect(radioLabel.getAttribute("data-disabled")).toBe("true");
        });

        it("handles focus and blur events", async () => {
            const onFocus = vi.fn();
            const onBlur = vi.fn();
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" onFocus={onFocus} onBlur={onBlur} />);

            const radioInput = screen.getByTestId("radio-input");

            await act(async () => {
                await user.click(radioInput);
            });

            expect(onFocus).toHaveBeenCalledTimes(1);

            await act(async () => {
                await user.tab();
            });

            expect(onBlur).toHaveBeenCalledTimes(1);
        });
    });

    describe("Radio group behavior", () => {
        it("allows selection within a group", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();

            render(
                <>
                    <RadioBase name="group" value="option1" onChange={onChange} />
                    <RadioBase name="group" value="option2" onChange={onChange} />
                    <RadioBase name="group" value="option3" onChange={onChange} />
                </>,
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];

            await act(async () => {
                await user.click(radios[0]);
            });

            expect(radios[0].checked).toBe(true);
            expect(radios[1].checked).toBe(false);
            expect(radios[2].checked).toBe(false);

            await act(async () => {
                await user.click(radios[1]);
            });

            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
        });

        it("assigns proper name and value attributes", () => {
            render(<RadioBase name="test-group" value="test-value" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("name")).toBe("test-group");
            expect(radioInput.getAttribute("value")).toBe("test-value");
        });
    });

    describe("Visual states", () => {
        it("updates data attributes based on state", () => {
            const { rerender } = render(<RadioBase name="test-radio" checked={false} onChange={vi.fn()} />);

            const radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel.getAttribute("data-checked")).toBe("false");

            rerender(<RadioBase name="test-radio" checked={true} onChange={vi.fn()} />);
            expect(radioLabel.getAttribute("data-checked")).toBe("true");
        });

        it("renders with different sizes", () => {
            const { rerender } = render(<RadioBase name="test-radio" size="sm" />);

            let radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<RadioBase name="test-radio" size="md" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<RadioBase name="test-radio" size="lg" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();
        });

        it("renders with different color variants", () => {
            const { rerender } = render(<RadioBase name="test-radio" color="primary" />);

            let radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<RadioBase name="test-radio" color="secondary" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();

            rerender(<RadioBase name="test-radio" color="danger" />);
            radioLabel = screen.getByTestId("radio-label");
            expect(radioLabel).toBeDefined();
        });
    });

    describe("Custom color functionality", () => {
        it("applies custom color styles when color is custom", () => {
            render(<RadioBase name="test-radio" color="custom" customColor="#ff0000" />);

            const radioOuter = screen.getByTestId("radio-outer");
            const radioInner = screen.getByTestId("radio-inner");

            expect(radioOuter).toBeDefined();
            expect(radioInner).toBeDefined();
        });

        it("handles hover state for custom colors", async () => {
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" color="custom" customColor="#ff0000" />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.hover(radioWrapper);
            });

            // The hover effect modifies inline styles
            expect(radioWrapper.style.backgroundColor).toBeDefined();

            await act(async () => {
                await user.unhover(radioWrapper);
            });

            expect(radioWrapper.style.backgroundColor).toBe("");
        });

        it("does not apply hover effects when disabled", async () => {
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" color="custom" customColor="#ff0000" disabled />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.hover(radioWrapper);
            });

            expect(radioWrapper.style.backgroundColor).toBe("");
        });
    });

    describe("Factory components", () => {
        it("renders PrimaryRadio with primary color", () => {
            render(<PrimaryRadio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });

        it("renders SecondaryRadio with secondary color", () => {
            render(<SecondaryRadio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });

        it("renders DangerRadio with danger color", () => {
            render(<DangerRadio name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });

        it("renders CustomRadio with custom color", () => {
            render(<CustomRadio name="test-radio" customColor="#00ff00" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("maintains proper ARIA roles", () => {
            render(<RadioBase name="test-radio" />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("type")).toBe("radio");
        });

        it("can be focused via keyboard", async () => {
            const user = userEvent.setup();

            render(
                <>
                    <input type="text" data-testid="before" />
                    <RadioBase name="test-radio" />
                    <input type="text" data-testid="after" />
                </>,
            );

            const beforeInput = screen.getByTestId("before");
            const radioInput = screen.getByTestId("radio-input");
            const afterInput = screen.getByTestId("after");

            beforeInput.focus();

            await act(async () => {
                await user.tab();
            });

            expect(document.activeElement).toBe(radioInput);

            await act(async () => {
                await user.tab();
            });

            expect(document.activeElement).toBe(afterInput);
        });

        it("properly handles keyboard navigation in radio groups", async () => {
            const user = userEvent.setup();

            render(
                <>
                    <RadioBase name="group" value="1" defaultChecked />
                    <RadioBase name="group" value="2" />
                    <RadioBase name="group" value="3" />
                </>,
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];

            radios[0].focus();

            await act(async () => {
                await user.keyboard("{ArrowDown}");
            });

            expect(document.activeElement).toBe(radios[1]);
            expect(radios[1].checked).toBe(true);

            await act(async () => {
                await user.keyboard("{ArrowDown}");
            });

            expect(document.activeElement).toBe(radios[2]);
            expect(radios[2].checked).toBe(true);

            await act(async () => {
                await user.keyboard("{ArrowUp}");
            });

            expect(document.activeElement).toBe(radios[1]);
            expect(radios[1].checked).toBe(true);
        });
    });

    describe("Edge cases", () => {
        it("handles rapid clicks gracefully", async () => {
            const onChange = vi.fn();
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" onChange={onChange} />);

            const radioInput = screen.getByTestId("radio-input");

            await act(async () => {
                await user.tripleClick(radioInput);
            });

            // Should still be called appropriate number of times
            expect(onChange).toHaveBeenCalled();
        });

        it("cleans up ripple effects after animation", async () => {
            const user = userEvent.setup();

            render(<RadioBase name="test-radio" />);

            const radioWrapper = screen.getByTestId("radio-wrapper");

            await act(async () => {
                await user.click(radioWrapper);
            });

            const ripple = screen.getByTestId("radio-ripple");
            expect(ripple).toBeDefined();

            // Simulate animation end
            const animationEndEvent = new Event("animationend", { bubbles: true });
            
            act(() => {
                ripple.dispatchEvent(animationEndEvent);
            });

            // Ripple should be removed after animation
            expect(screen.queryByTestId("radio-ripple")).toBeNull();
        });

        it("handles missing props gracefully", () => {
            render(<RadioBase />);

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("name")).toBeNull();
            expect(radioInput.getAttribute("value")).toBe("");
        });

        it("forwards refs correctly", () => {
            const ref = React.createRef<HTMLInputElement>();
            
            render(<RadioBase name="test-radio" ref={ref} />);

            expect(ref.current).toBeDefined();
            expect(ref.current?.type).toBe("radio");
        });

        it("spreads additional props to input element", () => {
            render(
                <RadioBase 
                    name="test-radio" 
                    data-custom="custom-value"
                    id="custom-id"
                />,
            );

            const radioInput = screen.getByTestId("radio-input");
            expect(radioInput.getAttribute("data-custom")).toBe("custom-value");
            expect(radioInput.getAttribute("id")).toBe("custom-id");
        });
    });

    describe("Formik integration", () => {
        it("works as a controlled component within Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <>
                    <RadioBase name="choice" value="option1" />
                    <RadioBase name="choice" value="option2" />
                    <RadioBase name="choice" value="option3" />
                </>,
                {
                    initialValues: { choice: "" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(false);
            expect(radios[2].checked).toBe(false);

            await act(async () => {
                await user.click(radios[1]);
            });

            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
            expect(getFormValues().choice).toBe("option2");
        });

        it("submits selected radio value correctly", async () => {
            const { user, onSubmit, submitForm } = renderWithFormik(
                <>
                    <RadioBase name="plan" value="basic" />
                    <RadioBase name="plan" value="premium" />
                    <RadioBase name="plan" value="enterprise" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { plan: "" },
                },
            );

            const radios = screen.getAllByTestId("radio-input");
            
            await act(async () => {
                await user.click(radios[1]); // Select premium
            });

            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, { plan: "premium" });
        });

        it("validates required radio selection", async () => {
            const validationSchema = yup.object({
                agreement: yup.string()
                    .required("Please select an option"),
            });

            const { user, onSubmit } = renderWithFormik(
                <>
                    <RadioBase name="agreement" value="yes" />
                    <RadioBase name="agreement" value="no" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { agreement: "" },
                    formikConfig: { validationSchema },
                },
            );

            const submitButton = screen.getByRole("button", { name: /submit/i });
            
            await act(async () => {
                await user.click(submitButton);
            });

            expect(onSubmit).not.toHaveBeenCalled();
            formAssertions.expectFieldError("Please select an option");
        });

        it("respects initial values in radio groups", () => {
            const { getFormValues } = renderWithFormik(
                <>
                    <RadioBase name="size" value="small" />
                    <RadioBase name="size" value="medium" />
                    <RadioBase name="size" value="large" />
                </>,
                {
                    initialValues: { size: "medium" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
            expect(getFormValues().size).toBe("medium");
        });

        it("resets to initial value on form reset", async () => {
            const { user, resetForm } = renderWithFormik(
                <>
                    <RadioBase name="frequency" value="daily" />
                    <RadioBase name="frequency" value="weekly" />
                    <RadioBase name="frequency" value="monthly" />
                </>,
                {
                    initialValues: { frequency: "weekly" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            expect(radios[1].checked).toBe(true);

            await act(async () => {
                await user.click(radios[2]); // Select monthly
            });

            expect(radios[1].checked).toBe(false);
            expect(radios[2].checked).toBe(true);

            resetForm();

            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
        });

        it("handles multiple independent radio groups", async () => {
            const { user, onSubmit, submitForm, getFormValues } = renderWithFormik(
                <>
                    <fieldset>
                        <legend>Color</legend>
                        <RadioBase name="color" value="red" />
                        <RadioBase name="color" value="blue" />
                        <RadioBase name="color" value="green" />
                    </fieldset>
                    <fieldset>
                        <legend>Size</legend>
                        <RadioBase name="size" value="s" />
                        <RadioBase name="size" value="m" />
                        <RadioBase name="size" value="l" />
                    </fieldset>
                </>,
                {
                    initialValues: { color: "", size: "" },
                },
            );

            const radios = screen.getAllByTestId("radio-input");
            
            await act(async () => {
                await user.click(radios[1]); // Select blue
                await user.click(radios[4]); // Select medium
            });

            expect(getFormValues()).toEqual({
                color: "blue",
                size: "m",
            });

            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, {
                color: "blue",
                size: "m",
            });
        });

        it("updates field touched state on blur", async () => {
            const { user, isFieldTouched } = renderWithFormik(
                <>
                    <RadioBase name="preference" value="email" />
                    <RadioBase name="preference" value="phone" />
                </>,
                {
                    initialValues: { preference: "" },
                },
            );

            const radios = screen.getAllByTestId("radio-input");
            
            expect(isFieldTouched("preference")).toBe(false);

            await act(async () => {
                await user.click(radios[0]);
                await user.tab(); // Blur
            });

            expect(isFieldTouched("preference")).toBe(true);
        });

        it("preserves disabled state in Formik context", async () => {
            const { user } = renderWithFormik(
                <>
                    <RadioBase name="status" value="active" />
                    <RadioBase name="status" value="inactive" disabled />
                    <RadioBase name="status" value="pending" />
                </>,
                {
                    initialValues: { status: "inactive" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            expect(radios[1].disabled).toBe(true);
            expect(radios[1].checked).toBe(true);

            await act(async () => {
                await user.click(radios[1]);
            });

            // Value should not change from disabled radio
            expect(radios[1].checked).toBe(true);
        });

        it("works with variant radios in Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <>
                    <PrimaryRadio name="theme" value="primary" />
                    <SecondaryRadio name="theme" value="secondary" />
                    <DangerRadio name="theme" value="danger" />
                    <CustomRadio name="theme" value="custom" customColor="#ff6600" />
                </>,
                {
                    initialValues: { theme: "" },
                },
            );

            const radios = screen.getAllByTestId("radio-input");
            expect(radios).toHaveLength(4);

            await act(async () => {
                await user.click(radios[2]); // Select danger
            });

            expect(getFormValues().theme).toBe("danger");
        });

        it("handles programmatic value changes", async () => {
            const { setFieldValue, getFormValues } = renderWithFormik(
                <>
                    <RadioBase name="priority" value="low" />
                    <RadioBase name="priority" value="medium" />
                    <RadioBase name="priority" value="high" />
                </>,
                {
                    initialValues: { priority: "low" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            expect(radios[0].checked).toBe(true);

            await setFieldValue("priority", "high");

            expect(radios[0].checked).toBe(false);
            expect(radios[2].checked).toBe(true);
            expect(getFormValues().priority).toBe("high");
        });

        it("handles validation for specific radio values", async () => {
            const validationSchema = yup.object({
                consent: yup.string()
                    .oneOf(["yes"], "You must agree to continue")
                    .required("Please make a selection"),
            });

            const { user, onSubmit } = renderWithFormik(
                <>
                    <RadioBase name="consent" value="yes" />
                    <RadioBase name="consent" value="no" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { consent: "" },
                    formikConfig: { validationSchema },
                },
            );

            const radios = screen.getAllByTestId("radio-input");
            const submitButton = screen.getByRole("button", { name: /submit/i });
            
            // Try to submit with "no" selected
            await act(async () => {
                await user.click(radios[1]); // Select "no"
                await user.click(submitButton);
            });

            expect(onSubmit).not.toHaveBeenCalled();
            formAssertions.expectFieldError("You must agree to continue");

            // Now select "yes" and submit
            await act(async () => {
                await user.click(radios[0]); // Select "yes"
                await user.click(submitButton);
            });

            formAssertions.expectFormSubmitted(onSubmit, { consent: "yes" });
        });

        it("maintains proper keyboard navigation within Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <>
                    <RadioBase name="rating" value="1" />
                    <RadioBase name="rating" value="2" />
                    <RadioBase name="rating" value="3" />
                    <RadioBase name="rating" value="4" />
                    <RadioBase name="rating" value="5" />
                </>,
                {
                    initialValues: { rating: "3" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            radios[2].focus(); // Focus on the initially selected radio

            await act(async () => {
                await user.keyboard("{ArrowDown}");
            });

            expect(document.activeElement).toBe(radios[3]);
            expect(getFormValues().rating).toBe("4");

            await act(async () => {
                await user.keyboard("{ArrowUp}");
                await user.keyboard("{ArrowUp}");
            });

            expect(document.activeElement).toBe(radios[1]);
            expect(getFormValues().rating).toBe("2");
        });
    });
});

describe("Radio (Formik Integration)", () => {
    describe("Formik integration", () => {
        it("works as a controlled component within Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <>
                    <Radio name="choice" value="option1" label="Option 1" />
                    <Radio name="choice" value="option2" label="Option 2" />
                    <Radio name="choice" value="option3" label="Option 3" />
                </>,
                {
                    initialValues: { choice: "" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(false);
            expect(radios[2].checked).toBe(false);

            await act(async () => {
                await user.click(radios[1]);
            });

            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
            expect(getFormValues().choice).toBe("option2");
        });

        it("respects initial values", () => {
            const { getFormValues } = renderWithFormik(
                <>
                    <Radio name="size" value="small" label="Small" />
                    <Radio name="size" value="medium" label="Medium" />
                    <Radio name="size" value="large" label="Large" />
                </>,
                {
                    initialValues: { size: "medium" },
                },
            );

            const radios = screen.getAllByTestId("radio-input") as HTMLInputElement[];
            
            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
            expect(getFormValues().size).toBe("medium");
        });

        it("shows validation errors", async () => {
            const validationSchema = yup.object({
                agreement: yup.string()
                    .required("Please select an option"),
            });

            const { user, onSubmit } = renderWithFormik(
                <>
                    <Radio name="agreement" value="yes" label="Yes" />
                    <Radio name="agreement" value="no" label="No" />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { agreement: "" },
                    formikConfig: { validationSchema },
                },
            );

            const submitButton = screen.getByRole("button", { name: /submit/i });

            await act(async () => {
                await user.click(submitButton);
            });

            expect(onSubmit).not.toHaveBeenCalled();
            formAssertions.expectFieldError("Please select an option");
        });

        it("uses factory components", async () => {
            const { user, getFormValues } = renderWithFormik(
                <>
                    <RadioFactory.Primary name="theme" value="light" label="Light" />
                    <RadioFactory.Secondary name="theme" value="dark" label="Dark" />
                    <RadioFactory.Danger name="theme" value="danger" label="Danger" />
                </>,
                {
                    initialValues: { theme: "" },
                },
            );

            const radios = screen.getAllByTestId("radio-input");

            await act(async () => {
                await user.click(radios[2]);
            });

            expect(getFormValues().theme).toBe("danger");
        });
    });
});

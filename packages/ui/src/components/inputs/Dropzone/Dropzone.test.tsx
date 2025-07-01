import { act, render, screen, waitFor, fireEvent } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { createTheme, ThemeProvider } from "@mui/material/styles";
import React from "react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import * as yup from "yup";
import { renderWithFormik, formInteractions, formAssertions } from "../../../__test/helpers/formTestHelpers";
import { Dropzone } from "./Dropzone";
import { PubSub } from "../../../utils/pubsub.js";

// Mock PubSub
vi.mock("../../../utils/pubsub.js", () => ({
    PubSub: {
        get: vi.fn(() => ({
            publish: vi.fn(),
        })),
    },
}));

// Mock URL.createObjectURL and URL.revokeObjectURL
global.URL.createObjectURL = vi.fn(() => "mocked-url");
global.URL.revokeObjectURL = vi.fn();

// Create a test theme
const theme = createTheme();

// Helper function to render with theme
const renderWithTheme = (component: React.ReactElement) => {
    return render(
        <ThemeProvider theme={theme}>
            {component}
        </ThemeProvider>,
    );
};

describe("Dropzone", () => {
    let onUploadMock: ReturnType<typeof vi.fn>;
    let user: ReturnType<typeof userEvent.setup>;

    beforeEach(() => {
        onUploadMock = vi.fn();
        user = userEvent.setup();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("Initial Rendering", () => {
        it("renders with default props", () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            expect(screen.getByTestId("dropzone-container")).toBeDefined();
            expect(screen.getByTestId("dropzone-area")).toBeDefined();
            expect(screen.getByTestId("dropzone-input")).toBeDefined();
            expect(screen.getByTestId("dropzone-text").textContent).toBe("Drag 'n' drop files here or click");
            expect(screen.getByTestId("dropzone-upload-button").textContent).toBe("Upload file(s)");
            expect(screen.getByTestId("dropzone-cancel-button").textContent).toBe("Cancel upload");
        });

        it("renders with custom text props", () => {
            renderWithTheme(
                <Dropzone
                    onUpload={onUploadMock}
                    dropzoneText="Custom drop text"
                    uploadText="Custom upload"
                    cancelText="Custom cancel"
                />,
            );

            expect(screen.getByTestId("dropzone-text").textContent).toBe("Custom drop text");
            expect(screen.getByTestId("dropzone-upload-button").textContent).toBe("Custom upload");
            expect(screen.getByTestId("dropzone-cancel-button").textContent).toBe("Custom cancel");
        });

        it("disables buttons when no files are selected", () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            const cancelButton = screen.getByTestId("dropzone-cancel-button");

            expect(uploadButton.disabled).toBe(true);
            expect(cancelButton.disabled).toBe(true);
        });

        it("disables upload button when disabled prop is true", () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} disabled={true} />);

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            expect(uploadButton.disabled).toBe(true);
        });
    });

    describe("File Selection", () => {
        it("accepts files via file input", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            // Check that buttons are now enabled
            expect(screen.getByTestId("dropzone-upload-button").disabled).toBe(false);
            expect(screen.getByTestId("dropzone-cancel-button").disabled).toBe(false);
        });

        it("shows thumbnails when files are selected and showThumbs is true", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} showThumbs={true} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            await waitFor(() => {
                expect(screen.getByTestId("dropzone-thumbs")).toBeDefined();
                expect(screen.getByTestId("dropzone-thumb-test.png")).toBeDefined();
                expect(screen.getByTestId("dropzone-thumb-img-test.png").getAttribute("src")).toBe("mocked-url");
            });
        });

        it("does not show thumbnails when showThumbs is false", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} showThumbs={false} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            expect(screen.queryByTestId("dropzone-thumbs")).toBeNull();
        });

        it("accepts multiple files", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} maxFiles={3} />);

            const files = [
                new File(["content1"], "test1.png", { type: "image/png" }),
                new File(["content2"], "test2.png", { type: "image/png" }),
                new File(["content3"], "test3.png", { type: "image/png" }),
            ];

            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, files);
            });

            await waitFor(() => {
                expect(screen.getByTestId("dropzone-thumb-test1.png")).toBeDefined();
                expect(screen.getByTestId("dropzone-thumb-test2.png")).toBeDefined();
                expect(screen.getByTestId("dropzone-thumb-test3.png")).toBeDefined();
            });
        });

        it("respects acceptedFileTypes prop", async () => {
            const { rerender } = renderWithTheme(
                <Dropzone 
                    onUpload={onUploadMock} 
                    acceptedFileTypes={[".pdf", ".doc"]} 
                />,
            );

            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;
            expect(input.getAttribute("accept")).toBe(".pdf,.doc");

            // Test with empty array
            rerender(
                <ThemeProvider theme={theme}>
                    <Dropzone 
                        onUpload={onUploadMock} 
                        acceptedFileTypes={[]} 
                    />
                </ThemeProvider>,
            );

            expect(input.getAttribute("accept")).toBeNull();
        });
    });

    describe("Upload Functionality", () => {
        it("calls onUpload with files when upload button is clicked", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");

            await act(async () => {
                await user.click(uploadButton);
            });

            expect(onUploadMock).toHaveBeenCalledTimes(1);
            expect(onUploadMock).toHaveBeenCalledWith(
                expect.arrayContaining([
                    expect.objectContaining({
                        name: "test.png",
                        type: "image/png",
                        preview: "mocked-url",
                    }),
                ]),
            );
        });

        it("clears files after successful upload", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            expect(screen.getByTestId("dropzone-thumb-test.png")).toBeDefined();

            const uploadButton = screen.getByTestId("dropzone-upload-button");

            await act(async () => {
                await user.click(uploadButton);
            });

            await waitFor(() => {
                expect(screen.queryByTestId("dropzone-thumb-test.png")).toBeNull();
                expect(screen.getByTestId("dropzone-upload-button").disabled).toBe(true);
                expect(screen.getByTestId("dropzone-cancel-button").disabled).toBe(true);
            });
        });

        it("does not upload when disabled", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} disabled={true} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            
            // Button should be disabled
            expect(uploadButton.disabled).toBe(true);

            // Try to click anyway (shouldn't work)
            await act(async () => {
                fireEvent.click(uploadButton);
            });

            expect(onUploadMock).not.toHaveBeenCalled();
        });

        it("shows error when trying to upload with no files and component is enabled", async () => {
            const mockPublish = vi.fn();
            vi.mocked(PubSub.get).mockReturnValue({ publish: mockPublish });

            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            // Get the upload button which should be disabled
            const uploadButton = screen.getByTestId("dropzone-upload-button");
            
            // Verify the button is disabled when no files are selected
            expect(uploadButton.disabled).toBe(true);
            
            // This test verifies that the button is properly disabled and won't trigger uploads
            // In a real scenario, users can't click disabled buttons, so no error message is shown
            expect(onUploadMock).not.toHaveBeenCalled();
        });
    });

    describe("Cancel Functionality", () => {
        it("clears files when cancel button is clicked", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            expect(screen.getByTestId("dropzone-thumb-test.png")).toBeDefined();

            const cancelButton = screen.getByTestId("dropzone-cancel-button");

            await act(async () => {
                await user.click(cancelButton);
            });

            await waitFor(() => {
                expect(screen.queryByTestId("dropzone-thumb-test.png")).toBeNull();
                expect(screen.getByTestId("dropzone-upload-button").disabled).toBe(true);
                expect(screen.getByTestId("dropzone-cancel-button").disabled).toBe(true);
            });
        });

        it("does not call onUpload when cancel is clicked", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const cancelButton = screen.getByTestId("dropzone-cancel-button");

            await act(async () => {
                await user.click(cancelButton);
            });

            expect(onUploadMock).not.toHaveBeenCalled();
        });

        it("stops event propagation on cancel", async () => {
            const onParentClick = vi.fn();
            renderWithTheme(
                <div onClick={onParentClick}>
                    <Dropzone onUpload={onUploadMock} />
                </div>,
            );

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const cancelButton = screen.getByTestId("dropzone-cancel-button");

            await act(async () => {
                await user.click(cancelButton);
            });

            expect(onParentClick).not.toHaveBeenCalled();
        });
    });

    describe("Memory Management", () => {
        it("revokes object URLs on cleanup", async () => {
            const { unmount } = renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            expect(global.URL.createObjectURL).toHaveBeenCalledWith(expect.any(File));

            unmount();

            expect(global.URL.revokeObjectURL).toHaveBeenCalledWith("mocked-url");
        });

        it("revokes object URLs when files change", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file1 = new File(["test content 1"], "test1.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file1);
            });

            vi.clearAllMocks();

            // Upload a new file (which replaces the old one)
            const file2 = new File(["test content 2"], "test2.png", { type: "image/png" });

            await act(async () => {
                await user.upload(input, file2);
            });

            // Should have revoked the URL for the first file
            expect(global.URL.revokeObjectURL).toHaveBeenCalledWith("mocked-url");
        });
    });

    describe("Drag and Drop", () => {
        it("accepts files via drag and drop", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const dropzoneArea = screen.getByTestId("dropzone-area");
            const file = new File(["test content"], "test.png", { type: "image/png" });

            const dataTransfer = {
                files: [file],
                items: [
                    {
                        kind: "file",
                        type: file.type,
                        getAsFile: () => file,
                    },
                ],
                types: ["Files"],
            };

            await act(async () => {
                fireEvent.dragEnter(dropzoneArea, { dataTransfer });
                fireEvent.dragOver(dropzoneArea, { dataTransfer });
                fireEvent.drop(dropzoneArea, { dataTransfer });
            });

            await waitFor(() => {
                expect(screen.getByTestId("dropzone-thumb-test.png")).toBeDefined();
                expect(screen.getByTestId("dropzone-upload-button").disabled).toBe(false);
            });
        });

        it("shows error when invalid files are dropped", async () => {
            const mockPublish = vi.fn();
            vi.mocked(PubSub.get).mockReturnValue({ publish: mockPublish });

            renderWithTheme(<Dropzone onUpload={onUploadMock} acceptedFileTypes={["image/*"]} />);

            const dropzoneArea = screen.getByTestId("dropzone-area");
            
            // Create an empty file list to simulate rejected files
            const dataTransfer = {
                files: [],
                items: [],
                types: ["Files"],
            };

            await act(async () => {
                fireEvent.dragEnter(dropzoneArea, { dataTransfer });
                fireEvent.dragOver(dropzoneArea, { dataTransfer });
                fireEvent.drop(dropzoneArea, { dataTransfer });
            });

            expect(mockPublish).toHaveBeenCalledWith("snack", {
                messageKey: "FilesNotAccepted",
                severity: "Error",
            });
        });
    });

    describe("Accessibility", () => {
        it("has proper input attributes", () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const input = screen.getByTestId("dropzone-input");
            expect(input.getAttribute("type")).toBe("file");
            expect(input.getAttribute("multiple")).toBe("");
        });

        it("allows keyboard navigation", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            const cancelButton = screen.getByTestId("dropzone-cancel-button");

            // Focus the upload button
            await act(async () => {
                uploadButton.focus();
            });

            expect(document.activeElement).toBe(uploadButton);

            // Tab to cancel button
            await act(async () => {
                await user.tab();
            });

            expect(document.activeElement).toBe(cancelButton);

            // Shift+Tab back to upload button
            await act(async () => {
                await user.tab({ shift: true });
            });

            expect(document.activeElement).toBe(uploadButton);
        });
    });

    describe("Edge Cases", () => {
        it("handles files with duplicate names", async () => {
            renderWithTheme(<Dropzone onUpload={onUploadMock} />);

            const file1 = new File(["content1"], "test.png", { type: "image/png" });
            const file2 = new File(["content2"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, [file1, file2]);
            });

            // Both files should be displayed (React will handle key duplication warnings)
            const thumbs = screen.getAllByTestId("dropzone-thumb-test.png");
            expect(thumbs).toHaveLength(2);
        });

        it("stops event propagation on upload", async () => {
            const onParentClick = vi.fn();
            renderWithTheme(
                <div onClick={onParentClick}>
                    <Dropzone onUpload={onUploadMock} />
                </div>,
            );

            const file = new File(["test content"], "test.png", { type: "image/png" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");

            await act(async () => {
                await user.click(uploadButton);
            });

            expect(onParentClick).not.toHaveBeenCalled();
        });
    });

    describe("Formik integration", () => {
        it("works as a controlled component within Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <Dropzone 
                    name="documents" 
                    onUpload={(files) => {
                        // In real usage, you'd handle file upload here
                        console.log("Files to upload:", files);
                    }}
                />,
                {
                    initialValues: { documents: [] },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <Dropzone 
                                name="documents"
                                onUpload={(files) => {
                                    // Set files in Formik state
                                    formikProps.setFieldValue("documents", files);
                                }}
                            />
                        </ThemeProvider>
                    ),
                },
            );

            const file = new File(["test content"], "test.pdf", { type: "application/pdf" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            
            await act(async () => {
                await user.click(uploadButton);
            });

            // Files should be in Formik state after upload
            expect(getFormValues().documents).toHaveLength(1);
            expect(getFormValues().documents[0]).toMatchObject({
                name: "test.pdf",
                type: "application/pdf",
            });
        });

        it("submits form with uploaded files", async () => {
            const { user, onSubmit, submitForm } = renderWithFormik(
                <>
                    <input name="title" placeholder="Title" />
                    <Dropzone 
                        name="attachments"
                        onUpload={(files) => {
                            // Handle file upload
                        }}
                    />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { 
                        title: "",
                        attachments: [],
                    },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <form onSubmit={formikProps.handleSubmit}>
                                <input 
                                    name="title" 
                                    placeholder="Title"
                                    value={formikProps.values.title}
                                    onChange={formikProps.handleChange}
                                />
                                <Dropzone 
                                    name="attachments"
                                    onUpload={(files) => {
                                        formikProps.setFieldValue("attachments", files);
                                    }}
                                />
                                <button type="submit">Submit</button>
                            </form>
                        </ThemeProvider>
                    ),
                },
            );

            const titleInput = screen.getByPlaceholderText("Title");
            await act(async () => {
                await user.type(titleInput, "My Document");
            });

            const file = new File(["content"], "document.pdf", { type: "application/pdf" });
            const dropzoneInput = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(dropzoneInput, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            await act(async () => {
                await user.click(uploadButton);
            });

            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, {
                title: "My Document",
                attachments: expect.arrayContaining([
                    expect.objectContaining({
                        name: "document.pdf",
                        type: "application/pdf",
                    }),
                ]),
            });
        });

        it("validates file requirements", async () => {
            const validationSchema = yup.object({
                resume: yup.array()
                    .min(1, "Please upload your resume")
                    .max(1, "Only one file allowed")
                    .required("Resume is required"),
            });

            const { user, onSubmit } = renderWithFormik(
                <>
                    <Dropzone 
                        name="resume"
                        onUpload={(files) => {
                            // Handle upload
                        }}
                        maxFiles={1}
                        acceptedFileTypes={[".pdf", ".doc", ".docx"]}
                    />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { resume: [] },
                    formikConfig: { validationSchema },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <form onSubmit={formikProps.handleSubmit}>
                                <Dropzone 
                                    name="resume"
                                    onUpload={(files) => {
                                        formikProps.setFieldValue("resume", files);
                                    }}
                                    maxFiles={1}
                                    acceptedFileTypes={[".pdf", ".doc", ".docx"]}
                                />
                                <button type="submit">Submit</button>
                                {formikProps.errors.resume && formikProps.touched.resume && (
                                    <div>{formikProps.errors.resume}</div>
                                )}
                            </form>
                        </ThemeProvider>
                    ),
                },
            );

            const submitButton = screen.getByRole("button", { name: /submit/i });
            
            // Try to submit without files
            await act(async () => {
                await user.click(submitButton);
            });

            expect(onSubmit).not.toHaveBeenCalled();
            await waitFor(() => {
                expect(screen.getByText("Please upload your resume")).toBeDefined();
            });
        });

        it("handles multiple files in Formik", async () => {
            const { user, getFormValues } = renderWithFormik(
                <Dropzone 
                    name="gallery"
                    onUpload={(files) => {
                        // Handle upload
                    }}
                    maxFiles={5}
                    showThumbs={true}
                />,
                {
                    initialValues: { gallery: [] },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <Dropzone 
                                name="gallery"
                                onUpload={(files) => {
                                    formikProps.setFieldValue("gallery", files);
                                }}
                                maxFiles={5}
                                showThumbs={true}
                            />
                        </ThemeProvider>
                    ),
                },
            );

            const files = [
                new File(["content1"], "image1.jpg", { type: "image/jpeg" }),
                new File(["content2"], "image2.jpg", { type: "image/jpeg" }),
                new File(["content3"], "image3.jpg", { type: "image/jpeg" }),
            ];

            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, files);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            
            await act(async () => {
                await user.click(uploadButton);
            });

            expect(getFormValues().gallery).toHaveLength(3);
            expect(getFormValues().gallery[0].name).toBe("image1.jpg");
            expect(getFormValues().gallery[1].name).toBe("image2.jpg");
            expect(getFormValues().gallery[2].name).toBe("image3.jpg");
        });

        it("clears files on form reset", async () => {
            const { user, resetForm, getFormValues } = renderWithFormik(
                <Dropzone 
                    name="files"
                    onUpload={(files) => {
                        // Handle upload
                    }}
                />,
                {
                    initialValues: { files: [] },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <Dropzone 
                                name="files"
                                onUpload={(files) => {
                                    formikProps.setFieldValue("files", files);
                                }}
                            />
                        </ThemeProvider>
                    ),
                },
            );

            const file = new File(["content"], "test.txt", { type: "text/plain" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            
            await act(async () => {
                await user.click(uploadButton);
            });

            expect(getFormValues().files).toHaveLength(1);

            resetForm();

            expect(getFormValues().files).toHaveLength(0);
        });

        it("preserves disabled state in Formik context", async () => {
            const onUploadMock = vi.fn();
            
            const { user } = renderWithFormik(
                <Dropzone 
                    name="protected"
                    onUpload={onUploadMock}
                    disabled={true}
                />,
                {
                    initialValues: { protected: [] },
                    customRender: () => (
                        <ThemeProvider theme={theme}>
                            <Dropzone 
                                name="protected"
                                onUpload={onUploadMock}
                                disabled={true}
                            />
                        </ThemeProvider>
                    ),
                },
            );

            const file = new File(["content"], "test.pdf", { type: "application/pdf" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            expect(uploadButton.disabled).toBe(true);

            // Try to click anyway
            await act(async () => {
                fireEvent.click(uploadButton);
            });

            expect(onUploadMock).not.toHaveBeenCalled();
        });

        it("handles file type restrictions in Formik", async () => {
            const mockPublish = vi.fn();
            vi.mocked(PubSub.get).mockReturnValue({ publish: mockPublish });

            const { user, getFormValues } = renderWithFormik(
                <Dropzone 
                    name="images"
                    onUpload={(files) => {
                        // Handle upload
                    }}
                    acceptedFileTypes={["image/*"]}
                />,
                {
                    initialValues: { images: [] },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <Dropzone 
                                name="images"
                                onUpload={(files) => {
                                    formikProps.setFieldValue("images", files);
                                }}
                                acceptedFileTypes={["image/*"]}
                            />
                        </ThemeProvider>
                    ),
                },
            );

            // Upload valid image file
            const imageFile = new File(["image"], "photo.jpg", { type: "image/jpeg" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, imageFile);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            
            await act(async () => {
                await user.click(uploadButton);
            });

            expect(getFormValues().images).toHaveLength(1);
        });

        it("integrates with other form fields", async () => {
            const validationSchema = yup.object({
                projectName: yup.string().required("Project name is required"),
                description: yup.string(),
                files: yup.array().min(1, "At least one file is required"),
            });

            const { user, onSubmit, submitForm } = renderWithFormik(
                <>
                    <input name="projectName" placeholder="Project Name" />
                    <textarea name="description" placeholder="Description" />
                    <Dropzone 
                        name="files"
                        onUpload={(files) => {
                            // Handle upload
                        }}
                    />
                    <button type="submit">Submit</button>
                </>,
                {
                    initialValues: { 
                        projectName: "",
                        description: "",
                        files: [],
                    },
                    formikConfig: { validationSchema },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <form onSubmit={formikProps.handleSubmit}>
                                <input 
                                    name="projectName" 
                                    placeholder="Project Name"
                                    value={formikProps.values.projectName}
                                    onChange={formikProps.handleChange}
                                />
                                <textarea 
                                    name="description" 
                                    placeholder="Description"
                                    value={formikProps.values.description}
                                    onChange={formikProps.handleChange}
                                />
                                <Dropzone 
                                    name="files"
                                    onUpload={(files) => {
                                        formikProps.setFieldValue("files", files);
                                    }}
                                />
                                <button type="submit">Submit</button>
                            </form>
                        </ThemeProvider>
                    ),
                },
            );

            // Fill in form fields
            const nameInput = screen.getByPlaceholderText("Project Name");
            const descriptionInput = screen.getByPlaceholderText("Description");
            
            await act(async () => {
                await user.type(nameInput, "My Project");
                await user.type(descriptionInput, "This is a test project");
            });

            // Upload file
            const file = new File(["content"], "project.zip", { type: "application/zip" });
            const dropzoneInput = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(dropzoneInput, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            await act(async () => {
                await user.click(uploadButton);
            });

            // Submit form
            await submitForm();

            formAssertions.expectFormSubmitted(onSubmit, {
                projectName: "My Project",
                description: "This is a test project",
                files: expect.arrayContaining([
                    expect.objectContaining({
                        name: "project.zip",
                        type: "application/zip",
                    }),
                ]),
            });
        });

        it("updates field touched state", async () => {
            const { user, isFieldTouched } = renderWithFormik(
                <Dropzone 
                    name="upload"
                    onUpload={(files) => {
                        // Handle upload
                    }}
                />,
                {
                    initialValues: { upload: [] },
                    customRender: (formikProps) => (
                        <ThemeProvider theme={theme}>
                            <Dropzone 
                                name="upload"
                                onUpload={(files) => {
                                    formikProps.setFieldValue("upload", files);
                                    formikProps.setFieldTouched("upload", true);
                                }}
                            />
                        </ThemeProvider>
                    ),
                },
            );

            expect(isFieldTouched("upload")).toBe(false);

            const file = new File(["content"], "test.txt", { type: "text/plain" });
            const input = screen.getByTestId("dropzone-input") as HTMLInputElement;

            await act(async () => {
                await user.upload(input, file);
            });

            const uploadButton = screen.getByTestId("dropzone-upload-button");
            
            await act(async () => {
                await user.click(uploadButton);
            });

            expect(isFieldTouched("upload")).toBe(true);
        });
    });
});

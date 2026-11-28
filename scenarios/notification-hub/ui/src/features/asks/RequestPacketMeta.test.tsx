import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { RequestPacketMeta } from "./RequestPacketMeta";

describe("RequestPacketMeta", () => {
  it("shows nothing for a plain decision", () => {
    const { container } = renderWithProviders(<RequestPacketMeta ask={{ attachmentCount: 0, questionCount: 0, openContextRequestCount: 0 }} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("counts attachments and questions and says the operator is waiting on the requester", () => {
    renderWithProviders(<RequestPacketMeta ask={{ attachmentCount: 3, questionCount: 2, openContextRequestCount: 1 }} />);
    expect(screen.getByText(strings.pages.history.attachments)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.history.questions)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.history.askedContext)).toBeInTheDocument();
  });
});

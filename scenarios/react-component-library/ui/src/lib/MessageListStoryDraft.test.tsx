import { describe, expect, it } from "vitest";
import { appendStreamingDetail } from "../../../library/components/MessageList/versions/1.1.3/story";
import { renderWithProviders } from "../test-utils";

describe("MessageList story draft", () => {
  it("preserves rich React content when adding streaming detail", () => {
    const content = <strong>Structured response</strong>;
    const { getByText } = renderWithProviders(<p>{appendStreamingDetail(content)}</p>);

    expect(getByText("Structured response").tagName).toBe("STRONG");
    expect(getByText("Additional streaming detail.")).toBeInTheDocument();
  });
});

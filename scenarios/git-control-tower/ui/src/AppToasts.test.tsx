import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { PushToast } from "./AppToasts";

describe("PushToast", () => {
  it("shows the git output behind a failure, not just a status word", async () => {
    render(
      <PushToast
        notice={{
          tone: "error",
          message: "Push to origin/agi failed",
          detail: "git push timed out before the transfer finished: context deadline exceeded",
          sticky: true
        }}
        positionClass="fixed bottom-4 right-4"
        onDismiss={vi.fn()}
      />
    );

    expect(screen.getByTestId("push-toast")).toHaveTextContent("Push to origin/agi failed");
    expect(screen.getByTestId("push-toast-detail")).toHaveTextContent(
      "timed out before the transfer finished"
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("only offers dismissal for notices that must be acknowledged", async () => {
    const onDismiss = vi.fn();
    const { rerender } = render(
      <PushToast
        notice={{ tone: "success", message: "Pushed to origin/agi" }}
        positionClass="fixed bottom-4 right-4"
        onDismiss={onDismiss}
      />
    );
    expect(screen.queryByTestId("push-toast-dismiss")).not.toBeInTheDocument();

    rerender(
      <PushToast
        notice={{ tone: "error", message: "Push to origin/agi failed", sticky: true }}
        positionClass="fixed bottom-4 right-4"
        onDismiss={onDismiss}
      />
    );
    await userEvent.click(screen.getByTestId("push-toast-dismiss"));
    expect(onDismiss).toHaveBeenCalled();
  });
});

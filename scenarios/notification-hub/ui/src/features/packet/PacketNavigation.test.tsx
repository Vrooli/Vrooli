import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation, useParams } from "react-router-dom";
import { renderWithProviders } from "../../test-utils";
import { conversationsClient } from "../../api/notifications";
import { strings } from "../../consts/strings";
import { PacketNavigation, usePacketNavigation, usePacketSwipe } from "./PacketNavigation";

vi.mock("../../api/notifications", () => ({ conversationsClient: { listAsks: vi.fn() } }));

function Example() {
  const location = useLocation();
  const { askId } = useParams();
  const current = askId ?? location.pathname.split("/").pop() ?? "";
  const navigation = usePacketNavigation(current, "/asks");
  usePacketSwipe(navigation);
  return <><PacketNavigation navigation={navigation} /><output data-testid="location">{location.pathname}</output></>;
}

describe("packet navigation", () => {
  it("moves through all pages without answering, including by swipe", async () => {
    vi.mocked(conversationsClient.listAsks).mockResolvedValueOnce({ asks: [{ id: "a" }, { id: "b" }], nextCursor: "more" } as never).mockResolvedValueOnce({ asks: [{ id: "c" }], nextCursor: "" } as never);
    renderWithProviders(<MemoryRouter initialEntries={["/asks/b"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Example /></MemoryRouter>, { withoutRouter: true });
    expect(await screen.findByText(strings.packet.navigation.position)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: strings.packet.navigation.next }));
    expect(screen.getByTestId("location")).toHaveTextContent("/asks/c");
    fireEvent.touchStart(document, { touches: [{ clientX: 10, clientY: 20 }] });
    fireEvent.touchEnd(document, { changedTouches: [{ clientX: 120, clientY: 25 }] });
    expect(screen.getByTestId("location")).toHaveTextContent("/asks/b");
  });
});

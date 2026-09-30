import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { EmbeddedServerSection } from "./EmbeddedServerSection";

describe("EmbeddedServerSection", () => {
  it("keeps all local-runtime parameters explicit and editable", () => {
    const onServerPortChange = vi.fn();
    const onLocalServerPathChange = vi.fn();
    const onLocalApiEndpointChange = vi.fn();
    render(
      <EmbeddedServerSection
        serverPort={3900}
        onServerPortChange={onServerPortChange}
        localServerPath="api/main.js"
        onLocalServerPathChange={onLocalServerPathChange}
        localApiEndpoint="http://127.0.0.1:3900"
        onLocalApiEndpointChange={onLocalApiEndpointChange}
      />,
    );

    expect(
      screen.getByText(/Embedded servers require more manual work/),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Server Port"), {
      target: { value: "4567" },
    });
    fireEvent.change(screen.getByLabelText("Server Entry"), {
      target: { value: "dist/server.js" },
    });
    fireEvent.change(screen.getByLabelText("API Endpoint"), {
      target: { value: "http://127.0.0.1:4567" },
    });
    expect(onServerPortChange).toHaveBeenCalledWith(4567);
    expect(onLocalServerPathChange).toHaveBeenCalledWith("dist/server.js");
    expect(onLocalApiEndpointChange).toHaveBeenCalledWith(
      "http://127.0.0.1:4567",
    );
  });
});

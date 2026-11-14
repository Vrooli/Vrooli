import { fireEvent, render, screen } from "@testing-library/react";
import { forwardRef, type ButtonHTMLAttributes } from "react";
import { describe, expect, it, vi } from "vitest";

import { Button } from "@vrooli/react-component-library/Button/2";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1.1.5";

describe("SettingsList rows that open something", () => {
  it("opens the row from its label and names the button after the label", () => {
    const onSelect = vi.fn();
    render(
      <SettingsList>
        <SettingsList.Group label="Services">
          <SettingsList.Row label="Voice" hint="Dictation and read-aloud" onSelect={onSelect}>
            <span>Ready</span>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    const open = screen.getByRole("button", { name: "Voice" });
    expect(open).toHaveAccessibleDescription("Dictation and read-aloud Ready");
    fireEvent.click(open);
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("keeps a control inside the row separate from opening the row", () => {
    const onSelect = vi.fn();
    const onStart = vi.fn();
    render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Remote terminals" onSelect={onSelect}>
            <button type="button" onClick={onStart}>Start</button>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).toHaveBeenCalledTimes(1);
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("draws a chevron only on rows that open something", () => {
    const { container } = render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Voice" onSelect={() => undefined} />
          <SettingsList.Row label="Standard sessions">
            <span>Ready</span>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    expect(container.querySelectorAll("[data-rcl-settings-row-chevron]")).toHaveLength(1);
    expect(screen.getByText("Standard sessions").tagName).toBe("DIV");
  });

  it("renders a leading icon beside the label", () => {
    const { container } = render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Voice" leading={<span data-testid="icon" />}>
            <span>Ready</span>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    const row = container.querySelector("[data-rcl-settings-row]");
    expect(row).toHaveAttribute("data-has-leading");
    expect(row?.firstElementChild).toContainElement(screen.getByTestId("icon"));
  });

  it("leaves out the control area when a row's children render nothing", () => {
    const { container } = render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Wake word">{false}</SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    expect(container.querySelector("[data-rcl-settings-row-control]")).toBeNull();
  });

  it("still labels a bare control from its row", () => {
    render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Adaptive chrome">
            <input type="checkbox" />
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    expect(screen.getByRole("checkbox", { name: "Adaptive chrome" })).toBeInTheDocument();
  });
  it("keeps the library Button named by its own text", () => {
    render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Balance">
            <Button type="button">Refresh</Button>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    expect(screen.getByRole("button", { name: "Refresh" })).toBeInTheDocument();
  });

  it("keeps an app's own button and link named by their own text", () => {
    const AppButton = forwardRef<HTMLButtonElement, ButtonHTMLAttributes<HTMLButtonElement>>((props, ref) => (
      <button ref={ref} type="button" {...props} />
    ));
    const AppLink = (props: { href: string; children: string }) => <a {...props} />;
    render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Profile status">
            <AppButton>Save profile</AppButton>
          </SettingsList.Row>
          <SettingsList.Row label="Privacy">
            <AppLink href="/privacy">Read the policy</AppLink>
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    expect(screen.getByRole("button", { name: "Save profile" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Read the policy" })).toBeInTheDocument();
  });

  it("still names a switch and an icon-only button after the row", () => {
    const IconButton = (props: { "aria-labelledby"?: string }) => (
      <button type="button" {...props}>
        <svg aria-hidden="true" />
      </button>
    );
    const SwitchButton = (props: { "aria-labelledby"?: string }) => (
      <button type="button" role="switch" aria-checked="false" {...props}>
        Off
      </button>
    );
    render(
      <SettingsList>
        <SettingsList.Group>
          <SettingsList.Row label="Use my credits">
            <SwitchButton />
          </SettingsList.Row>
          <SettingsList.Row label="Reload">
            <IconButton />
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>,
    );

    expect(screen.getByRole("switch", { name: "Use my credits" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reload" })).toBeInTheDocument();
  });
});

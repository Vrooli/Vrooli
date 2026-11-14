import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { externalHost } from "../features/asks/askView";
import { SourceLink } from "./SourceLink";

describe("externalHost", () => {
  const origin = window.location.origin;

  it("names the host a link leaves this site for, with its port", () => {
    expect(externalHost("https://agent-manager.example.test/api/v1/authorizations/1/approve")).toBe("agent-manager.example.test");
    expect(externalHost("http://127.0.0.1:18080/x")).toBe("127.0.0.1:18080");
  });

  it("shows the address the browser will open for a look-alike name", () => {
    expect(externalHost("https://exаmple.test/")).toBe("xn--exmple-4nf.test");
  });

  it("names nothing for a link that stays on the Hub", () => {
    expect(externalHost("/delivery/abc")).toBeNull();
    expect(externalHost(`${origin}/asks/abc`)).toBeNull();
  });

  it("names nothing for text that is not a link", () => {
    expect(externalHost("http://")).toBeNull();
  });
});

describe("SourceLink", () => {
  beforeEach(async () => { await setLocale("en"); });
  afterEach(cleanup);

  it("puts the destination host beside the words, isolated as left-to-right text", () => {
    renderWithProviders(<SourceLink href="https://agent-manager.example.test/runs/1" />);
    const link = screen.getByRole("link", { name: "Open source · agent-manager.example.test" });
    expect(link).toHaveAttribute("href", "https://agent-manager.example.test/runs/1");
    const host = link.querySelector("bdi");
    expect(host).toHaveTextContent("agent-manager.example.test");
    expect(host).toHaveAttribute("dir", "ltr");
  });

  it("adds no host for a link on the Hub itself", () => {
    renderWithProviders(<SourceLink href="/delivery/abc" />);
    const link = screen.getByRole("link", { name: "Open source" });
    expect(link.querySelector("bdi")).toBeNull();
  });
});

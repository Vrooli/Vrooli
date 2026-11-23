import { screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../test-utils";
import { absolute, relative } from "../lib/format";
import { DerivedTimeSchema } from "@vrooli/proto-types/research-desk/v1/shared/evidence_pb";
import { DerivedWhen, LastObserved, When } from "./When";

describe("When", () => {
  it("uses shared RelativeTime while retaining relative text, ISO value, and absolute title", () => {
    const date = new Date("2026-11-19T12:30:00.000Z");
    renderWithProviders(<When date={date} />);
    const time = screen.getByTestId("primitives.relative-time");
    expect(time).toHaveTextContent(relative(date));
    expect(time).toHaveAttribute("dateTime", date.toISOString());
    expect(time).toHaveAttribute("title", absolute(date));
  });

  it("keeps Research Desk labels and unknown/at-least interpretation in its thin wrappers", () => {
    renderWithProviders(
      <>
        <LastObserved />
        <DerivedWhen />
      </>,
    );
    expect(screen.getByLabelText("common.lastObserved")).toHaveTextContent("common.unknown");
    expect(screen.getByLabelText("common.lastObserved")).toHaveTextContent("common.unknown");
    expect(screen.queryByTestId("primitives.relative-time")).not.toBeInTheDocument();
  });

  it("keeps the at-least qualification outside the shared timestamp", () => {
    const date = new Date("2026-11-19T12:30:00.000Z");
    const time = create(DerivedTimeSchema, { value: timestampFromDate(date), atLeast: true });
    renderWithProviders(<DerivedWhen time={time} />);
    const renderedTime = screen.getByTestId("primitives.relative-time");
    expect(renderedTime).toHaveAttribute("dateTime", date.toISOString());
    expect(renderedTime.parentElement).toHaveTextContent("common.atLeast");
    expect(renderedTime.parentElement).toHaveTextContent(relative(date));
  });
});

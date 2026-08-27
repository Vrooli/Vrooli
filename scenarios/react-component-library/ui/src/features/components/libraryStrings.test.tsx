import { render, screen } from "@testing-library/react";
import { LibraryStringsProvider, defineStrings, useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";

function Probe() {
  return <span>{useStrings("controls.example.save", "Save changes")}</span>;
}

describe("library strings seam", () => {
  it("uses the declared English default without a provider", () => {
    render(<Probe />);
    expect(screen.getByText("Save changes")).toBeInTheDocument();
  });

  it("allows a provider to translate a declared key", () => {
    render(
      <LibraryStringsProvider
        strings={defineStrings("controls.example", { "controls.example.save": "Save changes" })}
        translate={(key, fallback) => (key === "controls.example.save" ? "Guardar cambios" : fallback)}
      >
        <Probe />
      </LibraryStringsProvider>,
    );
    expect(screen.getByText("Guardar cambios")).toBeInTheDocument();
  });
});

import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";

vi.mock("@react-three/fiber", () => ({
  Canvas: ({ children }: { children: ReactNode }) => (
    <div data-testid="canvas">{children}</div>
  ),
  useFrame: () => undefined,
}));

// Import after mock so the mock is applied.
import { SceneCanvas } from "./SceneCanvas";

describe("SceneCanvas", () => {
  it("renders the default fallback when the scene suspends", () => {
    // A lazy component that never resolves — forces Suspense to show the fallback.
    // React Suspense uses thrown Promises as a suspense signal; eslint's
    // only-throw-error doesn't understand this idiom, so we suppress locally.
    const Pending = (): ReactNode => {
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw new Promise<void>(() => undefined);
    };
    render(<SceneCanvas scene={<Pending />} />);
    expect(screen.getByTestId("scene-fallback")).toBeInTheDocument();
  });

  it("renders a custom fallback when provided and scene suspends", () => {
    const Pending = (): ReactNode => {
      // eslint-disable-next-line @typescript-eslint/only-throw-error
      throw new Promise<void>(() => undefined);
    };
    render(
      <SceneCanvas
        scene={<Pending />}
        fallback={<div data-testid="custom-fallback">custom</div>}
      />,
    );
    expect(screen.getByTestId("custom-fallback")).toBeInTheDocument();
  });

  it("renders the scene inside the mocked Canvas when it does not suspend", () => {
    render(<SceneCanvas scene={<mesh data-testid="ready-scene" />} />);
    expect(screen.getByTestId("canvas")).toBeInTheDocument();
  });
});

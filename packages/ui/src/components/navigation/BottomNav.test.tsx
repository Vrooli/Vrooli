import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BottomNav } from "./BottomNav.js";

// Mock keyboard state
let mockKeyboardOpen = false;

// Mock dependencies
vi.mock("../../contexts/session.js", () => ({
    SessionContext: {
        Provider: ({ children }: { children: React.ReactNode }) => children,
        Consumer: ({ children }: { children: (value: any) => React.ReactNode }) => children({}),
    },
}));

vi.mock("../../hooks/useKeyboardOpen.js", () => ({
    useKeyboardOpen: () => mockKeyboardOpen,
}));

vi.mock("../../route/router.js", () => ({
    useLocation: () => [{ pathname: "/" }, vi.fn()],
}));

vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key,
    }),
}));

describe("BottomNav", () => {
    it("renders navigation items for logged out users", () => {
        const { container } = render(<BottomNav />);
        const nav = container.querySelector("nav");
        expect(nav).toBeTruthy();
        
        // Should have Home, Search, About, Pricing, and Login for logged out users
        const links = container.querySelectorAll("a");
        expect(links).toHaveLength(5);
    });

    it("applies correct Tailwind classes", () => {
        const { container } = render(<BottomNav />);
        const nav = container.querySelector("nav");
        
        expect(nav?.className).toContain("tw-fixed");
        expect(nav?.className).toContain("tw-bottom-0");
        expect(nav?.className).toContain("tw-z-50");
        expect(nav?.className).toContain("md:tw-hidden");
    });

    it("hides when keyboard is open", () => {
        // Set keyboard to open
        mockKeyboardOpen = true;
        
        const { container } = render(<BottomNav />);
        const nav = container.querySelector("nav");
        expect(nav).toBeFalsy();
        
        // Reset keyboard state
        mockKeyboardOpen = false;
    });
});
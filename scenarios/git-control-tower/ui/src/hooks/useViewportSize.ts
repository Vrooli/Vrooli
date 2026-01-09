import { useState, useEffect, useCallback } from "react";

export type Breakpoint = "mobile" | "tablet" | "desktop";

export interface ViewportSize {
  width: number;
  height: number;
  breakpoint: Breakpoint;
  isMobile: boolean;
  isTablet: boolean;
  isDesktop: boolean;
  isTouchDevice: boolean;
}

// Tailwind breakpoints: sm=640, md=768, lg=1024
const MOBILE_MAX = 640;
const TABLET_MAX = 1024;

function getBreakpoint(width: number): Breakpoint {
  if (width < MOBILE_MAX) return "mobile";
  if (width < TABLET_MAX) return "tablet";
  return "desktop";
}

function detectTouchDevice(): boolean {
  if (typeof window === "undefined") return false;
  return (
    "ontouchstart" in window ||
    navigator.maxTouchPoints > 0 ||
    // @ts-expect-error - msMaxTouchPoints is IE-specific
    navigator.msMaxTouchPoints > 0
  );
}

function getViewportSize(): ViewportSize {
  if (typeof window === "undefined") {
    return {
      width: 1024,
      height: 768,
      breakpoint: "desktop",
      isMobile: false,
      isTablet: false,
      isDesktop: true,
      isTouchDevice: false
    };
  }

  const width = window.innerWidth;
  const height = window.innerHeight;
  const breakpoint = getBreakpoint(width);

  return {
    width,
    height,
    breakpoint,
    isMobile: breakpoint === "mobile",
    isTablet: breakpoint === "tablet",
    isDesktop: breakpoint === "desktop",
    isTouchDevice: detectTouchDevice()
  };
}

export function useViewportSize(): ViewportSize {
  const [size, setSize] = useState<ViewportSize>(getViewportSize);

  const handleResize = useCallback(() => {
    const newSize = getViewportSize();
    setSize((prev) => {
      // Only update if values actually changed to prevent unnecessary rerenders
      if (
        prev.width === newSize.width &&
        prev.height === newSize.height &&
        prev.breakpoint === newSize.breakpoint
      ) {
        return prev;
      }
      return newSize;
    });
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") return;

    // Update on mount in case SSR values differ
    handleResize();

    window.addEventListener("resize", handleResize);
    window.addEventListener("orientationchange", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      window.removeEventListener("orientationchange", handleResize);
    };
  }, [handleResize]);

  return size;
}

// Hook for just checking if mobile (lighter weight)
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.innerWidth < MOBILE_MAX;
  });

  useEffect(() => {
    if (typeof window === "undefined") return;

    const handleResize = () => {
      setIsMobile(window.innerWidth < MOBILE_MAX);
    };

    handleResize();
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  return isMobile;
}

// Hook for tablet and below (mobile + tablet)
export function useIsMobileOrTablet(): boolean {
  const [isMobileOrTablet, setIsMobileOrTablet] = useState(() => {
    if (typeof window === "undefined") return false;
    return window.innerWidth < TABLET_MAX;
  });

  useEffect(() => {
    if (typeof window === "undefined") return;

    const handleResize = () => {
      setIsMobileOrTablet(window.innerWidth < TABLET_MAX);
    };

    handleResize();
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  return isMobileOrTablet;
}

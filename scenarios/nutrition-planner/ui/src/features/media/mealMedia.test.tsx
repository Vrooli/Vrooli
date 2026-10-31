import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { MealMediaFrame, type MealMediaRecord } from "./mealMedia";

const approved: MealMediaRecord = {
  id: "owner-reviewed-meal-a",
  src: "/assets/meal-media/owner-reviewed-meal-a.webp",
  contentHash: `sha256:${"a".repeat(64)}`,
  source: { kind: "owned", reference: "owner-media-record-1", rights: "owner-approved-use" },
  approval: { status: "approved", reviewer: "owner-review-record-1", reviewedAt: "2026-10-01" },
  compatibility: { recipeId: "recipe-1", recipeRevision: 3, appearance: "any" },
  crop: { focalPoint: { x: 0.5, y: 0.5 }, bounds: { x: 0, y: 0, width: 1, height: 1 } },
};

const surfaces = [
  { name: "Today", className: "today-hero-scene", fallback: "Meal photo unavailable" },
  { name: "Explore", className: "explore-card-art", fallback: "Recipe photo unavailable" },
];

describe.each(surfaces)("$name meal imagery", ({ className, fallback }) => {
  it("shows only fully approved compatible local media", () => {
    renderWithProviders(<MealMediaFrame className={className} recipeId="recipe-1" recipeRevision={3} label="Lentil bowl" fallbackText={fallback} records={[approved]} />);
    expect(screen.getByRole("img", { name: "Lentil bowl" }).querySelector("img")?.getAttribute("src")).toBe(approved.src);
  });

  it("keeps the honest fallback when no asset exists", () => {
    const { container } = renderWithProviders(<MealMediaFrame className={className} recipeId="recipe-1" recipeRevision={3} label="Lentil bowl" fallbackText={fallback} records={[]} />);
    expect(screen.getByText(fallback)).toBeTruthy();
    expect(container.querySelector("img")).toBeNull();
  });

  it("rejects incomplete or incompatible metadata", () => {
    const incomplete = { ...approved, approval: { status: "pending" as "approved", reviewer: "", reviewedAt: "" } };
    const incompatible = { ...approved, compatibility: { ...approved.compatibility, recipeRevision: 4 } };
    renderWithProviders(<MealMediaFrame className={className} recipeId="recipe-1" recipeRevision={3} label="Lentil bowl" fallbackText={fallback} records={[incomplete, incompatible]} />);
    expect(screen.getByText(fallback)).toBeTruthy();
    expect(document.querySelector("img")).toBeNull();
  });

  it("falls back after a local image load failure without changing the frame", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    const { container } = renderWithProviders(<MealMediaFrame className={className} recipeId="recipe-1" recipeRevision={3} label="Lentil bowl" fallbackText={fallback} records={[approved]} />);
    const frame = container.firstElementChild;
    expect(frame?.classList.contains(className)).toBe(true);
    const image = container.querySelector("img");
    expect(image?.getAttribute("src")).toMatch(/^\/assets\/meal-media\//);
    fireEvent.error(image!);
    expect(screen.getByText(fallback)).toBeTruthy();
    expect(container.firstElementChild?.classList.contains(className)).toBe(true);
    expect(container.querySelector("img")).toBeNull();
    expect(fetchSpy).not.toHaveBeenCalled();
    fetchSpy.mockRestore();
  });
});

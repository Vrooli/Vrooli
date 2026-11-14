/**
 * Compile-only contract for the typed translation boundary.
 * Included by tsconfig so key safety is checked alongside the application.
 */
import type { TFunction } from "i18next";
import { strings } from "./strings";

declare const t: TFunction<"translation">;

// Valid generated registry keys remain accepted by the strict t() signature.
t(strings.pages.history.today);

// Hand-written keys must receive the same compile-time protection.
// @ts-expect-error unknown translation key must stay rejected
t("pages.history.todai");

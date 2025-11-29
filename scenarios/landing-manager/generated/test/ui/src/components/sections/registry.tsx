/**
 * Section Registry - Single source of truth for section components
 *
 * AGENT INSTRUCTIONS:
 * To add a new section, simply:
 * 1. Create your component file: {SectionName}Section.tsx
 * 2. Import and add to SECTION_REGISTRY below
 * 3. That's it! Types and rendering are handled automatically.
 *
 * The registry pattern eliminates the need to:
 * - Update switch statements
 * - Manually sync TypeScript unions
 * - Update multiple files
 */
import type { ComponentType } from 'react';
import { HeroSection } from './HeroSection';
import { FeaturesSection } from './FeaturesSection';
import { PricingSection } from './PricingSection';
import { CTASection } from './CTASection';
import { TestimonialsSection } from './TestimonialsSection';
import { FAQSection } from './FAQSection';
import { FooterSection } from './FooterSection';
import { VideoSection } from './VideoSection';

/**
 * Props interface for all section components
 * Every section receives its content as a flexible Record
 */
export interface SectionProps {
  content: Record<string, unknown>;
}

/**
 * Registry entry for a section type
 */
interface SectionRegistryEntry {
  /** The React component that renders this section */
  component: ComponentType<SectionProps>;
  /** Human-readable name for the section */
  name: string;
  /** Category for grouping (above-fold, conversion, etc.) */
  category: string;
  /** Suggested default order position */
  defaultOrder: number;
}

/**
 * SECTION_REGISTRY - Add new sections here
 *
 * This is the ONLY place you need to register a new section component.
 * The section type (key) must match:
 * - The `section_type` in the database
 * - The `id` in api/templates/sections/_index.json
 *
 * @example
 * // To add a new section:
 * 'my-section': {
 *   component: MySectionSection,
 *   name: 'My Section',
 *   category: 'value-proposition',
 *   defaultOrder: 50,
 * },
 */
export const SECTION_REGISTRY = {
  hero: {
    component: HeroSection,
    name: 'Hero Section',
    category: 'above-fold',
    defaultOrder: 1,
  },
  features: {
    component: FeaturesSection,
    name: 'Features Section',
    category: 'value-proposition',
    defaultOrder: 2,
  },
  video: {
    component: VideoSection,
    name: 'Video Section',
    category: 'engagement',
    defaultOrder: 4,
  },
  pricing: {
    component: PricingSection,
    name: 'Pricing Section',
    category: 'conversion',
    defaultOrder: 5,
  },
  testimonials: {
    component: TestimonialsSection,
    name: 'Testimonials Section',
    category: 'social-proof',
    defaultOrder: 6,
  },
  faq: {
    component: FAQSection,
    name: 'FAQ Section',
    category: 'trust-building',
    defaultOrder: 8,
  },
  cta: {
    component: CTASection,
    name: 'CTA Section',
    category: 'conversion',
    defaultOrder: 11,
  },
  footer: {
    component: FooterSection,
    name: 'Footer Section',
    category: 'navigation',
    defaultOrder: 99,
  },
} as const satisfies Record<string, SectionRegistryEntry>;

/**
 * Derived SectionType from registry keys
 * This type updates automatically when you add/remove sections from SECTION_REGISTRY
 */
export type SectionType = keyof typeof SECTION_REGISTRY;

/**
 * Array of all valid section types (useful for validation)
 */
export const SECTION_TYPES = Object.keys(SECTION_REGISTRY) as SectionType[];

/**
 * Check if a string is a valid section type
 */
export function isValidSectionType(type: string): type is SectionType {
  return type in SECTION_REGISTRY;
}

/**
 * Get the component for a section type
 * Returns undefined if the section type is not registered
 */
export function getSectionComponent(
  type: string
): ComponentType<SectionProps> | undefined {
  if (isValidSectionType(type)) {
    return SECTION_REGISTRY[type].component;
  }
  return undefined;
}

/**
 * Get registry metadata for a section type
 */
export function getSectionMeta(type: SectionType): SectionRegistryEntry {
  return SECTION_REGISTRY[type];
}

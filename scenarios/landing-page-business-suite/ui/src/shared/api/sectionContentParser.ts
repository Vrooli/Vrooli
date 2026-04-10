import { z } from 'zod';
import {
  SectionContentSchemas,
  HeroContentSchema,
  FeaturesContentSchema,
  PricingContentSchema,
  CTAContentSchema,
  TestimonialsContentSchema,
  FAQContentSchema,
  FooterContentSchema,
  VideoContentSchema,
  DownloadContentSchema,
  GenericSectionContentSchema,
  type HeroContent,
  type FeaturesContent,
  type PricingContent,
  type CTAContent,
  type TestimonialsContent,
  type FAQContent,
  type FooterContent,
  type VideoContent,
  type DownloadContent,
} from './schemas';
import { parseOrNull, safeParse, type ParseResult } from './safeParse';

/**
 * Type-safe section content parsing.
 *
 * This module provides utilities for validating section content
 * against the expected schema for each section type.
 */

export type SectionContentMap = {
  hero: HeroContent;
  features: FeaturesContent;
  pricing: PricingContent;
  cta: CTAContent;
  testimonials: TestimonialsContent;
  faq: FAQContent;
  footer: FooterContent;
  video: VideoContent;
  downloads: DownloadContent;
};

type SectionType = keyof SectionContentMap;

/**
 * Get the Zod schema for a given section type.
 */
function getSchemaForType(sectionType: string): z.ZodSchema {
  const schemas: Record<string, z.ZodSchema> = {
    hero: HeroContentSchema,
    features: FeaturesContentSchema,
    pricing: PricingContentSchema,
    cta: CTAContentSchema,
    testimonials: TestimonialsContentSchema,
    faq: FAQContentSchema,
    footer: FooterContentSchema,
    video: VideoContentSchema,
    downloads: DownloadContentSchema,
  };
  return schemas[sectionType] ?? GenericSectionContentSchema;
}

/**
 * Parse section content with full result (success/failure information).
 *
 * @param sectionType - The type of section (hero, features, etc.)
 * @param content - The raw content object from the API
 * @returns ParseResult with validated data or error details
 *
 * @example
 * const result = parseSectionContent('hero', section.content);
 * if (result.success) {
 *   // result.data is typed as HeroContent
 * } else {
 *   // Show error UI
 * }
 */
export function parseSectionContent<T extends SectionType>(
  sectionType: T,
  content: unknown
): ParseResult<SectionContentMap[T]> {
  const schema = getSchemaForType(sectionType);
  return safeParse(schema, content, `${sectionType}SectionContent`) as ParseResult<SectionContentMap[T]>;
}

/**
 * Parse section content, returning null on failure.
 * Logs validation errors to console.
 *
 * @param sectionType - The type of section
 * @param content - The raw content object
 * @returns Validated content or null
 *
 * @example
 * const heroContent = parseSectionContentOrNull('hero', section.content);
 * if (heroContent) {
 *   return <HeroSection content={heroContent} />;
 * } else {
 *   return <SectionErrorFallback />;
 * }
 */
export function parseSectionContentOrNull<T extends SectionType>(
  sectionType: T,
  content: unknown
): SectionContentMap[T] | null {
  const schema = getSchemaForType(sectionType);
  return parseOrNull(schema, content, `${sectionType}SectionContent`) as SectionContentMap[T] | null;
}

/**
 * Parse section content with a default fallback.
 * Returns the default if validation fails.
 *
 * @param sectionType - The type of section
 * @param content - The raw content object
 * @param defaultContent - Fallback content if validation fails
 * @returns Validated content or the default
 *
 * @example
 * const heroContent = parseSectionContentWithDefault('hero', section.content, {
 *   title: 'Welcome',
 *   subtitle: 'Default subtitle',
 * });
 * return <HeroSection content={heroContent} />;
 */
export function parseSectionContentWithDefault<T extends SectionType>(
  sectionType: T,
  content: unknown,
  defaultContent: SectionContentMap[T]
): SectionContentMap[T] {
  const result = parseSectionContentOrNull(sectionType, content);
  return result ?? defaultContent;
}

/**
 * Type guard to check if a section type is known.
 */
export function isKnownSectionType(sectionType: string): sectionType is SectionType {
  return sectionType in SectionContentSchemas;
}

/**
 * Parse any section content with loose typing.
 * Use when the section type is dynamic/unknown at compile time.
 *
 * @param sectionType - The type of section as a string
 * @param content - The raw content object
 * @returns ParseResult with data or error
 */
export function parseDynamicSectionContent(
  sectionType: string,
  content: unknown
): ParseResult<Record<string, unknown>> {
  const schema = getSchemaForType(sectionType);
  return safeParse(schema, content, `${sectionType}SectionContent`);
}

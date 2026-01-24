/**
 * Zod schema exports for API response validation.
 *
 * These schemas provide runtime type validation for API responses,
 * ensuring data matches expected types even when TypeScript types
 * are erased at runtime.
 */

// Common schemas and types
export * from './common.schema';

// Landing page schemas and types
export * from './landing.schema';

// Billing schemas and types
export * from './billing.schema';

// Variant schemas and types
export * from './variants.schema';

// Downloads schemas and types
export * from './downloads.schema';

// Auth schemas and types
export * from './auth.schema';

// Section content schemas and types (SectionType exported as SectionContentType to avoid collision)
export {
  HeroContentSchema,
  FeatureItemSchema,
  FeaturesContentSchema,
  PricingContentSchema,
  CTAContentSchema,
  TestimonialItemSchema,
  TestimonialsContentSchema,
  FAQItemSchema,
  FAQContentSchema,
  FooterLinkSchema,
  FooterLinkGroupSchema,
  FooterContentSchema,
  VideoContentSchema,
  DownloadContentSchema,
  GenericSectionContentSchema,
  SectionContentSchemas,
  SectionTypeEnumSchema,
  ContentSectionSchema,
  SectionsListResponseSchema,
  UpdateSectionResponseSchema,
  type SectionType as SectionContentType,
  type HeroContent,
  type FeatureItem,
  type FeaturesContent,
  type PricingContent,
  type CTAContent,
  type TestimonialItem,
  type TestimonialsContent,
  type FAQItem,
  type FAQContent,
  type FooterLink,
  type FooterLinkGroup,
  type FooterContent,
  type VideoContent,
  type DownloadContent,
  type ContentSection as ContentSectionType,
  type SectionsListResponse,
  type UpdateSectionResponse,
} from './sections.schema';

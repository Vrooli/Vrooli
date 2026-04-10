import { z } from 'zod';

/**
 * Section content Zod schemas for runtime validation.
 * These schemas define the expected shape of section.content for each section type.
 */

// Hero section content schema
export const HeroContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  cta_text: z.string().optional(),
  cta_url: z.string().optional(),
  image_url: z.string().optional(),
  background_style: z.enum(['gradient', 'solid', 'image']).optional(),
  secondary_cta_text: z.string().optional(),
  secondary_cta_url: z.string().optional(),
});

// Feature item schema
export const FeatureItemSchema = z.object({
  title: z.string(),
  description: z.string(),
  bullets: z.array(z.string()).optional(),
  icon: z.string().optional(),
});

// Features section content schema
export const FeaturesContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  features: z.array(FeatureItemSchema).optional(),
});

// Pricing section content schema
export const PricingContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  show_monthly: z.boolean().optional(),
  show_yearly: z.boolean().optional(),
  highlight_plan: z.string().optional(),
});

// CTA section content schema
export const CTAContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  cta_text: z.string().optional(),
  cta_url: z.string().optional(),
  secondary_cta_text: z.string().optional(),
  secondary_cta_url: z.string().optional(),
  background_style: z.enum(['gradient', 'solid', 'accent']).optional(),
});

// Testimonial item schema
export const TestimonialItemSchema = z.object({
  quote: z.string(),
  author: z.string(),
  role: z.string().optional(),
  company: z.string().optional(),
  avatar_url: z.string().optional(),
  rating: z.number().min(1).max(5).optional(),
});

// Testimonials section content schema
export const TestimonialsContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  testimonials: z.array(TestimonialItemSchema).optional(),
});

// FAQ item schema
export const FAQItemSchema = z.object({
  question: z.string(),
  answer: z.string(),
});

// FAQ section content schema
export const FAQContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  faqs: z.array(FAQItemSchema).optional(),
});

// Footer link schema
export const FooterLinkSchema = z.object({
  label: z.string(),
  url: z.string(),
});

// Footer link group schema
export const FooterLinkGroupSchema = z.object({
  title: z.string(),
  links: z.array(FooterLinkSchema),
});

// Footer section content schema
export const FooterContentSchema = z.object({
  copyright: z.string().optional(),
  links: z.array(FooterLinkGroupSchema).optional(),
  social_links: z.array(z.object({
    platform: z.string(),
    url: z.string(),
    icon: z.string().optional(),
  })).optional(),
  logo_url: z.string().optional(),
  tagline: z.string().optional(),
});

// Video section content schema
export const VideoContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  video_url: z.string().optional(),
  video_embed_html: z.string().optional(),
  thumbnail_url: z.string().optional(),
  autoplay: z.boolean().optional(),
  loop: z.boolean().optional(),
  muted: z.boolean().optional(),
});

// Download section content schema
export const DownloadContentSchema = z.object({
  title: z.string().optional(),
  subtitle: z.string().optional(),
  description: z.string().optional(),
  show_platforms: z.array(z.string()).optional(),
  show_storefronts: z.boolean().optional(),
});

// Generic section content (for unknown section types)
export const GenericSectionContentSchema = z.record(z.string(), z.unknown());

// Section type enum
export const SectionTypeEnumSchema = z.enum([
  'hero',
  'features',
  'pricing',
  'cta',
  'testimonials',
  'faq',
  'footer',
  'video',
  'downloads',
]);

// Content section schema (full section with metadata)
export const ContentSectionSchema = z.object({
  id: z.number(),
  variant_id: z.number(),
  section_type: SectionTypeEnumSchema,
  content: GenericSectionContentSchema,
  order: z.number(),
  enabled: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
});

// Sections list response schema
export const SectionsListResponseSchema = z.object({
  sections: z.array(ContentSectionSchema),
});

// Update section response schema
export const UpdateSectionResponseSchema = z.object({
  success: z.boolean(),
  message: z.string().optional(),
});

// Map of section type to schema
export const SectionContentSchemas = {
  hero: HeroContentSchema,
  features: FeaturesContentSchema,
  pricing: PricingContentSchema,
  cta: CTAContentSchema,
  testimonials: TestimonialsContentSchema,
  faq: FAQContentSchema,
  footer: FooterContentSchema,
  video: VideoContentSchema,
  downloads: DownloadContentSchema,
} as const;

export type SectionType = keyof typeof SectionContentSchemas;

// Type exports
export type HeroContent = z.infer<typeof HeroContentSchema>;
export type FeatureItem = z.infer<typeof FeatureItemSchema>;
export type FeaturesContent = z.infer<typeof FeaturesContentSchema>;
export type PricingContent = z.infer<typeof PricingContentSchema>;
export type CTAContent = z.infer<typeof CTAContentSchema>;
export type TestimonialItem = z.infer<typeof TestimonialItemSchema>;
export type TestimonialsContent = z.infer<typeof TestimonialsContentSchema>;
export type FAQItem = z.infer<typeof FAQItemSchema>;
export type FAQContent = z.infer<typeof FAQContentSchema>;
export type FooterLink = z.infer<typeof FooterLinkSchema>;
export type FooterLinkGroup = z.infer<typeof FooterLinkGroupSchema>;
export type FooterContent = z.infer<typeof FooterContentSchema>;
export type VideoContent = z.infer<typeof VideoContentSchema>;
export type DownloadContent = z.infer<typeof DownloadContentSchema>;
export type ContentSection = z.infer<typeof ContentSectionSchema>;
export type SectionsListResponse = z.infer<typeof SectionsListResponseSchema>;
export type UpdateSectionResponse = z.infer<typeof UpdateSectionResponseSchema>;

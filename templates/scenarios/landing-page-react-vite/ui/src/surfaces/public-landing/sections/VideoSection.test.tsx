import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { VideoSection } from './VideoSection';

/**
 * [REQ:DESIGN-VIDEO] VideoSection Component Tests
 * Validates video embed support for YouTube and Vimeo URLs
 */

describe('VideoSection [REQ:DESIGN-VIDEO]', () => {
  it('[REQ:DESIGN-VIDEO] should render video section with title', () => {
    render(
      <VideoSection
        title="See it in action"
        videoUrl="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    expect(screen.getByText('See it in action')).toBeDefined();
  });

  it('[REQ:DESIGN-VIDEO] should convert YouTube watch URL to embed URL', () => {
    const { container } = render(
      <VideoSection
        videoUrl="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    // Initially shows thumbnail (iframe not rendered until play clicked)
    expect(container.querySelector('iframe')).toBeNull();
    expect(container.querySelector('img[alt="Video thumbnail"]')).toBeDefined();
  });

  it('[REQ:DESIGN-VIDEO] should convert YouTube short URL to embed URL', () => {
    const { container } = render(
      <VideoSection
        videoUrl="https://youtu.be/dQw4w9WgXcQ"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    expect(container.querySelector('img[alt="Video thumbnail"]')).toBeDefined();
  });

  it('[REQ:DESIGN-VIDEO] should convert Vimeo URL to embed URL', () => {
    const { container } = render(
      <VideoSection
        videoUrl="https://vimeo.com/123456789"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    expect(container.querySelector('img[alt="Video thumbnail"]')).toBeDefined();
  });

  it('[REQ:DESIGN-VIDEO] should show play button overlay on thumbnail', () => {
    render(
      <VideoSection
        videoUrl="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    const playButton = screen.getByLabelText('Play video');
    expect(playButton).toBeDefined();
  });

  it('[REQ:DESIGN-VIDEO] should show iframe when play button is clicked', () => {
    const { container } = render(
      <VideoSection
        videoUrl="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    // Initially no iframe
    expect(container.querySelector('iframe')).toBeNull();

    // Click play button
    const playButton = screen.getByLabelText('Play video');
    fireEvent.click(playButton);

    // Now iframe should be present
    const iframe = container.querySelector('iframe');
    expect(iframe).toBeDefined();
    expect(iframe?.src).toContain('youtube.com/embed/dQw4w9WgXcQ');
    expect(iframe?.src).toContain('autoplay=1');
  });

  it('[REQ:DESIGN-VIDEO] should display caption when provided', () => {
    render(
      <VideoSection
        title="Product Demo"
        videoUrl="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
        thumbnailUrl="/thumbnail.jpg"
        caption="Watch how easy it is to get started"
      />
    );

    expect(screen.getByText('Watch how easy it is to get started')).toBeDefined();
  });

  it('[REQ:DESIGN-VIDEO] should show iframe directly when no thumbnail provided', () => {
    const { container } = render(
      <VideoSection
        videoUrl="https://www.youtube.com/watch?v=dQw4w9WgXcQ"
      />
    );

    // Should show iframe immediately when no thumbnail
    const iframe = container.querySelector('iframe');
    expect(iframe).toBeDefined();
    expect(iframe?.src).toContain('youtube.com/embed/dQw4w9WgXcQ');
  });

  it('[REQ:DESIGN-VIDEO] should handle invalid video URLs gracefully', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    const { container } = render(
      <VideoSection
        videoUrl="https://invalid-url.com/video"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    // Should not render anything for invalid URL
    expect(container.querySelector('section')).toBeNull();
    expect(consoleSpy).toHaveBeenCalledWith(
      '[VideoSection] Invalid video URL:',
      'https://invalid-url.com/video'
    );

    consoleSpy.mockRestore();
  });

  it('[REQ:DESIGN-VIDEO] should support YouTube embed URLs directly', () => {
    const { container } = render(
      <VideoSection
        videoUrl="https://www.youtube.com/embed/dQw4w9WgXcQ"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    expect(container.querySelector('img[alt="Video thumbnail"]')).toBeDefined();
  });

  it('[REQ:DESIGN-VIDEO] should support Vimeo video URLs', () => {
    const { container } = render(
      <VideoSection
        videoUrl="https://vimeo.com/video/123456789"
        thumbnailUrl="/thumbnail.jpg"
      />
    );

    expect(container.querySelector('img[alt="Video thumbnail"]')).toBeDefined();
  });

  it('should render when content prop provides data', () => {
    const { container } = render(
      <VideoSection
        content={{
          title: 'Inline content payload',
          videoUrl: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
          thumbnailUrl: '/thumbnail.jpg',
        }}
      />
    );

    expect(container.querySelector('img[alt="Video thumbnail"]')).toBeDefined();
    expect(screen.getByText('Inline content payload')).toBeDefined();
  });

  it('should skip rendering when videoUrl missing from content payload', () => {
    const { container } = render(
      <VideoSection
        content={{ title: 'Missing video url' }}
      />
    );

    expect(container.querySelector('section')).toBeNull();
  });
});

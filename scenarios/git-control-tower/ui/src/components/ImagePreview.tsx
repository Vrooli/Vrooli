interface ImagePreviewProps {
  src: string; // Base64 data URL or file path
  alt: string;
}

export function ImagePreview({ src, alt }: ImagePreviewProps) {
  return (
    <div
      className="flex items-center justify-center p-8 bg-slate-900/50 min-h-[200px]"
      data-testid="image-preview"
    >
      <img
        src={src}
        alt={alt}
        className="max-w-full max-h-[70vh] object-contain rounded-lg shadow-lg"
      />
    </div>
  );
}

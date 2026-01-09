import { Globe, Twitter, Facebook } from 'lucide-react';

export interface SEOPreviewProps {
  title?: string;
  description?: string;
  url?: string;
  ogImage?: string;
  siteName?: string;
  favicon?: string;
  twitterCard?: 'summary' | 'summary_large_image';
}

export function SEOPreview({
  title = 'Page Title',
  description = 'Page description will appear here. Make sure to keep it under 160 characters for optimal display in search results.',
  url = 'https://example.com',
  ogImage,
  siteName,
  favicon,
  twitterCard = 'summary_large_image',
}: SEOPreviewProps) {
  const displayUrl = url.replace(/^https?:\/\//, '').replace(/\/$/, '');
  const truncatedDescription = description.length > 160
    ? description.substring(0, 157) + '...'
    : description;

  return (
    <div className="space-y-6">
      {/* Google Search Preview */}
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <Globe className="h-4 w-4" />
          Google Search Preview
        </div>
        <div className="rounded-lg border border-white/10 bg-white p-4">
          <div className="flex items-center gap-2 mb-1">
            {favicon ? (
              <img src={favicon} alt="" className="h-7 w-7 rounded-full" />
            ) : (
              <div className="h-7 w-7 rounded-full bg-slate-200 flex items-center justify-center">
                <Globe className="h-4 w-4 text-slate-400" />
              </div>
            )}
            <div>
              <div className="text-sm text-slate-600">{siteName || displayUrl}</div>
              <div className="text-xs text-slate-400">{displayUrl}</div>
            </div>
          </div>
          <h3 className="text-xl text-blue-600 hover:underline cursor-pointer mb-1">
            {title}
          </h3>
          <p className="text-sm text-slate-600 line-clamp-2">
            {truncatedDescription}
          </p>
        </div>
      </div>

      {/* Twitter/X Card Preview */}
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <Twitter className="h-4 w-4" />
          Twitter/X Card Preview
        </div>
        <div className="rounded-xl border border-slate-700 bg-slate-800 overflow-hidden max-w-[500px]">
          {twitterCard === 'summary_large_image' && ogImage && (
            <div className="aspect-[1.91/1] bg-slate-700">
              <img
                src={ogImage}
                alt=""
                className="w-full h-full object-cover"
                onError={(e) => {
                  (e.target as HTMLImageElement).style.display = 'none';
                }}
              />
            </div>
          )}
          <div className="p-3 flex gap-3">
            {twitterCard === 'summary' && (
              <div className="flex-shrink-0">
                {ogImage ? (
                  <img
                    src={ogImage}
                    alt=""
                    className="w-32 h-32 object-cover rounded"
                    onError={(e) => {
                      (e.target as HTMLImageElement).src = '';
                    }}
                  />
                ) : (
                  <div className="w-32 h-32 bg-slate-700 rounded flex items-center justify-center">
                    <Globe className="h-8 w-8 text-slate-500" />
                  </div>
                )}
              </div>
            )}
            <div className="flex-1 min-w-0">
              <div className="text-sm text-slate-400">{displayUrl}</div>
              <h4 className="font-medium text-white truncate">{title}</h4>
              <p className="text-sm text-slate-400 line-clamp-2">{truncatedDescription}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Facebook/Open Graph Preview */}
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <Facebook className="h-4 w-4" />
          Facebook Preview
        </div>
        <div className="rounded-lg border border-slate-300 bg-white overflow-hidden max-w-[500px]">
          {ogImage && (
            <div className="aspect-[1.91/1] bg-slate-100">
              <img
                src={ogImage}
                alt=""
                className="w-full h-full object-cover"
                onError={(e) => {
                  (e.target as HTMLImageElement).style.display = 'none';
                }}
              />
            </div>
          )}
          <div className="p-3 bg-slate-50">
            <div className="text-xs text-slate-500 uppercase tracking-wide">{displayUrl}</div>
            <h4 className="font-semibold text-slate-900 line-clamp-2 mt-1">{title}</h4>
            <p className="text-sm text-slate-600 line-clamp-2 mt-1">{truncatedDescription}</p>
          </div>
        </div>
      </div>
    </div>
  );
}

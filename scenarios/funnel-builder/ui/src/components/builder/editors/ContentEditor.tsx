import { ContentStep } from '../../../types'
import { Image, Video } from 'lucide-react'

interface ContentEditorProps {
  content: ContentStep
  onChange: (content: ContentStep) => void
}

const ContentEditor = ({ content, onChange }: ContentEditorProps) => {
  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Content Step</h3>

      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Headline
          </label>
          <input
            type="text"
            value={content.headline || ''}
            onChange={(e) => onChange({ ...content, headline: e.target.value })}
            className="input w-full"
            placeholder="Enter a compelling headline"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Body Content
          </label>
          <textarea
            value={content.body || ''}
            onChange={(e) => onChange({ ...content, body: e.target.value })}
            className="input w-full h-32 resize-none"
            placeholder="Enter your content here. Support for basic markdown formatting."
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Media (Optional)
          </label>
          <div className="space-y-3">
            <div className="flex gap-2">
              <select
                value={content.media?.type || 'image'}
                onChange={(e) => onChange({
                  ...content,
                  media: {
                    ...content.media,
                    type: e.target.value as 'image' | 'video',
                    url: content.media?.url || ''
                  }
                })}
                className="input w-32"
              >
                <option value="image">Image</option>
                <option value="video">Video</option>
              </select>
              <input
                type="url"
                value={content.media?.url || ''}
                onChange={(e) => onChange({
                  ...content,
                  media: e.target.value ? {
                    ...content.media,
                    type: content.media?.type || 'image',
                    url: e.target.value
                  } : undefined
                })}
                className="input flex-1"
                placeholder="Enter media URL"
              />
            </div>
            {content.media?.url && (
              <div className="p-4 bg-gray-50 rounded-lg">
                <div className="flex items-center justify-center h-32 bg-white rounded border border-gray-200">
                  {content.media.type === 'image' ? (
                    <div className="text-center">
                      <Image className="w-12 h-12 mx-auto text-gray-400 mb-2" />
                      <p className="text-xs text-gray-500">Image Preview</p>
                    </div>
                  ) : (
                    <div className="text-center">
                      <Video className="w-12 h-12 mx-auto text-gray-400 mb-2" />
                      <p className="text-xs text-gray-500">Video Preview</p>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Button Text
          </label>
          <input
            type="text"
            value={content.buttonText || ''}
            onChange={(e) => onChange({ ...content, buttonText: e.target.value })}
            className="input w-full"
            placeholder="Continue / Next / Learn More"
          />
        </div>
      </div>
    </div>
  )
}

export default ContentEditor
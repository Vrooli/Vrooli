import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { LucideIcon } from 'lucide-react'
import { FileText, Users, Zap, TrendingUp, Gift, Calendar, ShoppingCart, Rocket } from 'lucide-react'
import { funnelTemplates, TemplateIcon } from '../data/funnelTemplates'

const iconMap: Record<TemplateIcon, LucideIcon> = {
  users: Users,
  rocket: Rocket,
  calendar: Calendar,
  zap: Zap,
  gift: Gift,
  shoppingCart: ShoppingCart,
  fileText: FileText,
  trendingUp: TrendingUp,
}

const Templates = () => {
  const navigate = useNavigate()
  const [activeCategory, setActiveCategory] = useState('All')

  const categories = useMemo(() => {
    const unique = new Set(funnelTemplates.map((template) => template.category))
    return ['All', ...Array.from(unique)]
  }, [])

  const filteredTemplates = useMemo(() => {
    if (activeCategory === 'All') {
      return funnelTemplates
    }
    return funnelTemplates.filter((template) => template.category === activeCategory)
  }, [activeCategory])

  const handleUseTemplate = (templateId: string) => {
    // Create new funnel from template
    navigate(`/builder/new?template=${templateId}`)
  }

  return (
    <div className="p-8">
      <div className="mb-8">
        <h2 className="text-3xl font-bold text-gray-900 mb-2">Funnel Templates</h2>
        <p className="text-gray-600">Start with a proven template and customize it to your needs</p>
      </div>

      <div className="mb-6">
        <div className="flex gap-2 flex-wrap">
          {categories.map((category) => (
            <button
              key={category}
              onClick={() => setActiveCategory(category)}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                category === activeCategory
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {category}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filteredTemplates.map((template) => {
          const Icon = iconMap[template.icon]
          const stepCount = template.funnel.steps.length

          return (
            <div
              key={template.id}
              className="card hover:shadow-lg transition-shadow cursor-pointer group"
              onClick={() => handleUseTemplate(template.id)}
            >
              <div className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className={`p-3 rounded-lg ${template.color}`}>
                    <Icon className="w-6 h-6 text-white" />
                  </div>
                  <span className="px-2 py-1 bg-gray-100 text-gray-600 text-xs rounded-full">
                    {template.category}
                  </span>
                </div>

                <h3 className="text-lg font-semibold text-gray-900 mb-2 group-hover:text-primary-600 transition-colors">
                  {template.name}
                </h3>

                <p className="text-sm text-gray-600 mb-4">
                  {template.description}
                </p>

                <div className="flex justify-between items-center pt-4 border-t border-gray-100">
                  <div className="flex gap-4 text-sm">
                    <span className="text-gray-500">
                      <span className="font-medium text-gray-700">{stepCount}</span> steps
                    </span>
                    <span className="text-gray-500">
                      <span className="font-medium text-green-600">{template.conversionRate}</span> conv.
                    </span>
                  </div>
                </div>
              </div>

              <div className="px-6 pb-4">
                <button className="btn btn-primary w-full opacity-0 group-hover:opacity-100 transition-opacity">
                  Use Template
                </button>
              </div>
            </div>
          )
        })}
      </div>

      <div className="mt-12 p-8 bg-gradient-to-r from-primary-50 to-blue-50 rounded-2xl">
        <div className="max-w-3xl">
          <h3 className="text-2xl font-bold text-gray-900 mb-3">
            Can't find what you're looking for?
          </h3>
          <p className="text-gray-600 mb-6">
            Start from scratch and build your own custom funnel with our intuitive drag-and-drop builder.
          </p>
          <button
            onClick={() => navigate('/builder')}
            className="btn btn-primary"
          >
            Create Custom Funnel
          </button>
        </div>
      </div>
    </div>
  )
}

export default Templates

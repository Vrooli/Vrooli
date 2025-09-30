// import { useParams } from 'react-router-dom' // TODO: Use when fetching specific funnel analytics
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts'
import { TrendingUp, Users, Target, Clock, ArrowUp, ArrowDown } from 'lucide-react'

const Analytics = () => {
  // const { id } = useParams() // TODO: Use when fetching specific funnel analytics

  // Mock data for demonstration
  const dailyData = [
    { date: 'Mon', views: 120, leads: 45, conversions: 12 },
    { date: 'Tue', views: 145, leads: 52, conversions: 15 },
    { date: 'Wed', views: 170, leads: 58, conversions: 18 },
    { date: 'Thu', views: 135, leads: 48, conversions: 14 },
    { date: 'Fri', views: 190, leads: 65, conversions: 22 },
    { date: 'Sat', views: 160, leads: 55, conversions: 17 },
    { date: 'Sun', views: 140, leads: 50, conversions: 15 },
  ]

  const dropOffData = [
    { step: 'Quiz', visitors: 1000, dropOff: 0 },
    { step: 'Form', visitors: 750, dropOff: 250 },
    { step: 'Content', visitors: 450, dropOff: 300 },
    { step: 'CTA', visitors: 320, dropOff: 130 },
    { step: 'Complete', visitors: 280, dropOff: 40 },
  ]

  const sourceData = [
    { name: 'Direct', value: 35, color: '#0ea5e9' },
    { name: 'Social Media', value: 30, color: '#8b5cf6' },
    { name: 'Email', value: 20, color: '#10b981' },
    { name: 'Referral', value: 15, color: '#f59e0b' },
  ]

  const stats = [
    {
      label: 'Total Views',
      value: '1,060',
      change: '+12.5%',
      trend: 'up',
      icon: Target,
      color: 'text-blue-600',
      bgColor: 'bg-blue-50'
    },
    {
      label: 'Total Leads',
      value: '373',
      change: '+8.2%',
      trend: 'up',
      icon: Users,
      color: 'text-green-600',
      bgColor: 'bg-green-50'
    },
    {
      label: 'Conversion Rate',
      value: '35.2%',
      change: '-2.1%',
      trend: 'down',
      icon: TrendingUp,
      color: 'text-purple-600',
      bgColor: 'bg-purple-50'
    },
    {
      label: 'Avg. Time',
      value: '2m 45s',
      change: '+0.3s',
      trend: 'up',
      icon: Clock,
      color: 'text-orange-600',
      bgColor: 'bg-orange-50'
    },
  ]

  return (
    <div className="p-8">
      <div className="mb-8">
        <h2 className="text-3xl font-bold text-gray-900 mb-2">Funnel Analytics</h2>
        <p className="text-gray-600">Track performance and optimize conversions</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {stats.map((stat) => (
          <div key={stat.label} className="card p-6">
            <div className="flex items-center justify-between mb-4">
              <div className={`p-3 rounded-lg ${stat.bgColor}`}>
                <stat.icon className={`w-6 h-6 ${stat.color}`} />
              </div>
              <div className={`flex items-center gap-1 text-sm ${
                stat.trend === 'up' ? 'text-green-600' : 'text-red-600'
              }`}>
                {stat.trend === 'up' ? (
                  <ArrowUp className="w-4 h-4" />
                ) : (
                  <ArrowDown className="w-4 h-4" />
                )}
                {stat.change}
              </div>
            </div>
            <p className="text-2xl font-bold text-gray-900 mb-1">{stat.value}</p>
            <p className="text-sm text-gray-600">{stat.label}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Daily Performance</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={dailyData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="date" stroke="#666" />
              <YAxis stroke="#666" />
              <Tooltip />
              <Line
                type="monotone"
                dataKey="views"
                stroke="#0ea5e9"
                strokeWidth={2}
                dot={{ fill: '#0ea5e9', r: 4 }}
              />
              <Line
                type="monotone"
                dataKey="leads"
                stroke="#10b981"
                strokeWidth={2}
                dot={{ fill: '#10b981', r: 4 }}
              />
              <Line
                type="monotone"
                dataKey="conversions"
                stroke="#8b5cf6"
                strokeWidth={2}
                dot={{ fill: '#8b5cf6', r: 4 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Funnel Drop-off</h3>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={dropOffData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="step" stroke="#666" />
              <YAxis stroke="#666" />
              <Tooltip />
              <Bar dataKey="visitors" fill="#0ea5e9" />
              <Bar dataKey="dropOff" fill="#ef4444" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Traffic Sources</h3>
          <ResponsiveContainer width="100%" height={250}>
            <PieChart>
              <Pie
                data={sourceData}
                cx="50%"
                cy="50%"
                innerRadius={60}
                outerRadius={80}
                paddingAngle={5}
                dataKey="value"
              >
                {sourceData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
          <div className="mt-4 space-y-2">
            {sourceData.map((source) => (
              <div key={source.name} className="flex justify-between items-center">
                <div className="flex items-center gap-2">
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: source.color }}
                  />
                  <span className="text-sm text-gray-600">{source.name}</span>
                </div>
                <span className="text-sm font-medium text-gray-900">{source.value}%</span>
              </div>
            ))}
          </div>
        </div>

        <div className="lg:col-span-2 card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Top Performing Steps</h3>
          <div className="space-y-3">
            {dropOffData.slice(0, 4).map((step, index) => {
              const conversionRate = index === 0 ? 100 : (step.visitors / dropOffData[0].visitors) * 100
              return (
                <div key={step.step} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-primary-100 text-primary-600 rounded-full flex items-center justify-center font-semibold text-sm">
                      {index + 1}
                    </div>
                    <div>
                      <p className="font-medium text-gray-900">{step.step}</p>
                      <p className="text-xs text-gray-500">{step.visitors} visitors</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold text-gray-900">{conversionRate.toFixed(1)}%</p>
                    <p className="text-xs text-gray-500">conversion</p>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}

export default Analytics
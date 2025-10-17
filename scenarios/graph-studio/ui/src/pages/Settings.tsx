import { useState } from 'react'
import type { ChangeEvent } from 'react'
import { Button } from '../components/ui/button.tsx'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../components/ui/card.tsx'
import { Input } from '../components/ui/input.tsx'
import { Label } from '../components/ui/label.tsx'
import { Select } from '../components/ui/select.tsx'
import { Switch } from '../components/ui/switch.tsx'
import { Settings2, RotateCcw } from 'lucide-react'

const SETTINGS_KEY = 'settings'

type ThemeMode = 'light' | 'dark' | 'auto'

interface StudioSettings {
  theme: ThemeMode
  autoSave: boolean
  autoValidate: boolean
  defaultFormat: 'svg' | 'png' | 'html'
  maxGraphSize: number
}

function Settings() {
  const [settings, setSettings] = useState<StudioSettings>(() => {
    const stored = localStorage.getItem(SETTINGS_KEY)
    if (stored) {
      try {
        return JSON.parse(stored) as StudioSettings
      } catch (error) {
        console.warn('[GraphStudio] Failed to parse stored settings', error)
      }
    }
    return {
      theme: 'light',
      autoSave: true,
      autoValidate: true,
      defaultFormat: 'svg',
      maxGraphSize: 10000,
    }
  })

  const handleChange = <Key extends keyof StudioSettings>(
    key: Key,
    value: StudioSettings[Key],
  ) => {
    setSettings((prev) => ({ ...prev, [key]: value }))
  }

  const handleSave = () => {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings))
    window.alert('Settings saved successfully')
  }

  const handleReset = () => {
    const defaults: StudioSettings = {
      theme: 'light',
      autoSave: true,
      autoValidate: true,
      defaultFormat: 'svg',
      maxGraphSize: 10000,
    }
    setSettings(defaults)
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(defaults))
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="space-y-2">
          <div className="inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            <Settings2 className="h-4 w-4" />
            Personalize your workspace
          </div>
          <h1 className="text-3xl font-semibold tracking-tight text-foreground">
            Studio preferences
          </h1>
          <p className="text-sm text-muted-foreground md:max-w-2xl">
            Configure editor defaults, automation behaviour, and performance guardrails to match your delivery workflow.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" className="gap-2" onClick={handleReset}>
            <RotateCcw className="h-4 w-4" />
            Reset
          </Button>
          <Button className="gap-2" onClick={handleSave}>
            Save changes
          </Button>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Appearance</CardTitle>
            <CardDescription>Fine-tune theming and interface density.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="theme">Theme mode</Label>
              <Select
                id="theme"
                value={settings.theme}
                onChange={(event: ChangeEvent<HTMLSelectElement>) =>
                  handleChange('theme', event.target.value as ThemeMode)
                }
              >
                <option value="light">Light</option>
                <option value="dark">Dark</option>
                <option value="auto">Auto</option>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="defaultFormat">Default export format</Label>
              <Select
                id="defaultFormat"
                value={settings.defaultFormat}
                onChange={(event: ChangeEvent<HTMLSelectElement>) =>
                  handleChange('defaultFormat', event.target.value as StudioSettings['defaultFormat'])
                }
              >
                <option value="svg">SVG</option>
                <option value="png">PNG</option>
                <option value="html">HTML</option>
              </Select>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Automation</CardTitle>
            <CardDescription>Guard against surprises with realtime validation.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between gap-4 rounded-lg border border-border/60 bg-muted/40 px-4 py-3">
              <div>
                <p className="text-sm font-medium text-foreground">Auto-save graphs</p>
                <p className="text-xs text-muted-foreground">
                  Persist drafts every few seconds so nothing gets lost.
                </p>
              </div>
              <Switch
                checked={settings.autoSave}
                onCheckedChange={(checked: boolean) => handleChange('autoSave', checked)}
              />
            </div>
            <div className="flex items-center justify-between gap-4 rounded-lg border border-border/60 bg-muted/40 px-4 py-3">
              <div>
                <p className="text-sm font-medium text-foreground">Auto-validate changes</p>
                <p className="text-xs text-muted-foreground">
                  Run structural checks after each change to catch regressions instantly.
                </p>
              </div>
              <Switch
                checked={settings.autoValidate}
                onCheckedChange={(checked: boolean) => handleChange('autoValidate', checked)}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="xl:col-span-2">
          <CardHeader>
            <CardTitle>Performance</CardTitle>
            <CardDescription>Keep graph rendering balanced against hardware constraints.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="maxGraphSize">Max graph size (nodes)</Label>
              <Input
                id="maxGraphSize"
                type="number"
                min={100}
                step={100}
                value={settings.maxGraphSize}
                onChange={(event: ChangeEvent<HTMLInputElement>) =>
                  handleChange('maxGraphSize', Number(event.target.value))
                }
              />
            </div>
            <div className="space-y-2">
              <Label>Realtime collaboration</Label>
              <p className="text-xs text-muted-foreground">
                Coming soon: tune concurrency, presence indicators, and conflict resolution strategies.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default Settings

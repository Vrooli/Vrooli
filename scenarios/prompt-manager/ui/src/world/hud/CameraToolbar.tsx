import { useSyncExternalStore } from 'react'
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp, Crosshair, Home, Minus, Plus, RotateCcw, RotateCw } from 'lucide-react'
import { NAV_VISUALS, type NavigationCommand, type NavigationMode, type NavigationPreferences, type NavigationPreset, type NavigationTool } from '../config/navigation'
import type { NavigationTelemetry } from './navigationState'

interface Props {
  telemetry: NavigationTelemetry
  preferences: NavigationPreferences
  onPreferences: (patch: Partial<NavigationPreferences>) => void
  tool: NavigationTool
  onTool: (tool: NavigationTool) => void
  onMode: (mode: NavigationMode) => void
  onCommand: (command: NavigationCommand) => void
  onPreset: (preset: NavigationPreset) => void
  onHome: () => void
  onFrame: () => void
  canFrame: boolean
  onLock: () => void
  zoomTarget: 'cursor' | 'center'
  onZoomTarget: (value: 'cursor' | 'center') => void
}
const button = 'inline-flex h-8 min-w-8 items-center justify-center rounded border border-border bg-background px-2 text-xs hover:bg-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-500 disabled:opacity-40'
export function CameraToolbar(props: Props) {
  const state = useSyncExternalStore(props.telemetry.subscribe, props.telemetry.read)
  const walking = state.mode !== 'explore'
  const movement = [
    ['left', ArrowLeft, walking ? 'Step left' : 'Pan left'], ['up', ArrowUp, walking ? 'Step forward' : 'Pan up'],
    ['down', ArrowDown, walking ? 'Step back' : 'Pan down'], ['right', ArrowRight, walking ? 'Step right' : 'Pan right'],
    ['orbit-left', RotateCcw, walking ? 'Look left' : 'Orbit left'], ['orbit-right', RotateCw, walking ? 'Look right' : 'Orbit right'],
  ] as const
  return (
    <section aria-label="Camera navigation" className="pointer-events-auto absolute left-3 top-16 z-30 w-60 max-w-[calc(100%-1.5rem)] rounded-xl border border-border bg-background/95 p-2 shadow-md backdrop-blur">
      <div className="mb-2 flex items-center justify-between text-xs"><strong>Camera</strong><span title="Heading clockwise from north">N · {state.heading}°</span></div>
      <label className="flex items-center justify-between gap-2 text-xs">Mode
        <select aria-label="Camera mode" value={state.mode} onChange={e => props.onMode(e.target.value as NavigationMode)} className="h-8 rounded border border-border bg-background px-1">
          <option value="explore">Explore</option><option value="first-person">First person</option><option value="third-person">Third person</option>
        </select>
      </label>
      <div className="mt-2 text-xs"><label className="flex items-center justify-between">Device<select aria-label="Navigation device" value={props.preferences.device} onChange={e => props.onPreferences({ device: e.target.value as NavigationPreferences['device'] })} className="rounded border bg-background p-1"><option value="mouse">Mouse</option><option value="trackpad">Trackpad</option></select></label></div>
      <div className="my-2 flex flex-wrap gap-1" role="group" aria-label="Camera actions">
        <button className={button} aria-label="Home view" title="Restore the full home view" onClick={props.onHome}><Home size={NAV_VISUALS.iconPixels} /></button>
        <button className={button} aria-label="Frame selection" title="Frame selected agent" disabled={!props.canFrame} onClick={props.onFrame}><Crosshair size={NAV_VISUALS.iconPixels} /></button>
        <button className={button} aria-label="Zoom in" disabled={state.mode === 'first-person'} onClick={() => props.onCommand('zoom-in')}><Plus size={NAV_VISUALS.iconPixels} /></button>
        <button className={button} aria-label="Zoom out" disabled={state.mode === 'first-person'} onClick={() => props.onCommand('zoom-out')}><Minus size={NAV_VISUALS.iconPixels} /></button>
        <button className={button} onClick={() => props.onCommand('stop')}>Stop</button>
      </div>
      <div className="flex flex-wrap gap-1" role="group" aria-label={walking ? 'Walking controls' : 'Pan and orbit'}>
        {movement.map(([command, Icon, label]) => <button key={command} className={button} aria-label={label} title={label} onClick={() => props.onCommand(command)}><Icon size={NAV_VISUALS.iconPixels} /></button>)}
      </div>
      {!walking ? <>
        <div className="my-2 flex gap-1" role="group" aria-label="Drag tool">
          {(['orbit', 'pan'] as const).map(tool => <button key={tool} className={`${button} flex-1 ${props.tool === tool ? 'border-sky-500 text-sky-600' : ''}`} aria-pressed={props.tool === tool} onClick={() => props.onTool(tool)}>{tool === 'orbit' ? 'Orbit drag' : 'Pan drag'}</button>)}
        </div>
        <div className="flex gap-1" role="group" aria-label="Camera views">
          {(['top', 'front', 'isometric'] as const).map(view => <button key={view} className={`${button} flex-1 capitalize`} onClick={() => props.onPreset(view)}>{view}</button>)}
        </div>
        <p className="mt-2 text-[11px] text-muted-foreground">{props.preferences.device === 'trackpad' ? 'Two-finger scroll: pan · Pinch: zoom' : 'Wheel: zoom · Right drag: pan'}<br />Click world, then arrows: orbit · WASD: pan.<br />The blue ring marks the orbit pivot.</p>
      </> : <>
        <button className={`${button} mt-2 w-full`} onClick={() => props.onCommand('jump')}>Jump · Space</button>
        <button className={`${button} mt-2 w-full`} onClick={props.onLock} disabled={state.locked}>{state.locked ? 'Mouse captured · Esc releases' : 'Capture mouse to look'}</button>
        <p className="mt-2 text-[11px] text-muted-foreground">Click world: WASD / arrows to walk. Shift: run. Space: jump. Drag to look. Esc releases mouse; Esc again returns to Explore.</p>
        <button className={`${button} mt-2 w-full`} onClick={() => props.onMode('explore')}>Return to Explore</button>
      </>}
      <p role="status" aria-live="polite" className="mt-1 text-xs text-amber-700">{state.message || (state.blocked ? 'Movement blocked by terrain, furniture, or a wall.' : '')}</p>
      <details className="mt-2 border-t border-border pt-2 text-xs">
        <summary className="cursor-pointer">Input settings</summary>
        <div className="mt-2 space-y-2">

          <label className="block">Sensitivity · {props.preferences.sensitivity.toFixed(2)}×<input aria-label="Camera sensitivity" className="w-full" type="range" min="0.25" max="3" step="0.25" value={props.preferences.sensitivity} onChange={e => props.onPreferences({ sensitivity: Number(e.target.value) })} /></label>
          <label className="flex items-center gap-2"><input type="checkbox" checked={props.preferences.invertZoom} onChange={e => props.onPreferences({ invertZoom: e.target.checked })} />Invert zoom</label>
          <label className="flex items-center gap-2"><input type="checkbox" checked={props.preferences.invertLook} onChange={e => props.onPreferences({ invertLook: e.target.checked })} />Invert vertical look</label>
          <label className="flex items-center gap-2"><input type="checkbox" checked={props.preferences.smoothing} onChange={e => props.onPreferences({ smoothing: e.target.checked })} />Gentle drag smoothing</label>
          <label className="flex items-center justify-between">Zoom toward<select aria-label="Navigation zoom target" value={props.zoomTarget} onChange={e => props.onZoomTarget(e.target.value as 'cursor' | 'center')} className="rounded border bg-background p-1"><option value="cursor">Pointer</option><option value="center">Center</option></select></label>
          <p className="text-muted-foreground">Device settings are saved in this browser. Reduced motion disables smoothing.</p>
        </div>
      </details>
    </section>
  )
}

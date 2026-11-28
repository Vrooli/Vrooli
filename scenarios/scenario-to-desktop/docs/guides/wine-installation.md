# Wine for Windows desktop builds

Windows cross-builds require working host Wine. Generated packages use
Electron Builder's host Wine path; the standard artifact is an NSIS `.exe`
installer. Native Windows builds do not require Wine.

The published `wine@1.0.1` bundle lacks PE builtins and cannot run Windows
tools on its own ([upstream toolset documentation](https://www.electron.build/docs/toolsets/)).
Generated packages therefore do not pin it. For an older generated package
that has the pin, Electron Builder 26 supports `USE_SYSTEM_WINE=true`.

Run the supported build pipeline after the host tool is available:

```bash
scenario-to-desktop pipeline run web-console --platforms windows-amd64 --deployment-mode bundled --stages bundle,generate,build --location-mode staging
```

`scenario-to-desktop wine check --json` observes an existing host Wine
installation. A version string alone does not prove that it can execute
Windows tools; probe it with `wine cmd /c ver` using an isolated `WINEPREFIX`.
A portable Wine profile can initialize without hardware-driver services using
`WINEDLLOVERRIDES='winemenubuilder.exe=d;mscoree=d;mshtml=d;winebus.sys=d;wineusb.sys=d;winebth.sys=d'`.
The initialized profile can then use Electron Builder's default overrides.

Host installation and repair belong to the control plane. The scenario does
not install Wine, overwrite wrappers, download shared executables or patch
shared build caches. Windows code signing remains an operator-owned decision;
an unsigned local build does not authorize publication.

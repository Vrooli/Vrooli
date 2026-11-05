"""Choose Later in the actual packaged application's native update dialog."""
import subprocess


def dialog(title, key):
    try:
        found = subprocess.check_output(
            ['xdotool', 'search', '--sync', '--onlyvisible', '--name', '^' + title + '$'],
            text=True, timeout=75).splitlines()
        if len(found) != 1:
            raise RuntimeError(f'Expected one {title} dialog, found {found}')
        subprocess.run(['xdotool', 'windowfocus', '--sync', found[0]], check=True, timeout=5)
        # XTest releases the key on the current focus; a dialog may destroy its
        # X window on keydown, so sending keyup to the old window is invalid.
        subprocess.run(['xdotool', 'key', '--clearmodifiers', key], check=True, timeout=5)
    except Exception:
        subprocess.run(['xwininfo', '-root', '-tree'], timeout=5)
        raise


dialog('Update Available', 'Return')
# Electron treats Escape as cancellation; Later is the cancel button.
dialog('Update Ready', 'Escape')
print('Accepted update notice and chose Later in the native dialog', flush=True)

# ghost-pause

Automatically pauses MPRIS media players when a browser starts playing audio, and resumes them when the browser stops.

## How it works

ghost-pause subscribes to PulseAudio sink-input events via `pactl subscribe`. When a configured browser application starts producing audio (a non-corked sink input appears), it pauses all active MPRIS2-compatible media players via D-Bus. When the browser goes silent, it resumes the previously paused players.

## Requirements

- PulseAudio or PipeWire with `pipewire-pulse`
- `pactl` (provided by `libpulse`)
- D-Bus session bus
- An MPRIS2-compatible media player (Spotify, Rhythmbox, Elisa, MPD with mpdris2, etc.)

## Installation

### AUR

```bash
paru -S ghost-pause-git
```

### Manual

```bash
git clone https://github.com/mentigen/ghost-pause
cd ghost-pause
go build -o ghost-pause .
install -Dm755 ghost-pause ~/.local/bin/ghost-pause
```

To also install the systemd user service:

```bash
install -Dm644 ghost-pause.service ~/.config/systemd/user/ghost-pause.service
```

## Configuration

ghost-pause reads its config from `~/.config/ghost-pause/config.json`.

```bash
mkdir -p ~/.config/ghost-pause
cp config.example.json ~/.config/ghost-pause/config.json
```

If no config file is found, cannot be parsed, or `target_apps` is empty, ghost-pause falls back to these defaults:

```json
{
  "target_apps": ["Zen", "Firefox", "Google Chrome", "Chromium", "Brave"]
}
```

### `target_apps`

List of `application.name` values that PulseAudio reports for your browsers. To find the correct name while a browser is playing audio:

```bash
pactl -f json list sink-inputs | python3 -m json.tool | grep '"application.name"'
```

### `ignore_players`

List of substrings to match against MPRIS2 D-Bus names (`org.mpris.MediaPlayer2.*`). Matched players will never be paused. Matching is case-insensitive.

```json
{
  "ignore_players": ["spotify", "mpd"]
}
```

### `pause_delay_ms`

How many milliseconds to wait after browser audio starts before pausing players. Useful if browser audio triggers and stops briefly (e.g. notification sounds). Default is `0` (pause immediately).

```json
{
  "pause_delay_ms": 1500
}
```

## Usage

### Run manually

```bash
ghost-pause
```

### Flags

| Flag | Description |
|------|-------------|
| `--version` | Print version and exit |
| `--config <path>` | Use a custom config file instead of the default |

```bash
ghost-pause --version
ghost-pause --config /path/to/config.json
```

### Commands

While the daemon is running, you can control it without restarting:

```bash
ghost-pause stop    # disable auto-pausing (players won't be paused)
ghost-pause start   # re-enable auto-pausing
```

### Run as a systemd user service (recommended)

```bash
systemctl --user enable --now ghost-pause
```

Check status and logs:

```bash
systemctl --user status ghost-pause
journalctl --user -u ghost-pause -f
```

## Troubleshooting

**ghost-pause exits unexpectedly**
Check the logs: `journalctl --user -u ghost-pause`. Make sure `pactl` is installed and PulseAudio or PipeWire-pulse is running.

**Browser audio is not detected**
Find the correct application name while the browser is playing something:
```bash
pactl -f json list sink-inputs | python3 -m json.tool | grep '"application.name"'
```
Add the result to `target_apps` in your config file.

**Media player is not pausing**
The player must expose an MPRIS2 interface on the session D-Bus. Verify it is visible:
```bash
dbus-send --session --print-reply \
  --dest=org.freedesktop.DBus /org/freedesktop/DBus \
  org.freedesktop.DBus.ListNames | grep mpris
```

## License

MIT — see [LICENSE](LICENSE).

# WPCtrl
A tool for controlling Windows Phone OS 7.0 and newer. <br>
<br>

<strong>Command support</strong>
| Command | Windows Phone 7.x | Windows Phone 8.x | Windows 10 Mobile |
| :--- | :---: | :---: | :---: |
| `getinfo` | ✅ | ✅ | ✅ |
| `ping` | ❌ | ✅ (8.1+) | ✅ |
| `battery` | ❌ | ✅ (8.1+) | ✅ |
| `reboot` | ❌ | ✅ (8.1+) | ✅ |
| `ls` | ❌ | ✅ | ✅ |
| `copy` | ❌ | ✅ (8.1+) | ✅ |
| `install` | ❌ | ✅ (8.1+) | ✅ |
| `list-apps` | ❌ | ❌ | ✅ |
| `uninstall` | ❌ | ❌ | ✅ |
| `launch` | ❌ | ❌ | ✅ |
| `terminate` | ❌ | ❌ | ✅ |
| `list-processes` | ❌ | ❌ | ✅ |
| `kill` | ❌ | ❌ | ✅ |
| `screenshot` | ❌ | ❌ | ✅ |
<br>
<h1>Usage</h1>
<br>
<strong>Requirements</strong>:
<li>A PC with Windows 10 or newer</li>
<li>A Microsoft/Nokia Windows Phone with Windows Phone OS 7.0 or newer</li>
<hr>
1. Download all files <br>
2. Run `go build -o wpctrl.exe` in the current directory <br>
3. Plug in your Windows Phone device <br>
4. Run `wpctrl <command>` in the current directory <br>
If it errors, you must use the commands above matched with your WP version <br>

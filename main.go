// main.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: wpctrl <command> [arguments]")
		fmt.Println("Commands: ping, getinfo, install, list-apps, uninstall, launch, terminate, list-processes, kill, battery, ls, screenshot, reboot, copy")
		os.Exit(1)
	}

	cmd := os.Args[1]
	client := NewClient()

	// Universal device presence validation across WDP and Legacy USB/PnP stacks
	isWDPOnline := checkWDP(client)
	isWP7 := false
	isWP8 := false

	if !isWDPOnline {
		isWP7 = detectWP7Device()
		if !isWP7 {
			isWP8 = detectWP8Device()
		}

		if !isWP7 && !isWP8 {
			fmt.Println("Error: No Windows Phone device detected. Please ensure the device is plugged in via USB and unlocked.")
			os.Exit(1)
		}
	}

	if isWP7 {
		switch cmd {
		case "getinfo":
			client.HandleWP7GetInfo()
			return
		default:
			fmt.Printf("Error: Command '%s' is not supported on Windows Phone 7.x (only 'getinfo' is available).\n", cmd)
			os.Exit(1)
		}
	}

	if isWP8 && !isWDPOnline {
		switch cmd {
		case "getinfo":
			client.HandleWP8GetInfo()
			return
		case "ls":
			loc := "\\"
			if len(os.Args) >= 3 {
				loc = os.Args[2]
			}
			client.HandleWP8Ls(loc)
			return
		case "copy":
			if len(os.Args) < 4 {
				fmt.Println("Error: Missing arguments for copy.")
				fmt.Println("Usage (PC to WP8): wpctrl copy local_file.txt Documents\\file.txt")
				os.Exit(1)
			}
			src := os.Args[2]
			dest := os.Args[3]
			client.HandleWP8CopyPC(src, dest)
			return
		default:
			fmt.Printf("Error: Command '%s' requires WDP (WP8.1/W10M) or is not supported on WP8.0 MTP fallback.\n", cmd)
			os.Exit(1)
		}
	}

	// Standard WDP flow for WP8.1 / W10M
	client.VerifyDeviceConnectionAndVersion(cmd)

	switch cmd {
	case "ping":
		client.HandlePing()
	case "getinfo":
		client.HandleGetInfo()
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing <appx-path> argument.")
			os.Exit(1)
		}
		client.HandleInstall(os.Args[2])
	case "list-apps":
		client.HandleListApps()
	case "uninstall":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing <package-full-name> argument.")
			os.Exit(1)
		}
		client.HandleUninstall(os.Args[2])
	case "launch":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing <aumid> argument.")
			os.Exit(1)
		}
		client.HandleLaunch(os.Args[2])
	case "terminate":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing <aumid/family-name> argument.")
			os.Exit(1)
		}
		client.HandleTerminate(os.Args[2])
	case "list-processes":
		client.HandleListProcesses()
	case "kill":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing <pid> argument.")
			os.Exit(1)
		}
		client.HandleKill(os.Args[2])
	case "battery":
		client.HandleBattery()
	case "ls":
		loc := "\\"
		if len(os.Args) >= 3 {
			loc = os.Args[2]
		}
		client.HandleLs(loc)
	case "screenshot":
		out := "screenshot.png"
		if len(os.Args) >= 3 {
			out = os.Args[2]
		}
		client.HandleScreenshot(out)
	case "reboot":
		client.HandleReboot()
	case "copy":
		if len(os.Args) < 4 {
			fmt.Println("Error: Missing arguments for copy.")
			fmt.Println("Usage (PC to Phone): wpctrl copy local_file.txt \\Documents\\file.txt")
			fmt.Println("Usage (Phone to PC): wpctrl copy \\Documents\\file.txt local_file.txt")
			os.Exit(1)
		}
		src := os.Args[2]
		dest := os.Args[3]
		if stringsHasPrefix(src, "\\") || stringsHasPrefix(src, "/") {
			client.HandleCopyMobile(src, dest)
		} else {
			client.HandleCopyPC(src, dest)
		}
	default:
		fmt.Printf("Error: Unknown command '%s'\n", cmd)
		os.Exit(1)
	}
}

func checkWDP(c *Client) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(c.BaseURL + "/api/control/deviceinfo")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func detectWP7Device() bool {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", 
		"Get-PnpDevice -Present | Where-Object {$_.HardwareID -like '*Zune*' -or $_.FriendlyName -like '*Trophy*' -or $_.FriendlyName -like '*Focus*'} | Select-Object -ExpandProperty FriendlyName")
	out, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return true
	}
	return false
}

func detectWP8Device() bool {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", 
		"Get-PnpDevice -Present | Where-Object {$_.Class -eq 'WPD' -or $_.HardwareID -like '*MSFT_WP*'} | Select-Object -ExpandProperty FriendlyName")
	out, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return true
	}
	return false
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

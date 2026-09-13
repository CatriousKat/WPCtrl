// wp7.go
package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type WP7Client struct{}

type WP7DeviceInfo struct {
	DeviceName   string `json:"DeviceName"`
	Platform     string `json:"Platform"`
	FormFactor   string `json:"FormFactor"`
	Architecture string `json:"Architecture"`
	OsVersion    string `json:"OsVersion"`
}

func (c *Client) HandleWP7GetInfo() {
	fmt.Println("[*] Inspecting Windows Phone 7.x device via host-side USB/WPD Zune-era enumeration...")

	deviceName := "Unknown WP7 Device"
	platform := "Windows Phone 7 (Unknown VID/PID)"

	// Query connected USB devices via PowerShell to find Zune/WP7 signatures
	cmd := exec.Command("powershell", "-NoProfile", "-Command", 
		"Get-PnpDevice -Present | Where-Object {$_.Class -eq 'WPD' -or $_.HardwareID -like '*Zune*'} | Select-Object FriendlyName, InstanceId | ConvertTo-Json")
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		strOut := string(out)
		if strings.Contains(strOut, "Lumia") || strings.Contains(strOut, "HTC") || strings.Contains(strOut, "Samsung") {
			deviceName = "Zune/WP7 Device (Detected via USB)"
		}
		if strings.Contains(strOut, "VID_045E") {
			platform = "Windows Phone 7 (Microsoft Zune/WP7 Transport)"
		} else {
			platform = "Windows Phone 7 (Standard USB Bridge)"
		}
	}

	info := WP7DeviceInfo{
		DeviceName:   deviceName,
		Platform:     platform,
		FormFactor:   "Phone",
		Architecture: "ARM",
		OsVersion:    "Unknown",
	}

	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting device info: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}